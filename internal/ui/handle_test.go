package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/dhth/hours/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleCopyTaskSummary(t *testing.T) {
	testCases := []struct {
		name            string
		setupModel      func() Model
		expectedMsg     string
		expectedMsgKind userMsgKind
	}{
		{
			name: "success - active task list",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				task := createTestTask(1, "Test task summary", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectedMsg:     "Copied to clipboard",
			expectedMsgKind: userMsgInfo,
		},
		{
			name: "success - inactive task list",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = inactiveTaskListView
				task := createTestTask(1, "Archived task", false, false, m.timeProvider)
				m.inactiveTasksList.SetItems([]list.Item{task})
				m.inactiveTasksList.Select(0)
				return m
			},
			expectedMsg:     "Copied to clipboard",
			expectedMsgKind: userMsgInfo,
		},
		{
			name: "no task selected - active task list",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				return m
			},
			expectedMsg:     "No task selected",
			expectedMsgKind: userMsgErr,
		},
		{
			name: "no task selected - inactive task list",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = inactiveTaskListView
				return m
			},
			expectedMsg:     "No task selected",
			expectedMsgKind: userMsgErr,
		},
		{
			name: "wrong view - task log view",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				return m
			},
			expectedMsg:     "",
			expectedMsgKind: userMsgInfo,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			m.handleCopyTaskSummary()

			assert.Equal(t, tt.expectedMsg, m.message.value)
			if tt.expectedMsg != "" {
				assert.Equal(t, tt.expectedMsgKind, m.message.kind)
			}
		})
	}
}

// T-082: handle.go async message handler tests

func TestHandleTasksFetchedMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := tasksFetchedMsg{err: errors.New("db failure"), active: true}

		cmd := m.handleTasksFetchedMsg(msg)

		assert.Nil(t, cmd)
		assert.Equal(t, userMsgErr, m.message.kind)
		assert.Contains(t, m.message.value, "db failure")
	})

	t.Run("active tasks populate taskMap and taskIndexMap", func(t *testing.T) {
		m := createTestModel()
		tasks := []types.Task{
			{ID: 1, Summary: "task one", Active: true, UpdatedAt: referenceTime},
			{ID: 2, Summary: "task two", Active: true, UpdatedAt: referenceTime},
		}
		msg := tasksFetchedMsg{tasks: tasks, active: true}

		cmd := m.handleTasksFetchedMsg(msg)

		// returns fetchActiveTask cmd
		require.NotNil(t, cmd)
		assert.True(t, m.tasksFetched)
		assert.Len(t, m.taskMap, 2)
		assert.Contains(t, m.taskMap, 1)
		assert.Contains(t, m.taskMap, 2)
		assert.Equal(t, 0, m.taskIndexMap[1])
		assert.Equal(t, 1, m.taskIndexMap[2])
		assert.Len(t, m.activeTasksList.Items(), 2)
	})

	t.Run("inactive tasks populate inactiveTasksList", func(t *testing.T) {
		m := createTestModel()
		tasks := []types.Task{
			{ID: 3, Summary: "archived task", Active: false, UpdatedAt: referenceTime},
		}
		msg := tasksFetchedMsg{tasks: tasks, active: false}

		cmd := m.handleTasksFetchedMsg(msg)

		assert.Nil(t, cmd)
		assert.Len(t, m.inactiveTasksList.Items(), 1)
		// taskMap should remain untouched
		assert.Empty(t, m.taskMap)
	})
}

func TestHandleManualTLInsertedMsg(t *testing.T) {
	t.Run("error sets error message and returns nil", func(t *testing.T) {
		m := createTestModel()
		msg := manualTLInsertedMsg{err: errors.New("insert failed"), taskID: 1}

		cmds := m.handleManualTLInsertedMsg(msg)

		assert.Nil(t, cmds)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("success with known task returns updateTaskRep and fetchTLS cmds", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "my task", true, false, m.timeProvider)
		m.taskMap[1] = task
		msg := manualTLInsertedMsg{taskID: 1}

		cmds := m.handleManualTLInsertedMsg(msg)

		// updateTaskRep + fetchTLS = 2 cmds
		require.Len(t, cmds, 2)
	})

	t.Run("success with unknown task returns only fetchTLS cmd", func(t *testing.T) {
		m := createTestModel()
		// taskMap is empty — taskID 99 is not known
		msg := manualTLInsertedMsg{taskID: 99}

		cmds := m.handleManualTLInsertedMsg(msg)

		require.Len(t, cmds, 1)
	})
}

