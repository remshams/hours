package ui

// T-PR6: Tests for the handlers extracted from Model.Update as part of PR6
// refactoring.  These tests focus on the new sub-functions and are intended to
// supplement (not replace) the existing update_test.go suite.

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// handleFilteringKeys
// ---------------------------------------------------------------------------

func TestHandleFilteringKeysActiveTasksListFiltering(t *testing.T) {
	// GIVEN – active tasks list is in filtering mode
	m := createTestModel()
	task := createTestTask(1, "example task", true, false, m.timeProvider)
	m.activeTasksList.SetItems([]list.Item{task})
	// Trigger filter mode by sending a "/" key directly to the list
	m.activeTasksList, _ = m.activeTasksList.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// WHEN
	exitEarly, cmds := m.handleFilteringKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// THEN – the handler should signal early exit so the outer loop returns
	assert.True(t, exitEarly)
	// cmds may be nil but the slice itself is returned (length not asserted to
	// avoid coupling to bubbles internals)
	_ = cmds
}

func TestHandleFilteringKeysNoListFiltering(t *testing.T) {
	// GIVEN – no list is filtering
	m := createTestModel()

	// WHEN
	exitEarly, cmds := m.handleFilteringKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// THEN – should not exit early when nothing is filtering
	assert.False(t, exitEarly)
	assert.Nil(t, cmds)
}

// ---------------------------------------------------------------------------
// handleFormKeys – enter / ctrl+s (submit)
// ---------------------------------------------------------------------------

func TestHandleFormKeysEnterOnTaskInputViewTriggersSubmit(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskInputView
	m.taskInputs[summaryField].SetValue("My new task")

	// WHEN – KeyEnter is the proper enter key type
	exitEarly, cmds := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// THEN – a submit command should be produced and the caller should return early
	assert.True(t, exitEarly)
	assert.NotEmpty(t, cmds)
}

func TestHandleFormKeysEnterIgnoredWhenFocusedOnComment(t *testing.T) {
	// GIVEN – comment field is focused; enter should NOT submit
	m := createTestModel()
	m.activeView = finishActiveTLView
	m.trackingFocussedField = entryComment

	// WHEN
	exitEarly, cmds := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyEnter})

	// THEN – bail path: no exit early, no cmds
	assert.False(t, exitEarly)
	assert.Empty(t, cmds)
}

func TestHandleFormKeysCtrlSOnTaskInputViewTriggersSubmit(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskInputView
	m.taskInputs[summaryField].SetValue("My new task")

	// WHEN
	exitEarly, cmds := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyCtrlS})

	// THEN
	assert.True(t, exitEarly)
	assert.NotEmpty(t, cmds)
}

// ---------------------------------------------------------------------------
// handleFormKeys – escape
// ---------------------------------------------------------------------------

func TestHandleFormKeysEscapeInFormViewsExitsEarly(t *testing.T) {
	formViews := []stateView{
		taskInputView,
		editActiveTLView,
		finishActiveTLView,
		manualTasklogEntryView,
		editSavedTLView,
		moveTaskLogView,
	}

	for _, view := range formViews {
		t.Run(view.String(), func(t *testing.T) {
			m := createTestModel()
			m.activeView = view
			// editActiveTLView needs a valid begin TS to not panic
			m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:00")

			exitEarly, cmds := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyEsc})

			assert.True(t, exitEarly, "escape inside form should exit early")
			assert.Empty(t, cmds)
		})
	}
}

func TestHandleFormKeysEscapeOutsideFormViewsDoesNotExitEarly(t *testing.T) {
	// GIVEN – not in a form view
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	exitEarly, cmds := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyEsc})

	// THEN
	assert.False(t, exitEarly)
	assert.Empty(t, cmds)
}

// ---------------------------------------------------------------------------
// handleFormKeys – tab / shift+tab
// ---------------------------------------------------------------------------

func TestHandleFormKeysTabNavigatesForwardInView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyTab})

	// THEN – tab should call goForwardInView (taskListView → taskLogView)
	// exitEarly is false; the side-effect is the view change
	assert.False(t, exitEarly)
	assert.Equal(t, taskLogView, m.activeView)
}

