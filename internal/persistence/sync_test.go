package persistence

import (
	"testing"
	"time"

	"github.com/dhth/hours/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchSyncTaskByID(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	beforeInsert := time.Now().UTC()
	taskID, err := InsertTask(db, "sync task")
	require.NoError(t, err)
	afterInsert := time.Now().UTC()

	record, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)

	assert.Equal(t, taskID, record.LocalID)
	assert.Equal(t, "sync task", record.Summary)
	assert.True(t, record.Active)
	assert.NotEmpty(t, record.SyncID)
	assert.False(t, record.CreatedAt.Before(beforeInsert))
	assert.False(t, record.CreatedAt.After(afterInsert))
	assert.False(t, record.UpdatedAt.Before(beforeInsert))
	assert.False(t, record.UpdatedAt.After(afterInsert))

	records, err := FetchSyncTasks(db)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, record.SyncID, records[0].SyncID)
}

func TestFetchSyncTaskLogByIDPreservesStableIdentityAcrossEdits(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	taskID, err := InsertTask(db, "sync task")
	require.NoError(t, err)

	taskRecord, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)

	comment := "initial"
	beginTS := time.Date(2026, time.February, 1, 10, 0, 0, 0, time.UTC)
	endTS := beginTS.Add(2 * time.Hour)
	beforeInsert := time.Now().UTC()
	taskLogID, err := InsertManualTL(db, taskID, beginTS, endTS, &comment)
	require.NoError(t, err)
	afterInsert := time.Now().UTC()

	record, err := FetchSyncTaskLogByID(db, taskLogID)
	require.NoError(t, err)
	require.NotNil(t, record.EndTS)

	assert.Equal(t, taskLogID, record.LocalID)
	assert.Equal(t, taskID, record.TaskLocalID)
	assert.Equal(t, taskRecord.SyncID, record.TaskSyncID)
	assert.Equal(t, beginTS, record.BeginTS)
	assert.Equal(t, endTS, *record.EndTS)
	assert.Equal(t, 2*60*60, record.SecsSpent)
	assert.False(t, record.Active)
	assert.NotEmpty(t, record.SyncID)
	assert.False(t, record.CreatedAt.Before(beforeInsert))
	assert.False(t, record.CreatedAt.After(afterInsert))
	assert.False(t, record.UpdatedAt.Before(beforeInsert))
	assert.False(t, record.UpdatedAt.After(afterInsert))

	originalSyncID := record.SyncID
	oldUpdatedAt := time.Date(2025, time.December, 31, 8, 0, 0, 0, time.UTC)
	_, err = db.Exec(`UPDATE task_log SET updated_at = ? WHERE id = ?;`, oldUpdatedAt, taskLogID)
	require.NoError(t, err)

	editedComment := "edited"
	_, err = EditSavedTL(db, taskLogID, beginTS.Add(-30*time.Minute), endTS, &editedComment)
	require.NoError(t, err)

	editedRecord, err := FetchSyncTaskLogByID(db, taskLogID)
	require.NoError(t, err)
	assert.Equal(t, originalSyncID, editedRecord.SyncID)
	assert.True(t, editedRecord.UpdatedAt.After(oldUpdatedAt))
	assert.Equal(t, taskRecord.SyncID, editedRecord.TaskSyncID)

	records, err := FetchSyncTaskLogs(db)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, originalSyncID, records[0].SyncID)
}

func TestApplySyncBundleRecomputesSecsSpentAndIsIdempotent(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	taskID, err := InsertTask(db, "sync task")
	require.NoError(t, err)

	taskRecord, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)

	beginTS := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	endTS := beginTS.Add(time.Hour)
	incomingLog := types.SyncTaskLogRecord{
		SyncID:     "remote-log",
		TaskSyncID: taskRecord.SyncID,
		BeginTS:    beginTS,
		EndTS:      &endTS,
		SecsSpent:  int(time.Hour.Seconds()),
		Active:     false,
		CreatedAt:  beginTS,
		UpdatedAt:  endTS,
	}

	require.NoError(t, ApplySyncBundle(db, []types.SyncTaskRecord{taskRecord}, []types.SyncTaskLogRecord{incomingLog}))
	require.NoError(t, ApplySyncBundle(db, []types.SyncTaskRecord{taskRecord}, []types.SyncTaskLogRecord{incomingLog}))

	updatedTask, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)
	assert.Equal(t, int(time.Hour.Seconds()), updatedTask.SecsSpent)

	taskLogs, err := FetchSyncTaskLogs(db)
	require.NoError(t, err)
	require.Len(t, taskLogs, 1)
}

func TestApplySyncBundleBreaksTimestampTiesDeterministically(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	taskID, err := InsertTask(db, "aaa")
	require.NoError(t, err)

	taskRecord, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)

	tieTS := time.Date(2026, time.March, 3, 10, 0, 0, 0, time.UTC)
	_, err = db.Exec(`UPDATE task SET updated_at = ? WHERE id = ?;`, tieTS, taskID)
	require.NoError(t, err)

	taskRecord.UpdatedAt = tieTS
	taskRecord.Summary = "zzz"

	require.NoError(t, ApplySyncBundle(db, []types.SyncTaskRecord{taskRecord}, nil))

	updatedTask, err := FetchSyncTaskByID(db, taskID)
	require.NoError(t, err)
	assert.Equal(t, "zzz", updatedTask.Summary)
}
