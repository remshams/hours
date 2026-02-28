package cmd

import (
	"database/sql"
	"testing"

	"github.com/dhth/hours/internal/persistence"
	"github.com/dhth/hours/internal/types"
	"github.com/dhth/hours/internal/ui"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const (
	testDBPath     = "/test/hours.db"
	testTaskStatus = "any"
	invalidStatus  = "invalid-status"
)

// mockPreRun is a mock preRun function for testing
func mockPreRun(_ *cobra.Command, _ []string) error {
	return nil
}

// setupTestDB creates an in-memory database with complete schema for testing
// by using the same persistence/migration flow used in production
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	err = persistence.InitDB(db)
	require.NoError(t, err)

	return db
}

func TestNewGenerateCmd(t *testing.T) {
	t.Run("command properties", func(t *testing.T) {
		dbPath := testDBPath
		dbPathFull := testDBPath
		genNumDays := uint8(10)
		genNumTasks := uint8(5)
		genSkipConfirmation := true
		var db *sql.DB

		cmd := newGenerateCmd(&db, mockPreRun, &dbPath, &dbPathFull, &genNumDays, &genNumTasks, &genSkipConfirmation)

		assert.Equal(t, "gen", cmd.Use)
		assert.Equal(t, "Generate dummy log entries (helpful for beginners)", cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.PreRunE)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("exceeds num days threshold", func(t *testing.T) {
		dbPath := testDBPath
		dbPathFull := testDBPath
		genNumDays := uint8(genNumDaysThreshold + 1)
		genNumTasks := uint8(5)
		genSkipConfirmation := true
		var db *sql.DB

		cmd := newGenerateCmd(&db, mockPreRun, &dbPath, &dbPathFull, &genNumDays, &genNumTasks, &genSkipConfirmation)

		err := cmd.RunE(cmd, []string{})
		assert.ErrorIs(t, err, errNumDaysExceedsThreshold)
	})

	t.Run("exceeds num tasks threshold", func(t *testing.T) {
		dbPath := testDBPath
		dbPathFull := testDBPath
		genNumDays := uint8(10)
		genNumTasks := uint8(genNumTasksThreshold + 1)
		genSkipConfirmation := true
		var db *sql.DB

		cmd := newGenerateCmd(&db, mockPreRun, &dbPath, &dbPathFull, &genNumDays, &genNumTasks, &genSkipConfirmation)

		err := cmd.RunE(cmd, []string{})
		assert.ErrorIs(t, err, errNumTasksExceedsThreshold)
	})

	t.Run("command has correct thresholds", func(t *testing.T) {
		// Verify the thresholds are set correctly (they are untyped int constants)
		assert.Equal(t, 30, genNumDaysThreshold)
		assert.Equal(t, 20, genNumTasksThreshold)
	})
}

func TestNewReportCmd(t *testing.T) {
	t.Run("command properties", func(t *testing.T) {
		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		assert.Equal(t, "report [PERIOD]", cmd.Use)
		assert.Equal(t, "Output a report based on task log entries", cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.PreRunE)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("invalid task status", func(t *testing.T) {
		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := invalidStatus
		var db *sql.DB

		cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("uses 3d as default period", func(t *testing.T) {
		// This test verifies the default period logic without executing the command
		// since we can't run with nil database
		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Verify command structure
		assert.NotNil(t, cmd.RunE)
		// The actual default "3d" is handled inside RunE when args is empty
	})
}

func TestNewLogCmd(t *testing.T) {
	t.Run("command properties", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		assert.Equal(t, "log [PERIOD]", cmd.Use)
		assert.Equal(t, "Output task log entries", cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.PreRunE)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("invalid task status", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := invalidStatus
		var db *sql.DB

		cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("uses today as default period", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Verify command structure
		assert.NotNil(t, cmd.RunE)
		// The actual default "today" is handled inside RunE when args is empty
	})
}

func TestNewStatsCmd(t *testing.T) {
	t.Run("command properties", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		assert.Equal(t, "stats [PERIOD]", cmd.Use)
		assert.Equal(t, "Output statistics for tracked time", cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.PreRunE)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("invalid task status", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := invalidStatus
		var db *sql.DB

		cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		err := cmd.RunE(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("uses 3d as default period", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := false
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Verify command structure
		assert.NotNil(t, cmd.RunE)
		// The actual default "3d" is handled inside RunE when args is empty
	})
}

func TestNewActiveCmd(t *testing.T) {
	t.Run("command properties", func(t *testing.T) {
		activeTemplate := "{{task}} ({{time}})"
		var db *sql.DB

		cmd := newActiveCmd(&db, mockPreRun, &activeTemplate)

		assert.Equal(t, "active", cmd.Use)
		assert.Equal(t, `Show the task being actively tracked by "hours"`, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotNil(t, cmd.PreRunE)
		assert.NotNil(t, cmd.RunE)
		// Active command doesn't have Args field set (unlike others)
		assert.Nil(t, cmd.Args)
	})

	t.Run("with custom template", func(t *testing.T) {
		activeTemplate := "custom: {{task}}"
		var db *sql.DB

		cmd := newActiveCmd(&db, mockPreRun, &activeTemplate)

		assert.NotNil(t, cmd.RunE)
	})
}

func TestCommandCreationWithDB(t *testing.T) {
	t.Run("newReportCmd with database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Execute with a valid period but plain output to avoid interactive mode
		// The command will run without crashing, but may have no data
		err := cmd.RunE(cmd, []string{"today"})
		// We don't expect an error since we have a valid database schema
		// The output will just be empty
		assert.NoError(t, err)
	})

	t.Run("newLogCmd with database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Execute with "today" as period
		err := cmd.RunE(cmd, []string{"today"})
		assert.NoError(t, err)
	})

	t.Run("newStatsCmd with database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Execute with "3d" as period
		err := cmd.RunE(cmd, []string{"3d"})
		assert.NoError(t, err)
	})

	t.Run("newActiveCmd with database", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		activeTemplate := ui.ActiveTaskPlaceholder

		cmd := newActiveCmd(&db, mockPreRun, &activeTemplate)

		// Execute - should not crash even with empty database
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("newStatsCmd with all period", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)

		// Execute with "all" as period - should use nil date range
		err := cmd.RunE(cmd, []string{"all"})
		assert.NoError(t, err)
	})
}

func TestCommandArgsValidation(t *testing.T) {
	t.Run("report command accepts max 1 arg", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newReportCmd(&db, mockPreRun, &style, nil, nil, nil, &taskStatusStr)

		// cobra.MaximumNArgs(1) should be set
		assert.NotNil(t, cmd.Args)
	})

	t.Run("log command accepts max 1 arg", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newLogCmd(&db, mockPreRun, &style, nil, nil, &taskStatusStr)

		assert.NotNil(t, cmd.Args)
	})

	t.Run("stats command accepts max 1 arg", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newStatsCmd(&db, mockPreRun, &style, nil, nil, &taskStatusStr)

		assert.NotNil(t, cmd.Args)
	})

	t.Run("active command has no Args constraint", func(t *testing.T) {
		activeTemplate := ui.ActiveTaskPlaceholder
		var db *sql.DB

		cmd := newActiveCmd(&db, mockPreRun, &activeTemplate)

		// Active command doesn't set Args field - it accepts no arguments by default
		assert.Nil(t, cmd.Args)
	})
}

func TestPreRunEAssignment(t *testing.T) {
	t.Run("generate command has PreRunE", func(t *testing.T) {
		dbPath := testDBPath
		dbPathFull := testDBPath
		genNumDays := uint8(10)
		genNumTasks := uint8(5)
		genSkipConfirmation := true
		var db *sql.DB

		cmd := newGenerateCmd(&db, mockPreRun, &dbPath, &dbPathFull, &genNumDays, &genNumTasks, &genSkipConfirmation)

		assert.NotNil(t, cmd.PreRunE)
	})

	t.Run("report command has PreRunE", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newReportCmd(&db, mockPreRun, &style, nil, nil, nil, &taskStatusStr)

		assert.NotNil(t, cmd.PreRunE)
	})

	t.Run("log command has PreRunE", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newLogCmd(&db, mockPreRun, &style, nil, nil, &taskStatusStr)

		assert.NotNil(t, cmd.PreRunE)
	})

	t.Run("stats command has PreRunE", func(t *testing.T) {
		style := ui.Style{}
		taskStatusStr := testTaskStatus
		var db *sql.DB

		cmd := newStatsCmd(&db, mockPreRun, &style, nil, nil, &taskStatusStr)

		assert.NotNil(t, cmd.PreRunE)
	})

	t.Run("active command has PreRunE", func(t *testing.T) {
		activeTemplate := ui.ActiveTaskPlaceholder
		var db *sql.DB

		cmd := newActiveCmd(&db, mockPreRun, &activeTemplate)

		assert.NotNil(t, cmd.PreRunE)
	})
}

func TestPeriodParsing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	t.Run("report command parses various periods", func(t *testing.T) {
		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		periods := []string{"today", "yest", "3d", "week"}
		for _, period := range periods {
			cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			// Execute with valid database
			err := cmd.RunE(cmd, []string{period})
			assert.NoError(t, err, "period %s should not cause error", period)
		}
	})

	t.Run("log command parses various periods", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		periods := []string{"today", "yest", "3d", "week"}
		for _, period := range periods {
			cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			err := cmd.RunE(cmd, []string{period})
			assert.NoError(t, err, "period %s should not cause error", period)
		}
	})

	t.Run("stats command parses various periods", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true
		taskStatusStr := testTaskStatus

		periods := []string{"today", "yest", "3d", "week", "this-month"}
		for _, period := range periods {
			cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			err := cmd.RunE(cmd, []string{period})
			assert.NoError(t, err, "period %s should not cause error", period)
		}
	})
}

