package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
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
		}
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

// handleMsg handles all typed (non-key) messages produced by async commands.
func (m *Model) handleMsg(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case taskCreatedMsg:
		if msg.err != nil {
			m.message = errMsg(fmt.Sprintf("Error creating task: %s", msg.err))
		} else {
			cmds = append(cmds, fetchTasks(m.db, true))
		}
	case staleTasksArchivedMsg:
		if msg.err != nil {
			m.message = errMsg(fmt.Sprintf("Error archiving tasks: %s", msg.err))
		} else {
			m.message = infoMsg(fmt.Sprintf("Archived %d tasks", msg.count))
			cmds = append(cmds, fetchTasks(m.db, true))
			cmds = append(cmds, fetchTasks(m.db, false))
		}
	case taskUpdatedMsg:
		if msg.err != nil {
			m.message = errMsg(fmt.Sprintf("Error updating task: %s", msg.err))
		} else {
			msg.tsk.Summary = msg.summary
			msg.tsk.UpdateListTitle()
		}
	case tasksFetchedMsg:
		if handleCmd := m.handleTasksFetchedMsg(msg); handleCmd != nil {
			cmds = append(cmds, handleCmd)
		}
	case activeTLUpdatedMsg:
		if msg.err != nil {
			m.message = errMsg(msg.err.Error())
		} else {
			m.activeTLBeginTS = msg.beginTS
			m.activeTLComment = msg.comment
		}
	case manualTLInsertedMsg:
		if handleCmds := m.handleManualTLInsertedMsg(msg); handleCmds != nil {
			cmds = append(cmds, handleCmds...)
		}
	case savedTLEditedMsg:
		if handleCmds := m.handleSavedTLEditedMsg(msg); handleCmds != nil {
			cmds = append(cmds, handleCmds...)
		}
	case tLsFetchedMsg:
		m.handleTLSFetchedMsg(msg)
	case activeTaskFetchedMsg:
		m.handleActiveTaskFetchedMsg(msg)
	case trackingToggledMsg:
		if updateCmds := m.handleTrackingToggledMsg(msg); updateCmds != nil {
			cmds = append(cmds, updateCmds...)
		}
	case activeTLSwitchedMsg:
		if updateCmd := m.handleActiveTLSwitchedMsg(msg); updateCmd != nil {
			cmds = append(cmds, updateCmd)
		}
	case taskRepUpdatedMsg:
		if msg.err != nil {
			m.message = errMsg(fmt.Sprintf("Error updating task status: %s", msg.err))
		} else {
			msg.tsk.UpdateListDesc(m.timeProvider)
		}
	case tLDeletedMsg:
		if updateCmds := m.handleTLDeleted(msg); updateCmds != nil {
			cmds = append(cmds, updateCmds...)
		}
	case taskLogMovedMsg:
		if msg.err != nil {
			m.message = errMsg(fmt.Sprintf("Error moving task log: %s", msg.err))
		} else {
			cmds = append(cmds, fetchTLS(m.db, nil))
			cmds = append(cmds, fetchTasks(m.db, true))
		}
		m.activeView = taskLogView
		m.targetTasksList.ResetFilter()
	case activeTaskLogDeletedMsg:
		m.handleActiveTLDeletedMsg(msg)
	case taskActiveStatusUpdatedMsg:
		if msg.err != nil {
			m.message = errMsg("Error updating task's active status: " + msg.err.Error())
		} else {
			cmds = append(cmds, fetchTasks(m.db, true))
			cmds = append(cmds, fetchTasks(m.db, false))
		}
	case hideHelpMsg:
		m.showHelpIndicator = false
	}
	return cmds
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

func (m recordsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case ctrlC, "q", escape:
			m.quitting = true
			return m, tea.Quit
		case "left", "h":
			if !m.busy {
				var dr types.DateRange

				switch m.period {
				case types.TimePeriodWeek:
					weekday := m.dateRange.Start.Weekday()
					offset := (7 + weekday - time.Monday) % 7
					startOfPrevWeek := m.dateRange.Start.AddDate(0, 0, -int(offset+7))
					dr.Start = time.Date(startOfPrevWeek.Year(), startOfPrevWeek.Month(), startOfPrevWeek.Day(), 0, 0, 0, 0, startOfPrevWeek.Location())
				default:
					dr.Start = m.dateRange.Start.AddDate(0, 0, -m.dateRange.NumDays)
				}

				dr.NumDays = m.dateRange.NumDays
				dr.End = dr.Start.AddDate(0, 0, m.dateRange.NumDays)
				cmds = append(cmds, getRecordsData(m.kind, m.db, m.style, dr, m.taskStatus, m.plain))
				m.busy = true
			}
		case "right", "l":
			if !m.busy {
				var dr types.DateRange

				switch m.period {
				case types.TimePeriodWeek:
					weekday := m.dateRange.Start.Weekday()
					offset := (7 + weekday - time.Monday) % 7
					startOfNextWeek := m.dateRange.Start.AddDate(0, 0, 7-int(offset))
					dr.Start = time.Date(startOfNextWeek.Year(), startOfNextWeek.Month(), startOfNextWeek.Day(), 0, 0, 0, 0, startOfNextWeek.Location())
					dr.NumDays = 7

				default:
					dr.Start = m.dateRange.Start.AddDate(0, 0, 1*(m.dateRange.NumDays))
				}

				dr.NumDays = m.dateRange.NumDays
				dr.End = dr.Start.AddDate(0, 0, dr.NumDays)
				cmds = append(cmds, getRecordsData(m.kind, m.db, m.style, dr, m.taskStatus, m.plain))
				m.busy = true
			}
		case "ctrl+t":
			if !m.busy {
				var dr types.DateRange

				now := m.timeProvider.Now()
				switch m.period {
				case types.TimePeriodWeek:
					weekday := now.Weekday()
					offset := (7 + weekday - time.Monday) % 7
					startOfWeek := now.AddDate(0, 0, -int(offset))
					dr.Start = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())
					dr.NumDays = 7
				default:
					nDaysBack := now.AddDate(0, 0, -1*(m.dateRange.NumDays-1))

					dr.Start = time.Date(nDaysBack.Year(), nDaysBack.Month(), nDaysBack.Day(), 0, 0, 0, 0, nDaysBack.Location())
				}

				dr.NumDays = m.dateRange.NumDays
				dr.End = dr.Start.AddDate(0, 0, dr.NumDays)
				cmds = append(cmds, getRecordsData(m.kind, m.db, m.style, dr, m.taskStatus, m.plain))
				m.busy = true
			}
		}
	case recordsDataFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.quitting = true
			return m, tea.Quit
		}

		m.dateRange = msg.dateRange
		m.report = msg.report
		m.busy = false
	}
	return m, tea.Batch(cmds...)
}
