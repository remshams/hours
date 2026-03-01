package ui

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
)

// handleFormKeys handles key events that are only meaningful while a form view
// is active: enter/ctrl+s (submit), esc (cancel), tab/shift+tab (field
// navigation), and j/k/J/K/h/l (time-shifting).  Returns exitEarly=true when
// the caller should return immediately after processing.
func (m *Model) handleFormKeys(keyMsg tea.KeyMsg) (exitEarly bool, cmds []tea.Cmd) {
	switch keyMsg.String() {
	case enter, "ctrl+s":
		var bail bool
		if keyMsg.String() == enter {
			switch m.activeView {
			case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
				if m.trackingFocussedField == entryComment {
					bail = true
				}
			}
		}

		if bail {
			return false, nil
		}

		var updateCmd tea.Cmd
		switch m.activeView {
		case taskInputView:
			updateCmd = m.getCmdToCreateOrUpdateTask()
		case editActiveTLView:
			updateCmd = m.getCmdToUpdateActiveTL()
		case finishActiveTLView:
			updateCmd = m.getCmdToFinishTrackingActiveTL()
		case manualTasklogEntryView, editSavedTLView:
			updateCmd = m.getCmdToCreateOrEditTL()
		case moveTaskLogView:
			if keyMsg.String() == enter {
				updateCmd = m.handleTargetTaskSelection()
			}
		}
		if updateCmd != nil {
			return true, []tea.Cmd{updateCmd}
		}

	case escape:
		switch m.activeView {
		case taskInputView, editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView, moveTaskLogView:
			m.handleEscapeInForms()
			return true, nil
		}

	case "tab":
		m.goForwardInView()

	case "shift+tab":
		m.goBackwardInView()

	case "k":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftBackward, types.ShiftMinute); err != nil {
				return true, nil
			}
		}

	case "j":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftForward, types.ShiftMinute); err != nil {
				return true, nil
			}
		}

	case "K":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftBackward, types.ShiftFiveMinutes); err != nil {
				return true, nil
			}
		}

	case "J":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftForward, types.ShiftFiveMinutes); err != nil {
				return true, nil
			}
		}

	case "h":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftBackward, types.ShiftDay); err != nil {
				return true, nil
			}
		case taskLogDetailsView:
			m.taskLogList.CursorUp()
			m.handleRequestToViewTLDetails()
			return true, nil
		}

	case "l":
		switch m.activeView {
		case editActiveTLView, finishActiveTLView, manualTasklogEntryView, editSavedTLView:
			if err := m.shiftTime(types.ShiftForward, types.ShiftDay); err != nil {
				return true, nil
			}
		case taskLogDetailsView:
			m.taskLogList.CursorDown()
			m.handleRequestToViewTLDetails()
			return true, nil
		}
	}

	return false, nil
}

func (m *Model) getCmdToUpdateActiveTL() tea.Cmd {
	beginTS, err := time.ParseInLocation(timeFormat, m.tLInputs[entryBeginTS].Value(), time.Local)
	if err != nil {
		m.message = errMsgQuick(err.Error())
		return nil
	}

	if beginTS.After(m.timeProvider.Now()) {
		m.message = errMsgQuick(beginTsCannotBeInTheFutureMsg)
		return nil
	}

	comment := commentPtrFromInput(m.tLCommentInput)

	m.activeView = taskListView
	return updateActiveTL(m.db, beginTS, comment)
}

func (m *Model) getCmdToFinishTrackingActiveTL() tea.Cmd {
	beginTS, endTS, err := types.ParseTaskLogTimes(m.tLInputs[entryBeginTS].Value(), m.tLInputs[entryEndTS].Value())
	if err != nil {
		m.message = errMsg(err.Error())
		return nil
	}

	m.activeTLBeginTS = beginTS
	m.activeTLEndTS = endTS

	comment := commentPtrFromInput(m.tLCommentInput)

	m.activeView = taskListView

	return toggleTracking(m.db, m.activeTaskID, m.activeTLBeginTS, m.activeTLEndTS, comment)
}

