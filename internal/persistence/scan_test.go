package persistence

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // sqlite driver
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err, "failed to open in-memory DB")
	require.NoError(t, InitDB(db), "failed to initialize DB")
	require.NoError(t, UpgradeDB(db, 1), "failed to upgrade DB")
	return db
}

// TestScanTask verifies that scanTask correctly reads a task row including
// timezone conversion.
func TestScanTask(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT id, summary, secs_spent, created_at, updated_at, active
FROM task
WHERE id = 1`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "expected at least one row")

	entry, err := scanTask(rows)
	require.NoError(t, err)

	assert.Equal(t, 1, entry.ID)
	assert.Equal(t, "seeded task 1", entry.Summary)
	assert.Equal(t, 5*secsInOneHour, entry.SecsSpent)
	assert.True(t, entry.Active)
	createdAt := referenceTS.UTC().Add(time.Hour * 24 * 7 * -1)
	updatedAt := createdAt.Add(time.Hour * 9)
	// Timezone conversion: Local() must be applied
	assert.Equal(t, createdAt.Local(), entry.CreatedAt)
	assert.Equal(t, updatedAt.Local(), entry.UpdatedAt)
}

// TestScanTaskLogEntry verifies that scanTaskLogEntry correctly reads a task
// log row, including nullable comment and timezone conversion.
func TestScanTaskLogEntry(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT tl.id, tl.task_id, t.summary, tl.begin_ts, tl.end_ts, tl.secs_spent, tl.comment
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.id = 1`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "expected at least one row")

	entry, err := scanTaskLogEntry(rows)
	require.NoError(t, err)

	assert.Equal(t, 1, entry.ID)
	assert.Equal(t, 1, entry.TaskID)
	assert.Equal(t, "seeded task 1", entry.TaskSummary)
	assert.Equal(t, 2*secsInOneHour, entry.SecsSpent)
	require.NotNil(t, entry.Comment)
	assert.Equal(t, "task 1 tl 1", *entry.Comment)
	// Timezone conversion
	assert.Equal(t, entry.BeginTS, entry.BeginTS.Local())
	assert.Equal(t, entry.EndTS, entry.EndTS.Local())
}

// TestScanTaskLogEntry_NilComment verifies that a NULL comment is scanned as nil.
func TestScanTaskLogEntry_NilComment(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	beginTS := referenceTS.Add(-2 * time.Hour)
	endTS := referenceTS.Add(-1 * time.Hour)

	_, err := db.Exec(`INSERT INTO task (id, summary, secs_spent, active, created_at, updated_at) VALUES (1, 'task', 3600, true, ?, ?)`,
		referenceTS, referenceTS)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO task_log (id, task_id, begin_ts, end_ts, secs_spent, comment, active) VALUES (1, 1, ?, ?, 3600, NULL, false)`,
		beginTS, endTS)
	require.NoError(t, err)

	rows, err := db.Query(`
SELECT tl.id, tl.task_id, t.summary, tl.begin_ts, tl.end_ts, tl.secs_spent, tl.comment
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.id = 1`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next())

	entry, err := scanTaskLogEntry(rows)
	require.NoError(t, err)
	assert.Nil(t, entry.Comment)
}

// TestScanTaskReportEntry verifies that scanTaskReportEntry correctly reads an
// aggregated report row.
func TestScanTaskReportEntry(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT tl.task_id, t.summary, COUNT(tl.id) as num_entries, t.secs_spent
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.task_id = 1
GROUP BY tl.task_id`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "expected at least one row")

	entry, err := scanTaskReportEntry(rows)
	require.NoError(t, err)

	assert.Equal(t, 1, entry.TaskID)
	assert.Equal(t, "seeded task 1", entry.TaskSummary)
	assert.Equal(t, 2, entry.NumEntries) // 2 task log entries for task 1
	assert.Equal(t, 5*secsInOneHour, entry.SecsSpent)
}

// TestCollectTasks verifies that collectTasks accumulates all rows correctly.
func TestCollectTasks(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT id, summary, secs_spent, created_at, updated_at, active
FROM task
ORDER BY id ASC`)
	require.NoError(t, err)
	defer rows.Close()

	tasks, err := collectTasks(rows)
	require.NoError(t, err)

	assert.Len(t, tasks, 2)
	assert.Equal(t, 1, tasks[0].ID)
	assert.Equal(t, 2, tasks[1].ID)
}

// TestCollectTasks_Empty verifies that collectTasks returns nil (not an error)
// when there are no matching rows.
func TestCollectTasks_Empty(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	rows, err := db.Query(`SELECT id, summary, secs_spent, created_at, updated_at, active FROM task`)
	require.NoError(t, err)
	defer rows.Close()

	tasks, err := collectTasks(rows)
	require.NoError(t, err)
	assert.Nil(t, tasks)
}

// TestCollectTaskLogEntries verifies that collectTaskLogEntries accumulates all
// rows correctly.
func TestCollectTaskLogEntries(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT tl.id, tl.task_id, t.summary, tl.begin_ts, tl.end_ts, tl.secs_spent, tl.comment
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.active = false
ORDER BY tl.id ASC`)
	require.NoError(t, err)
	defer rows.Close()

	entries, err := collectTaskLogEntries(rows)
	require.NoError(t, err)

	assert.Len(t, entries, 3)
	assert.Equal(t, 1, entries[0].ID)
	assert.Equal(t, 2, entries[1].ID)
	assert.Equal(t, 3, entries[2].ID)
}

// TestCollectTaskLogEntries_Empty verifies that collectTaskLogEntries returns
// nil when there are no matching rows.
func TestCollectTaskLogEntries_Empty(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	rows, err := db.Query(`
SELECT tl.id, tl.task_id, t.summary, tl.begin_ts, tl.end_ts, tl.secs_spent, tl.comment
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.active = false`)
	require.NoError(t, err)
	defer rows.Close()

	entries, err := collectTaskLogEntries(rows)
	require.NoError(t, err)
	assert.Nil(t, entries)
}

// TestCollectTaskReportEntries verifies that collectTaskReportEntries
// accumulates all aggregated rows correctly.
func TestCollectTaskReportEntries(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	referenceTS := time.Now().Truncate(time.Second).UTC()
	seedDB(t, db, getTestData(referenceTS))

	rows, err := db.Query(`
SELECT tl.task_id, t.summary, COUNT(tl.id) as num_entries, t.secs_spent
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
GROUP BY tl.task_id
ORDER BY tl.task_id ASC`)
	require.NoError(t, err)
	defer rows.Close()

	entries, err := collectTaskReportEntries(rows)
	require.NoError(t, err)

	assert.Len(t, entries, 2)
	assert.Equal(t, 1, entries[0].TaskID)
	assert.Equal(t, 2, entries[1].TaskID)
}

// TestCollectTaskReportEntries_Empty verifies that collectTaskReportEntries
// returns nil when there are no matching rows.
func TestCollectTaskReportEntries_Empty(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	rows, err := db.Query(`
SELECT tl.task_id, t.summary, COUNT(tl.id) as num_entries, t.secs_spent
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
GROUP BY tl.task_id`)
	require.NoError(t, err)
	defer rows.Close()

	entries, err := collectTaskReportEntries(rows)
	require.NoError(t, err)
	assert.Nil(t, entries)
}
