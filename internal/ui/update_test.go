package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-020: Navigation and view transitions

func TestNavigationKey1SwitchesToTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskListView, model.activeView)
}

func TestNavigationKey1DoesNothingWhenAlreadyOnTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskListView, model.activeView)
}

func TestNavigationKey2SwitchesToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskLogView, model.activeView)
}

func TestNavigationKey2DoesNothingWhenAlreadyOnTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskLogView, model.activeView)
}

func TestNavigationKey3SwitchesToInactiveTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, inactiveTaskListView, model.activeView)
}

func TestNavigationKey3DoesNothingWhenAlreadyOnInactiveTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = inactiveTaskListView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, inactiveTaskListView, model.activeView)
}

func TestHelpKeyShowsHelpView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, helpView, model.activeView)
	assert.Equal(t, taskListView, model.lastView)
}

func TestEscapeFromHelpViewReturnsToLastView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = helpView
	m.lastView = taskLogView

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go back to taskLogView
	assert.Equal(t, taskLogView, model.activeView)
	// Command should not be quit
	assert.Nil(t, cmd)
}

func TestEscapeFromTaskLogViewReturnsToTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskListView, model.activeView)
	assert.Nil(t, cmd)
}

func TestEscapeFromInactiveTaskListViewReturnsToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = inactiveTaskListView

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskLogView, model.activeView)
	assert.Nil(t, cmd)
}

func TestEscapeFromMoveTaskLogViewReturnsToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = moveTaskLogView

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskLogView, model.activeView)
	assert.Nil(t, cmd)
}

func TestEscapeFromTaskLogDetailsViewReturnsToTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogDetailsView

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskLogView, model.activeView)
	assert.Nil(t, cmd)
}

func TestEscapeFromTaskInputViewReturnsToTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskInputView
	m.taskInputs[summaryField].SetValue("some input")

	// WHEN - simulate escape key
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN
	assert.Equal(t, taskListView, model.activeView)
	assert.Empty(t, model.taskInputs[summaryField].Value())
	assert.Nil(t, cmd)
}

func TestTabForwardNavigationInTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - press tab
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go to taskLogView
	assert.Equal(t, taskLogView, model.activeView)
}

func TestTabForwardNavigationInTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN - press tab
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go to inactiveTaskListView
	assert.Equal(t, inactiveTaskListView, model.activeView)
}

func TestTabForwardNavigationInInactiveTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = inactiveTaskListView

	// WHEN - press tab
	keyMsg := tea.KeyMsg{Type: tea.KeyTab}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go back to taskListView (cycle)
	assert.Equal(t, taskListView, model.activeView)
}

func TestShiftTabBackwardNavigationInTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - press shift+tab
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go to inactiveTaskListView
	assert.Equal(t, inactiveTaskListView, model.activeView)
}

func TestShiftTabBackwardNavigationInTaskLogView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN - press shift+tab
	keyMsg := tea.KeyMsg{Type: tea.KeyShiftTab}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should go to taskListView
	assert.Equal(t, taskListView, model.activeView)
}

func TestQuitKeyFromTaskListView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - press 'q'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(keyMsg)

	// THEN - should get tea.Quit command
	assert.NotNil(t, cmd)
}

func TestQuitKeyFromHelpViewReturnsToLastView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = helpView
	m.lastView = taskListView

	// WHEN - press 'q'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should return to last view, not quit
	assert.Equal(t, taskListView, model.activeView)
	assert.Nil(t, cmd)
}

func TestCtrlCQuitsFromAnyView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - press ctrl+c
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(keyMsg)

	// THEN - should get tea.Quit command
	assert.NotNil(t, cmd)
}

func TestCtrlCQuitsFromInsufficientDimensionsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView

	// WHEN - press ctrl+c
	keyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(keyMsg)

	// THEN - should get tea.Quit command
	assert.NotNil(t, cmd)
}

func TestEscapeQuitsFromInsufficientDimensionsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView

	// WHEN - press escape
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := m.Update(keyMsg)

	// THEN - should get tea.Quit command
	assert.NotNil(t, cmd)
}

func TestQQuitsFromInsufficientDimensionsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView

	// WHEN - press 'q'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(keyMsg)

	// THEN - should get tea.Quit command
	assert.NotNil(t, cmd)
}

func TestOtherKeysIgnoredInInsufficientDimensionsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView

	// WHEN - press a random key
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newM, cmd := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should stay in insufficient dimensions view
	assert.Equal(t, insufficientDimensionsView, model.activeView)
	assert.Nil(t, cmd)
}

// T-023: Resize and viewport edge cases

func TestWindowSizeBelowMinimumWidthEntersInsufficientDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - window resize with too small width
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded - 1,
		Height: minHeightNeeded,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should switch to insufficient dimensions view
	assert.Equal(t, insufficientDimensionsView, model.activeView)
	assert.Equal(t, taskListView, model.lastViewBeforeInsufficientDims)
}

func TestWindowSizeBelowMinimumHeightEntersInsufficientDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - window resize with too small height
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded,
		Height: minHeightNeeded - 1,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should switch to insufficient dimensions view
	assert.Equal(t, insufficientDimensionsView, model.activeView)
	assert.Equal(t, taskListView, model.lastViewBeforeInsufficientDims)
}

func TestWindowSizeBelowBothMinimumsEntersInsufficientDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskListView

	// WHEN - window resize with too small width and height
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded - 10,
		Height: minHeightNeeded - 5,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should switch to insufficient dimensions view
	assert.Equal(t, insufficientDimensionsView, model.activeView)
	assert.Equal(t, taskListView, model.lastViewBeforeInsufficientDims)
}

