package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleListKeys handles key events that operate on lists and views (navigation
// shortcuts, task/log actions, viewport scrolling, help).
func (m *Model) handleListKeys(keyMsg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd
	switch keyMsg.String() {
	case "q", escape:
		if m.handleRequestToGoBackOrQuit() {
			return []tea.Cmd{tea.Quit}
		}
	case "1":
		if m.activeView != taskListView {
			m.activeView = taskListView
		}
	case "2":
		if m.activeView != taskLogView {
			m.activeView = taskLogView
		}
	case "3":
		if m.activeView != inactiveTaskListView {
			m.activeView = inactiveTaskListView
		}
	case "ctrl+r":
		if reloadCmd := m.getCmdToReloadData(); reloadCmd != nil {
			cmds = append(cmds, reloadCmd)
		}
	case "ctrl+t":
		m.goToActiveTask()
	case "f":
		if m.activeView != taskListView {
			break
		}

		if !m.trackingActive {
			m.message = errMsg("Nothing is being tracked right now")
			break
		}

		if handleCmd := m.getCmdToFinishActiveTL(); handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case "ctrl+s":
		switch m.activeView {
		case taskListView:
			switch m.trackingActive {
			case true:
				m.handleRequestToEditActiveTL()
			case false:
				m.handleRequestToCreateManualTL()
			}
		case taskLogView:
			m.handleRequestToEditSavedTL()
		}
	case "u":
		switch m.activeView {
		case taskListView:
			m.handleRequestToUpdateTask()
		case taskLogView:
			m.handleRequestToEditSavedTL()
		}
	case "ctrl+d":
		var handleCmd tea.Cmd
		switch m.activeView {
		case taskListView:
			handleCmd = m.getCmdToDeactivateTask()
		case taskLogView:
			handleCmd = m.getCmdToDeleteTL()
		case inactiveTaskListView:
			handleCmd = m.getCmdToActivateDeactivatedTask()
		}
		if handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case "ctrl+x":
		if m.activeView == taskListView && m.trackingActive {
			cmds = append(cmds, deleteActiveTL(m.db))
		}
	case "s":
		if m.activeView == taskListView {
			switch m.lastTrackingChange {
			case trackingFinished:
				if trackCmd := m.getCmdToStartTracking(); trackCmd != nil {
					cmds = append(cmds, trackCmd)
				}
			case trackingStarted:
				m.handleRequestToStopTracking()
			}
		}
	case "S":
		if m.activeView != taskListView {
			break
		}
		if quickSwitchCmd := m.getCmdToQuickSwitchTracking(); quickSwitchCmd != nil {
			cmds = append(cmds, quickSwitchCmd)
		}
	case "a":
		if m.activeView == taskListView {
			m.handleRequestToCreateTask()
		}
	case "c":
		if m.activeView == taskListView || m.activeView == inactiveTaskListView {
			m.handleCopyTaskSummary()
		}
	case "k":
		m.handleRequestToScrollVPUp()
	case "j":
		m.handleRequestToScrollVPDown()
	case "d":
		if m.activeView == taskLogView {
			m.handleRequestToViewTLDetails()
		}
	case "m":
		if m.activeView == taskLogView {
			if cmd := m.handleRequestToMoveTaskLog(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case "A":
		if m.activeView == taskListView {
			twoWeeksAgo := m.timeProvider.Now().AddDate(0, 0, -14)
			cmds = append(cmds, archiveStaleTasks(m.db, twoWeeksAgo))
		}
	case "?":
		m.lastView = m.activeView
		m.activeView = helpView
	}
	return cmds
}
