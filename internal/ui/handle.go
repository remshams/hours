package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	c "github.com/dhth/hours/internal/common"
	"github.com/dhth/hours/internal/types"
)

// commentPtrFromInput returns a pointer to the trimmed textarea value, or nil if it is empty.
func commentPtrFromInput(input textarea.Model) *string {
	v := strings.TrimSpace(input.Value())
	if v == "" {
		return nil
	}
	return &v
}

const (
	genericErrorMsg               = "Something went wrong"
	removeFilterMsg               = "Remove filter first"
	beginTsCannotBeInTheFutureMsg = "Begin timestamp cannot be in the future"
)

var suggestReloadingMsg = fmt.Sprintf("Something went wrong, please restart hours; let %s know about this error via %s.", c.Author, c.RepoIssuesURL)

func (m *Model) handleRequestToGoBackOrQuit() bool {
	var shouldQuit bool
	switch m.activeView {
	case taskListView:
		fs := m.activeTasksList.FilterState()
		if fs == list.Filtering || fs == list.FilterApplied {
			m.activeTasksList.ResetFilter()
		} else {
			shouldQuit = true
		}
	case taskLogView:
		fs := m.taskLogList.FilterState()
		if fs == list.Filtering || fs == list.FilterApplied {
			m.taskLogList.ResetFilter()
		} else {
			m.activeView = taskListView
		}
	case taskLogDetailsView:
		m.activeView = taskLogView
	case inactiveTaskListView:
		fs := m.inactiveTasksList.FilterState()
		if fs == list.Filtering || fs == list.FilterApplied {
			m.inactiveTasksList.ResetFilter()
		} else {
			m.activeView = taskLogView
		}
	case helpView:
		m.activeView = m.lastView
	case moveTaskLogView:
		m.activeView = taskLogView
		m.targetTasksList.ResetFilter()
	}

	return shouldQuit
}

func (m *Model) getCmdToReloadData() tea.Cmd {
	var cmd tea.Cmd
	switch m.activeView {
	case taskListView:
		cmd = fetchTasks(m.db, true)
	case taskLogView:
		cmd = fetchTLS(m.db, nil)
		m.taskLogList.ResetSelected()
	case inactiveTaskListView:
		cmd = fetchTasks(m.db, false)
		m.inactiveTasksList.ResetSelected()
	}

	return cmd
}

func (m *Model) handleRequestToScrollVPUp() {
	switch m.activeView {
	case helpView:
		if m.helpVP.AtTop() {
			return
		}
		m.helpVP.ScrollUp(viewPortMoveLineCount)
	case taskLogDetailsView:
		if m.tLDetailsVP.AtTop() {
			return
		}
		m.tLDetailsVP.ScrollUp(viewPortMoveLineCount)
	default:
		return
	}
}

func (m *Model) handleRequestToScrollVPDown() {
	switch m.activeView {
	case helpView:
		if m.helpVP.AtBottom() {
			return
		}
		m.helpVP.ScrollDown(viewPortMoveLineCount)
	case taskLogDetailsView:
		if m.tLDetailsVP.AtBottom() {
			return
		}
		m.tLDetailsVP.ScrollDown(viewPortMoveLineCount)
	default:
		return
	}
}

func (m *Model) handleWindowResizing(msg tea.WindowSizeMsg) {
	w, h := m.style.list.GetFrameSize()

	m.terminalWidth = msg.Width
	m.terminalHeight = msg.Height

	if msg.Width < minWidthNeeded || msg.Height < minHeightNeeded {
		if m.activeView != insufficientDimensionsView {
			m.lastViewBeforeInsufficientDims = m.activeView
			m.activeView = insufficientDimensionsView
		}
		return
	}

	if m.activeView == insufficientDimensionsView {
		m.activeView = m.lastViewBeforeInsufficientDims
	}

	m.taskLogList.SetWidth(msg.Width - w)
	m.taskLogList.SetHeight(msg.Height - h - 2)

	m.activeTasksList.SetWidth(msg.Width - w)
	m.activeTasksList.SetHeight(msg.Height - h - 2)

	m.inactiveTasksList.SetWidth(msg.Width - w)
	m.inactiveTasksList.SetHeight(msg.Height - h - 2)

	m.targetTasksList.SetWidth(msg.Width - w)
	m.targetTasksList.SetHeight(msg.Height - h - 2)

	if !m.helpVPReady {
		m.helpVP = viewport.New(msg.Width-4, m.terminalHeight-7)
		m.helpVP.SetContent(getHelpText(m.style))
		m.helpVP.KeyMap.Up.SetEnabled(false)
		m.helpVP.KeyMap.Down.SetEnabled(false)
		m.helpVPReady = true
	} else {
		m.helpVP.Height = m.terminalHeight - 7
		m.helpVP.Width = msg.Width - 4
	}

	if !m.tLDetailsVPReady {
		m.tLDetailsVP = viewport.New(msg.Width-4, m.terminalHeight-6)
		m.tLDetailsVP.KeyMap.Up.SetEnabled(false)
		m.tLDetailsVP.KeyMap.Down.SetEnabled(false)
		m.tLDetailsVPReady = true
	} else {
		m.tLDetailsVP.Height = m.terminalHeight - 6
		m.tLDetailsVP.Width = msg.Width - 4
	}
}

