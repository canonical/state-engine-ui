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
//
// The server provides a Server-Sent Events (SSE) endpoint that streams
// real-time updates for changes and tasks under a change. Every SSE event
// carries the complete list of changes and tasks so that clients can
// replace their entire state on each event without merging deltas.
package taskdebug

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/overlord/state"
)

type stepState struct {
	allowNext string
	continued bool
}

type Manager struct {
	state    *state.State
	addr     string
	listener net.Listener
	server   *http.Server
	mu       sync.Mutex
	started  bool

	hub               *eventHub
	taskHandlerID     int
	changeHandlerID   int
	changeTaskAddedID int
	changeRemovedID   int

	keepaliveDone chan struct{}

	runner   *state.TaskRunner
	stepping map[string]*stepState
}

func NewManager(st *state.State) *Manager {
	return &Manager{
		state:    st,
		addr:     os.Getenv("SNAPD_TASK_DEBUG_ADDR"),
		hub:      newEventHub(100),
		stepping: make(map[string]*stepState),
	}
}

func (m *Manager) SetRunner(r *state.TaskRunner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runner = r
}

func (m *Manager) Ensure() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started || m.addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", m.handleTasks)
	mux.HandleFunc("/api/v1/tasks/", m.handleTasksPrefix)
	mux.HandleFunc("/api/v1/changes", m.handleChanges)
	mux.HandleFunc("/api/v1/changes/", m.handleChangesPrefix)
	m.server = &http.Server{Addr: m.addr, Handler: mux}
	lis, err := net.Listen("tcp", m.addr)
	if err != nil {
		logger.Noticef("cannot start task debug server on %s: %v", m.addr, err)
		return nil
	}
	m.listener = lis
	m.started = true
	logger.Noticef("task debug server listening on %s", lis.Addr().String())

	m.state.Lock()
	m.taskHandlerID = m.state.AddTaskStatusChangedHandler(func(t *state.Task, old, new state.Status) bool {
		snap := m.buildSnapshot()
		snap.Trigger = "task-status-changed"
		snap.TriggerID = t.ID()
		snap.ChangeID = t.Change().ID()
		snap.OldStatus = visibleStatus(old)
		snap.NewStatus = visibleStatus(new)
		m.hub.publish(sseEvent{Event: "task-status-changed", Data: snap})
		return false
	})
	m.changeHandlerID = m.state.AddChangeStatusChangedHandler(func(chg *state.Change, old, new state.Status) {
		snap := m.buildSnapshot()
		snap.Trigger = "change-status-changed"
		snap.TriggerID = chg.ID()
		snap.OldStatus = visibleStatus(old)
		snap.NewStatus = visibleStatus(new)
		m.hub.publish(sseEvent{Event: "change-status-changed", Data: snap})
	})
	m.changeTaskAddedID = m.state.AddChangeTaskAddedHandler(func(chg *state.Change, t *state.Task) {
		snap := m.buildSnapshot()
		snap.Trigger = "change-task-added"
		snap.TriggerID = chg.ID()
		m.hub.publish(sseEvent{Event: "change-task-added", Data: snap})
	})
	m.changeRemovedID = m.state.AddChangeRemovedHandler(func(chg *state.Change) {
		snap := m.buildSnapshot()
		snap.Trigger = "change-removed"
		snap.TriggerID = chg.ID()
		m.hub.publish(sseEvent{Event: "change-removed", Data: snap})
	})
	m.state.Unlock()

	if m.runner != nil {
		m.runner.AddBlocked(func(t *state.Task, running []*state.Task) bool {
			m.mu.Lock()
			defer m.mu.Unlock()
			ss, ok := m.stepping[t.Change().ID()]
			if !ok {
				return true
			}
			if ss.continued {
				return false
			}
			if ss.allowNext == t.ID() {
				ss.allowNext = ""
				return false
			}
			return true
		})
	}

	m.keepaliveDone = make(chan struct{})
	go m.keepalive()

	go func() {
		if err := m.server.Serve(lis); err != nil && err != http.ErrServerClosed {
			logger.Noticef("task debug server error: %v", err)
		}
	}()
	return nil
}

func (m *Manager) keepalive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.hub.publish(sseEvent{Event: "keepalive"})
		case <-m.keepaliveDone:
			return
		}
	}
}

func (m *Manager) Addr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listener == nil {
		return ""
	}
	return m.listener.Addr().String()
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return
	}
	close(m.keepaliveDone)
	m.state.Lock()
	m.state.RemoveTaskStatusChangedHandler(m.taskHandlerID)
	m.state.RemoveChangeStatusChangedHandler(m.changeHandlerID)
	m.state.RemoveChangeTaskAddedHandler(m.changeTaskAddedID)
	m.state.RemoveChangeRemovedHandler(m.changeRemovedID)
	m.state.Unlock()
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

