package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// assertTrackingCmdResult is a helper to reduce duplication in tracking command tests
func assertTrackingCmdResult(t *testing.T, cmd tea.Cmd, expectCmd bool, expectLocked bool, expectMsg string, changesLocked bool, messageValue string) {
	t.Helper()
	if expectCmd {
		assert.NotNil(t, cmd)
	} else {
		assert.Nil(t, cmd)
	}
	assert.Equal(t, expectLocked, changesLocked)
	if expectMsg != "" {
		assert.Equal(t, expectMsg, messageValue)
	}
}

// T-021: Task and tracking flow tests

func TestHandleRequestToCreateTask(t *testing.T) {
	testCases := []struct {
		name          string
		setupModel    func() Model
		expectedView  stateView
		expectedMsg   string
		expectedCtx   taskMgmtContext
		expectedField taskInputField
	}{
		{
			name: "success - creates task input view",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				return m
			},
			expectedView:  taskInputView,
			expectedCtx:   taskCreateCxt,
			expectedField: summaryField,
		},
		{
			name: "filtered list shows error message",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				m.activeTasksList.SetFilterText("filter")
				return m
			},
			expectedView: taskListView,
			expectedMsg:  removeFilterMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			m.handleRequestToCreateTask()

			assert.Equal(t, tt.expectedView, m.activeView)
			assert.Equal(t, tt.expectedCtx, m.taskMgmtContext)
			assert.Equal(t, tt.expectedField, m.taskInputFocussedField)
			if tt.expectedMsg != "" {
				assert.Equal(t, tt.expectedMsg, m.message.value)
			}
		})
	}
}

func TestHandleRequestToUpdateTask(t *testing.T) {
	testCases := []struct {
		name          string
		setupModel    func() Model
		expectedView  stateView
		expectedMsg   string
		expectedCtx   taskMgmtContext
		expectedValue string
	}{
		{
			name: "success - creates update view with task summary",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				task := createTestTask(1, "Task to update", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectedView:  taskInputView,
			expectedCtx:   taskUpdateCxt,
			expectedValue: "Task to update",
		},
		{
			name: "filtered list shows error message",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				m.activeTasksList.SetFilterText("filter")
				return m
			},
			expectedView: taskListView,
			expectedMsg:  removeFilterMsg,
		},
		{
			name: "no task selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskListView
				return m
			},
			expectedView: taskListView,
			expectedMsg:  genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			m.handleRequestToUpdateTask()

			assert.Equal(t, tt.expectedView, m.activeView)
			assert.Equal(t, tt.expectedCtx, m.taskMgmtContext)
			assert.Equal(t, tt.expectedValue, m.taskInputs[summaryField].Value())
			if tt.expectedMsg != "" {
				assert.Equal(t, tt.expectedMsg, m.message.value)
			}
		})
	}
}

func TestHandleRequestToStopTracking(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView
	m.trackingActive = true
	m.activeTLBeginTS = m.timeProvider.Now().Add(-time.Hour)

	// WHEN
	m.handleRequestToStopTracking()

	// THEN
	assert.Equal(t, finishActiveTLView, m.activeView)
	assert.Equal(t, entryComment, m.trackingFocussedField)
	assert.NotEmpty(t, m.tLInputs[entryBeginTS].Value())
	assert.NotEmpty(t, m.tLInputs[entryEndTS].Value())
}

func TestGetCmdToStartTracking(t *testing.T) {
	testCases := []struct {
		name         string
		setupModel   func() Model
		expectCmd    bool
		expectMsg    string
		expectLocked bool
	}{
		{
			name: "success - starts tracking",
			setupModel: func() Model {
				m := createTestModel()
				task := createTestTask(1, "Task to track", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd:    true,
			expectLocked: true,
		},
		{
			name: "no task selected - shows error",
			setupModel: func() Model {
				m := createTestModel()
				return m
			},
			expectCmd: false,
			expectMsg: genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.getCmdToStartTracking()
			assertTrackingCmdResult(t, cmd, tt.expectCmd, tt.expectLocked, tt.expectMsg, m.changesLocked, m.message.value)
		})
	}
}

func TestGetCmdToFinishActiveTLWithoutComment(t *testing.T) {
	testCases := []struct {
		name           string
		setupModel     func() Model
		expectCmd      bool
		expectMsg      string
		expectMsgKind  userMsgKind
		expectEndTSSet bool
	}{
		{
			name: "success - valid duration returns command",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 1
				// Begin 2 hours ago - valid duration
				m.activeTLBeginTS = m.timeProvider.Now().Add(-2 * time.Hour)
				return m
			},
			expectCmd:      true,
			expectEndTSSet: true,
		},
		{
			name: "duration too short - shows info message",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 1
				// Begin 1 second ago - too short
				m.activeTLBeginTS = m.timeProvider.Now().Add(-1 * time.Second)
				return m
			},
			expectCmd:      false,
			expectMsg:      "Task log duration is too short to save; press <ctrl+x> if you want to discard it",
			expectMsgKind:  userMsgInfo,
			expectEndTSSet: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			originalEndTS := m.activeTLEndTS

			cmd := m.getCmdToFinishActiveTL()

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
				assert.Equal(t, tt.expectMsgKind, m.message.kind)
			}
			if tt.expectEndTSSet {
				assert.NotEqual(t, originalEndTS, m.activeTLEndTS)
			}
		})
	}
}

