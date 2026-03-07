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

	return m.getCmdToStartTrackingTask(task.ID)
}

func (m *Model) getCmdToStartTrackingTask(taskID int) tea.Cmd {
	return m.getCmdToStartTrackingTaskAt(taskID, time.Time{})
}

func (m *Model) normalizedTrackingTS(ts time.Time) time.Time {
	if ts.IsZero() {
		ts = m.timeProvider.Now()
	}

	return ts.Truncate(time.Second)
}

func (m *Model) getCmdToStartTrackingTaskAt(taskID int, startedAt time.Time) tea.Cmd {
	if _, ok := m.taskMap[taskID]; !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	m.autoResumeNoticePending = false
	m.autoResumePauseDuration = 0
	m.autoStopTaskID = -1
	m.autoResumeTaskID = -1
	m.autoResumeAt = time.Time{}
	m.changesLocked = true
	m.activeTLBeginTS = m.normalizedTrackingTS(startedAt)
	return toggleTracking(m.db, taskID, m.activeTLBeginTS, m.activeTLEndTS, nil)
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
		return m.getCmdToStartTrackingTask(task.ID)
	}

	return quickSwitchActiveIssue(m.db, task.ID, m.timeProvider.Now())
}

func (m *Model) getCmdToAutoStopTracking() tea.Cmd {
	return m.getCmdToAutoStopTrackingAt(time.Time{})
}

func (m *Model) getCmdToAutoStopTrackingAt(stoppedAt time.Time) tea.Cmd {
	if !m.trackingActive || m.activeTaskID < 0 {
		return nil
	}

	m.autoResumeNoticePending = false
	m.autoResumePauseDuration = 0
	m.autoStopTaskID = m.activeTaskID
	m.autoResumeTaskID = -1
	m.autoResumeAt = time.Time{}
	m.changesLocked = true
	m.activeTLEndTS = m.normalizedTrackingTS(stoppedAt)

	return toggleTracking(m.db, m.activeTaskID, m.activeTLBeginTS, m.activeTLEndTS, m.activeTLComment)
}

func (m *Model) getCmdToResumeAutoStoppedTask() tea.Cmd {
	return m.getCmdToResumeAutoStoppedTaskAt(time.Time{})
}

func (m *Model) getCmdToResumeAutoStoppedTaskAt(resumedAt time.Time) tea.Cmd {
	if m.trackingActive || m.autoResumeTaskID < 0 {
		return nil
	}

	taskID := m.autoResumeTaskID
	if resumedAt.IsZero() {
		resumedAt = m.autoResumeAt
	}
	cmd := m.getCmdToStartTrackingTaskAt(taskID, resumedAt)
	if cmd != nil {
		pauseDuration := time.Duration(0)
		if !m.activeTLEndTS.IsZero() && !m.activeTLBeginTS.Before(m.activeTLEndTS) {
			pauseDuration = m.activeTLBeginTS.Sub(m.activeTLEndTS)
		}
		m.autoResumeNoticePending = true
		m.autoResumePauseDuration = pauseDuration
		m.autoResumeTaskID = -1
		m.autoResumeAt = time.Time{}
	}

	return cmd
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