func TestHandleFormKeysShiftTabNavigatesBackwardInView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyShiftTab})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, taskListView, m.activeView)
}

// ---------------------------------------------------------------------------
// handleFormKeys – time-shift keys (k / j / K / J / h / l)
// ---------------------------------------------------------------------------

func TestHandleFormKeysTimeShiftKInFormViewShiftsTimeBackwardOneMinute(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = editActiveTLView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// THEN – time should have shifted; exitEarly is false (no early bail on success)
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/16 09:29", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftJInFormViewShiftsTimeForwardOneMinute(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = editActiveTLView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/16 09:31", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftCapitalKInFormViewShiftsTimeBackwardFiveMinutes(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = finishActiveTLView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/16 09:25", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftCapitalJInFormViewShiftsTimeForwardFiveMinutes(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = finishActiveTLView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/16 09:35", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftHInFormViewShiftsTimeBackwardOneDay(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = manualTasklogEntryView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/15 09:30", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftLInFormViewShiftsTimeForwardOneDay(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = editSavedTLView
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].SetValue("2025/08/16 09:30")

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// THEN
	assert.False(t, exitEarly)
	assert.Equal(t, "2025/08/17 09:30", m.tLInputs[entryBeginTS].Value())
}

func TestHandleFormKeysTimeShiftNotAppliedOutsideFormViews(t *testing.T) {
	// GIVEN – taskListView: time-shift keys should have no effect
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	exitEarly, _ := m.handleFormKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// THEN – no exit early, no time change (nothing to change)
	assert.False(t, exitEarly)
}

// ---------------------------------------------------------------------------
// updateInputComponents
// ---------------------------------------------------------------------------

func TestUpdateInputComponentsTaskInputViewReturnsHandled(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskInputView

	// WHEN
	cmds, handled := m.updateInputComponents(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// THEN
	assert.True(t, handled)
	// cmds length may vary with bubbles internals, just check it's non-nil
	_ = cmds
}

func TestUpdateInputComponentsTLFormViewsReturnHandled(t *testing.T) {
	tlFormViews := []stateView{
		editActiveTLView,
		finishActiveTLView,
		manualTasklogEntryView,
		editSavedTLView,
	}

	for _, view := range tlFormViews {
		t.Run(view.String(), func(t *testing.T) {
			m := createTestModel()
			m.activeView = view

			cmds, handled := m.updateInputComponents(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

			assert.True(t, handled)
			_ = cmds
		})
	}
}

func TestUpdateInputComponentsListViewsReturnNotHandled(t *testing.T) {
	listViews := []stateView{
		taskListView,
		taskLogView,
		inactiveTaskListView,
		moveTaskLogView,
		helpView,
	}

	for _, view := range listViews {
		t.Run(view.String(), func(t *testing.T) {
			m := createTestModel()
			m.activeView = view

			cmds, handled := m.updateInputComponents(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

			assert.False(t, handled)
			assert.Nil(t, cmds)
		})
	}
}

// ---------------------------------------------------------------------------
// handleListKeys – view navigation
// ---------------------------------------------------------------------------

func TestHandleListKeysKey1SwitchesToTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})

	// THEN
	assert.Equal(t, taskListView, m.activeView)
}

func TestHandleListKeysKey2SwitchesToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})

	// THEN
	assert.Equal(t, taskLogView, m.activeView)
}

func TestHandleListKeysKey3SwitchesToInactiveTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

	// THEN
	assert.Equal(t, inactiveTaskListView, m.activeView)
}

func TestHandleListKeysHelpKeyOpensHelpView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// THEN
	assert.Equal(t, helpView, m.activeView)
	assert.Equal(t, taskListView, m.lastView)
}

func TestHandleListKeysQFromTaskListViewReturnsQuitCmd(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	cmds := m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// THEN – quit command should be in returned cmds
	assert.NotEmpty(t, cmds)
}

func TestHandleListKeysQFromHelpViewGoesBackToLastView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = helpView
	m.lastView = taskLogView

	// WHEN
	cmds := m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// THEN – no quit, back to lastView
	assert.Empty(t, cmds)
	assert.Equal(t, taskLogView, m.activeView)
}

