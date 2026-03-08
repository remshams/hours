package ui

import (
	"time"

	syncpkg "github.com/dhth/hours/internal/sync"
)

const defaultSyncInterval = syncpkg.DefaultInterval

type SyncConfig = syncpkg.Config

func DefaultSyncConfig() SyncConfig {
	return syncpkg.DefaultConfig()
}

func parseSyncInterval(raw string) (time.Duration, error) {
	return syncpkg.ParseInterval(raw)
}