func (m *Model) getCmdToFinishActiveTL() tea.Cmd {
	now := m.timeProvider.Now().Truncate(time.Second)
	err := types.IsTaskLogDurationValid(m.activeTLBeginTS, now)

	if errors.Is(err, types.ErrDurationNotLongEnough) {
		m.message = infoMsg("Task log duration is too short to save; press <ctrl+x> if you want to discard it")
		return nil
	}

	if err != nil {
		m.message = errMsg(fmt.Sprintf("Error: %s", err.Error()))
		return nil
	}

	m.activeTLEndTS = now

	return toggleTracking(m.db, m.activeTaskID, m.activeTLBeginTS, m.activeTLEndTS, m.activeTLComment)
}

func (m *Model) getCmdToCreateOrEditTL() tea.Cmd {
	beginTS, endTS, err := types.ParseTaskLogTimes(m.tLInputs[entryBeginTS].Value(), m.tLInputs[entryEndTS].Value())
	if err != nil {
		m.message = errMsg(err.Error())
		return nil
	}

	comment := commentPtrFromInput(m.tLCommentInput)

	m.blurTLTrackingInputs()
	m.tLCommentInput.SetValue("")
	m.activeTLComment = nil

	var cmd tea.Cmd
	switch m.tasklogSaveType {
	case tasklogInsert:
		m.activeView = taskListView
		task, ok := m.selectedActiveTask()
		if !ok {
			m.message = errMsg(genericErrorMsg)
			return nil
		}
		cmd = insertManualTL(m.db, task.ID, beginTS, endTS, comment)
	case tasklogUpdate:
		m.activeView = taskLogView
		tl, ok := m.selectedTaskLogEntry()
		if !ok {
			m.message = errMsg(genericErrorMsg)
			return nil
		}
		cmd = editSavedTL(m.db, tl.ID, tl.TaskID, beginTS, endTS, comment)
	}

	return cmd
}

func (m *Model) handleRequestToEditActiveTL() {
	m.clearAllTaskLogInputs()
	m.activeView = editActiveTLView
	m.tLInputs[entryBeginTS].SetValue(m.activeTLBeginTS.Format(timeFormat))
	if m.activeTLComment != nil {
		m.tLCommentInput.SetValue(*m.activeTLComment)
	} else {
		m.tLCommentInput.SetValue("")
	}

	m.blurTLTrackingInputs()
	m.tLInputs[entryBeginTS].Focus()
	m.trackingFocussedField = entryBeginTS
}

func (m *Model) handleRequestToCreateManualTL() {
	m.clearAllTaskLogInputs()
	m.activeView = manualTasklogEntryView
	m.tasklogSaveType = tasklogInsert
	currentTime := m.timeProvider.Now()
	currentTimeStr := currentTime.Format(timeFormat)

	m.tLInputs[entryBeginTS].SetValue(currentTimeStr)
	m.tLInputs[entryEndTS].SetValue(currentTimeStr)

	m.blurTLTrackingInputs()
	m.trackingFocussedField = entryBeginTS
	m.tLInputs[entryBeginTS].Focus()
}

func (m *Model) handleRequestToStopTracking() {
	m.clearAllTaskLogInputs()
	m.activeView = finishActiveTLView
	m.activeTLEndTS = m.timeProvider.Now()

	beginTimeStr := m.activeTLBeginTS.Format(timeFormat)
	currentTimeStr := m.activeTLEndTS.Format(timeFormat)

	m.tLInputs[entryBeginTS].SetValue(beginTimeStr)
	m.tLInputs[entryEndTS].SetValue(currentTimeStr)
	if m.activeTLComment != nil {
		m.tLCommentInput.SetValue(*m.activeTLComment)
	}
	m.trackingFocussedField = entryComment

	m.blurTLTrackingInputs()
	m.tLCommentInput.Focus()
}

