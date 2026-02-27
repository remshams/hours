package ui

import (
	"database/sql"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/persistence"
	"github.com/dhth/hours/internal/types"
	"github.com/dhth/hours/internal/ui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// journeyTestHarness provides utilities for creating deterministic TUI journey tests
// as specified in T-040 of the testing plan.
type journeyTestHarness struct {
	t            *testing.T
	db           *sql.DB
	model        Model
	timeProvider types.TestTimeProvider
}

// newJourneyTestHarness creates a new test harness with:
// - In-memory SQLite database with seeded test data
// - Model initialized with fixed time provider
// - Reference time set to a deterministic value
func newJourneyTestHarness(t *testing.T) *journeyTestHarness {
	t.Helper()

	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Initialize database schema
	err = persistence.InitDB(db)
	require.NoError(t, err)

	// Use a fixed reference time for deterministic tests
	referenceTime := time.Date(2025, 8, 16, 9, 0, 0, 0, time.UTC)
	timeProvider := types.TestTimeProvider{FixedTime: referenceTime}

	// Initialize model with test time provider
	defaultTheme := theme.Default()
	style := NewStyle(defaultTheme)
	m := InitialModel(db, style, timeProvider, false, logFramesConfig{})

	// Set up minimum window size for proper initialization
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded,
		Height: minHeightNeeded,
	}
	m.handleWindowResizing(msg)

	return &journeyTestHarness{
		t:            t,
		db:           db,
		model:        m,
		timeProvider: timeProvider,
	}
}

// cleanup closes the database connection
func (h *journeyTestHarness) cleanup() {
	if h.db != nil {
		h.db.Close()
	}
}

// insertTask creates a new task in the database and returns its ID
func (h *journeyTestHarness) insertTask(summary string, active bool) int {
	id, err := persistence.InsertTask(h.db, summary)
	require.NoError(h.t, err)

	if !active {
		_, err = h.db.Exec("UPDATE task SET active = ? WHERE id = ?", active, id)
		require.NoError(h.t, err)
	}

	return id
}

// insertTaskLog creates a completed (non-active) task log entry using persistence layer
func (h *journeyTestHarness) insertTaskLog(taskID int, beginTS, endTS time.Time, comment string) int {
	tlogID, err := persistence.InsertManualTL(h.db, taskID, beginTS, endTS, &comment)
	require.NoError(h.t, err)

	return int(tlogID)
}

