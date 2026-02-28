package cmd

import (
	"testing"

	"github.com/dhth/hours/internal/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandTilde(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		homeDir  string
		expected string
	}{
		{
			name:     "a simple case",
			path:     "~/some/path",
			homeDir:  "/Users/trinity",
			expected: "/Users/trinity/some/path",
		},
		{
			name:     "path with no ~",
			path:     "some/path",
			homeDir:  "/Users/trinity",
			expected: "some/path",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := expandTilde(tt.path, tt.homeDir)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAddDBPathFlag(t *testing.T) {
	testCases := []struct {
		name              string
		defaultPath       string
		expectedName      string
		expectedShorthand string
	}{
		{
			name:              "adds dbpath flag with default",
			defaultPath:       "/home/user/hours.db",
			expectedName:      "dbpath",
			expectedShorthand: "d",
		},
		{
			name:              "adds dbpath flag with empty default",
			defaultPath:       "",
			expectedName:      "dbpath",
			expectedShorthand: "d",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var dbPath string

			addDBPathFlag(cmd, &dbPath, tt.defaultPath)

			flag := cmd.Flags().Lookup(tt.expectedName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.expectedShorthand, flag.Shorthand)
			assert.Equal(t, tt.defaultPath, flag.DefValue)
			assert.Equal(t, "location of hours' database file", flag.Usage)
		})
	}
}

func TestAddThemeFlag(t *testing.T) {
	testCases := []struct {
		name              string
		defaultTheme      string
		usage             string
		expectedName      string
		expectedShorthand string
	}{
		{
			name:              "adds theme flag with default",
			defaultTheme:      "default",
			usage:             "UI theme to use",
			expectedName:      "theme",
			expectedShorthand: "t",
		},
		{
			name:              "adds theme flag with custom default",
			defaultTheme:      "catppuccin",
			usage:             "custom theme usage",
			expectedName:      "theme",
			expectedShorthand: "t",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			var themeName string

			addThemeFlag(cmd, &themeName, tt.defaultTheme, tt.usage)

			flag := cmd.Flags().Lookup(tt.expectedName)
			require.NotNil(t, flag)
			assert.Equal(t, tt.expectedShorthand, flag.Shorthand)
			assert.Equal(t, tt.defaultTheme, flag.DefValue)
			assert.Equal(t, tt.usage, flag.Usage)
		})
	}
}

func TestAddTaskStatusFlag(t *testing.T) {
	t.Run("adds task-status flag", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var taskStatusStr string

		addTaskStatusFlag(cmd, &taskStatusStr)

		flag := cmd.Flags().Lookup("task-status")
		require.NotNil(t, flag)
		assert.Equal(t, "s", flag.Shorthand)
		assert.Equal(t, "any", flag.DefValue)
		assert.Contains(t, flag.Usage, "only show data for tasks with this status")
		assert.Contains(t, flag.Usage, "active")
		assert.Contains(t, flag.Usage, "inactive")
	})

	t.Run("task status flag references valid values", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var taskStatusStr string

		addTaskStatusFlag(cmd, &taskStatusStr)

		flag := cmd.Flags().Lookup("task-status")
		require.NotNil(t, flag)

		// Verify the usage message mentions all valid status values
		for _, status := range types.ValidTaskStatusValues {
			assert.Contains(t, flag.Usage, status)
		}
	})
}

func TestResolveThemeFromEnvOrFlag(t *testing.T) {
	testCases := []struct {
		name          string
		flagValue     string
		envValue      string
		flagChanged   bool
		expectedTheme string
	}{
		{
			name:          "uses flag when explicitly set",
			flagValue:     "bubblegum",
			envValue:      "catppuccin",
			flagChanged:   true,
			expectedTheme: "bubblegum",
		},
		{
			name:          "uses env when flag not changed",
			flagValue:     "default",
			envValue:      "catppuccin",
			flagChanged:   false,
			expectedTheme: "catppuccin",
		},
		{
			name:          "keeps default when no env and flag not changed",
			flagValue:     "default",
			envValue:      "",
			flagChanged:   false,
			expectedTheme: "default",
		},
		{
			name:          "ignores empty env value",
			flagValue:     "default",
			envValue:      "   ",
			flagChanged:   false,
			expectedTheme: "default",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().StringVarP(&tt.flagValue, "theme", "t", tt.flagValue, "theme flag")

			if tt.flagChanged {
				_ = cmd.Flags().Set("theme", tt.flagValue)
			}

			t.Setenv("HOURS_THEME", tt.envValue)

			resolveThemeFromEnvOrFlag(cmd, &tt.flagValue, "HOURS_THEME")

			assert.Equal(t, tt.expectedTheme, tt.flagValue)
		})
	}
}

func TestResolveThemeFromEnvOrFlagWithNoThemeFlag(t *testing.T) {
	t.Run("works without theme flag registered", func(t *testing.T) {
		// Create a command without registering the theme flag
		cmd := &cobra.Command{Use: "test"}
		themeName := defaultThemeName

		t.Setenv("HOURS_THEME", "dracula")

		// This should complete without panic and use the env value
		resolveThemeFromEnvOrFlag(cmd, &themeName, "HOURS_THEME")

		assert.Equal(t, "dracula", themeName)
	})

	t.Run("keeps default when env not set and no flag registered", func(t *testing.T) {
		// Create a command without registering the theme flag
		cmd := &cobra.Command{Use: "test"}
		themeName := defaultThemeName

		// No env var set

		// This should complete without panic and keep the default
		resolveThemeFromEnvOrFlag(cmd, &themeName, "HOURS_THEME")

		assert.Equal(t, defaultThemeName, themeName)
	})
}

func TestAllUtilityFunctionsIntegration(t *testing.T) {
	t.Run("can add all flags to a command", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		var dbPath string
		var themeName string
		var taskStatusStr string

		addDBPathFlag(cmd, &dbPath, "/default/path.db")
		addThemeFlag(cmd, &themeName, "default", "theme to use")
		addTaskStatusFlag(cmd, &taskStatusStr)

		// Verify all flags exist
		require.NotNil(t, cmd.Flags().Lookup("dbpath"))
		require.NotNil(t, cmd.Flags().Lookup("theme"))
		require.NotNil(t, cmd.Flags().Lookup("task-status"))

		// Verify short flags
		dbFlag := cmd.Flags().Lookup("dbpath")
		assert.Equal(t, "d", dbFlag.Shorthand)

		themeFlag := cmd.Flags().Lookup("theme")
		assert.Equal(t, "t", themeFlag.Shorthand)

		statusFlag := cmd.Flags().Lookup("task-status")
		assert.Equal(t, "s", statusFlag.Shorthand)
	})
}