type sseEvent struct {
	Event string
	Data  any
}

type sseEventData struct {
	Trigger   string       `json:"trigger"`
	TriggerID string       `json:"trigger_id,omitempty"`
	ChangeID  string       `json:"change_id,omitempty"`
	OldStatus string       `json:"old_status,omitempty"`
	NewStatus string       `json:"new_status,omitempty"`
	Changes   []changeInfo `json:"changes"`
	Tasks     []taskInfo   `json:"tasks"`
}

func visibleStatus(s state.Status) string {
	if s == state.DefaultStatus {
		return state.DoStatus.String()
	}
	return s.String()
}

func (m *Manager) buildSnapshot() sseEventData {
	st := m.state
	var snap sseEventData
	for _, chg := range st.Changes() {
		snap.Changes = append(snap.Changes, changeToInfo(chg))
	}
	for _, chg := range st.Changes() {
		for _, t := range chg.Tasks() {
			snap.Tasks = append(snap.Tasks, taskToInfo(t))
		}
	}
	return snap
}

type eventHub struct {
	mu      sync.Mutex
	subs    []*subscriber
	bufSize int
}

type subscriber struct {
	ch     chan sseEvent
	done   <-chan struct{}
	filter func(sseEvent) bool
}

func newEventHub(bufSize int) *eventHub {
	return &eventHub{bufSize: bufSize}
}

func (h *eventHub) subscribe(done <-chan struct{}, filter func(sseEvent) bool) *subscriber {
	sub := &subscriber{
		ch:     make(chan sseEvent, h.bufSize),
		done:   done,
		filter: filter,
	}
	h.mu.Lock()
	h.subs = append(h.subs, sub)
	h.mu.Unlock()
	return sub
}

func (h *eventHub) unsubscribe(sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, s := range h.subs {
		if s == sub {
			h.subs = append(h.subs[:i], h.subs[i+1:]...)
			return
		}
	}
}

func (h *eventHub) publish(ev sseEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sub := range h.subs {
		if sub.filter != nil && !sub.filter(ev) {
			continue
		}
		select {
		case sub.ch <- ev:
		default:
		}
	}
}

func (m *Manager) handleTasksPrefix(w http.ResponseWriter, r *http.Request) {
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
	m.handleTaskDetailByID(w, r, id)
}

func (m *Manager) handleChangesPrefix(w http.ResponseWriter, r *http.Request) {
	remainder := strings.TrimPrefix(r.URL.Path, "/api/v1/changes/")
	if remainder == "" {
		http.NotFound(w, r)
		return
	}
	segments := strings.Split(remainder, "/")
	chgID := segments[0]

	switch {
	case len(segments) == 1:
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.handleChangeDetailByID(w, r, chgID)
	case len(segments) == 2 && segments[1] == "event":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		st := m.state
		st.Lock()
		if chg := st.Change(chgID); chg == nil {
			st.Unlock()
			http.NotFound(w, r)
			return
		}
		st.Unlock()
		m.serveSSE(w, r, changeEventFilter(chgID))
	case len(segments) == 2 && segments[1] == "tasks":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.handleChangeTasksByID(w, r, chgID)
	case len(segments) == 2 && segments[1] == "action":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.handleAction(w, r, chgID)
	case len(segments) == 3 && segments[1] == "tasks":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		m.handleTaskDetailByID(w, r, segments[2])
	default:
		http.NotFound(w, r)
	}
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

func (m *Manager) handleTaskDetailByID(w http.ResponseWriter, r *http.Request, id string) {
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

type changeEntry struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Ready  bool   `json:"ready"`
}

func (m *Manager) handleChanges(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var filter map[string]bool
	if vals, ok := r.URL.Query()["status"]; ok && len(vals) > 0 {
		filter = make(map[string]bool)
		for _, v := range strings.Split(vals[0], ",") {
			filter[v] = true
		}
	}
	st := m.state
	st.Lock()
	var entries []changeEntry
	for _, chg := range st.Changes() {
		status := visibleStatus(chg.Status())
		if filter != nil && !filter[status] {
			continue
		}
		entries = append(entries, changeEntry{
			ID:     chg.ID(),
			Status: status,
			Ready:  chg.Status().Ready(),
		})
	}
	st.Unlock()
	writeJSON(w, entries)
}

func (m *Manager) handleChangeDetailByID(w http.ResponseWriter, r *http.Request, chgID string) {
	st := m.state
	st.Lock()
	chg := st.Change(chgID)
	if chg == nil {
		st.Unlock()
		http.NotFound(w, r)
		return
	}
	info := changeToInfo(chg)
	st.Unlock()
	writeJSON(w, info)
}

func (m *Manager) handleChangeTasksByID(w http.ResponseWriter, r *http.Request, chgID string) {
	st := m.state
	st.Lock()
	chg := st.Change(chgID)
	if chg == nil {
		st.Unlock()
		http.NotFound(w, r)
		return
	}
	var infos []taskInfo
	for _, t := range chg.Tasks() {
		infos = append(infos, taskToInfo(t))
	}
	st.Unlock()
	writeJSON(w, infos)
}

func (m *Manager) handleAction(w http.ResponseWriter, r *http.Request, chgID string) {
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "cannot decode request body", http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "step":
		m.handleStep(w, r, chgID)
	case "continue":
		m.handleContinue(w, r, chgID)
	case "pause":
		m.handlePause(w, r, chgID)
	default:
		writeJSONWithStatus(w, http.StatusBadRequest, map[string]string{"error": "unknown action: " + req.Action})
	}
}

