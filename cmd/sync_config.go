package cmd

import (
	"path/filepath"

	clientpkg "github.com/dhth/hours/internal/client"
	syncpkg "github.com/dhth/hours/internal/sync"
)

const syncConfigFileName = "sync.json"

func getSyncConfigPath(userConfigDir string) string {
	return filepath.Join(userConfigDir, configDirName, syncConfigFileName)
}

func loadSyncConfig(path string) (syncpkg.Config, string) {
	return clientpkg.LoadSyncConfig(path)
}

func saveSyncConfig(path string, config syncpkg.Config) error {
	return clientpkg.SaveSyncConfig(path, config)
}
