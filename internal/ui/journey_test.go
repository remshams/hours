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

// insertTaskLog creates a completed (non-active) task log entry
func (h *journeyTestHarness) insertTaskLog(taskID int, beginTS, endTS time.Time, comment string) int {
	secsSpent := int(endTS.Sub(beginTS).Seconds())
	result, err := h.db.Exec(
		"INSERT INTO task_log (task_id, begin_ts, end_ts, secs_spent, comment, active) VALUES (?, ?, ?, ?, ?, ?)",
		taskID, beginTS, endTS, secsSpent, comment, false,
	)
	require.NoError(h.t, err)

	id, err := result.LastInsertId()
	require.NoError(h.t, err)

	return int(id)
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
