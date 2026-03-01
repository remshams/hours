package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) getCmdToActivateDeactivatedTask() tea.Cmd {
	if m.inactiveTasksList.IsFiltered() {
		m.message = errMsg(removeFilterMsg)
		return nil
	}

	task, ok := m.selectedInactiveTask()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	return updateTaskActiveStatus(m.db, task, true)
}
