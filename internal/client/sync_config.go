package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	syncpkg "github.com/dhth/hours/internal/sync"
)

var errCouldntWriteSyncConfig = errors.New("couldn't write sync config")

func LoadSyncConfig(path string) (syncpkg.Config, string) {
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return syncpkg.DefaultConfig(), ""
	}
	if err != nil {
		return syncpkg.DefaultConfig(), fmt.Sprintf("couldn't read sync settings at %s: %s", path, err)
	}

	config := syncpkg.DefaultConfig()
	if err := json.Unmarshal(content, &config); err != nil {
		return syncpkg.DefaultConfig(), fmt.Sprintf("couldn't parse sync settings at %s: %s", path, err)
	}

	if err := config.Validate(); err != nil {
		return config, err.Error()
	}

	return config, ""
}

func SaveSyncConfig(path string, config syncpkg.Config) error {
	if err := config.Validate(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return fmt.Errorf("%w: %w", errCouldntWriteSyncConfig, err)
	}

	return nil
}
