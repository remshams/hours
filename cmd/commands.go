package cmd

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/dhth/hours/internal/types"
	"github.com/dhth/hours/internal/ui"
	"github.com/spf13/cobra"
)

// resolvePeriodAndRange resolves the period and date range from command arguments
// It takes the incoming args slice, the recordsInteractive flag pointer, and a pointer
// to the upper bound (reportNumDaysThreshold), decides the default period when args is empty,
// sets fullWeek based on *recordsInteractive, calls types.GetDateRangeFromPeriod
// and returns the resolved period and dateRange (or an error).
func resolvePeriodAndRange(
	args []string,
	defaultPeriod string,
	recordsInteractive *bool,
	numDaysUpperBound *int,
) (string, types.DateRange, error) {
	var period string
	if len(args) == 0 {
		period = defaultPeriod
	} else {
		period = args[0]
	}

	var fullWeek bool
	if *recordsInteractive {
		fullWeek = true
	}

	dateRange, err := types.GetDateRangeFromPeriod(period, types.RealTimeProvider{}.Now(), fullWeek, numDaysUpperBound)
	if err != nil {
		return "", types.DateRange{}, err
	}

	return period, dateRange, nil
}

// newGenerateCmd creates the generate command (gen)
func newGenerateCmd(
	db **sql.DB,
	preRun func(cmd *cobra.Command, args []string) error,
	dbPath *string,
	dbPathFull *string,
	genNumDays *uint8,
	genNumTasks *uint8,
	genSkipConfirmation *bool,
) *cobra.Command {
	return &cobra.Command{
		Use:   "gen",
		Short: "Generate dummy log entries (helpful for beginners)",
		Long: `Generate dummy log entries.
This is intended for new users of 'hours' so they can get a sense of its
capabilities without actually tracking any time. It's recommended to always use
this with a --dbpath/-d flag that points to a throwaway database.
`,
		PreRunE: preRun,
		RunE: func(_ *cobra.Command, _ []string) error {
			if *genNumDays > genNumDaysThreshold {
				return fmt.Errorf("%w (%d)", errNumDaysExceedsThreshold, genNumDaysThreshold)
			}
			if *genNumTasks > genNumTasksThreshold {
				return fmt.Errorf("%w (%d)", errNumTasksExceedsThreshold, genNumTasksThreshold)
			}

			if !*genSkipConfirmation {
				fmt.Print(lipgloss.NewStyle().Foreground(lipgloss.Color(warningColor)).Render(`
WARNING: You shouldn't run 'gen' on hours' actively used database as it'll
create dummy entries in it. You can run it on a throwaway database by passing a
path for it via --dbpath/-d (use it for all further invocations of 'hours' as
well).
`))
				fmt.Printf(`
The 'gen' subcommand is intended for new users of 'hours' so they can get a
sense of its capabilities without actually tracking any time.

Running with --dbpath set to: %q

---

`, *dbPathFull)
				confirm, err := getConfirmation()
				if err != nil {
					return err
				}
				if !confirm {
					return fmt.Errorf("%w", errIncorrectCodeEntered)
				}
			}

			genErr := ui.GenerateData(*db, *genNumDays, *genNumTasks)
			if genErr != nil {
				return fmt.Errorf("%w: %s", errCouldntGenerateData, genErr.Error())
			}
			fmt.Printf(`
Successfully generated dummy data in the database file: %s

Go ahead and try the following!

hours --dbpath=%s
hours --dbpath=%s report week -i
hours --dbpath=%s log today -i
hours --dbpath=%s stats today -i
`, *dbPath, *dbPath, *dbPath, *dbPath, *dbPath)
			return nil
		},
	}
}

// newReportCmd creates the report command
func newReportCmd(
	db **sql.DB,
	preRun func(cmd *cobra.Command, args []string) error,
	style *ui.Style,
	reportAgg *bool,
	recordsInteractive *bool,
	recordsOutputPlain *bool,
	taskStatusStr *string,
) *cobra.Command {
	return &cobra.Command{
		Use:   "report [PERIOD]",
		Short: "Output a report based on task log entries",
		Long: fmt.Sprintf(`Output a report based on task log entries.

Reports show time spent on tasks per day in the time period you specify. These
can also be aggregated (using -a) to consolidate all task entries and show the
cumulative time spent on each task per day.

Accepts an argument, which can be one of the following:

  today      for today's report
  yest       for yesterday's report
  3d         for a report on the last 3 days (default)
  week       for a report on the current week
  date       for a report for a specific date (eg. "2024/06/08")
  range      for a report for a date range (eg. "2024/06/08...2024/06/12", "2024/06/08...today", "2024/06/08..."; shouldn't be greater than %d days)

Note: If a task log continues past midnight in your local timezone, it
will be reported on the day it ends.
`, reportNumDaysThreshold),
		Args:    cobra.MaximumNArgs(1),
		PreRunE: preRun,
		RunE: func(_ *cobra.Command, args []string) error {
			taskStatus, err := types.ParseTaskStatus(*taskStatusStr)
			if err != nil {
				return err
			}

			numDaysUpperBound := reportNumDaysThreshold
			period, dateRange, err := resolvePeriodAndRange(args, "3d", recordsInteractive, &numDaysUpperBound)
			if err != nil {
				return err
			}

			return ui.RenderReport(*db, *style, os.Stdout, *recordsOutputPlain, dateRange, period, taskStatus, *reportAgg, *recordsInteractive)
		},
	}
}

