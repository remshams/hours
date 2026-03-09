package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dhth/hours/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSyncConfigPath(t *testing.T) {
	t.Run("darwin uses home dot config", func(t *testing.T) {
		assert.Equal(
			t,
			filepath.Join("/tmp/home", macOSConfigParentDirName, configDirName, syncConfigFileName),
			getSyncConfigPath("darwin", "/tmp/home", "/tmp/config"),
		)
	})

	t.Run("non-darwin uses user config dir", func(t *testing.T) {
		assert.Equal(
			t,
			filepath.Join("/tmp/config", configDirName, syncConfigFileName),
			getSyncConfigPath("linux", "/tmp/home", "/tmp/config"),
		)
	})
}

func TestLoadSyncConfigReturnsDefaultWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncConfigFileName)

	config, statusErr := loadSyncConfig(path)

	assert.Equal(t, ui.DefaultSyncConfig(), config)
	assert.Empty(t, statusErr)
}

func TestLoadSyncConfigReportsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncConfigFileName)
	require.NoError(t, os.WriteFile(path, []byte("{not-json"), 0o644))

	config, statusErr := loadSyncConfig(path)

	assert.Equal(t, ui.DefaultSyncConfig(), config)
	assert.Contains(t, statusErr, "couldn't parse sync settings")
}

func TestLoadSyncConfigKeepsValuesWhenValidationFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncConfigFileName)
	require.NoError(
		t,
		os.WriteFile(path, []byte(`{"enabled":true,"interval":"30s"}`), 0o644),
	)

	config, statusErr := loadSyncConfig(path)

	assert.Equal(t, ui.SyncConfig{Enabled: true, Interval: "30s"}, config)
	assert.Contains(t, statusErr, "sync interval must be at least")
	assert.Contains(t, statusErr, "sync server URL is required")
}

func TestSaveSyncConfigWritesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncConfigFileName)
	config := ui.SyncConfig{
		Enabled:   true,
		ServerURL: "https://sync.example.com",
		Interval:  "15m",
	}

	require.NoError(t, saveSyncConfig(path, config))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"enabled": true`)
	assert.Contains(t, string(content), `"serverURL": "https://sync.example.com"`)
	assert.Contains(t, string(content), `"interval": "15m"`)
}

func TestSaveSyncConfigRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), syncConfigFileName)
	config := ui.SyncConfig{
		Enabled:  true,
		Interval: "15m",
	}

	err := saveSyncConfig(path, config)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "sync server URL is required")
	_, statErr := os.Stat(path)
	assert.Error(t, statErr)
}
