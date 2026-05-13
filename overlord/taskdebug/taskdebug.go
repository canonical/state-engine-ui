// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2026 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

// Package taskdebug exposes a lightweight HTTP server for debugging the
// state engine. It shows the status of tasks and changes, including custom
// data associated with tasks. It is intended for use by a debugging user
// interface. The server is disabled by default and can be enabled by setting
// the SNAPD_TASK_DEBUG_ADDR environment variable.
package taskdebug

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/overlord/state"
)

// Manager is a state manager that runs a debug HTTP server.
type Manager struct {
	state    *state.State
	addr     string
	listener net.Listener
	server   *http.Server
	mu       sync.Mutex
	started  bool
}

// NewManager returns a new Manager.
func NewManager(st *state.State) *Manager {
	return &Manager{state: st, addr: os.Getenv("SNAPD_TASK_DEBUG_ADDR")}
}

// Ensure starts the debug HTTP server if it has not already been started.
func (m *Manager) Ensure() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started || m.addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", m.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", m.handleTaskDetail)
	mux.HandleFunc("/api/v1/changes", m.handleChanges)
	mux.HandleFunc("/api/v1/changes/", m.handleChangeDetail)
	m.server = &http.Server{Addr: m.addr, Handler: mux}
	lis, err := net.Listen("tcp", m.addr)
	if err != nil {
		logger.Noticef("cannot start task debug server on %s: %v", m.addr, err)
		return nil
	}
	m.listener = lis
	m.started = true
	logger.Noticef("task debug server listening on %s", lis.Addr().String())
	go func() {
		if err := m.server.Serve(lis); err != nil && err != http.ErrServerClosed {
			logger.Noticef("task debug server error: %v", err)
		}
	}()
	return nil
}

// Addr returns the actual listener address of the debug HTTP server, or an empty string if it has not started.
func (m *Manager) Addr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listener == nil {
		return ""
	}
	return m.listener.Addr().String()
}

// Stop shuts down the debug HTTP server.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.server.Shutdown(ctx); err != nil {
		logger.Noticef("task debug server shutdown error: %v", err)
	}
	m.started = false
}

type taskInfo struct {
	ID          string                      `json:"id"`
	Kind        string                      `json:"kind"`
	Summary     string                      `json:"summary"`
	Status      string                      `json:"status"`
	ChangeID    string                      `json:"change_id,omitempty"`
	Progress    progressInfo                `json:"progress"`
	Data        map[string]*json.RawMessage `json:"data,omitempty"`
	WaitTasks   []string                    `json:"wait_tasks,omitempty"`
	HaltTasks   []string                    `json:"halt_tasks,omitempty"`
	Lanes       []int                       `json:"lanes,omitempty"`
	Log         []string                    `json:"log,omitempty"`
	SpawnTime   time.Time                   `json:"spawn_time,omitzero"`
	ReadyTime   *time.Time                  `json:"ready_time,omitempty"`
	AtTime      *time.Time                  `json:"at_time,omitempty"`
	DoingTime   time.Duration               `json:"doing_time,omitempty"`
	UndoingTime time.Duration               `json:"undoing_time,omitempty"`
	Clean       bool                        `json:"clean,omitempty"`
}

type progressInfo struct {
	Label string `json:"label"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
}

type changeInfo struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Summary   string     `json:"summary"`
	Status    string     `json:"status"`
	Ready     bool       `json:"ready"`
	Err       string     `json:"err,omitempty"`
	SpawnTime time.Time  `json:"spawn_time,omitzero"`
	ReadyTime *time.Time `json:"ready_time,omitempty"`
	TaskIDs   []string   `json:"task_ids,omitempty"`
}

func (m *Manager) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := m.state
	st.Lock()
	var infos []taskInfo
	for _, chg := range st.Changes() {
		for _, t := range chg.Tasks() {
			infos = append(infos, taskToInfo(t))
		}
	}
	st.Unlock()
	writeJSON(w, infos)
}

func (m *Manager) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	prefix := "/api/v1/tasks/"
	if len(path) < len(prefix)+1 || path[:len(prefix)] != prefix {
		http.NotFound(w, r)
		return
	}
	id := path[len(prefix):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	st := m.state
	st.Lock()
	t := st.Task(id)
	if t == nil {
		st.Unlock()
		http.NotFound(w, r)
		return
	}
	info := taskToInfo(t)
	st.Unlock()
	writeJSON(w, info)
}

func (m *Manager) handleChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := m.state
	st.Lock()
	var infos []changeInfo
	for _, chg := range st.Changes() {
		infos = append(infos, changeToInfo(chg))
	}
	st.Unlock()
	writeJSON(w, infos)
}

func (m *Manager) handleChangeDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	prefix := "/api/v1/changes/"
	if len(path) < len(prefix)+1 || path[:len(prefix)] != prefix {
		http.NotFound(w, r)
		return
	}
	remainder := path[len(prefix):]
	parts := strings.SplitN(remainder, "/", 2)
	chgID := parts[0]
	if chgID == "" {
		http.NotFound(w, r)
		return
	}

	st := m.state
	st.Lock()
	chg := st.Change(chgID)
	if chg == nil {
		st.Unlock()
		http.NotFound(w, r)
		return
	}

	// /api/v1/changes/{id}/tasks
	if len(parts) == 2 && parts[1] == "tasks" {
		var infos []taskInfo
		for _, t := range chg.Tasks() {
			infos = append(infos, taskToInfo(t))
		}
		st.Unlock()
		writeJSON(w, infos)
		return
	}

	// /api/v1/changes/{id}
	info := changeToInfo(chg)
	st.Unlock()
	writeJSON(w, info)
}

func taskToInfo(t *state.Task) taskInfo {
	label, done, total := t.Progress()
	info := taskInfo{
		ID:          t.ID(),
		Kind:        t.Kind(),
		Summary:     t.Summary(),
		Status:      t.Status().String(),
		ChangeID:    t.Change().ID(),
		Progress:    progressInfo{Label: label, Done: done, Total: total},
		Data:        t.AllData(),
		Log:         t.Log(),
		SpawnTime:   t.SpawnTime(),
		DoingTime:   t.DoingTime(),
		UndoingTime: t.UndoingTime(),
		Clean:       t.IsClean(),
	}
	for _, wt := range t.WaitTasks() {
		info.WaitTasks = append(info.WaitTasks, wt.ID())
	}
	for _, ht := range t.HaltTasks() {
		info.HaltTasks = append(info.HaltTasks, ht.ID())
	}
	info.Lanes = t.Lanes()
	rt := t.ReadyTime()
	if !rt.IsZero() {
		info.ReadyTime = &rt
	}
	at := t.AtTime()
	if !at.IsZero() {
		info.AtTime = &at
	}
	return info
}

func changeToInfo(chg *state.Change) changeInfo {
	status := chg.Status()
	info := changeInfo{
		ID:        chg.ID(),
		Kind:      chg.Kind(),
		Summary:   chg.Summary(),
		Status:    status.String(),
		Ready:     status.Ready(),
		SpawnTime: chg.SpawnTime(),
	}
	if err := chg.Err(); err != nil {
		info.Err = err.Error()
	}
	rt := chg.ReadyTime()
	if !rt.IsZero() {
		info.ReadyTime = &rt
	}
	for _, t := range chg.Tasks() {
		info.TaskIDs = append(info.TaskIDs, t.ID())
	}
	return info
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		logger.Noticef("cannot encode task debug response: %v", err)
	}
}
