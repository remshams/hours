package ui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	ctrlC                 = "ctrl+c"
	enter                 = "enter"
	escape                = "esc"
	viewPortMoveLineCount = 3
	msgCouldntSelectATask = "Couldn't select a task"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.frameCounter++
	var cmds []tea.Cmd

	// early check for window resizing and handling insufficient dimensions
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowResizing(msg)
	case tea.KeyMsg:
		if msg.String() == ctrlC {
			return m, tea.Quit
		}

		if m.activeView == insufficientDimensionsView {
			switch msg.String() {
			case "q", escape:
				return m, tea.Quit
			default:
				return m, tea.Batch(cmds...)
			}
		}
	}

	if m.activeView != insufficientDimensionsView {
		if m.message.framesLeft > 0 {
			m.message.framesLeft--
		}

		if m.message.framesLeft == 0 {
			m.message.value = ""
		}
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Delegate filter key handling to the appropriate list when filtering.
		if exitEarly, exitCmds := m.handleFilteringKeys(keyMsg); exitEarly {
			return m, tea.Batch(exitCmds...)
		}

		// Keys that only make sense inside forms (submit, escape, tab, time-shifts).
		if exitEarly, exitCmds := m.handleFormKeys(keyMsg); exitEarly {
			return m, tea.Batch(exitCmds...)
		}
	}

	// Propagate msg to active input components (forms and lists).
	if inputCmds, handled := m.updateInputComponents(msg); handled {
		return m, tea.Batch(inputCmds...)
	}

	// Handle typed messages and list-level key actions.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		listCmds := m.handleListKeys(msg)
		cmds = append(cmds, listCmds...)
	default:
		msgCmds := m.handleMsg(msg)
		cmds = append(cmds, msgCmds...)
	}

	// Propagate msg to the currently focused list or viewport.
	viewCmds := m.updateActiveView(msg)
	cmds = append(cmds, viewCmds...)

	return m, tea.Batch(cmds...)
}

// handleFilteringKeys routes key events to whichever list is currently in
// filter mode and signals the caller to return early.
func (m *Model) handleFilteringKeys(keyMsg tea.KeyMsg) (exitEarly bool, cmds []tea.Cmd) {
	var cmd tea.Cmd
	if m.activeTasksList.FilterState() == list.Filtering {
		m.activeTasksList, cmd = m.activeTasksList.Update(keyMsg)
		return true, []tea.Cmd{cmd}
	}
	if m.targetTasksList.FilterState() == list.Filtering {
		m.targetTasksList, cmd = m.targetTasksList.Update(keyMsg)
		return true, []tea.Cmd{cmd}
	}
	return false, nil
}

// updateInputComponents propagates an input event to the active form's input
// widgets and signals the caller to return early.  Returns handled=true only
// when a form view is active AND msg is an input event (tea.KeyMsg or
// tea.MouseMsg); for all other message types it returns handled=false so that
// async messages (e.g. taskCreatedMsg) are not silently dropped and can reach
// handleMsg.
func (m *Model) updateInputComponents(msg tea.Msg) (cmds []tea.Cmd, handled bool) {
	switch msg.(type) {
	case tea.KeyMsg, tea.MouseMsg:
	default:
		return nil, false
	}

	var cmd tea.Cmd
	switch m.activeView {
	case taskInputView:
		for i := range m.taskInputs {
			m.taskInputs[i], cmd = m.taskInputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		return cmds, true
	case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
		for i := range m.tLInputs {
			m.tLInputs[i], cmd = m.tLInputs[i].Update(msg)
			cmds = append(cmds, cmd)
		}
		m.tLCommentInput, cmd = m.tLCommentInput.Update(msg)
		cmds = append(cmds, cmd)
		return cmds, true
	}
	return nil, false
}

// updateActiveView propagates a message to the list or viewport that
// corresponds to the currently active view.
func (m *Model) updateActiveView(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	var cmd tea.Cmd
	switch m.activeView {
	case taskListView:
		m.activeTasksList, cmd = m.activeTasksList.Update(msg)
		cmds = append(cmds, cmd)
	case taskLogView:
		m.taskLogList, cmd = m.taskLogList.Update(msg)
		cmds = append(cmds, cmd)
	case inactiveTaskListView:
		m.inactiveTasksList, cmd = m.inactiveTasksList.Update(msg)
		cmds = append(cmds, cmd)
	case moveTaskLogView:
		m.targetTasksList, cmd = m.targetTasksList.Update(msg)
		cmds = append(cmds, cmd)
	case helpView:
		m.helpVP, cmd = m.helpVP.Update(msg)
		cmds = append(cmds, cmd)
	}
	return cmds
}
