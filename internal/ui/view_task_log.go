package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
)

func (m *Model) getCmdToDeleteTL() tea.Cmd {
	entry, ok := m.selectedTaskLogEntry()
	if !ok {
		m.message = errMsg("Couldn't delete task log entry")
		return nil
	}
	return deleteTL(m.db, &entry)
}

func (m *Model) handleRequestToEditSavedTL() {
	if len(m.taskLogList.Items()) == 0 {
		return
	}

	tl, ok := m.selectedTaskLogEntry()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return
	}

	m.activeView = editSavedTLView
	m.tasklogSaveType = tasklogUpdate

	beginTimeStr := tl.BeginTS.Format(timeFormat)
	endTimeStr := tl.EndTS.Format(timeFormat)

	var comment string
	if tl.Comment != nil {
		comment = *tl.Comment
	}

	m.tLInputs[entryBeginTS].SetValue(beginTimeStr)
	m.tLInputs[entryEndTS].SetValue(endTimeStr)
	m.tLCommentInput.SetValue(comment)

	m.blurTLTrackingInputs()
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].Focus()
}

func (m *Model) handleRequestToMoveTaskLog() tea.Cmd {
	if m.taskLogList.IsFiltered() {
		m.message = errMsg(removeFilterMsg)
		return nil
	}

	entry, ok := m.selectedTaskLogEntry()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	// Store the log entry details
	m.moveTLID = entry.ID
	m.moveOldTaskID = entry.TaskID
	m.moveSecsSpent = entry.SecsSpent

	// Initialize target list with active tasks, excluding current parent
	items := m.activeTasksList.Items()
	targetItems := []list.Item{}
	for i := range items {
		task, ok := items[i].(*types.Task)
		if !ok {
			continue
		}
		// Exclude the current parent task
		if task.ID != entry.TaskID {
			targetItems = append(targetItems, task)
		}
	}
	if len(targetItems) == 0 {
		m.message = errMsg("No other active tasks to move this log to")
		return nil
	}

	m.targetTasksList.SetItems(targetItems)

	m.activeView = moveTaskLogView
	return nil
}

func (m *Model) handleTargetTaskSelection() tea.Cmd {
	task, ok := m.selectedTargetTask()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	return moveTaskLog(m.db, m.moveTLID, m.moveOldTaskID, task.ID, m.moveSecsSpent)
}

func (m *Model) handleRequestToViewTLDetails() {
	if len(m.taskLogList.Items()) == 0 {
		return
	}

	tl, ok := m.selectedTaskLogEntry()
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return
	}

	var taskDetails string
	task, tOk := m.taskMap[tl.TaskID]
	if tOk {
		taskDetails = task.Summary
	}

	timeSpentStr := types.HumanizeDuration(tl.SecsSpent)

	details := fmt.Sprintf(`Task: %s

%s → %s (%s)

---

%s
`, taskDetails,
		tl.BeginTS.Format(timeFormat),
		tl.EndTS.Format(timeFormat),
		timeSpentStr,
		tl.GetComment())

	m.tLDetailsVP.SetContent(details)
	m.activeView = taskLogDetailsView
}
