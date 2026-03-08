package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncConfigValidate(t *testing.T) {
	testCases := []struct {
		name       string
		config     SyncConfig
		errSnippet string
	}{
		{
			name: "valid disabled config",
			config: SyncConfig{
				Enabled:  false,
				Interval: "15m",
			},
		},
		{
			name: "enabled requires server url",
			config: SyncConfig{
				Enabled:  true,
				Interval: "15m",
			},
			errSnippet: "sync server URL is required",
		},
		{
			name: "invalid scheme rejected",
			config: SyncConfig{
				Enabled:   true,
				ServerURL: "ftp://sync.example.com",
				Interval:  "15m",
			},
			errSnippet: "must use http or https",
		},
		{
			name: "interval must be at least a minute",
			config: SyncConfig{
				Enabled:   true,
				ServerURL: "https://sync.example.com",
				Interval:  "30s",
			},
			errSnippet: "must be at least 1m0s",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.errSnippet == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errSnippet)
		})
	}
}

func TestNavigationKey4SwitchesToSyncSettingsView(t *testing.T) {
	m := createTestModel()
	m.activeView = taskListView

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	assert.Equal(t, syncSettingsView, model.activeView)
	assert.Equal(t, taskListView, model.lastView)
	assert.Equal(t, syncEnabledField, model.syncInputFocussedField)
}

func TestEscapeFromSyncSettingsReturnsToLastView(t *testing.T) {
	m := createTestModel()
	m.activeView = syncSettingsView
	m.lastView = taskLogView
	m.focusSyncInput(syncEnabledField)

	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := newM.(Model)

	assert.Equal(t, taskLogView, model.activeView)
	assert.Nil(t, cmd)
}

func TestSaveSyncSettingsRejectsInvalidInput(t *testing.T) {
	m := createTestModel()
	m.handleRequestToOpenSyncSettings()
	m.syncInputs[syncEnabledField].SetValue("on")
	m.syncInputs[syncServerURLField].SetValue("")
	m.syncInputs[syncIntervalField].SetValue("30s")

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model := newM.(Model)

	assert.Equal(t, userMsgErr, model.message.kind)
	assert.Contains(t, model.message.value, "sync interval must be at least")
	assert.Contains(t, model.message.value, "sync server URL is required")
	assert.Contains(t, model.syncConfigStatusErr, "sync server URL is required")
	assert.NotEqual(t, SyncConfig{Enabled: true, ServerURL: "", Interval: "30s"}, model.syncConfig)
}

func TestSaveSyncSettingsPersistsValidInput(t *testing.T) {
	m := createTestModel()
	m.handleRequestToOpenSyncSettings()

	var saved SyncConfig
	m.saveSyncConfig = func(config SyncConfig) error {
		saved = config
		return nil
	}

	m.syncInputs[syncEnabledField].SetValue("on")
	m.syncInputs[syncServerURLField].SetValue("https://sync.example.com")
	m.syncInputs[syncIntervalField].SetValue("30m")

	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model := newM.(Model)

	assert.Equal(t, userMsgInfo, model.message.kind)
	assert.Equal(t, "Sync settings saved", model.message.value)
	assert.Equal(t, SyncConfig{Enabled: true, ServerURL: "https://sync.example.com", Interval: "30m"}, saved)
	assert.Equal(t, saved, model.syncConfig)
	assert.Empty(t, model.syncConfigStatusErr)
}

func TestSyncSettingsViewShowsStatusAndPath(t *testing.T) {
	m := createTestModel()
	m.activeView = syncSettingsView
	m.setSyncInputs(SyncConfig{Enabled: true, ServerURL: "https://sync.example.com", Interval: "15m"})
	m.syncConfigPath = "testdata/sync.json"

	view := stripANSI(m.View())
	lines := strings.Split(view, "\n")
	nonEmptyLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines = append(nonEmptyLines, trimmed)
		}
	}

	assert.NotContains(t, view, `\n`)
	assert.Contains(t, nonEmptyLines, "Sync Settings")
	assert.Contains(t, nonEmptyLines, "Configure sync behavior for hours.")
	assert.Contains(t, nonEmptyLines, "Enabled* (on/off)")
	assert.Contains(t, view, "Sync Settings")
	assert.Contains(t, view, "Will sync with https://sync.example.com every 15m.")
	assert.Contains(t, view, "Settings file: testdata/sync.json")
}