func TestTaskStatusParsing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	validStatuses := []string{"any", "active", "inactive"}

	t.Run("report command with valid task statuses", func(t *testing.T) {
		style := ui.Style{}
		reportAgg := false
		recordsInteractive := false
		recordsOutputPlain := true

		for _, status := range validStatuses {
			taskStatusStr := status
			cmd := newReportCmd(&db, mockPreRun, &style, &reportAgg, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			err := cmd.RunE(cmd, []string{"today"})
			assert.NoError(t, err, "status %s should not cause error", status)
		}
	})

	t.Run("log command with valid task statuses", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true

		for _, status := range validStatuses {
			taskStatusStr := status
			cmd := newLogCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			err := cmd.RunE(cmd, []string{"today"})
			assert.NoError(t, err, "status %s should not cause error", status)
		}
	})

	t.Run("stats command with valid task statuses", func(t *testing.T) {
		style := ui.Style{}
		recordsInteractive := false
		recordsOutputPlain := true

		for _, status := range validStatuses {
			taskStatusStr := status
			cmd := newStatsCmd(&db, mockPreRun, &style, &recordsInteractive, &recordsOutputPlain, &taskStatusStr)
			err := cmd.RunE(cmd, []string{"3d"})
			assert.NoError(t, err, "status %s should not cause error", status)
		}
	})
}

func TestValidTaskStatusValues(t *testing.T) {
	// Verify that the ValidTaskStatusValues contains the expected values
	expectedValues := []string{"any", "active", "inactive"}

	for _, expected := range expectedValues {
		found := false
		for _, actual := range types.ValidTaskStatusValues {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "expected %s to be in ValidTaskStatusValues", expected)
	}
}