// startTracking starts tracking time on the currently selected task
func (h *journeyTestHarness) startTracking() {
	cmd := h.model.getCmdToStartTracking()
	require.NotNil(h.t, cmd, "startTracking command should not be nil")

	// Execute the command to get the message
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model with tracking toggled message
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// stopTracking stops tracking and opens the finish form
func (h *journeyTestHarness) stopTracking() {
	h.model.handleRequestToStopTracking()
	assert.Equal(h.t, finishActiveTLView, h.model.activeView)
}

// finishTracking saves the active task log with the given end time and comment
func (h *journeyTestHarness) finishTracking(endTS time.Time, comment string) {
	// Set end time and comment in the form
	h.model.tLInputs[entryEndTS].SetValue(endTS.Format(timeFormat))
	h.model.tLCommentInput.SetValue(comment)

	// Submit the form
	cmd := h.model.getCmdToFinishTrackingActiveTL()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// createTask creates a new task via the TUI (simulates pressing 'a')
func (h *journeyTestHarness) createTask(summary string) {
	// Enter task creation view
	h.model.handleRequestToCreateTask()
	assert.Equal(h.t, taskInputView, h.model.activeView)

	// Enter task summary
	h.model.taskInputs[summaryField].SetValue(summary)

	// Submit the form
	cmd := h.model.getCmdToCreateOrUpdateTask()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model with task created message
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// updateTask updates the currently selected task's summary
func (h *journeyTestHarness) updateTask(newSummary string) {
	// Enter task update view
	h.model.handleRequestToUpdateTask()
	assert.Equal(h.t, taskInputView, h.model.activeView)

	// Set new summary
	h.model.taskInputs[summaryField].SetValue(newSummary)

	// Submit the form
	cmd := h.model.getCmdToCreateOrUpdateTask()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// selectTask selects a task in the active tasks list by index
func (h *journeyTestHarness) selectTask(index int) {
	h.model.activeTasksList.Select(index)
}

// getActiveTaskID returns the ID of the currently active (being tracked) task
func (h *journeyTestHarness) getActiveTaskID() int {
	return h.model.activeTaskID
}

// isTrackingActive returns true if time tracking is currently active
func (h *journeyTestHarness) isTrackingActive() bool {
	return h.model.trackingActive
}

// getTaskCount returns the number of tasks in the database
func (h *journeyTestHarness) getTaskCount() int {
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM task").Scan(&count)
	require.NoError(h.t, err)
	return count
}

// getTaskLogCount returns the number of task log entries in the database
func (h *journeyTestHarness) getTaskLogCount() int {
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM task_log WHERE active = 0").Scan(&count)
	require.NoError(h.t, err)
	return count
}

// getTaskByID retrieves a task from the database by ID
func (h *journeyTestHarness) getTaskByID(id int) (*types.Task, error) {
	row := h.db.QueryRow("SELECT id, summary, secs_spent, active, created_at, updated_at FROM task WHERE id = ?", id)

	var task types.Task
	err := row.Scan(&task.ID, &task.Summary, &task.SecsSpent, &task.Active, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

// getTaskSecsSpent returns the secs_spent for a task from the database
func (h *journeyTestHarness) getTaskSecsSpent(id int) int {
	var secsSpent int
	err := h.db.QueryRow("SELECT secs_spent FROM task WHERE id = ?", id).Scan(&secsSpent)
	require.NoError(h.t, err)
	return secsSpent
}

// goToTaskListView navigates to the task list view (simulates going back from taskLogView)
func (h *journeyTestHarness) goToTaskListView() {
	// From taskLogView, goBackward goes to taskListView
	// From inactiveTaskListView, goForward goes to taskListView
	switch h.model.activeView {
	case taskLogView:
		h.model.goBackwardInView()
	case inactiveTaskListView:
		h.model.goForwardInView()
	}
	require.Equal(h.t, taskListView, h.model.activeView)
}

// goToTaskLogView navigates to the task log view
func (h *journeyTestHarness) goToTaskLogView() {
	// From taskListView, goForward goes to taskLogView
	// From inactiveTaskListView, goBackward goes to taskLogView
	switch h.model.activeView {
	case taskListView:
		h.model.goForwardInView()
	case inactiveTaskListView:
		h.model.goBackwardInView()
	}
	require.Equal(h.t, taskLogView, h.model.activeView)
}

// goToInactiveTaskView navigates to the inactive task view
func (h *journeyTestHarness) goToInactiveTaskView() {
	// From taskListView, goBackward goes to inactiveTaskListView
	// From taskLogView, goForward goes to inactiveTaskListView
	switch h.model.activeView {
	case taskListView:
		h.model.goBackwardInView()
	case taskLogView:
		h.model.goForwardInView()
	}
	require.Equal(h.t, inactiveTaskListView, h.model.activeView)
}

// refreshTaskList refreshes the task list from the database
func (h *journeyTestHarness) refreshTaskList() {
	tasks, err := persistence.FetchTasks(h.db, true, 50)
	require.NoError(h.t, err)

	listItems := make([]list.Item, len(tasks))
	for i := range tasks {
		tasks[i].UpdateListTitle()
		tasks[i].UpdateListDesc(h.timeProvider)
		listItems[i] = &tasks[i]
		h.model.taskMap[tasks[i].ID] = &tasks[i]
		h.model.taskIndexMap[tasks[i].ID] = i
	}
	h.model.activeTasksList.SetItems(listItems)
}

// refreshInactiveTaskList refreshes the inactive task list from the database
func (h *journeyTestHarness) refreshInactiveTaskList() {
	tasks, err := persistence.FetchTasks(h.db, false, 50)
	require.NoError(h.t, err)

	listItems := make([]list.Item, len(tasks))
	for i := range tasks {
		tasks[i].UpdateListTitle()
		tasks[i].UpdateListDesc(h.timeProvider)
		listItems[i] = &tasks[i]
	}
	h.model.inactiveTasksList.SetItems(listItems)
}

// refreshTaskLogList refreshes the task log list from the database
func (h *journeyTestHarness) refreshTaskLogList() {
	entries, err := persistence.FetchTLEntries(h.db, true, 50)
	require.NoError(h.t, err)

	listItems := make([]list.Item, len(entries))
	for i := range entries {
		entries[i].UpdateListTitle()
		entries[i].UpdateListDesc(h.timeProvider)
		listItems[i] = entries[i]
	}
	h.model.taskLogList.SetItems(listItems)
}

// editSavedTaskLog edits an existing task log entry
func (h *journeyTestHarness) editSavedTaskLog(newBeginTS, newEndTS time.Time, newComment string) {
	// Enter edit view
	h.model.handleRequestToEditSavedTL()
	require.Equal(h.t, editSavedTLView, h.model.activeView)

	// Set new values
	h.model.tLInputs[entryBeginTS].SetValue(newBeginTS.Format(timeFormat))
	h.model.tLInputs[entryEndTS].SetValue(newEndTS.Format(timeFormat))
	h.model.tLCommentInput.SetValue(newComment)

	// Submit the form
	cmd := h.model.getCmdToCreateOrEditTL()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// moveTaskLogToTask moves the currently selected task log to a target task
func (h *journeyTestHarness) moveTaskLogToTask(targetTaskIndex int) {
	// Enter move view (this directly changes the view, no command returned)
	cmd := h.model.handleRequestToMoveTaskLog()
	require.Nil(h.t, cmd, "handleRequestToMoveTaskLog should not return a command on success")
	require.Equal(h.t, moveTaskLogView, h.model.activeView)

	// Select target task
	h.model.targetTasksList.Select(targetTaskIndex)

	// Submit the move (this returns the command)
	cmd = h.model.handleTargetTaskSelection()
	require.NotNil(h.t, cmd, "handleTargetTaskSelection should return a command")

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// moveTaskLogToTaskByID moves the currently selected task log to a target task identified by task ID
func (h *journeyTestHarness) moveTaskLogToTaskByID(targetTaskID int) {
	// Enter move view (this directly changes the view, no command returned)
	cmd := h.model.handleRequestToMoveTaskLog()
	require.Nil(h.t, cmd, "handleRequestToMoveTaskLog should not return a command on success")
	require.Equal(h.t, moveTaskLogView, h.model.activeView)

	// Find the target task by ID in the targetTasksList
	targetItems := h.model.targetTasksList.Items()
	targetIndex := -1
	for i := 0; i < len(targetItems); i++ {
		task, ok := targetItems[i].(*types.Task)
		if ok && task.ID == targetTaskID {
			targetIndex = i
			break
		}
	}
	require.NotEqual(h.t, -1, targetIndex, "target task with ID %d should be found in targetTasksList", targetTaskID)

	// Select target task
	h.model.targetTasksList.Select(targetIndex)

	// Submit the move (this returns the command)
	cmd = h.model.handleTargetTaskSelection()
	require.NotNil(h.t, cmd, "handleTargetTaskSelection should return a command")

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// deactivateTask deactivates the currently selected active task
func (h *journeyTestHarness) deactivateTask() {
	cmd := h.model.getCmdToDeactivateTask()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// reactivateTask reactivates the currently selected inactive task
func (h *journeyTestHarness) reactivateTask() {
	cmd := h.model.getCmdToActivateDeactivatedTask()
	require.NotNil(h.t, cmd)

	// Execute command
	msg := cmd()
	require.NotNil(h.t, msg)

	// Update model
	newModel, _ := h.model.Update(msg)
	h.model = newModel.(Model)
}

// selectTaskLog selects a task log entry by index
func (h *journeyTestHarness) selectTaskLog(index int) {
	h.model.taskLogList.Select(index)
}

// selectInactiveTask selects an inactive task by index
func (h *journeyTestHarness) selectInactiveTask(index int) {
	h.model.inactiveTasksList.Select(index)
}

// assertTaskSecsSpent asserts the secs_spent for a task in the database
func (h *journeyTestHarness) assertTaskSecsSpent(taskID, expectedSecs int) {
	secs := h.getTaskSecsSpent(taskID)
	assert.Equal(h.t, expectedSecs, secs, "task secs_spent mismatch")
}

// getTaskLogByID retrieves a task log entry from the database by ID
func (h *journeyTestHarness) getTaskLogByID(id int) (*types.TaskLogEntry, error) {
	row := h.db.QueryRow(
		"SELECT id, task_id, begin_ts, end_ts, secs_spent, comment FROM task_log WHERE id = ? AND active = 0",
		id,
	)

	var entry types.TaskLogEntry
	var comment sql.NullString
	err := row.Scan(&entry.ID, &entry.TaskID, &entry.BeginTS, &entry.EndTS, &entry.SecsSpent, &comment)
	if err != nil {
		return nil, err
	}

	if comment.Valid {
		entry.Comment = &comment.String
	}

	return &entry, nil
}

// assertView asserts the current active view
func (h *journeyTestHarness) assertView(expected stateView, msgAndArgs ...interface{}) {
	assert.Equal(h.t, expected, h.model.activeView, msgAndArgs...)
}

// assertTrackingState asserts the tracking state
func (h *journeyTestHarness) assertTrackingState(expectedActive bool, expectedTaskID int) {
	assert.Equal(h.t, expectedActive, h.model.trackingActive, "tracking active state mismatch")
	if expectedActive {
		assert.Equal(h.t, expectedTaskID, h.model.activeTaskID, "active task ID mismatch")
	}
}

// assertMessage asserts the current user message
func (h *journeyTestHarness) assertMessage(expected string) {
	assert.Equal(h.t, expected, h.model.message.value)
}

// assertDBTaskCount asserts the number of tasks in the database
func (h *journeyTestHarness) assertDBTaskCount(expected int) {
	count := h.getTaskCount()
	assert.Equal(h.t, expected, count, "task count mismatch")
}

// assertDBTaskLogCount asserts the number of task log entries in the database
func (h *journeyTestHarness) assertDBTaskLogCount(expected int) {
	count := h.getTaskLogCount()
	assert.Equal(h.t, expected, count, "task log count mismatch")
}

// T-040: Journey harness tests

func TestJourneyHarnessCreation(t *testing.T) {
	// GIVEN / WHEN
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// THEN - harness should be properly initialized
	assert.NotNil(t, h.db)
	assert.NotNil(t, h.model.db)
	assert.Equal(t, h.db, h.model.db)
	assert.Equal(t, taskListView, h.model.activeView)
	assert.False(t, h.model.trackingActive)
}

func TestJourneyHarnessInsertTask(t *testing.T) {
	// GIVEN
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// WHEN
	taskID := h.insertTask("Test Task", true)

	// THEN
	assert.Equal(t, 1, taskID)
	h.assertDBTaskCount(1)

	task, err := h.getTaskByID(taskID)
	require.NoError(t, err)
	assert.Equal(t, "Test Task", task.Summary)
	assert.True(t, task.Active)
}

func TestJourneyHarnessInsertTaskLog(t *testing.T) {
	// GIVEN
	h := newJourneyTestHarness(t)
	defer h.cleanup()
	taskID := h.insertTask("Test Task", true)

	// WHEN
	beginTS := h.timeProvider.Now().Add(-2 * time.Hour)
	endTS := h.timeProvider.Now()
	logID := h.insertTaskLog(taskID, beginTS, endTS, "Test work")

	// THEN
	assert.Equal(t, 1, logID)
	h.assertDBTaskLogCount(1)

	log, err := h.getTaskLogByID(logID)
	require.NoError(t, err)
	assert.Equal(t, taskID, log.TaskID)
	assert.Equal(t, "Test work", *log.Comment)
}

func TestJourneyHarnessCreateTaskViaTUI(t *testing.T) {
	// GIVEN
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// WHEN - create task via TUI
	h.createTask("New Task via TUI")

	// THEN
	h.assertDBTaskCount(1)
	h.assertView(taskListView)
}

func TestJourneyHarnessTaskLifecycle(t *testing.T) {
	// GIVEN
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// Insert a task and select it
	taskID := h.insertTask("Task to Track", true)
	require.Equal(t, 1, taskID)

	// Refresh the model's task list by simulating a tasks fetch
	task := createTestTask(taskID, "Task to Track", true, false, h.timeProvider)
	h.model.taskMap[taskID] = task
	h.model.taskIndexMap[taskID] = 0
	h.model.activeTasksList.SetItems([]list.Item{task})
	h.model.activeTasksList.Select(0)

	// WHEN - start tracking
	h.startTracking()

	// THEN
	h.assertTrackingState(true, taskID)
	h.assertView(taskListView)

	// WHEN - stop tracking
	h.stopTracking()

	// THEN
	h.assertView(finishActiveTLView)

	// WHEN - finish tracking with comment (end time 2 hours after begin)
	endTS := h.timeProvider.Now().Add(2 * time.Hour)
	h.finishTracking(endTS, "Work completed")

	// THEN
	h.assertTrackingState(false, -1)
	h.assertView(taskListView)
	h.assertDBTaskLogCount(1)
}

// T-041: Journey Flow A - Create task -> Start tracking -> Stop/Save -> Verify
func TestJourneyFlowA_CreateTrackStopVerify(t *testing.T) {
	// GIVEN - fresh harness with no tasks
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// Verify initial state
	h.assertDBTaskCount(0)
	h.assertDBTaskLogCount(0)

	// STEP 1: Create a new task via TUI
	h.createTask("My New Task")
	h.assertDBTaskCount(1)
	h.assertView(taskListView)

	// Refresh task list to get the newly created task
	h.refreshTaskList()
	h.selectTask(0)

	// STEP 2: Start tracking on the newly created task
	taskID := h.getActiveTaskIDAtCurrentSelection()
	h.startTracking()
	h.assertTrackingState(true, taskID)

	// Simulate time passing (2 hours of work)
	workDuration := 2 * time.Hour
	endTS := h.timeProvider.Now().Add(workDuration)

	// STEP 3: Stop tracking and save with comment
	h.stopTracking()
	h.assertView(finishActiveTLView)

	h.finishTracking(endTS, "Completed important work")

	// THEN - Verify final state
	h.assertTrackingState(false, -1)
	h.assertView(taskListView)
	h.assertDBTaskLogCount(1)

	// Verify task has accumulated time
	expectedSecs := int(workDuration.Seconds())
	h.assertTaskSecsSpent(taskID, expectedSecs)

	// Verify the task log entry details
	logEntry, err := h.getTaskLogByID(1)
	require.NoError(t, err)
	assert.Equal(t, taskID, logEntry.TaskID)
	assert.Equal(t, expectedSecs, logEntry.SecsSpent)
	assert.Equal(t, "Completed important work", *logEntry.Comment)
}

// T-042: Journey Flow B - Edit log -> Move log -> Deactivate/Reactivate -> Verify
func TestJourneyFlowB_EditMoveDeactivateReactivate(t *testing.T) {
	// GIVEN - Setup with two tasks and one task log entry
	h := newJourneyTestHarness(t)
	defer h.cleanup()

	// Create two tasks
	task1ID := h.insertTask("Source Task", true)
	task2ID := h.insertTask("Target Task", true)

	// Create a task log entry for task1 (1 hour of work)
	beginTS := h.timeProvider.Now().Add(-2 * time.Hour)
	endTS := h.timeProvider.Now().Add(-1 * time.Hour)
	originalSecs := int(endTS.Sub(beginTS).Seconds())
	logID := h.insertTaskLog(task1ID, beginTS, endTS, "Original work")

	h.assertDBTaskCount(2)
	h.assertDBTaskLogCount(1)
	h.assertTaskSecsSpent(task1ID, originalSecs)
	h.assertTaskSecsSpent(task2ID, 0)

	// Refresh task log list and select the entry
	h.goToTaskLogView()
	h.refreshTaskLogList()
	h.selectTaskLog(0)

	// STEP 1: Edit the task log (change times and comment)
	newBeginTS := h.timeProvider.Now().Add(-3 * time.Hour)
	newEndTS := h.timeProvider.Now().Add(-1 * time.Hour)
	newSecs := int(newEndTS.Sub(newBeginTS).Seconds())

	h.editSavedTaskLog(newBeginTS, newEndTS, "Updated work description")

	// Verify the edit was applied
	updatedLog, err := h.getTaskLogByID(logID)
	require.NoError(t, err)
	assert.Equal(t, newSecs, updatedLog.SecsSpent)
	assert.Equal(t, "Updated work description", *updatedLog.Comment)

	// Refresh task list to get updated secs_spent values
	h.refreshTaskList()
	h.assertTaskSecsSpent(task1ID, newSecs)

	// Verify we're at taskLogView after editing (editSavedTaskLog ends there)
	h.assertView(taskLogView)

	// Re-select the task log after refresh
	h.refreshTaskLogList()
	h.selectTaskLog(0)

	// STEP 2: Move the task log from task1 to task2
	// First refresh the target tasks list
	h.refreshTaskList()
	h.refreshTaskLogList()
	h.selectTaskLog(0)

	h.moveTaskLogToTaskByID(task2ID) // Move to task2 by ID

	// Refresh lists after move (Update handler sets view to taskLogView)
	h.refreshTaskList()
	h.refreshTaskLogList()

	// From taskLogView, go backward once to get to taskListView
	h.model.goBackwardInView() // taskLogView -> taskListView
	h.assertView(taskListView)

	// Verify the move - secs should have transferred
	h.assertTaskSecsSpent(task1ID, 0)
	h.assertTaskSecsSpent(task2ID, newSecs)

	// Verify task log now belongs to task2
	movedLog, err := h.getTaskLogByID(logID)
	require.NoError(t, err)
	assert.Equal(t, task2ID, movedLog.TaskID)

	// STEP 3: Deactivate task1 (now with no time)
	h.refreshTaskList()

	// Find and select task1
	items := h.model.activeTasksList.Items()
	found := false
	for i := 0; i < len(items); i++ {
		task, ok := items[i].(*types.Task)
		if ok && task.ID == task1ID {
			h.selectTask(i)
			found = true
			break
		}
	}
	require.True(t, found, "task1 should be found in active tasks list")

	h.deactivateTask()
	h.refreshTaskList()
	h.refreshInactiveTaskList()

	// Verify task1 is now inactive
	task1, err := h.getTaskByID(task1ID)
	require.NoError(t, err)
	assert.False(t, task1.Active)

	// STEP 4: Reactivate task1
	h.goToInactiveTaskView()

	// Find and select task1 in inactive list
	inactiveItems := h.model.inactiveTasksList.Items()
	found = false
	for i := 0; i < len(inactiveItems); i++ {
		task, ok := inactiveItems[i].(*types.Task)
		if ok && task.ID == task1ID {
			h.selectInactiveTask(i)
			found = true
			break
		}
	}
	require.True(t, found, "task1 should be found in inactive tasks list")

	h.reactivateTask()
	h.refreshTaskList()
	h.refreshInactiveTaskList()

	// Verify task1 is active again
	reactivatedTask1, err := h.getTaskByID(task1ID)
	require.NoError(t, err)
	assert.True(t, reactivatedTask1.Active)

	// Final verification - task2 should still have the time
	h.assertTaskSecsSpent(task2ID, newSecs)
	h.assertTaskSecsSpent(task1ID, 0)
}

// Helper to get task ID from current selection
func (h *journeyTestHarness) getActiveTaskIDAtCurrentSelection() int {
	task, ok := h.model.activeTasksList.SelectedItem().(*types.Task)
	if !ok {
		h.t.Fatal("No task selected")
	}
	return task.ID
}