func (m *Model) handleEscapeInForms() {
	switch m.activeView {
	case taskInputView:
		m.activeView = taskListView
		for i := range m.taskInputs {
			m.taskInputs[i].SetValue("")
		}
	case editActiveTLView:
		m.tLInputs[entryBeginTS].SetValue("")
		m.activeView = taskListView
	case finishActiveTLView:
		m.activeView = taskListView
		m.tLCommentInput.SetValue("")
	case manualTasklogEntryView:
		if m.tasklogSaveType == tasklogInsert {
			m.activeView = taskListView
		}
	case editSavedTLView:
		m.activeView = taskLogView
	case moveTaskLogView:
		m.activeView = taskLogView
		m.targetTasksList.ResetFilter()
	}
}

func (m *Model) goForwardInView() {
	switch m.activeView {
	case taskListView:
		m.activeView = taskLogView
	case taskLogView:
		m.activeView = inactiveTaskListView
	case inactiveTaskListView:
		m.activeView = taskListView
	case editActiveTLView:
		switch m.trackingFocussedField {
		case entryBeginTS:
			m.trackingFocussedField = entryComment
			m.tLInputs[entryBeginTS].Blur()
			m.tLCommentInput.Focus()
		case entryComment:
			m.trackingFocussedField = entryBeginTS
			m.tLInputs[entryBeginTS].Focus()
			m.tLCommentInput.Blur()
		}
	case finishActiveTLView, manualTasklogEntryView, editSavedTLView:
		switch m.trackingFocussedField {
		case entryBeginTS:
			m.trackingFocussedField = entryEndTS
			m.tLInputs[entryBeginTS].Blur()
			m.tLInputs[entryEndTS].Focus()
		case entryEndTS:
			m.trackingFocussedField = entryComment
			m.tLInputs[entryEndTS].Blur()
			m.tLCommentInput.Focus()
		case entryComment:
			m.trackingFocussedField = entryBeginTS
			m.tLCommentInput.Blur()
			m.tLInputs[entryBeginTS].Focus()
		}
	}
}

func (m *Model) goBackwardInView() {
	switch m.activeView {
	case taskLogView:
		m.activeView = taskListView
	case taskListView:
		m.activeView = inactiveTaskListView
	case inactiveTaskListView:
		m.activeView = taskLogView
	case editActiveTLView:
		switch m.trackingFocussedField {
		case entryBeginTS:
			m.trackingFocussedField = entryComment
			m.tLCommentInput.Focus()
			m.tLInputs[entryBeginTS].Blur()
		case entryComment:
			m.trackingFocussedField = entryBeginTS
			m.tLInputs[entryBeginTS].Focus()
			m.tLCommentInput.Blur()
		}
	case finishActiveTLView, manualTasklogEntryView, editSavedTLView:
		switch m.trackingFocussedField {
		case entryBeginTS:
			m.trackingFocussedField = entryComment
			m.tLCommentInput.Focus()
			m.tLInputs[entryBeginTS].Blur()
		case entryEndTS:
			m.trackingFocussedField = entryBeginTS
			m.tLInputs[entryBeginTS].Focus()
			m.tLInputs[entryEndTS].Blur()
		case entryComment:
			m.trackingFocussedField = entryEndTS
			m.tLInputs[entryEndTS].Focus()
			m.tLCommentInput.Blur()
		}
	}
}

func (m *Model) shiftTime(direction types.TimeShiftDirection, duration types.TimeShiftDuration) error {
	switch m.trackingFocussedField {
	case entryBeginTS, entryEndTS:
		ts, err := time.ParseInLocation(timeFormat, m.tLInputs[m.trackingFocussedField].Value(), time.Local)
		if err != nil {
			return err
		}

		newTs := types.GetShiftedTime(ts, direction, duration)

		m.tLInputs[m.trackingFocussedField].SetValue(newTs.Format(timeFormat))
	}

	return nil
}

func (m *Model) clearAllTaskLogInputs() {
	for i := range m.tLInputs {
		m.tLInputs[i].SetValue("")
	}
	m.tLCommentInput.SetValue("")
}
