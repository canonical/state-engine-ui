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
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"

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
		"/api/v1/changes/abc/tasks/xyz/event",
	} {
		resp, err := http.Post(addr+path, "application/json", nil)
		c.Assert(err, IsNil)
		resp.Body.Close()
		c.Assert(resp.StatusCode, Equals, http.StatusMethodNotAllowed, Commentf("path %s", path))
	}
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

func (s *taskDebugSuite) TestSSEPerTaskUnderChangeNotFound(c *C) {
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

	for _, path := range []string{
		"/api/v1/changes/nonexistent/tasks/abc/event",
		"/api/v1/changes/" + chg.ID() + "/tasks/nonexistent/event",
		"/api/v1/changes/" + chg.ID() + "/tasks/" + t2.ID() + "/event",
	} {
		resp, err := http.Get("http://" + mgr.Addr() + path)
		c.Assert(err, IsNil)
		resp.Body.Close()
		c.Assert(resp.StatusCode, Equals, http.StatusNotFound, Commentf("path %s", path))
	}
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

func (s *taskDebugSuite) TestSSEPerTaskUnderChangeEventFilter(c *C) {
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
	t2 := st.NewTask("mount-snap", "mount snap foo")
	chg.AddTask(t2)
	st.Unlock()

	resp, err := http.Get("http://" + mgr.Addr() + "/api/v1/changes/" + chg.ID() + "/tasks/" + t1.ID() + "/event")
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	reader := bufio.NewReader(resp.Body)
	ev, err := readSSEEvent(reader)
	c.Assert(err, IsNil)
	c.Assert(ev.Event, Equals, "snapshot")

	time.Sleep(10 * time.Millisecond)

	st.Lock()
	t2.SetStatus(state.DoneStatus)
	t1.SetStatus(state.DoneStatus)
	st.Unlock()

	ev, err = readNextNonKeepalive(reader)
	c.Assert(err, IsNil)
	c.Assert(ev.Event, Equals, "task-status-changed")
	d := parseEventData(c, ev.Data)
	c.Assert(d["trigger_id"], Equals, t1.ID())
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
