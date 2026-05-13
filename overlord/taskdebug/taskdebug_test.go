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
	"encoding/json"
	"net/http"
	"os"
	"testing"

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

	var changes []map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	c.Assert(dec.Decode(&changes), IsNil)
	c.Assert(len(changes), Equals, 1)
	c.Assert(changes[0]["id"], Equals, chg.ID())
	c.Assert(changes[0]["kind"], Equals, "install")
	c.Assert(changes[0]["ready"], Equals, false)
	c.Assert(changes[0]["status"], Equals, "Do")

	taskIDs := changes[0]["task_ids"].([]interface{})
	c.Assert(len(taskIDs), Equals, 1)
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
	for _, path := range []string{"/api/v1/tasks", "/api/v1/changes", "/api/v1/changes/abc", "/api/v1/tasks/abc"} {
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
	// after stop, the server should no longer respond
	_, err := http.Get("http://" + mgr.Addr() + "/api/v1/tasks")
	c.Assert(err, NotNil)
}
