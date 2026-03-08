package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	syncpkg "github.com/dhth/hours/internal/sync"
)

const defaultSyncInterval = syncpkg.DefaultInterval

type SyncConfig = syncpkg.Config

func DefaultSyncConfig() SyncConfig {
	return syncpkg.DefaultConfig()
}

func parseSyncEnabled(raw string) (bool, error) {
	return syncpkg.ParseEnabled(raw)
}

func formatSyncEnabled(enabled bool) string {
	return syncpkg.FormatEnabled(enabled)
}

func parseSyncInterval(raw string) (time.Duration, error) {
	return syncpkg.ParseInterval(raw)
}

func (m *Model) syncConfigFromInputs() (SyncConfig, error) {
	enabled, err := parseSyncEnabled(m.syncInputs[syncEnabledField].Value())
	if err != nil {
		return SyncConfig{}, err
	}

	config := SyncConfig{
		Enabled:   enabled,
		ServerURL: m.syncInputs[syncServerURLField].Value(),
		Interval:  m.syncInputs[syncIntervalField].Value(),
	}.Sanitized()

	if err := config.Validate(); err != nil {
		return SyncConfig{}, err
	}

	return config, nil
}

func (m *Model) setSyncInputs(config SyncConfig) {
	config = config.Sanitized()
	m.syncInputs[syncEnabledField].SetValue(formatSyncEnabled(config.Enabled))
	m.syncInputs[syncServerURLField].SetValue(config.ServerURL)
	m.syncInputs[syncIntervalField].SetValue(config.Interval)
}

func (m *Model) focusSyncInput(field syncInputField) {
	m.syncInputFocussedField = field
	for i := range m.syncInputs {
		if i == int(field) {
			m.syncInputs[i].Focus()
			continue
		}
		m.syncInputs[i].Blur()
	}
}

func (m *Model) handleRequestToOpenSyncSettings() {
	m.lastView = m.activeView
	m.activeView = syncSettingsView
	m.focusSyncInput(syncEnabledField)
}

func (m *Model) saveSyncSettings() tea.Cmd {
	config, err := m.syncConfigFromInputs()
	if err != nil {
		m.syncConfigStatusErr = err.Error()
		m.message = errMsg(err.Error())
		return nil
	}

	if m.saveSyncConfig != nil {
		if err := m.saveSyncConfig(config); err != nil {
			m.syncConfigStatusErr = err.Error()
			m.message = errMsg(err.Error())
			return nil
		}
	}

	m.syncConfig = config
	m.syncConfigStatusErr = ""
	m.syncLastError = ""
	m.setSyncInputs(config)
	m.message = infoMsg("Sync settings saved")

	return m.scheduleBackgroundSyncCmd()
}

func (m Model) syncStatusForDisplay() (string, string, bool) {
	config, err := m.syncConfigFromInputs()
	if err != nil {
		return "Sync needs attention", err.Error(), true
	}

	if m.syncConfigStatusErr != "" {
		return "Sync needs attention", m.syncConfigStatusErr, true
	}

	if !config.Enabled {
		return "Sync disabled", fmt.Sprintf("Sync is off. Current interval is %s.", config.Interval), false
	}

	if m.syncInFlight {
		return "Sync in progress", fmt.Sprintf("Syncing with %s.", config.ServerURL), false
	}

	if m.syncLastError != "" {
		return "Last sync failed", fmt.Sprintf("%s at %s.", m.syncLastError, formatSyncTimestamp(m.syncLastAttemptAt)), true
	}

	if !m.syncLastSuccessAt.IsZero() {
		return "Last sync succeeded", fmt.Sprintf("Synced with %s at %s.", config.ServerURL, formatSyncTimestamp(m.syncLastSuccessAt)), false
	}

	return "Sync ready", fmt.Sprintf("Will sync with %s every %s.", config.ServerURL, config.Interval), false
}
