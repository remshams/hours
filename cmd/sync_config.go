package cmd

import (
	"path/filepath"

	clientpkg "github.com/dhth/hours/internal/client"
	syncpkg "github.com/dhth/hours/internal/sync"
)

const (
	syncConfigFileName       = "sync.json"
	macOSConfigParentDirName = ".config"
)

func getSyncConfigPath(goos, userHomeDir, userConfigDir string) string {
	if goos == "darwin" {
		return filepath.Join(userHomeDir, macOSConfigParentDirName, configDirName, syncConfigFileName)
	}

	return filepath.Join(userConfigDir, configDirName, syncConfigFileName)
}

func loadSyncConfig(path string) (syncpkg.Config, string) {
	return clientpkg.LoadSyncConfig(path)
}

func saveSyncConfig(path string, config syncpkg.Config) error {
	return clientpkg.SaveSyncConfig(path, config)
}
