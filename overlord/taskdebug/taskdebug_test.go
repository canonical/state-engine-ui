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

package taskdebug_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
	"gopkg.in/tomb.v2"

	"github.com/snapcore/snapd/overlord/state"
	"github.com/snapcore/snapd/overlord/taskdebug"
)

func Test(t *testing.T) { TestingT(t) }

type taskDebugSuite struct{}

var _ = Suite(&taskDebugSuite{})

func (s *taskDebugSuite) TestDisabledByDefault(c *C) {
	st := state.New(nil)
	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	c.Assert(mgr.Addr(), Equals, "")
}

func (s *taskDebugSuite) TestServerStarts(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()
	c.Assert(mgr.Addr(), Not(Equals), "")
}

func (s *taskDebugSuite) TestTasksEndpoint(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	t1.Set("snap-setup", map[string]string{"name": "foo"})
	chg.AddTask(t1)
	t2 := st.NewTask("mount-snap", "mount snap foo")
	chg.AddTask(t2)
	t2.WaitFor(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/tasks")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	c.Assert(resp.Header.Get("Content-Type"), Equals, "application/json")

	var tasks []map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&tasks), IsNil)
	c.Assert(len(tasks), Equals, 2)

	ids := []string{tasks[0]["id"].(string), tasks[1]["id"].(string)}
	c.Assert(ids, DeepEquals, []string{t1.ID(), t2.ID()})

	c.Assert(tasks[0]["kind"], Equals, "download-snap")
	c.Assert(tasks[0]["status"], Equals, "Do")
	c.Assert(tasks[0]["change_id"], Equals, chg.ID())
	data := tasks[0]["data"].(map[string]interface{})
	c.Assert(data["snap-setup"], DeepEquals, map[string]interface{}{"name": "foo"})

	waitTasks := tasks[1]["wait_tasks"].([]interface{})
	c.Assert(len(waitTasks), Equals, 1)
	c.Assert(waitTasks[0], Equals, t1.ID())
}

func (s *taskDebugSuite) TestTaskDetail(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/tasks/" + t1.ID())
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	var task map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&task), IsNil)
	c.Assert(task["id"], Equals, t1.ID())
	c.Assert(task["kind"], Equals, "download-snap")
}

func (s *taskDebugSuite) TestTaskDetailNotFound(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/tasks/nonexistent")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *taskDebugSuite) TestChangesEndpoint(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	var ids []string
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&ids), IsNil)
	c.Assert(ids, DeepEquals, []string{chg.ID()})
}

func (s *taskDebugSuite) TestChangeDetail(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID())
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	var change map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&change), IsNil)
	c.Assert(change["id"], Equals, chg.ID())
}

func (s *taskDebugSuite) TestChangeDetailNotFound(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/nonexistent")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *taskDebugSuite) TestChangeTasks(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	t2 := st.NewTask("mount-snap", "mount snap foo")
	chg.AddTask(t2)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/tasks")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	var tasks []map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&tasks), IsNil)
	c.Assert(len(tasks), Equals, 2)
	c.Assert(tasks[0]["id"], Equals, t1.ID())
}

func (s *taskDebugSuite) TestMethodNotAllowed(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	addr := "http://" + mgr.Addr()
	for _, path := range []string{
		"/api/v1/tasks",
		"/api/v1/changes",
		"/api/v1/changes/abc",
		"/api/v1/tasks/abc",
		"/api/v1/changes/abc/event",
	} {
		resp, err := http.Post(addr+path, "application/json", nil)
		c.Assert(err, IsNil)
		resp.Body.Close()
		c.Assert(resp.StatusCode, Equals, http.StatusMethodNotAllowed, Commentf("path %s", path))
	}

	resp, err := http.Get(addr + "/api/v1/changes/abc/action")
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusMethodNotAllowed)
}

func (s *taskDebugSuite) TestStop(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	c.Assert(mgr.Addr(), Not(Equals), "")
	mgr.Stop()
	_, err := http.Get("http://" + mgr.Addr() + "/api/v1/tasks")
	c.Assert(err, NotNil)
}

func (s *taskDebugSuite) TestSSEPerChangeFilteredSnapshot(c *C) {
	st := state.New(nil)
	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	chg2 := st.NewChange("remove", "remove bar")
	t2 := st.NewTask("remove-snap", "remove snap bar")
	chg2.AddTask(t2)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/event")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	ev, err := readSSEEvent(reader)
	c.Assert(err, IsNil)
	c.Assert(ev.Event, Equals, "snapshot")

	d := parseEventData(c, ev.Data)
	changes := d["changes"].([]interface{})
	c.Assert(len(changes), Equals, 2)
	tasks := d["tasks"].([]interface{})
	c.Assert(len(tasks), Equals, 2)
}