func (m *Model) handleTasksFetchedMsg(msg tasksFetchedMsg) tea.Cmd {
	if msg.err != nil {
		m.message = errMsg("Error fetching tasks : " + msg.err.Error())
		return nil
	}

	var cmd tea.Cmd
	switch msg.active {
	case true:
		m.taskMap = make(map[int]*types.Task)
		m.taskIndexMap = make(map[int]int)
		tasks := make([]list.Item, len(msg.tasks))
		for i, task := range msg.tasks {
			task.UpdateListTitle()
			task.UpdateListDesc(m.timeProvider)
			tasks[i] = &task
			m.taskMap[task.ID] = &task
			m.taskIndexMap[task.ID] = i
		}
		m.activeTasksList.SetItems(tasks)
		m.activeTasksList.Title = "Tasks"
		m.tasksFetched = true
		cmd = fetchActiveTask(m.db)

	case false:
		inactiveTasks := make([]list.Item, len(msg.tasks))
		for i, inactiveTask := range msg.tasks {
			inactiveTask.UpdateListTitle()
			inactiveTask.UpdateListDesc(m.timeProvider)
			inactiveTasks[i] = &inactiveTask
		}
		m.inactiveTasksList.SetItems(inactiveTasks)
	}

	return cmd
}

func (m *Model) handleManualTLInsertedMsg(msg manualTLInsertedMsg) []tea.Cmd {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		return nil
	}

	task, ok := m.taskMap[msg.taskID]

	var cmds []tea.Cmd
	if ok {
		cmds = append(cmds, updateTaskRep(m.db, task))
	}
	cmds = append(cmds, fetchTLS(m.db, nil))

	return cmds
}

func (m *Model) handleSavedTLEditedMsg(msg savedTLEditedMsg) []tea.Cmd {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		return nil
	}

	task, ok := m.taskMap[msg.taskID]

	var cmds []tea.Cmd
	if ok {
		cmds = append(cmds, updateTaskRep(m.db, task))
	}
	cmds = append(cmds, fetchTLS(m.db, &msg.tlID))

	return cmds
}

func (m *Model) handleTLSFetchedMsg(msg tLsFetchedMsg) {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		return
	}

	items := make([]list.Item, len(msg.entries))
	var indexToFocusOn *int
	var indexToFocusOnFound bool
	for i, e := range msg.entries {
		e.UpdateListTitle()
		e.UpdateListDesc(m.timeProvider)
		items[i] = e
		if !indexToFocusOnFound && msg.tlIDToFocusOn != nil && e.ID == *msg.tlIDToFocusOn {
			indexToFocusOn = &i
			indexToFocusOnFound = true
		}
	}
	m.taskLogList.SetItems(items)

	if indexToFocusOn != nil {
		m.taskLogList.Select(*indexToFocusOn)
	} else {
		m.taskLogList.Select(0)
	}
}

func (m *Model) handleActiveTaskFetchedMsg(msg activeTaskFetchedMsg) {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		return
	}

	if msg.noneActive {
		m.lastTrackingChange = trackingFinished
		return
	}

	m.lastTrackingChange = trackingStarted
	m.activeTaskID = msg.activeTask.TaskID
	m.activeTLBeginTS = msg.activeTask.CurrentLogBeginTS
	m.activeTLComment = msg.activeTask.CurrentLogComment

	activeTask, ok := m.taskMap[m.activeTaskID]
	if ok {
		activeTask.TrackingActive = true
		activeTask.UpdateListTitle()

		// go to tracked item on startup
		activeIndex, aOk := m.taskIndexMap[msg.activeTask.TaskID]
		if aOk {
			m.activeTasksList.Select(activeIndex)
		}
	}
	m.trackingActive = true
}

