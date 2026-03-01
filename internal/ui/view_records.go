package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dhth/hours/internal/types"
)

func (m recordsModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Something went wrong: %s\n", m.err)
	}
	var help string

	var dateRangeStr string
	var dateRange string
	if m.dateRange.NumDays > 1 {
		dateRangeStr = fmt.Sprintf(`
 range:             %s...%s
 `,
			m.dateRange.Start.Format(dateFormat), m.dateRange.End.AddDate(0, 0, -1).Format(dateFormat))
	} else {
		dateRangeStr = fmt.Sprintf(`
 date:              %s
`,
			m.dateRange.Start.Format(dateFormat))
	}

	helpStr := `
 go backwards:      h or <-
 go forwards:       l or ->
 go to today:       ctrl+t

 press ctrl+c/q to quit
`

	if m.plain {
		help = helpStr
		dateRange = dateRangeStr
	} else {
		help = m.style.recordsHelp.Render(helpStr)
		dateRange = m.style.recordsDateRange.Render(dateRangeStr)
	}

	return fmt.Sprintf("%s%s%s", m.report, dateRange, help)
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
				dr.End = dr.Start.AddDate(0, 0, dr.NumDays)
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