func (s *taskDebugSuite) TestSSEPerChangeNotFound(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/nonexistent/event")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *taskDebugSuite) TestSSETaskStatusChanged(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/event")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	ev, err := readSSEEvent(reader)
	c.Assert(err, IsNil)
	c.Assert(ev.Event, Equals, "snapshot")

	time.Sleep(10 * time.Millisecond)

	st.Lock()
	t1.SetStatus(state.DoneStatus)
	st.Unlock()

	for _, expectedEvent := range []string{"change-status-changed", "task-status-changed"} {
		ev, err = readNextNonKeepalive(reader)
		c.Assert(err, IsNil)
		c.Assert(ev.Event, Equals, expectedEvent)

		d := parseEventData(c, ev.Data)
		changes := d["changes"].([]interface{})
		tasks := d["tasks"].([]interface{})
		c.Assert(len(changes) > 0, Equals, true, Commentf("event %s should have changes", expectedEvent))
		c.Assert(len(tasks) > 0, Equals, true, Commentf("event %s should have tasks", expectedEvent))
	}

	c.Assert(ev.Event, Equals, "task-status-changed")
	d := parseEventData(c, ev.Data)
	c.Assert(d["trigger"], Equals, "task-status-changed")
	c.Assert(d["trigger_id"], Equals, t1.ID())
	c.Assert(d["change_id"], Equals, chg.ID())
	c.Assert(d["old_status"], Equals, "Do")
	c.Assert(d["new_status"], Equals, "Done")
}

func (s *taskDebugSuite) TestSSEPerChangeEventFilter(c *C) {
	st := state.New(nil)
	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/event")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	ev, err := readSSEEvent(reader)
	c.Assert(err, IsNil)
	c.Assert(ev.Event, Equals, "snapshot")

	time.Sleep(10 * time.Millisecond)

	st.Lock()
	chg2 := st.NewChange("remove", "remove bar")
	t2 := st.NewTask("remove-snap", "remove snap bar")
	chg2.AddTask(t2)
	t1.SetStatus(state.DoneStatus)
	st.Unlock()

	var taskEv parsedSSEEvent
	for {
		ev, err = readNextNonKeepalive(reader)
		c.Assert(err, IsNil)
		d := parseEventData(c, ev.Data)
		if d["trigger"] == "task-status-changed" && d["trigger_id"] == t1.ID() {
			taskEv = ev
			break
		}
	}
	c.Assert(taskEv.Event, Equals, "task-status-changed")
	d := parseEventData(c, taskEv.Data)
	c.Assert(d["trigger_id"], Equals, t1.ID())
	c.Assert(d["change_id"], Equals, chg.ID())
}

func (s *taskDebugSuite) TestTasksBlockedAtStartup(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)
	done := make(chan struct{})
	runner.AddHandler("download-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done)
		return nil
	}, nil)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	runner.Ensure()

	select {
	case <-done:
		c.Fatal("task should not have run — tasks must be blocked at startup")
	case <-time.After(200 * time.Millisecond):
	}

	st.Lock()
	c.Assert(t1.Status(), Equals, state.DoStatus)
	st.Unlock()
}

func (s *taskDebugSuite) TestStepChangeRunsOneTask(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	runner.AddHandler("download-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done1)
		return nil
	}, nil)
	runner.AddHandler("mount-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done2)
		return nil
	}, nil)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	t2 := st.NewTask("mount-snap", "mount snap foo")
	t2.WaitFor(t1)
	chg.AddTask(t2)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"step"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	var result map[string]string
	c.Assert(json.NewDecoder(resp.Body).Decode(&result), IsNil)
	c.Assert(result["allowed_task"], Equals, t1.ID())

	runner.Ensure()

	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		c.Fatal("first task didn't run after step")
	}

	select {
	case <-done2:
		c.Fatal("second task should not have run yet")
	case <-time.After(200 * time.Millisecond):
	}

	body = bytes.NewReader([]byte(`{"action":"step"}`))
	resp2, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp2.Body.Close()

	runner.Ensure()

	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		c.Fatal("second task didn't run after second step")
	}
}