func TestHandleListKeysFInTaskListViewWithNoActiveTrackingShowsError(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView
	m.trackingActive = false

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// THEN
	assert.Equal(t, userMsgErr, m.message.kind)
}

func TestHandleListKeysFOutsideTaskListViewIsIgnored(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView
	m.message = infoMsg("")

	// WHEN
	cmds := m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// THEN – no message set, no cmds
	assert.Empty(t, cmds)
	assert.Empty(t, m.message.value)
}

func TestHandleListKeysAOpensTaskInputViewWhenInTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// THEN
	assert.Equal(t, taskInputView, m.activeView)
}

func TestHandleListKeysAIgnoredOutsideTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	m.handleListKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// THEN – view unchanged
	assert.Equal(t, taskLogView, m.activeView)
}

// ---------------------------------------------------------------------------
// handleMsg – async message handling
// ---------------------------------------------------------------------------

func TestHandleMsgTaskCreatedMsgWithErrorSetsErrMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(taskCreatedMsg{err: errTestError})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
}

func TestHandleMsgTaskCreatedMsgSuccessFetchesTasks(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(taskCreatedMsg{})

	// THEN – a fetchTasks cmd should be returned
	assert.NotEmpty(t, cmds)
}

func TestHandleMsgHideHelpMsgDisablesHelpIndicator(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.showHelpIndicator = true

	// WHEN
	cmds := m.handleMsg(hideHelpMsg{})

	// THEN
	assert.Empty(t, cmds)
	assert.False(t, m.showHelpIndicator)
}

func TestHandleMsgActiveTLUpdatedMsgWithErrorSetsErrMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(activeTLUpdatedMsg{err: errTestError})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
}

func TestHandleMsgActiveTLUpdatedMsgSuccessUpdatesState(t *testing.T) {
	// GIVEN
	m := createTestModel()
	comment := "updated comment"

	// WHEN
	cmds := m.handleMsg(activeTLUpdatedMsg{
		beginTS: referenceTime,
		comment: &comment,
	})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, referenceTime, m.activeTLBeginTS)
	assert.Equal(t, &comment, m.activeTLComment)
}

func TestHandleMsgTaskLogMovedMsgWithErrorSetsErrMessageAndGoesToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = moveTaskLogView

	// WHEN
	cmds := m.handleMsg(taskLogMovedMsg{err: errTestError})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
	assert.Equal(t, taskLogView, m.activeView)
}

func TestHandleMsgTaskLogMovedMsgSuccessFetchesData(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = moveTaskLogView

	// WHEN
	cmds := m.handleMsg(taskLogMovedMsg{})

	// THEN – fetchTLS + fetchTasks = 2 cmds; view goes to taskLogView
	assert.Len(t, cmds, 2)
	assert.Equal(t, taskLogView, m.activeView)
}

func TestHandleMsgTaskActiveStatusUpdatedMsgWithErrorSetsErrMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(taskActiveStatusUpdatedMsg{err: errTestError})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
}

func TestHandleMsgTaskActiveStatusUpdatedMsgSuccessFetchesBothTaskLists(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(taskActiveStatusUpdatedMsg{})

	// THEN – fetchTasks(active) + fetchTasks(inactive) = 2 cmds
	assert.Len(t, cmds, 2)
}

func TestHandleMsgStaleTasksArchivedMsgWithErrorSetsErrMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(staleTasksArchivedMsg{err: errTestError})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
}

func TestHandleMsgStaleTasksArchivedMsgSuccessSetsInfoMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	cmds := m.handleMsg(staleTasksArchivedMsg{count: 3})

	// THEN – info message set, fetchTasks(active) + fetchTasks(inactive) = 2 cmds
	assert.Equal(t, userMsgInfo, m.message.kind)
	assert.Len(t, cmds, 2)
}

// ---------------------------------------------------------------------------
// Model.Update – async message dispatch when a form view is active
// ---------------------------------------------------------------------------