func (m *Manager) handleStep(w http.ResponseWriter, r *http.Request, chgID string) {
	st := m.state
	st.Lock()
	chg := st.Change(chgID)
	if chg == nil {
		st.Unlock()
		http.NotFound(w, r)
		return
	}
	nextTask := findNextRunnable(chg)
	st.Unlock()

	if nextTask == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	m.mu.Lock()
	if _, ok := m.stepping[chgID]; !ok {
		m.stepping[chgID] = &stepState{}
	}
	ss := m.stepping[chgID]
	ss.allowNext = nextTask
	ss.continued = false
	m.mu.Unlock()

	st.EnsureBefore(0)
	writeJSON(w, map[string]string{"allowed_task": nextTask})
}

func (m *Manager) handleContinue(w http.ResponseWriter, r *http.Request, chgID string) {
	m.mu.Lock()
	if ss, ok := m.stepping[chgID]; ok {
		ss.continued = true
		ss.allowNext = ""
	} else {
		m.stepping[chgID] = &stepState{continued: true}
	}
	m.mu.Unlock()

	m.state.EnsureBefore(0)
	w.WriteHeader(http.StatusOK)
}

func (m *Manager) handlePause(w http.ResponseWriter, r *http.Request, chgID string) {
	m.mu.Lock()
	if ss, ok := m.stepping[chgID]; ok {
		ss.continued = false
		ss.allowNext = ""
	} else {
		m.stepping[chgID] = &stepState{}
	}
	m.mu.Unlock()

	m.state.EnsureBefore(0)
	w.WriteHeader(http.StatusOK)
}

func findNextRunnable(chg *state.Change) string {
	for _, t := range chg.Tasks() {
		status := t.Status()
		if status.Ready() || status == state.WaitStatus {
			continue
		}
		if mustWaitTask(t) {
			continue
		}
		return t.ID()
	}
	return ""
}

func mustWaitTask(t *state.Task) bool {
	switch t.Status() {
	case state.DoStatus:
		for _, wt := range t.WaitTasks() {
			if wt.Status() != state.DoneStatus {
				return true
			}
		}
	case state.UndoStatus:
		for _, ht := range t.HaltTasks() {
			if !ht.Status().Ready() {
				return true
			}
		}
	}
	return false
}

func (m *Manager) serveSSE(w http.ResponseWriter, r *http.Request, filter func(sseEvent) bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := m.hub.subscribe(r.Context().Done(), filter)
	defer m.hub.unsubscribe(sub)

	st := m.state
	st.Lock()
	snap := m.buildSnapshot()
	st.Unlock()
	snap.Trigger = "snapshot"
	writeSSEEvent(w, flusher, sseEvent{Event: "snapshot", Data: snap})

	for {
		select {
		case ev, ok := <-sub.ch:
			if !ok {
				return
			}
			writeSSEEvent(w, flusher, ev)
		case <-r.Context().Done():
			return
		}
	}
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, ev sseEvent) {
	fmt.Fprintf(w, "event: %s\n", ev.Event)
	if ev.Data != nil {
		data, err := json.Marshal(ev.Data)
		if err != nil {
			logger.Noticef("cannot marshal SSE event data: %v", err)
			return
		}
		fmt.Fprintf(w, "data: %s\n", data)
	}
	fmt.Fprintf(w, "\n")
	flusher.Flush()
}

func changeEventFilter(chgID string) func(sseEvent) bool {
	return func(ev sseEvent) bool {
		switch ev.Event {
		case "snapshot", "keepalive":
			return true
		default:
			d, ok := ev.Data.(sseEventData)
			if !ok {
				return true
			}
			if d.ChangeID != "" {
				return d.ChangeID == chgID
			}
			return d.TriggerID == chgID
		}
	}
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

func writeJSONWithStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		logger.Noticef("cannot encode task debug response: %v", err)
	}
}