func TestHandleSavedTLEditedMsg(t *testing.T) {
	t.Run("error sets error message and returns nil", func(t *testing.T) {
		m := createTestModel()
		msg := savedTLEditedMsg{err: errors.New("edit failed"), taskID: 1, tlID: 1}

		cmds := m.handleSavedTLEditedMsg(msg)

		assert.Nil(t, cmds)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("success with known task returns updateTaskRep and fetchTLS cmds", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "my task", true, false, m.timeProvider)
		m.taskMap[1] = task
		tlID := 5
		msg := savedTLEditedMsg{taskID: 1, tlID: tlID}

		cmds := m.handleSavedTLEditedMsg(msg)

		require.Len(t, cmds, 2)
	})

	t.Run("success with unknown task returns only fetchTLS cmd", func(t *testing.T) {
		m := createTestModel()
		msg := savedTLEditedMsg{taskID: 99, tlID: 1}

		cmds := m.handleSavedTLEditedMsg(msg)

		require.Len(t, cmds, 1)
	})
}

func TestHandleTLSFetchedMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := tLsFetchedMsg{err: errors.New("fetch failed")}

		m.handleTLSFetchedMsg(msg)

		assert.Equal(t, userMsgErr, m.message.kind)
		assert.Contains(t, m.message.value, "fetch failed")
	})

	t.Run("success populates task log list", func(t *testing.T) {
		m := createTestModel()
		entries := []types.TaskLogEntry{
			*createTestTaskLogEntry(1, 1, "task one", m.timeProvider),
			*createTestTaskLogEntry(2, 1, "task one", m.timeProvider),
		}
		msg := tLsFetchedMsg{entries: entries}

		m.handleTLSFetchedMsg(msg)

		assert.Len(t, m.taskLogList.Items(), 2)
	})

	t.Run("focuses on specified tlID when present", func(t *testing.T) {
		m := createTestModel()
		entry1 := *createTestTaskLogEntry(10, 1, "task", m.timeProvider)
		entry2 := *createTestTaskLogEntry(20, 1, "task", m.timeProvider)
		focusID := 20
		msg := tLsFetchedMsg{entries: []types.TaskLogEntry{entry1, entry2}, tlIDToFocusOn: &focusID}

		m.handleTLSFetchedMsg(msg)

		assert.Equal(t, 1, m.taskLogList.Index())
	})
}

func TestHandleActiveTaskFetchedMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := activeTaskFetchedMsg{err: errors.New("db error")}

		m.handleActiveTaskFetchedMsg(msg)

		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("noneActive sets trackingFinished", func(t *testing.T) {
		m := createTestModel()
		msg := activeTaskFetchedMsg{noneActive: true}

		m.handleActiveTaskFetchedMsg(msg)

		assert.Equal(t, trackingFinished, m.lastTrackingChange)
		assert.False(t, m.trackingActive)
	})

	t.Run("active task sets tracking state", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "tracked task", true, false, m.timeProvider)
		m.taskMap[1] = task
		m.taskIndexMap[1] = 0
		m.activeTasksList.SetItems([]list.Item{task})

		beginTS := referenceTime.Add(-time.Hour)
		comment := "working on it"
		msg := activeTaskFetchedMsg{
			activeTask: types.ActiveTaskDetails{
				TaskID:            1,
				CurrentLogBeginTS: beginTS,
				CurrentLogComment: &comment,
			},
		}

		m.handleActiveTaskFetchedMsg(msg)

		assert.True(t, m.trackingActive)
		assert.Equal(t, 1, m.activeTaskID)
		assert.True(t, m.activeTLBeginTS.Equal(beginTS))
		require.NotNil(t, m.activeTLComment)
		assert.Equal(t, comment, *m.activeTLComment)
		assert.Equal(t, trackingStarted, m.lastTrackingChange)
		assert.True(t, task.TrackingActive)
	})
}

func TestHandleTrackingToggledMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := trackingToggledMsg{err: errors.New("toggle failed"), taskID: 1}

		cmds := m.handleTrackingToggledMsg(msg)

		assert.Nil(t, cmds)
		assert.Equal(t, userMsgErr, m.message.kind)
		assert.False(t, m.trackingActive)
	})

	t.Run("unknown task ID on success sets error", func(t *testing.T) {
		m := createTestModel()
		// taskMap empty, taskID 99 unknown
		msg := trackingToggledMsg{taskID: 99, finished: false}

		cmds := m.handleTrackingToggledMsg(msg)

		assert.Nil(t, cmds)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("finished=true clears tracking state", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "task", true, true, m.timeProvider)
		m.taskMap[1] = task
		m.trackingActive = true
		m.activeTaskID = 1
		msg := trackingToggledMsg{taskID: 1, finished: true}

		cmds := m.handleTrackingToggledMsg(msg)

		assert.False(t, m.trackingActive)
		assert.Equal(t, -1, m.activeTaskID)
		assert.Equal(t, trackingFinished, m.lastTrackingChange)
		assert.False(t, task.TrackingActive)
		assert.False(t, m.changesLocked)
		// updateTaskRep + fetchTLS = 2 cmds
		require.Len(t, cmds, 2)
	})

	t.Run("finished=false sets tracking started", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "task", true, false, m.timeProvider)
		m.taskMap[1] = task
		msg := trackingToggledMsg{taskID: 1, finished: false}

		cmds := m.handleTrackingToggledMsg(msg)

		assert.True(t, m.trackingActive)
		assert.Equal(t, 1, m.activeTaskID)
		assert.Equal(t, trackingStarted, m.lastTrackingChange)
		assert.True(t, task.TrackingActive)
		assert.Nil(t, cmds)
	})
}

func TestHandleActiveTLSwitchedMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := activeTLSwitchedMsg{err: errors.New("switch failed")}

		cmd := m.handleActiveTLSwitchedMsg(msg)

		assert.Nil(t, cmd)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("unknown last active task sets error", func(t *testing.T) {
		m := createTestModel()
		// taskMap empty
		msg := activeTLSwitchedMsg{lastActiveTaskID: 1, currentlyActiveTaskID: 2}

		cmd := m.handleActiveTLSwitchedMsg(msg)

		assert.Nil(t, cmd)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("unknown currently active task sets error", func(t *testing.T) {
		m := createTestModel()
		task1 := createTestTask(1, "old task", true, true, m.timeProvider)
		m.taskMap[1] = task1
		// task 2 not in taskMap
		msg := activeTLSwitchedMsg{lastActiveTaskID: 1, currentlyActiveTaskID: 2}

		cmd := m.handleActiveTLSwitchedMsg(msg)

		assert.Nil(t, cmd)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("success updates tracking state and returns fetchTLS cmd", func(t *testing.T) {
		m := createTestModel()
		task1 := createTestTask(1, "old task", true, true, m.timeProvider)
		task2 := createTestTask(2, "new task", true, false, m.timeProvider)
		m.taskMap[1] = task1
		m.taskMap[2] = task2

		newBeginTS := referenceTime.Add(-30 * time.Minute)
		msg := activeTLSwitchedMsg{
			lastActiveTaskID:      1,
			currentlyActiveTaskID: 2,
			ts:                    newBeginTS,
		}

		cmd := m.handleActiveTLSwitchedMsg(msg)

		require.NotNil(t, cmd) // fetchTLS cmd
		assert.False(t, task1.TrackingActive)
		assert.True(t, task2.TrackingActive)
		assert.Equal(t, 2, m.activeTaskID)
		assert.True(t, m.activeTLBeginTS.Equal(newBeginTS))
		assert.Nil(t, m.activeTLComment)
	})
}

func TestHandleTLDeleted(t *testing.T) {
	t.Run("error sets error message and returns nil", func(t *testing.T) {
		m := createTestModel()
		entry := createTestTaskLogEntry(1, 1, "task", m.timeProvider)
		msg := tLDeletedMsg{err: errors.New("delete failed"), entry: entry}

		cmds := m.handleTLDeleted(msg)

		assert.Nil(t, cmds)
		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("success with known task returns updateTaskRep and fetchTLS cmds", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "my task", true, false, m.timeProvider)
		m.taskMap[1] = task
		entry := createTestTaskLogEntry(1, 1, "my task", m.timeProvider)
		msg := tLDeletedMsg{entry: entry}

		cmds := m.handleTLDeleted(msg)

		require.Len(t, cmds, 2)
	})

	t.Run("success with unknown task returns only fetchTLS cmd", func(t *testing.T) {
		m := createTestModel()
		entry := createTestTaskLogEntry(1, 99, "unknown task", m.timeProvider)
		msg := tLDeletedMsg{entry: entry}

		cmds := m.handleTLDeleted(msg)

		require.Len(t, cmds, 1)
	})
}

func TestHandleActiveTLDeletedMsg(t *testing.T) {
	t.Run("error sets error message", func(t *testing.T) {
		m := createTestModel()
		msg := activeTaskLogDeletedMsg{err: errors.New("delete active failed")}

		m.handleActiveTLDeletedMsg(msg)

		assert.Equal(t, userMsgErr, m.message.kind)
		assert.Contains(t, m.message.value, "delete active failed")
	})

	t.Run("unknown activeTaskID sets error", func(t *testing.T) {
		m := createTestModel()
		m.activeTaskID = 99 // not in taskMap
		msg := activeTaskLogDeletedMsg{}

		m.handleActiveTLDeletedMsg(msg)

		assert.Equal(t, userMsgErr, m.message.kind)
	})

	t.Run("success clears tracking state", func(t *testing.T) {
		m := createTestModel()
		task := createTestTask(1, "tracked task", true, true, m.timeProvider)
		m.taskMap[1] = task
		m.activeTaskID = 1
		m.trackingActive = true

		msg := activeTaskLogDeletedMsg{}

		m.handleActiveTLDeletedMsg(msg)

		assert.False(t, task.TrackingActive)
		assert.False(t, m.trackingActive)
		assert.Equal(t, -1, m.activeTaskID)
		assert.Equal(t, trackingFinished, m.lastTrackingChange)
		assert.Nil(t, m.activeTLComment)
	})
}
