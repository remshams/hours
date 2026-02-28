package ui

import (
	"database/sql"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhth/hours/internal/types"
)

// setupList applies the shared defaults to a list model: title, status-bar item
// name, quit-keybinding, help, title style, and page-navigation keybindings.
// filteringEnabled controls whether the list supports filtering.
func setupList(l *list.Model, title, singularItem, pluralItem string, bgColor lipgloss.Color, fgColor lipgloss.Color, filteringEnabled bool) {
	l.Title = title
	l.SetStatusBarItemName(singularItem, pluralItem)
	if !filteringEnabled {
		l.SetFilteringEnabled(false)
	}
	l.DisableQuitKeybindings()
	l.SetShowHelp(false)
	l.Styles.Title = l.Styles.Title.
		Foreground(fgColor).
		Background(bgColor).
		Bold(true)
	l.KeyMap.PrevPage.SetKeys("left", "h", "pgup")
	l.KeyMap.NextPage.SetKeys("right", "l", "pgdown")
}

const (
	tlCommentLengthLimit = 3000
	textInputWidth       = 80
)

func InitialModel(db *sql.DB,
	style Style,
	timeProvider types.TimeProvider,
	debug bool,
	logFramesCfg logFramesConfig,
) Model {
	var activeTaskItems []list.Item
	var inactiveTaskItems []list.Item
	var tasklogListItems []list.Item

	tLInputs := make([]textinput.Model, 2)
	tLInputs[entryBeginTS] = textinput.New()
	tLInputs[entryBeginTS].Placeholder = "09:30"
	tLInputs[entryBeginTS].CharLimit = len(timeFormat)
	tLInputs[entryBeginTS].Width = 30

	tLInputs[entryEndTS] = textinput.New()
	tLInputs[entryEndTS].Placeholder = "12:30pm"
	tLInputs[entryEndTS].CharLimit = len(timeFormat)
	tLInputs[entryEndTS].Width = 30

	tLCommentInput := textarea.New()
	tLCommentInput.Placeholder = `Task log comment goes here.

This can be used to record details about your work on this task.`
	tLCommentInput.CharLimit = tlCommentLengthLimit
	tLCommentInput.SetWidth(textInputWidth)
	tLCommentInput.SetHeight(10)
	tLCommentInput.ShowLineNumbers = false
	tLCommentInput.Prompt = "  ┃ "

	taskInputs := make([]textinput.Model, 1)
	taskInputs[summaryField] = textinput.New()
	taskInputs[summaryField].Placeholder = "task summary goes here"
	taskInputs[summaryField].Focus()
	taskInputs[summaryField].CharLimit = 100
	taskInputs[entryBeginTS].Width = textInputWidth

	m := Model{
		db:           db,
		style:        style,
		timeProvider: timeProvider,
		activeTasksList: list.New(activeTaskItems,
			newItemDelegate(style.listItemTitleColor,
				style.listItemDescColor,
				lipgloss.Color(style.theme.ActiveTasks),
			), listWidth, 0),
		inactiveTasksList: list.New(inactiveTaskItems,
			newItemDelegate(style.listItemTitleColor,
				style.listItemDescColor,
				lipgloss.Color(style.theme.InactiveTasks),
			), listWidth, 0),
		taskMap:      make(map[int]*types.Task),
		taskIndexMap: make(map[int]int),
		taskLogList: list.New(tasklogListItems,
			newItemDelegate(style.listItemTitleColor,
				style.listItemDescColor,
				lipgloss.Color(style.theme.TaskLogList),
			), listWidth, 0),
		showHelpIndicator: true,
		tLInputs:          tLInputs,
		tLCommentInput:    tLCommentInput,
		taskInputs:        taskInputs,
		debug:             debug,
		logFramesCfg:      logFramesCfg,
	}
	titleFG := lipgloss.Color(style.theme.TitleForeground)
	setupList(&m.activeTasksList, "Tasks", "task", "tasks", lipgloss.Color(style.theme.ActiveTasks), titleFG, true)
	setupList(&m.taskLogList, "Task Logs (last 50)", "entry", "entries", lipgloss.Color(style.theme.TaskLogList), titleFG, false)
	setupList(&m.inactiveTasksList, "Inactive Tasks", "task", "tasks", lipgloss.Color(style.theme.InactiveTasks), titleFG, true)

	m.targetTasksList = list.New([]list.Item{},
		newItemDelegate(style.listItemTitleColor,
			style.listItemDescColor,
			lipgloss.Color(style.theme.ActiveTasks),
		), listWidth, 0)
	setupList(&m.targetTasksList, "Select Target Task", "task", "tasks", lipgloss.Color(style.theme.ActiveTasks), titleFG, false)

	return m
}

func initialRecordsModel(
	kind recordsKind,
	db *sql.DB,
	style Style,
	timeProvider types.TimeProvider,
	dateRange types.DateRange,
	period string,
	taskStatus types.TaskStatus,
	plain bool,
	initialData string,
) recordsModel {
	return recordsModel{
		kind:         kind,
		db:           db,
		style:        style,
		timeProvider: timeProvider,
		dateRange:    dateRange,
		period:       period,
		taskStatus:   taskStatus,
		plain:        plain,
		report:       initialData,
	}
}
