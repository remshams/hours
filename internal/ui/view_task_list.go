package ui

import (
	"os"
	"strings"
	"time"

	"github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
)

func (m *Model) goToActiveTask() {
	if m.activeView != taskListView {
		return
	}

	if !m.trackingActive {
		m.message = errMsg("Nothing is being tracked right now")
		return
	}

	if m.activeTasksList.IsFiltered() {
		m.activeTasksList.ResetFilter()
	}
	activeIndex, ok := m.taskIndexMap[m.activeTaskID]
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return
	}

	m.activeTasksList.Select(activeIndex)
}

func (m *Model) handleRequestToCreateTask() {
	if m.activeTasksList.IsFiltered() {
		m.message = errMsg(removeFilterMsg)
		return
	}

	m.activeView = taskInputView
	m.taskInputFocussedField = summaryField
	m.taskInputs[summaryField].Focus()
	m.taskMgmtContext = taskCreateCxt
}

func (m *Model) handleRequestToUpdateTask() {
	if m.activeTasksList.IsFiltered() {
		m.message = errMsg(removeFilterMsg)
		return
	}

	task, ok := m.selectedActiveTask()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return
	}

	m.activeView = taskInputView
	m.taskInputFocussedField = summaryField
	m.taskInputs[summaryField].Focus()
	m.taskInputs[summaryField].SetValue(task.Summary)
	m.taskMgmtContext = taskUpdateCxt
}

func (m *Model) getCmdToCreateOrUpdateTask() tea.Cmd {
	if strings.TrimSpace(m.taskInputs[summaryField].Value()) == "" {
		m.message = errMsg("Task summary cannot be empty")
		return nil
	}

	var cmd tea.Cmd
	switch m.taskMgmtContext {
	case taskCreateCxt:
		cmd = createTask(m.db, m.taskInputs[summaryField].Value())
		m.taskInputs[summaryField].SetValue("")
	case taskUpdateCxt:
		selectedTask, ok := m.selectedActiveTask()
		if !ok {
			m.message = errMsg("Something went wrong")
			return nil
		}
		cmd = updateTask(m.db, selectedTask, m.taskInputs[summaryField].Value())
		m.taskInputs[summaryField].SetValue("")
	}

	m.activeView = taskListView
	return cmd
}

func (m *Model) getCmdToStartTracking() tea.Cmd {
	task, ok := m.selectedActiveTask()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	m.changesLocked = true
	m.activeTLBeginTS = m.timeProvider.Now().Truncate(time.Second)
	return toggleTracking(m.db, task.ID, m.activeTLBeginTS, m.activeTLEndTS, nil)
}

func (m *Model) getCmdToQuickSwitchTracking() tea.Cmd {
	task, ok := m.selectedActiveTask()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	if task.ID == m.activeTaskID {
		return nil
	}

	if !m.trackingActive {
		m.changesLocked = true
		m.activeTLBeginTS = m.timeProvider.Now().Truncate(time.Second)
		return toggleTracking(m.db,
			task.ID,
			m.activeTLBeginTS,
			m.activeTLEndTS,
			nil,
		)
	}

	return quickSwitchActiveIssue(m.db, task.ID, m.timeProvider.Now())
}

func (m *Model) getCmdToDeactivateTask() tea.Cmd {
	if m.activeTasksList.IsFiltered() {
		m.message = errMsg(removeFilterMsg)
		return nil
	}

	if m.trackingActive {
		m.message = errMsg("Cannot deactivate a task being tracked; stop tracking and try again.")
		return nil
	}

	task, ok := m.selectedActiveTask()
	if !ok {
		m.message = errMsg(msgCouldntSelectATask)
		return nil
	}

	return updateTaskActiveStatus(m.db, task, false)
}

func (m *Model) handleCopyTaskSummary() {
	var selectedTask *types.Task
	var ok bool

	switch m.activeView {
	case taskListView:
		selectedTask, ok = m.selectedActiveTask()
	case inactiveTaskListView:
		selectedTask, ok = m.selectedInactiveTask()
	default:
		return
	}

	if !ok || selectedTask == nil {
		m.message = errMsg("No task selected")
		return
	}

	_, _ = osc52.New(selectedTask.Summary).WriteTo(os.Stderr)
	m.message = infoMsg("Copied to clipboard")
}
