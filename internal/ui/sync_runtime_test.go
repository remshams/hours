package ui

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
