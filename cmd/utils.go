package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhth/hours/internal/types"
	"github.com/spf13/cobra"
)

func expandTilde(path string, homeDir string) string {
	pathWithoutTilde, found := strings.CutPrefix(path, "~/")
	if !found {
		return path
	}
	return filepath.Join(homeDir, pathWithoutTilde)
}

// addDBPathFlag adds the --dbpath/-d flag to a command
func addDBPathFlag(cmd *cobra.Command, dbPath *string, defaultDBPath string) {
	cmd.Flags().StringVarP(dbPath, "dbpath", "d", defaultDBPath, "location of hours' database file")
}

// addThemeFlag adds the --theme/-t flag to a command
func addThemeFlag(cmd *cobra.Command, themeName *string, defaultThemeName string, usage string) {
	cmd.Flags().StringVarP(themeName, "theme", "t", defaultThemeName, usage)
}

// addTaskStatusFlag adds the --task-status/-s flag to a command
func addTaskStatusFlag(cmd *cobra.Command, taskStatusStr *string) {
	cmd.Flags().StringVarP(taskStatusStr, "task-status", "s", "any",
		fmt.Sprintf("only show data for tasks with this status [possible values: %q]", types.ValidTaskStatusValues))
}

// resolveThemeFromEnvOrFlag resolves the theme name from environment variable
// if the flag wasn't explicitly set by the user
func resolveThemeFromEnvOrFlag(cmd *cobra.Command, themeName *string, envVar string) {
	if !cmd.Flags().Changed("theme") {
		themeFromEnv := strings.TrimSpace(os.Getenv(envVar))
		if themeFromEnv != "" {
			*themeName = themeFromEnv
		}
	}
}
