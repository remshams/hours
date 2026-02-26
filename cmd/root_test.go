package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreRunE_InvalidDBExtension(t *testing.T) {
	// Create a temp directory for test
	tempDir := t.TempDir()

	// Set the dbpath flag to a file without .db extension
	testCases := []struct {
		name   string
		dbPath string
	}{
		{
			name:   "txt extension",
			dbPath: filepath.Join(tempDir, "hours.txt"),
		},
		{
			name:   "no extension",
			dbPath: filepath.Join(tempDir, "hours"),
		},
		{
			name:   "json extension",
			dbPath: filepath.Join(tempDir, "hours.json"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh root command for each subtest
			cmd, err := NewRootCommand()
			require.NoError(t, err)

			// Reset the flag to its default value first
			cmd.Flags().Set("dbpath", tt.dbPath)

			// Execute PreRunE
			preRunE := cmd.PreRunE
			require.NotNil(t, preRunE)
			err = preRunE(cmd, []string{})
			assert.ErrorIs(t, err, errDBFileExtIncorrect)
		})
	}
}

func TestPreRunE_ValidDBExtension(t *testing.T) {
	// Create a temp directory for test
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "hours.db")

	// Create a root command
	cmd, err := NewRootCommand()
	require.NoError(t, err)

	// Set the dbpath flag to a file with .db extension
	cmd.Flags().Set("dbpath", dbPath)

	// Execute PreRunE - should not fail due to extension
	preRunE := cmd.PreRunE
	require.NotNil(t, preRunE)

	// This will likely fail on DB setup, but not on extension validation
	err = preRunE(cmd, []string{})
	// We expect this to succeed in creating the DB since it's a temp path
	// or potentially fail on theme validation, but not on extension
	assert.NotErrorIs(t, err, errDBFileExtIncorrect)
}

func TestThemeEnvVarPrecedence(t *testing.T) {
	testCases := []struct {
		name          string
		flagValue     string
		envValue      string
		expectedTheme string
		description   string
	}{
		{
			name:          "flag takes precedence over env",
			flagValue:     "bubblegum",
			envValue:      "catppuccin",
			expectedTheme: "bubblegum",
			description:   "When --theme flag is explicitly set, it should override HOURS_THEME env var",
		},
		{
			name:          "env var used when flag is default",
			flagValue:     "default",
			envValue:      "catppuccin",
			expectedTheme: "catppuccin",
			description:   "When --theme flag is not explicitly changed, HOURS_THEME should be used",
		},
		{
			name:          "default theme when neither set",
			flagValue:     "default",
			envValue:      "",
			expectedTheme: "default",
			description:   "When neither flag nor env var is set, use default theme",
		},
		{
			name:          "empty env var ignored",
			flagValue:     "default",
			envValue:      "   ",
			expectedTheme: "default",
			description:   "Whitespace-only env var should be treated as empty",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment (always set, even to empty string to avoid host leakage)
			t.Setenv("HOURS_THEME", tt.envValue)

			// Create a root command
			cmd, err := NewRootCommand()
			require.NoError(t, err)

			// Set the theme flag (only when explicitly set in the test)
			if tt.flagValue != "default" {
				err = cmd.Flags().Set("theme", tt.flagValue)
				require.NoError(t, err)
			}

			// Execute PreRunE to run the resolution logic
			preRunE := cmd.PreRunE
			require.NotNil(t, preRunE)

			// Execute PreRunE - this should run the theme resolution logic
			// We use a temp db to avoid failures on DB setup
			tempDir := t.TempDir()
			dbPath := filepath.Join(tempDir, "test.db")
			cmd.Flags().Set("dbpath", dbPath)

			// The PreRunE may fail on theme validation but should not panic
			// Capture the error to check theme was resolved
			err = preRunE(cmd, []string{})

			// Assert the resolved theme equals the expected theme
			resolvedTheme := GetThemeName()
			assert.Equal(t, tt.expectedTheme, resolvedTheme, "resolved theme should match expected")

			// If there was an error, it should be a theme-related error, not DB-related
			if err != nil {
				assert.NotErrorIs(t, err, errDBFileExtIncorrect, "should not fail on DB extension")
				assert.NotErrorIs(t, err, errCouldntCreateDB, "should not fail on DB creation")
				assert.NotErrorIs(t, err, errCouldntInitializeDB, "should not fail on DB initialization")
			}
		})
	}
}

func TestNewRootCommand_DefaultValues(t *testing.T) {
	cmd, err := NewRootCommand()
	require.NoError(t, err)

	// Check that the command has expected flags
	dbPathFlag, err := cmd.Flags().GetString("dbpath")
	require.NoError(t, err)
	assert.NotEmpty(t, dbPathFlag, "dbpath should have a default value")

	themeFlag, err := cmd.Flags().GetString("theme")
	require.NoError(t, err)
	assert.Equal(t, "default", themeFlag, "theme should default to 'default'")
}

func TestNewRootCommand_Subcommands(t *testing.T) {
	cmd, err := NewRootCommand()
	require.NoError(t, err)

	// Check that expected subcommands exist
	expectedSubcommands := []string{"gen", "report", "log", "stats", "active", "themes"}
	for _, name := range expectedSubcommands {
		subCmd, _, err := cmd.Find([]string{name})
		assert.NoError(t, err, "subcommand %s should exist", name)
		assert.NotNil(t, subCmd, "subcommand %s should not be nil", name)
	}
}

func TestPreRunE_DBSetupAndThemeLoading(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	cmd, err := NewRootCommand()
	require.NoError(t, err)

	// Set the dbpath to our temp file
	cmd.Flags().Set("dbpath", dbPath)

	preRunE := cmd.PreRunE
	require.NotNil(t, preRunE)

	// Execute PreRunE - should create the database
	err = preRunE(cmd, []string{})

	// Should succeed (or fail on theme if themes dir doesn't exist, but not on DB creation)
	if err != nil {
		// Check that it's not a DB extension error
		assert.NotErrorIs(t, err, errDBFileExtIncorrect, "should not fail on DB extension")
		// Check that it's not a DB creation/initialization error
		assert.NotErrorIs(t, err, errCouldntCreateDB, "should not fail on DB creation")
		assert.NotErrorIs(t, err, errCouldntInitializeDB, "should not fail on DB initialization")
	}

	// Verify the database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "database file should have been created")
}