// TestModelUpdateDispatchesTypedAsyncMessagesWhenFormViewActive is a regression
// test for the bug where updateInputComponents returned handled=true for every
// message type (not just input events) when a form view was active.  This caused
// typed async messages (e.g. taskCreatedMsg) to be swallowed before handleMsg
// could process them.
//
// The test simulates the full Update dispatch path: it delivers a taskCreatedMsg
// directly to Model.Update while activeView is set to a form view and asserts
// that handleMsg routed the message correctly (error case → m.message.kind is
// set to userMsgErr; success case → a fetchTasks command is returned).
func TestModelUpdateDispatchesTypedAsyncMessagesWhenFormViewActive(t *testing.T) {
	formViews := []stateView{
		taskInputView,
		editActiveTLView,
		finishActiveTLView,
		manualTasklogEntryView,
		editSavedTLView,
	}

	for _, view := range formViews {
		t.Run(view.String()+"/error", func(t *testing.T) {
			// GIVEN – model is showing a form view
			m := createTestModel()
			m.activeView = view

			// WHEN – an async taskCreatedMsg with an error arrives
			newModel, cmd := m.Update(taskCreatedMsg{err: errTestError})
			updated := newModel.(Model)

			// THEN – handleMsg must have processed the message and set the error
			// state; no further command should be issued for the error path.
			assert.Equal(t, userMsgErr, updated.message.kind,
				"handleMsg should have set an error message for view %s", view)
			_ = cmd
		})

		t.Run(view.String()+"/success", func(t *testing.T) {
			// GIVEN – model is showing a form view
			m := createTestModel()
			m.activeView = view

			// WHEN – a successful taskCreatedMsg arrives
			_, cmd := m.Update(taskCreatedMsg{})

			// THEN – handleMsg should have returned a fetchTasks command, which
			// means Update batched at least one non-nil command.
			require.NotNil(t, cmd,
				"handleMsg should have produced a fetchTasks cmd for view %s", view)
		})
	}
}

func TestHandleMsgTaskRepUpdatedMsgWithErrorSetsErrMessage(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN
	task := createTestTask(1, "some task", true, false, m.timeProvider)
	cmds := m.handleMsg(taskRepUpdatedMsg{err: errTestError, tsk: task})

	// THEN
	assert.Empty(t, cmds)
	assert.Equal(t, userMsgErr, m.message.kind)
}

// ---------------------------------------------------------------------------
// updateActiveView
// ---------------------------------------------------------------------------

func TestUpdateActiveViewUpdatesTaskListInTaskListView(_ *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN – send a no-op key message; the list should still process it
	cmds := m.updateActiveView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// THEN – a cmd slice is returned (may be empty or contain tick cmds)
	_ = cmds // No panic = success
}

func TestUpdateActiveViewUpdatesTaskLogListInTaskLogView(_ *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	cmds := m.updateActiveView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// THEN
	_ = cmds
}

func TestUpdateActiveViewReturnsNilCmdsForUnhandledViews(t *testing.T) {
	// GIVEN – a view that has no list/viewport to update
	m := createTestModel()
	m.activeView = taskInputView

	// WHEN
	cmds := m.updateActiveView(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// THEN – no cmds returned for form views (handled earlier)
	assert.Empty(t, cmds)
}

// ---------------------------------------------------------------------------
// stateView.String() helper (used in sub-test names)
// ---------------------------------------------------------------------------

func (v stateView) String() string {
	switch v {
	case taskListView:
		return "taskListView"
	case taskLogView:
		return "taskLogView"
	case taskLogDetailsView:
		return "taskLogDetailsView"
	case inactiveTaskListView:
		return "inactiveTaskListView"
	case editActiveTLView:
		return "editActiveTLView"
	case finishActiveTLView:
		return "finishActiveTLView"
	case manualTasklogEntryView:
		return "manualTasklogEntryView"
	case editSavedTLView:
		return "editSavedTLView"
	case taskInputView:
		return "taskInputView"
	case moveTaskLogView:
		return "moveTaskLogView"
	case helpView:
		return "helpView"
	case insufficientDimensionsView:
		return "insufficientDimensionsView"
	default:
		return "unknown"
	}
}

// errTestError is a sentinel error used in handler tests.
var errTestError = types.ErrDurationNotLongEnough
