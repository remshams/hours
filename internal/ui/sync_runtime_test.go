package ui

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartupSyncStatusCmdSuccessShowsTransientBanner(t *testing.T) {
	m := createTestModel()
	m.checkSyncServerReachability = func(_ context.Context, serverURL string) error {
		assert.Equal(t, "http://sync.example.com", serverURL)
		return nil
	}
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}

	cmd := m.startupSyncStatusCmd()
	require.NotNil(t, cmd)

	updated, _ := m.Update(cmd())
	model := updated.(Model)
	assert.Equal(t, userMsgInfo, model.message.kind)
	assert.Equal(t, syncServerReachableMsg, model.message.value)
	assert.Equal(t, uint(userMsgDefaultFrames), model.message.framesLeft)
}

func TestStartupSyncStatusCmdFailureShowsTransientBanner(t *testing.T) {
	m := createTestModel()
	m.checkSyncServerReachability = func(context.Context, string) error {
		return errors.New("connection refused")
	}
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}

	cmd := m.startupSyncStatusCmd()
	require.NotNil(t, cmd)

	updated, _ := m.Update(cmd())
	model := updated.(Model)
	assert.Equal(t, userMsgErr, model.message.kind)
	assert.Equal(t, syncServerUnreachableMsg, model.message.value)
	assert.Equal(t, uint(userMsgDefaultFrames), model.message.framesLeft)
}

func TestStartupSyncStatusCmdOnlyRunsWhenSyncEnabledAndValid(t *testing.T) {
	testCases := []struct {
		name   string
		config SyncConfig
		want   bool
	}{
		{
			name:   "disabled sync",
			config: SyncConfig{Enabled: false, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval},
			want:   false,
		},
		{
			name:   "invalid config",
			config: SyncConfig{Enabled: true, ServerURL: "", Interval: defaultSyncInterval},
			want:   false,
		},
		{
			name:   "enabled valid config",
			config: SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval},
			want:   true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			m := createTestModel()
			m.syncConfig = tt.config

			cmd := m.startupSyncStatusCmd()
			if tt.want {
				require.NotNil(t, cmd)
				return
			}

			assert.Nil(t, cmd)
		})
	}
}

func TestInitDoesNotRunStartupSyncStatusCheckSynchronously(t *testing.T) {
	called := make(chan struct{}, 1)
	m := createTestModel()
	m.checkSyncServerReachability = func(context.Context, string) error {
		called <- struct{}{}
		return nil
	}
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}

	done := make(chan tea.Cmd, 1)
	go func() {
		done <- m.Init()
	}()

	select {
	case cmd := <-done:
		require.NotNil(t, cmd)
		select {
		case <-called:
			t.Fatal("startup sync status check ran during Init")
		default:
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Init blocked on startup sync status setup")
	}
}

func TestHandleMsgTaskCreatedSuccessQueuesSyncWhenEnabled(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	var calls int
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error {
		calls++
		return nil
	}

	cmds := m.handleMsg(taskCreatedMsg{})
	require.Len(t, cmds, 2)
	require.NotNil(t, cmds[1])

	msg := cmds[1]()
	_, ok := msg.(syncCompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
}

func TestHandleMsgManualTLInsertedSuccessQueuesSyncWhenEnabled(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	m.taskMap[1] = createTestTask(1, "task", true, false, m.timeProvider)
	var calls int
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error {
		calls++
		return nil
	}

	cmds := m.handleMsg(manualTLInsertedMsg{taskID: 1})
	require.Len(t, cmds, 3)
	require.NotNil(t, cmds[2])

	msg := cmds[2]()
	_, ok := msg.(syncCompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
}

func TestHandleMsgSavedTLEditedSuccessQueuesSyncWhenEnabled(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	m.taskMap[1] = createTestTask(1, "task", true, false, m.timeProvider)
	var calls int
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error {
		calls++
		return nil
	}

	cmds := m.handleMsg(savedTLEditedMsg{taskID: 1, tlID: 5})
	require.Len(t, cmds, 3)
	require.NotNil(t, cmds[2])

	msg := cmds[2]()
	_, ok := msg.(syncCompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
}

func TestHandleMsgStaleTasksArchivedSuccessQueuesSyncWhenEnabled(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	var calls int
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error {
		calls++
		return nil
	}

	cmds := m.handleMsg(staleTasksArchivedMsg{count: 3})
	require.Len(t, cmds, 3)
	require.NotNil(t, cmds[2])

	msg := cmds[2]()
	_, ok := msg.(syncCompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
}

func TestHandleTrackingToggledMsgStartSchedulesBackgroundSyncAndImmediateSync(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	m.taskMap[1] = createTestTask(1, "task", true, false, m.timeProvider)
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error { return nil }

	cmds := m.handleTrackingToggledMsg(trackingToggledMsg{taskID: 1})
	require.Len(t, cmds, 2)
	assert.NotNil(t, cmds[0])
	assert.NotNil(t, cmds[1])
}

func TestHandleSyncCompletedMsgRetriesDirtySyncAfterFailure(t *testing.T) {
	m := createTestModel()
	m.db = setupSyncRuntimeTestDB(t)
	m.syncConfig = SyncConfig{Enabled: true, ServerURL: "http://sync.example.com", Interval: defaultSyncInterval}
	m.syncInFlight = true
	m.syncDirty = true
	var calls int
	m.runSync = func(_ context.Context, _ *sql.DB, _ string) error {
		calls++
		return nil
	}

	cmds := m.handleSyncCompletedMsg(syncCompletedMsg{attemptedAt: referenceTime, err: assert.AnError})
	require.Len(t, cmds, 1)
	require.NotNil(t, cmds[0])

	msg := cmds[0]()
	_, ok := msg.(syncCompletedMsg)
	assert.True(t, ok)
	assert.Equal(t, 1, calls)
	assert.Equal(t, assert.AnError.Error(), m.syncLastError)
	assert.True(t, m.syncInFlight)
	assert.False(t, m.syncDirty)
}

func setupSyncRuntimeTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}
