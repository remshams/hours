package ui

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const syncRequestTimeout = 10 * time.Second

type syncRunFunc func(context.Context, *sql.DB, string) error

func (m *Model) canRunSync() bool {
	config := m.syncConfig.Sanitized()
	if m.db == nil || m.runSync == nil || !config.Enabled {
		return false
	}

	return config.Validate() == nil
}

func (m *Model) startSyncCmd() tea.Cmd {
	if !m.canRunSync() || m.syncInFlight {
		return nil
	}

	m.syncInFlight = true
	return syncNowCmd(m.db, m.syncConfig.ServerURL, m.runSync)
}

func (m *Model) requestSyncCmd() tea.Cmd {
	if !m.canRunSync() {
		return nil
	}

	if m.syncInFlight {
		m.syncDirty = true
		return nil
	}

	m.syncDirty = false
	return m.startSyncCmd()
}

func (m *Model) scheduleBackgroundSyncCmd() tea.Cmd {
	if !m.trackingActive || !m.canRunSync() {
		return nil
	}

	interval, err := parseSyncInterval(m.syncConfig.Interval)
	if err != nil {
		return nil
	}

	return tea.Tick(interval, func(time.Time) tea.Msg {
		return syncTickMsg{}
	})
}

func syncNowCmd(db *sql.DB, serverURL string, runSync syncRunFunc) tea.Cmd {
	if db == nil || runSync == nil || strings.TrimSpace(serverURL) == "" {
		return nil
	}

	return func() tea.Msg {
		attemptedAt := time.Now().UTC()
		ctx, cancel := context.WithTimeout(context.Background(), syncRequestTimeout)
		defer cancel()

		return syncCompletedMsg{attemptedAt: attemptedAt, err: runSync(ctx, db, serverURL)}
	}
}

func (m *Model) handleSyncCompletedMsg(msg syncCompletedMsg) []tea.Cmd {
	m.syncInFlight = false
	m.syncLastAttemptAt = msg.attemptedAt

	var cmds []tea.Cmd
	if msg.err != nil {
		m.syncLastError = msg.err.Error()
		m.message = errMsg(fmt.Sprintf("Sync failed: %s", msg.err.Error()))
	} else {
		m.syncLastError = ""
		m.syncLastSuccessAt = msg.attemptedAt
		cmds = append(cmds, fetchTasks(m.db, true))
		cmds = append(cmds, fetchTasks(m.db, false))
		cmds = append(cmds, fetchTLS(m.db, nil))
	}

	if m.syncDirty {
		m.syncDirty = false
		if retryCmd := m.startSyncCmd(); retryCmd != nil {
			cmds = append(cmds, retryCmd)
		}
	}

	return cmds
}