func (m *Model) handleTrackingToggledMsg(msg trackingToggledMsg) []tea.Cmd {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		m.trackingActive = false
		return nil
	}

	m.changesLocked = false

	task, ok := m.taskMap[msg.taskID]

	if !ok {
		m.message = errMsg(genericErrorMsg)
		return nil
	}

	var cmds []tea.Cmd
	switch msg.finished {
	case true:
		m.lastTrackingChange = trackingFinished
		task.TrackingActive = false
		m.activeTLComment = nil
		m.trackingActive = false
		m.activeTaskID = -1
		cmds = append(cmds, updateTaskRep(m.db, task))
		cmds = append(cmds, fetchTLS(m.db, nil))
	case false:
		m.lastTrackingChange = trackingStarted
		task.TrackingActive = true
		m.trackingActive = true
		m.activeTaskID = msg.taskID
	}

	task.UpdateListTitle()

	return cmds
}

func (m *Model) handleActiveTLSwitchedMsg(msg activeTLSwitchedMsg) tea.Cmd {
	if msg.err != nil {
		m.message = errMsg(msg.err.Error())
		return nil
	}

	lastActiveTask, ok := m.taskMap[msg.lastActiveTaskID]

	if !ok {
		m.message = errMsg(suggestReloadingMsg)
		return nil
	}

	lastActiveTask.TrackingActive = false
	lastActiveTask.UpdateListTitle()

	currentlyActiveTask, ok := m.taskMap[msg.currentlyActiveTaskID]

	if !ok {
		m.message = errMsg(suggestReloadingMsg)
		return nil
	}
	currentlyActiveTask.TrackingActive = true
	currentlyActiveTask.UpdateListTitle()

	m.activeTLComment = nil
	m.activeTaskID = msg.currentlyActiveTaskID
	m.activeTLBeginTS = msg.ts

	return fetchTLS(m.db, nil)
}

func (m *Model) handleTLDeleted(msg tLDeletedMsg) []tea.Cmd {
	if msg.err != nil {
		m.message = errMsg("Error deleting entry: " + msg.err.Error())
		return nil
	}

	var cmds []tea.Cmd
	task, ok := m.taskMap[msg.entry.TaskID]
	if ok {
		cmds = append(cmds, updateTaskRep(m.db, task))
	}
	cmds = append(cmds, fetchTLS(m.db, nil))

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

func (m *Model) handleActiveTLDeletedMsg(msg activeTaskLogDeletedMsg) {
	if msg.err != nil {
		m.message = errMsg(fmt.Sprintf("Error deleting active log entry: %s", msg.err))
		return
	}

	activeTask, ok := m.taskMap[m.activeTaskID]
	if !ok {
		m.message = errMsg(genericErrorMsg)
		return
	}

	activeTask.TrackingActive = false
	activeTask.UpdateListTitle()
	m.lastTrackingChange = trackingFinished
	m.trackingActive = false
	m.activeTLComment = nil
	m.activeTaskID = -1
}

// selectedActiveTask returns the currently selected item in the active tasks list cast to *types.Task.
func (m *Model) selectedActiveTask() (*types.Task, bool) {
	task, ok := m.activeTasksList.SelectedItem().(*types.Task)
	return task, ok
}

// selectedInactiveTask returns the currently selected item in the inactive tasks list cast to *types.Task.
func (m *Model) selectedInactiveTask() (*types.Task, bool) {
	task, ok := m.inactiveTasksList.SelectedItem().(*types.Task)
	return task, ok
}

// selectedTargetTask returns the currently selected item in the target tasks list cast to *types.Task.
func (m *Model) selectedTargetTask() (*types.Task, bool) {
	task, ok := m.targetTasksList.SelectedItem().(*types.Task)
	return task, ok
}

// selectedTaskLogEntry returns the currently selected item in the task log list cast to types.TaskLogEntry.
func (m *Model) selectedTaskLogEntry() (types.TaskLogEntry, bool) {
	entry, ok := m.taskLogList.SelectedItem().(types.TaskLogEntry)
	return entry, ok
}