func (s *taskDebugSuite) TestContinueUnblocksAll(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	runner.AddHandler("download-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done1)
		return nil
	}, nil)
	runner.AddHandler("mount-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done2)
		return nil
	}, nil)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	t2 := st.NewTask("mount-snap", "mount snap foo")
	t2.WaitFor(t1)
	chg.AddTask(t2)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"continue"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	runner.Ensure()

	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		c.Fatal("first task didn't run after continue")
	}

	runner.Ensure()

	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		c.Fatal("second task didn't run after continue")
	}
}

func (s *taskDebugSuite) TestPauseReblocksAfterContinue(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)
	done1 := make(chan struct{})
	done2 := make(chan struct{})
	runner.AddHandler("download-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done1)
		return nil
	}, nil)
	runner.AddHandler("mount-snap", func(t *state.Task, _ *tomb.Tomb) error {
		close(done2)
		return nil
	}, nil)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	t2 := st.NewTask("mount-snap", "mount snap foo")
	t2.WaitFor(t1)
	chg.AddTask(t2)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"continue"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	runner.Ensure()

	select {
	case <-done1:
	case <-time.After(2 * time.Second):
		c.Fatal("first task didn't run")
	}

	body = bytes.NewReader([]byte(`{"action":"pause"}`))
	resp, err = http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	runner.Ensure()

	select {
	case <-done2:
		c.Fatal("second task should not run after pause")
	case <-time.After(200 * time.Millisecond):
	}

	body = bytes.NewReader([]byte(`{"action":"continue"}`))
	resp, err = http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp.Body.Close()

	runner.Ensure()

	select {
	case <-done2:
	case <-time.After(2 * time.Second):
		c.Fatal("second task didn't run after re-continue")
	}
}

func (s *taskDebugSuite) TestStepChangeNotFound(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"step"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/nonexistent/action", "application/json", body)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
}

func (s *taskDebugSuite) TestStepNoRunnableTasks(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	t1.SetStatus(state.DoneStatus)
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"step"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusNoContent)
}

func (s *taskDebugSuite) TestActionMethodNotAllowed(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/action")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusMethodNotAllowed)
}

func (s *taskDebugSuite) TestUnknownAction(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"foobar"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusBadRequest)

	var result map[string]string
	c.Assert(json.NewDecoder(resp.Body).Decode(&result), IsNil)
	c.Assert(result["error"], Equals, "unknown action: foobar")
}

func (s *taskDebugSuite) TestContinueChange(c *C) {
	st := state.New(nil)
	runner := state.NewTaskRunner(st)

	st.Lock()
	chg := st.NewChange("install", "install foo")
	t1 := st.NewTask("download-snap", "download snap foo")
	chg.AddTask(t1)
	st.Unlock()

	os.Setenv("SNAPD_TASK_DEBUG_ADDR", "127.0.0.1:0")
	defer os.Unsetenv("SNAPD_TASK_DEBUG_ADDR")

	mgr := taskdebug.NewManager(st)
	mgr.SetRunner(runner)
	c.Assert(mgr.Ensure(), IsNil)
	defer mgr.Stop()

	body := bytes.NewReader([]byte(`{"action":"step"}`))
	resp, err := http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	body = bytes.NewReader([]byte(`{"action":"continue"}`))
	resp, err = http.Post("http://"+mgr.Addr()+"/api/v1/changes/"+chg.ID()+"/action", "application/json", body)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
}

func parseEventData(c *C, raw string) map[string]interface{} {
	var d map[string]interface{}
	c.Assert(json.Unmarshal([]byte(raw), &d), IsNil)
	return d
}

type parsedSSEEvent struct {
	Event string
	Data  string
}

func readSSEEvent(reader *bufio.Reader) (parsedSSEEvent, error) {
	var ev parsedSSEEvent
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return ev, err
		}
		line = strings.TrimRight(line, "\n\r")
		if strings.HasPrefix(line, "event: ") {
			ev.Event = line[len("event: "):]
		} else if strings.HasPrefix(line, "data: ") {
			ev.Data = line[len("data: "):]
		} else if line == "" {
			return ev, nil
		}
	}
}

func readNextNonKeepalive(reader *bufio.Reader) (parsedSSEEvent, error) {
	for {
		ev, err := readSSEEvent(reader)
		if err != nil {
			return ev, err
		}
		if ev.Event != "keepalive" {
			return ev, nil
		}
	}
}