// newLogCmd creates the log command
func newLogCmd(
	db **sql.DB,
	preRun func(cmd *cobra.Command, args []string) error,
	style *ui.Style,
	recordsInteractive *bool,
	recordsOutputPlain *bool,
	taskStatusStr *string,
) *cobra.Command {
	return &cobra.Command{
		Use:   "log [PERIOD]",
		Short: "Output task log entries",
		Long: `Output task log entries.

Accepts an argument, which can be one of the following:

  today      for log entries from today (default)
  yest       for log entries from yesterday
  3d         for log entries from the last 3 days
  week       for log entries from the current week
  date       for log entries from a specific date (eg. "2024/06/08")
  range      for log entries for a date range (eg. "2024/06/08...2024/06/12", "2024/06/08...today", "2024/06/08...")

Note: If a task log continues past midnight in your local timezone, it'll
appear in the log for the day it ends.
`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: preRun,
		RunE: func(_ *cobra.Command, args []string) error {
			taskStatus, err := types.ParseTaskStatus(*taskStatusStr)
			if err != nil {
				return err
			}

			period, dateRange, err := resolvePeriodAndRange(args, "today", recordsInteractive, nil)
			if err != nil {
				return err
			}

			return ui.RenderTaskLog(*db, *style, os.Stdout, *recordsOutputPlain, dateRange, period, taskStatus, *recordsInteractive)
		},
	}
}

// newStatsCmd creates the stats command
func newStatsCmd(
	db **sql.DB,
	preRun func(cmd *cobra.Command, args []string) error,
	style *ui.Style,
	recordsInteractive *bool,
	recordsOutputPlain *bool,
	taskStatusStr *string,
) *cobra.Command {
	return &cobra.Command{
		Use:   "stats [PERIOD]",
		Short: "Output statistics for tracked time",
		Long: `Output statistics for tracked time.

Accepts an argument, which can be one of the following:

  today       show stats for today
  yest        show stats for yesterday
  3d          show stats for the last 3 days (default)
  week        show stats for the current week
  this-month  show stats for the current month
  date        show stats for a specific date (eg. "2024/06/08")
  range       show stats for a date range (eg. "2024/06/08...2024/06/12", "2024/06/08...today", "2024/06/08...")
  all         show stats for all log entries

Note: If a task log continues past midnight in your local timezone, it'll
be considered in the stats for the day it ends.
`,
		Args:    cobra.MaximumNArgs(1),
		PreRunE: preRun,
		RunE: func(_ *cobra.Command, args []string) error {
			taskStatus, err := types.ParseTaskStatus(*taskStatusStr)
			if err != nil {
				return err
			}

			var period string
			if len(args) == 0 {
				period = "3d"
			} else {
				period = args[0]
			}

			var dateRangePtr *types.DateRange
			if period != "all" {
				_, dateRange, err := resolvePeriodAndRange(args, "3d", recordsInteractive, nil)
				if err != nil {
					return err
				}
				dateRangePtr = &dateRange
			}

			return ui.RenderStats(*db, *style, os.Stdout, *recordsOutputPlain, dateRangePtr, period, taskStatus, *recordsInteractive)
		},
	}
}

// newActiveCmd creates the active command
func newActiveCmd(
	db **sql.DB,
	preRun func(cmd *cobra.Command, args []string) error,
	activeTemplate *string,
) *cobra.Command {
	return &cobra.Command{
		Use:   "active",
		Short: `Show the task being actively tracked by "hours"`,
		Long: `Show the task being actively tracked by "hours".

You can pass in a template using the --template/-t flag, which supports the
following placeholders:

  {{task}}:  for the task summary
  {{time}}:  for the time spent so far on the active log entry

eg. hours active -t ' {{task}} ({{time}}) '
`,
		PreRunE: preRun,
		RunE: func(_ *cobra.Command, _ []string) error {
			return ui.ShowActiveTask(*db, os.Stdout, *activeTemplate)
		},
	}
}