func TestGetCmdToQuickSwitchTracking(t *testing.T) {
	testCases := []struct {
		name         string
		setupModel   func() Model
		expectCmd    bool
		expectMsg    string
		expectLocked bool
	}{
		{
			name: "success - quick switch to different task",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 1
				task := createTestTask(2, "Task to switch to", true, false, m.timeProvider)
				m.taskMap[2] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd: true,
		},
		{
			name: "same task - no command",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 1
				task := createTestTask(1, "Same task", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd: false,
		},
		{
			name: "not tracking - starts tracking new task",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = false
				task := createTestTask(1, "Task to track", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd:    true,
			expectLocked: true,
		},
		{
			name: "no task selected - shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				return m
			},
			expectCmd: false,
			expectMsg: genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.getCmdToQuickSwitchTracking()
			assertTrackingCmdResult(t, cmd, tt.expectCmd, tt.expectLocked, tt.expectMsg, m.changesLocked, m.message.value)
		})
	}
}

// T-022: Task log operation tests

func TestGetCmdToDeactivateTask(t *testing.T) {
	testCases := []struct {
		name       string
		setupModel func() Model
		expectCmd  bool
		expectMsg  string
	}{
		{
			name: "success - deactivates task",
			setupModel: func() Model {
				m := createTestModel()
				task := createTestTask(1, "Task to deactivate", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd: true,
		},
		{
			name: "filtered list shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeTasksList.SetFilterText("filter")
				return m
			},
			expectCmd: false,
			expectMsg: removeFilterMsg,
		},
		{
			name: "tracking active shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				task := createTestTask(1, "Tracked task", true, true, m.timeProvider)
				m.taskMap[1] = task
				m.activeTasksList.SetItems([]list.Item{task})
				m.activeTasksList.Select(0)
				return m
			},
			expectCmd: false,
			expectMsg: "Cannot deactivate a task being tracked; stop tracking and try again.",
		},
		{
			name: "no task selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				return m
			},
			expectCmd: false,
			expectMsg: msgCouldntSelectATask,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.getCmdToDeactivateTask()

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
		})
	}
}

func TestGetCmdToActivateDeactivatedTask(t *testing.T) {
	testCases := []struct {
		name       string
		setupModel func() Model
		expectCmd  bool
		expectMsg  string
	}{
		{
			name: "success - activates task",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = inactiveTaskListView
				task := createTestTask(1, "Task to activate", false, false, m.timeProvider)
				m.inactiveTasksList.SetItems([]list.Item{task})
				m.inactiveTasksList.Select(0)
				return m
			},
			expectCmd: true,
		},
		{
			name: "filtered list shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = inactiveTaskListView
				m.inactiveTasksList.SetFilterText("filter")
				return m
			},
			expectCmd: false,
			expectMsg: removeFilterMsg,
		},
		{
			name: "no task selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = inactiveTaskListView
				return m
			},
			expectCmd: false,
			expectMsg: genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.getCmdToActivateDeactivatedTask()

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
		})
	}
}

func TestGetCmdToDeleteTL(t *testing.T) {
	testCases := []struct {
		name       string
		setupModel func() Model
		expectCmd  bool
		expectMsg  string
	}{
		{
			name: "success - deletes task log",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				entry := createTestTaskLogEntry(1, 1, "Task", m.timeProvider)
				// Use value type, not pointer, matching the list.Item interface implementation
				m.taskLogList.SetItems([]list.Item{*entry})
				m.taskLogList.Select(0)
				return m
			},
			expectCmd: true,
		},
		{
			name: "no entry selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				return m
			},
			expectCmd: false,
			expectMsg: "Couldn't delete task log entry",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.getCmdToDeleteTL()

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
		})
	}
}

