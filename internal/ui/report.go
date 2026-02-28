package ui

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	pers "github.com/dhth/hours/internal/persistence"
	"github.com/dhth/hours/internal/types"
	"github.com/dhth/hours/internal/utils"
)

var errCouldntGenerateReport = errors.New("couldn't generate report")

const (
	reportTimeCharsBudget = 6
)

// reportSummaryBudget returns the character width budget for task summary cells
// in a report grid based on the number of days being displayed. Narrower budgets
// are used for wider grids (more days) so the table fits in a typical terminal.
func reportSummaryBudget(numDays int) int {
	switch numDays {
	case 7:
		return 8
	case 6:
		return 10
	case 5:
		return 14
	default:
		return 16
	}
}

// reportGridEntry is the minimal interface needed by renderReportGrid to render
// a single cell in the calendar-style report grid.
type reportGridEntry interface {
	reportTaskSummary() string
	reportSecsSpent() int
}

type taskLogEntryAdapter struct{ e types.TaskLogEntry }

func (a taskLogEntryAdapter) reportTaskSummary() string { return a.e.TaskSummary }
func (a taskLogEntryAdapter) reportSecsSpent() int      { return a.e.SecsSpent }

type taskReportEntryAdapter struct{ e types.TaskReportEntry }

func (a taskReportEntryAdapter) reportTaskSummary() string { return a.e.TaskSummary }
func (a taskReportEntryAdapter) reportSecsSpent() int      { return a.e.SecsSpent }

// perDayFetcher fetches the report entries for a single day [day, nextDay).
type perDayFetcher func(db *sql.DB, day, nextDay time.Time, taskStatus types.TaskStatus) ([]reportGridEntry, error)

func fetchTLEntriesForDay(db *sql.DB, day, nextDay time.Time, taskStatus types.TaskStatus) ([]reportGridEntry, error) {
	raw, err := pers.FetchTLEntriesBetweenTS(db, day, nextDay, taskStatus, 100)
	if err != nil {
		return nil, err
	}
	out := make([]reportGridEntry, len(raw))
	for i, e := range raw {
		out[i] = taskLogEntryAdapter{e}
	}
	return out, nil
}

func fetchReportEntriesForDay(db *sql.DB, day, nextDay time.Time, taskStatus types.TaskStatus) ([]reportGridEntry, error) {
	raw, err := pers.FetchReportBetweenTS(db, day, nextDay, taskStatus, 100)
	if err != nil {
		return nil, err
	}
	out := make([]reportGridEntry, len(raw))
	for i, e := range raw {
		out[i] = taskReportEntryAdapter{e}
	}
	return out, nil
}

// renderReportGrid is the shared rendering pipeline for both the plain and
// aggregated report views.
func renderReportGrid(db *sql.DB, style Style, start time.Time, numDays int, taskStatus types.TaskStatus, plain bool, fetch perDayFetcher) (string, error) {
	day := start
	var nextDay time.Time

	var maxEntryForADay int
	reportData := make(map[int][]reportGridEntry)

	noEntriesFound := true
	for i := range numDays {
		nextDay = day.AddDate(0, 0, 1)
		entries, err := fetch(db, day, nextDay, taskStatus)
		if err != nil {
			return "", err
		}
		if noEntriesFound && len(entries) > 0 {
			noEntriesFound = false
		}

		day = nextDay
		reportData[i] = entries
		if len(entries) > maxEntryForADay {
			maxEntryForADay = len(entries)
		}
	}

	if noEntriesFound {
		maxEntryForADay = 1
	}

	data := make([][]string, maxEntryForADay)
	totalSecsPerDay := make(map[int]int)

	for j := range numDays {
		totalSecsPerDay[j] = 0
	}

	rs := style.getReportStyles(plain)
	summaryBudget := reportSummaryBudget(numDays)

	styleCache := make(map[string]lipgloss.Style)
	for rowIndex := range maxEntryForADay {
		row := make([]string, numDays)
		for colIndex := range numDays {
			if rowIndex >= len(reportData[colIndex]) {
				row[colIndex] = fmt.Sprintf("%s  %s",
					utils.RightPadTrim("", summaryBudget, false),
					utils.RightPadTrim("", reportTimeCharsBudget, false),
				)
				continue
			}

			tr := reportData[colIndex][rowIndex]
			timeSpentStr := types.HumanizeDuration(tr.reportSecsSpent())

			if plain {
				row[colIndex] = fmt.Sprintf("%s  %s",
					utils.RightPadTrim(tr.reportTaskSummary(), summaryBudget, false),
					utils.RightPadTrim(timeSpentStr, reportTimeCharsBudget, false),
				)
			} else {
				rowStyle, ok := styleCache[tr.reportTaskSummary()]
				if !ok {
					rowStyle = style.getDynamicStyle(tr.reportTaskSummary())
					styleCache[tr.reportTaskSummary()] = rowStyle
				}

				row[colIndex] = fmt.Sprintf("%s  %s",
					rowStyle.Render(utils.RightPadTrim(tr.reportTaskSummary(), summaryBudget, false)),
					rowStyle.Render(utils.RightPadTrim(timeSpentStr, reportTimeCharsBudget, false)),
				)
			}
			totalSecsPerDay[colIndex] += tr.reportSecsSpent()
		}
		data[rowIndex] = row
	}

	totalTimePerDay := make([]string, numDays)
	for i, ts := range totalSecsPerDay {
		if ts != 0 {
			totalTimePerDay[i] = rs.footerStyle.Render(types.HumanizeDuration(ts))
		} else {
			totalTimePerDay[i] = " "
		}
	}

	headersValues := make([]string, numDays)
	day = start
	counter := 0
	for counter < numDays {
		headersValues[counter] = day.Format(dateFormat)
		day = day.AddDate(0, 0, 1)
		counter++
	}

	headers := make([]string, numDays)
	for i := range numDays {
		headers[i] = rs.headerStyle.Render(headersValues[i])
	}

	return renderRecordsTable(rs, headers, totalTimePerDay, data)
}

func RenderReport(db *sql.DB,
	style Style,
	writer io.Writer,
	plain bool,
	dateRange types.DateRange,
	period string,
	taskStatus types.TaskStatus,
	agg bool,
	interactive bool,
) error {
	var report string
	var analyticsType recordsKind
	var err error

	if agg {
		analyticsType = reportAggRecords
		report, err = renderReportGrid(db, style, dateRange.Start, dateRange.NumDays, taskStatus, plain, fetchReportEntriesForDay)
	} else {
		analyticsType = reportRecords
		report, err = renderReportGrid(db, style, dateRange.Start, dateRange.NumDays, taskStatus, plain, fetchTLEntriesForDay)
	}
	if err != nil {
		return fmt.Errorf("%w: %s", errCouldntGenerateReport, err.Error())
	}

	if interactive {
		p := tea.NewProgram(initialRecordsModel(
			analyticsType,
			db,
			style,
			types.RealTimeProvider{},
			dateRange,
			period,
			taskStatus,
			plain,
			report,
		))
		_, err := p.Run()
		if err != nil {
			return err
		}
	} else {
		fmt.Fprint(writer, report)
	}
	return nil
}