func TestWindowSizeRecoveryFromInsufficientDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView
	m.lastViewBeforeInsufficientDims = taskLogView

	// WHEN - window resize back to adequate dimensions
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded,
		Height: minHeightNeeded,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should return to previous view
	assert.Equal(t, taskLogView, model.activeView)
}

func TestWindowSizeUpdateTerminalDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()
	expectedWidth := 120
	expectedHeight := 60

	// WHEN - window resize
	msg := tea.WindowSizeMsg{
		Width:  expectedWidth,
		Height: expectedHeight,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should update terminal dimensions
	assert.Equal(t, expectedWidth, model.terminalWidth)
	assert.Equal(t, expectedHeight, model.terminalHeight)
}

func TestWindowSizeUpdateListDimensions(t *testing.T) {
	// GIVEN
	m := createTestModel()

	// WHEN - window resize
	msg := tea.WindowSizeMsg{
		Width:  120,
		Height: 60,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - list dimensions should be updated (accounting for frame size)
	// Frame width is 2, so list width = terminalWidth - 2
	expectedListWidth := 120 - 2
	assert.Equal(t, expectedListWidth, model.taskLogList.Width())
	assert.Equal(t, expectedListWidth, model.activeTasksList.Width())
	assert.Equal(t, expectedListWidth, model.inactiveTasksList.Width())
	assert.Equal(t, expectedListWidth, model.targetTasksList.Width())
}

func TestViewportScrollUpAtTopDoesNotScroll(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = helpView
	m.helpVPReady = true

	// WHEN - scroll up when at top (k key)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - viewport should still be at top
	assert.True(t, model.helpVP.AtTop())
}

func TestViewportScrollDownAtBottomDoesNotScroll(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = helpView
	m.helpVPReady = true

	// Set up content that exceeds viewport height so we can scroll
	longContent := "\nLine 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10\nLine 11\nLine 12\nLine 13\nLine 14\nLine 15\nLine 16\nLine 17\nLine 18\nLine 19\nLine 20\nLine 21\nLine 22\nLine 23\nLine 24\nLine 25\nLine 26\nLine 27\nLine 28\nLine 29\nLine 30"
	m.helpVP.SetContent(longContent)

	// Scroll down multiple times to reach the bottom
	for range 50 {
		m.handleRequestToScrollVPDown()
	}

	// Verify we're at the bottom
	require.True(t, m.helpVP.AtBottom(), "viewport should be at bottom after scrolling")
	initialYOffset := m.helpVP.YOffset

	// WHEN - attempt to scroll down more when already at bottom
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - viewport should still be at bottom and YOffset unchanged
	assert.True(t, model.helpVP.AtBottom(), "viewport should still be at bottom")
	assert.Equal(t, initialYOffset, model.helpVP.YOffset, "YOffset should not increase when scrolling at bottom")
}

func TestViewportScrollUpInTaskLogDetailsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogDetailsView
	m.tLDetailsVPReady = true
	m.tLDetailsVP.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	// WHEN - scroll up (k key)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should handle scroll (viewport guards prevent actual scroll when at top)
	assert.Equal(t, taskLogDetailsView, model.activeView)
}

func TestViewportScrollDownInTaskLogDetailsView(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogDetailsView
	m.tLDetailsVPReady = true
	m.tLDetailsVP.SetContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5")

	// WHEN - scroll down (j key)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - should handle scroll
	assert.Equal(t, taskLogDetailsView, model.activeView)
}

func TestLastViewPreservedWhenEnteringHelp(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = taskLogView

	// WHEN - show help
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	// THEN - lastView should be preserved
	assert.Equal(t, helpView, model.activeView)
	assert.Equal(t, taskLogView, model.lastView)
}

func TestLastViewPreservedWhenEnteringInsufficientDimsFromDifferentViews(t *testing.T) {
	testCases := []struct {
		name             string
		startingView     stateView
		expectedLastView stateView
	}{
		{
			name:             "from task list view",
			startingView:     taskListView,
			expectedLastView: taskListView,
		},
		{
			name:             "from task log view",
			startingView:     taskLogView,
			expectedLastView: taskLogView,
		},
		{
			name:             "from inactive task list view",
			startingView:     inactiveTaskListView,
			expectedLastView: inactiveTaskListView,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// GIVEN
			m := createTestModel()
			m.activeView = tt.startingView

			// WHEN - window resize to insufficient dimensions
			msg := tea.WindowSizeMsg{
				Width:  minWidthNeeded - 10,
				Height: minHeightNeeded - 5,
			}
			newM, _ := m.Update(msg)
			model := newM.(Model)

			// THEN - lastViewBeforeInsufficientDims should be preserved
			_ = newM.(Model)
			assert.Equal(t, insufficientDimensionsView, model.activeView)
			assert.Equal(t, tt.expectedLastView, model.lastViewBeforeInsufficientDims)
		})
	}
}

func TestWindowResizeFromInsufficientDimsDoesNotSwitchIfStillInsufficient(t *testing.T) {
	// GIVEN
	m := createTestModel()
	m.activeView = insufficientDimensionsView
	m.lastViewBeforeInsufficientDims = taskListView

	// WHEN - window resize but still insufficient
	msg := tea.WindowSizeMsg{
		Width:  minWidthNeeded - 5,
		Height: minHeightNeeded,
	}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// THEN - should stay in insufficient dimensions view
	assert.Equal(t, insufficientDimensionsView, model.activeView)
	assert.Equal(t, taskListView, model.lastViewBeforeInsufficientDims)
}