func TestHandleRequestToMoveTaskLog(t *testing.T) {
	testCases := []struct {
		name         string
		setupModel   func() Model
		expectedView stateView
		expectMsg    string
		expectItems  int
	}{
		{
			name: "success - shows move view with targets",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				// Create task log entry
				entry := createTestTaskLogEntry(1, 1, "Source Task", m.timeProvider)
				m.taskLogList.SetItems([]list.Item{*entry})
				m.taskLogList.Select(0)
				// Create target tasks (excluding source)
				task2 := createTestTask(2, "Target Task 1", true, false, m.timeProvider)
				task3 := createTestTask(3, "Target Task 2", true, false, m.timeProvider)
				m.taskMap[1] = createTestTask(1, "Source Task", true, false, m.timeProvider)
				m.taskMap[2] = task2
				m.taskMap[3] = task3
				m.activeTasksList.SetItems([]list.Item{task2, task3})
				return m
			},
			expectedView: moveTaskLogView,
			expectItems:  2,
		},
		{
			name: "filtered list shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				m.taskLogList.SetFilterText("filter")
				return m
			},
			expectedView: taskLogView,
			expectMsg:    removeFilterMsg,
			expectItems:  0,
		},
		{
			name: "no log entry selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				return m
			},
			expectedView: taskLogView,
			expectMsg:    genericErrorMsg,
			expectItems:  0,
		},
		{
			name: "no other tasks shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				entry := createTestTaskLogEntry(1, 1, "Only Task", m.timeProvider)
				m.taskLogList.SetItems([]list.Item{*entry})
				m.taskLogList.Select(0)
				// Only source task in active list
				task1 := createTestTask(1, "Only Task", true, false, m.timeProvider)
				m.taskMap[1] = task1
				m.activeTasksList.SetItems([]list.Item{task1})
				return m
			},
			expectedView: taskLogView,
			expectMsg:    "No other active tasks to move this log to",
			expectItems:  0,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			m.handleRequestToMoveTaskLog()

			assert.Equal(t, tt.expectedView, m.activeView)
			assert.Len(t, m.targetTasksList.Items(), tt.expectItems)
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
		})
	}
}

func TestHandleTargetTaskSelection(t *testing.T) {
	testCases := []struct {
		name       string
		setupModel func() Model
		expectCmd  bool
		expectMsg  string
	}{
		{
			name: "success - returns command to move task log",
			setupModel: func() Model {
				m := createTestModel()
				task := createTestTask(2, "Target Task", true, false, m.timeProvider)
				m.targetTasksList.SetItems([]list.Item{task})
				m.targetTasksList.Select(0)
				m.moveTLID = 1
				m.moveOldTaskID = 1
				m.moveSecsSpent = 3600
				return m
			},
			expectCmd: true,
		},
		{
			name: "no task selected shows error",
			setupModel: func() Model {
				m := createTestModel()
				return m
			},
			expectCmd: false,
			expectMsg: genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			cmd := m.handleTargetTaskSelection()

			if tt.expectCmd {
				assert.NotNil(t, cmd)
			} else {
				assert.Nil(t, cmd)
			}
			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
		})
	}
}

func TestGoToActiveTask(t *testing.T) {
	testCases := []struct {
		name           string
		setupModel     func() Model
		expectSelected bool
		expectMsg      string
		expectFiltered bool
	}{
		{
			name: "success - navigates to active task",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 1
				task := createTestTask(1, "Active task", true, false, m.timeProvider)
				m.taskMap[1] = task
				m.taskIndexMap[1] = 2
				m.activeTasksList.SetItems([]list.Item{
					createTestTask(2, "Other task", true, false, m.timeProvider),
					createTestTask(3, "Another task", true, false, m.timeProvider),
					task,
				})
				return m
			},
			expectSelected: true,
			expectFiltered: false,
		},
		{
			name: "resets filter and navigates",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 2
				// Set up multiple tasks so we can verify selection change
				task1 := createTestTask(1, "First task", true, false, m.timeProvider)
				task2 := createTestTask(2, "Active task", true, false, m.timeProvider)
				m.taskMap[2] = task2
				m.taskIndexMap[2] = 1 // Active task is at index 1
				m.activeTasksList.SetItems([]list.Item{task1, task2})
				m.activeTasksList.Select(0) // Start at index 0
				// Simulate a filter being applied
				m.activeTasksList.SetFilterText("filtered")
				return m
			},
			expectSelected: true,
			expectFiltered: false, // Filter should be reset
		},
		{
			name: "not in task list view - returns early",
			setupModel: func() Model {
				m := createTestModel()
				m.activeView = taskLogView
				m.trackingActive = true
				m.activeTaskID = 1
				return m
			},
			expectSelected: false,
			expectMsg:      "",
		},
		{
			name: "not tracking - shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = false
				return m
			},
			expectSelected: false,
			expectMsg:      "Nothing is being tracked right now",
		},
		{
			name: "task not in map - shows error",
			setupModel: func() Model {
				m := createTestModel()
				m.trackingActive = true
				m.activeTaskID = 999 // ID that doesn't exist in taskMap
				return m
			},
			expectSelected: false,
			expectMsg:      genericErrorMsg,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setupModel()
			originalIndex := m.activeTasksList.Index()

			m.goToActiveTask()

			if tt.expectMsg != "" {
				assert.Equal(t, tt.expectMsg, m.message.value)
			}
			if tt.expectSelected {
				// Should have selected the active task
				assert.NotEqual(t, originalIndex, m.activeTasksList.Index())
			}
			if !tt.expectFiltered {
				// Filter should be reset
				assert.Equal(t, list.Unfiltered, m.activeTasksList.FilterState())
			}
		})
	}
}
