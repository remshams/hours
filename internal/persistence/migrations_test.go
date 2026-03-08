package persistence

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // sqlite driver
)

func TestMigrationsAreSetupCorrectly(t *testing.T) {
	// GIVEN
	// WHEN
	migrations := getMigrations()

	// THEN
	for i := 2; i <= latestDBVersion; i++ {
		m, ok := migrations[i]
		if !ok {
			assert.True(t, ok, "couldn't get migration %d", i)
		}
		if m == "" {
			assert.NotEmpty(t, ok, "migration %d is empty", i)
		}
	}
}

func TestMigrationsWork(t *testing.T) {
	// GIVEN
	var testDB *sql.DB
	var err error
	testDB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Couldn't open database: %s", err.Error())
	}

	err = InitDB(testDB)
	if err != nil {
		t.Fatalf("Couldn't initialize database: %s", err.Error())
	}

	// WHEN
	err = UpgradeDB(testDB, 1)

	// THEN
	assert.NoError(t, err)
}

func TestRunMigrationFailsWhenGivenBadMigration(t *testing.T) {
	// GIVEN
	var testDB *sql.DB
	var err error
	testDB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Couldn't open database: %s", err.Error())
	}

	err = InitDB(testDB)
	if err != nil {
		t.Fatalf("Couldn't initialize database: %s", err.Error())
	}

	// WHEN
	query := "BAD SQL CODE;"
	migrateErr := runMigration(testDB, query, 1)

	// THEN
	assert.Error(t, migrateErr)
}

func TestMigrationBackfillsSyncMetadata(t *testing.T) {
	// GIVEN
	testDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	err = InitDB(testDB)
	require.NoError(t, err)

	createdAt := time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(3 * time.Hour)
	completedBegin := createdAt.Add(30 * time.Minute)
	completedEnd := createdAt.Add(90 * time.Minute)
	activeBegin := createdAt.Add(2 * time.Hour)

	_, err = testDB.Exec(`
INSERT INTO task (id, summary, secs_spent, active, created_at, updated_at)
VALUES (1, 'seed task', 5400, true, ?, ?);
	`, createdAt, updatedAt)
	require.NoError(t, err)

	_, err = testDB.Exec(`
INSERT INTO task_log (id, task_id, begin_ts, end_ts, secs_spent, comment, active)
VALUES (1, 1, ?, ?, 3600, 'completed', false),
	       (2, 1, ?, NULL, 0, 'active', true);
	`, completedBegin, completedEnd, activeBegin)
	require.NoError(t, err)

	// WHEN
	err = UpgradeDB(testDB, 1)
	require.NoError(t, err)

	// THEN
	latestVersion, err := fetchLatestDBVersion(testDB)
	require.NoError(t, err)
	assert.Equal(t, 2, latestVersion.version)

	var taskCount int
	var distinctTaskSyncIDs int
	err = testDB.QueryRow(`
SELECT COUNT(*), COUNT(DISTINCT sync_id)
FROM task;
	`).Scan(&taskCount, &distinctTaskSyncIDs)
	require.NoError(t, err)
	assert.Equal(t, taskCount, distinctTaskSyncIDs)

	var taskLogCount int
	var distinctTaskLogSyncIDs int
	err = testDB.QueryRow(`
SELECT COUNT(*), COUNT(DISTINCT sync_id)
FROM task_log;
	`).Scan(&taskLogCount, &distinctTaskLogSyncIDs)
	require.NoError(t, err)
	assert.Equal(t, taskLogCount, distinctTaskLogSyncIDs)

	var taskSyncID string
	err = testDB.QueryRow(`SELECT sync_id FROM task WHERE id = 1;`).Scan(&taskSyncID)
	require.NoError(t, err)
	assert.NotEmpty(t, taskSyncID)

	var completedSyncID string
	var completedCreatedAt time.Time
	var completedUpdatedAt time.Time
	err = testDB.QueryRow(`
SELECT sync_id, created_at, updated_at
FROM task_log
WHERE id = 1;
	`).Scan(&completedSyncID, &completedCreatedAt, &completedUpdatedAt)
	require.NoError(t, err)
	assert.NotEmpty(t, completedSyncID)
	assert.Equal(t, completedBegin, completedCreatedAt)
	assert.Equal(t, completedEnd, completedUpdatedAt)

	var activeSyncID string
	var activeCreatedAt time.Time
	var activeUpdatedAt time.Time
	err = testDB.QueryRow(`
SELECT sync_id, created_at, updated_at
FROM task_log
WHERE id = 2;
	`).Scan(&activeSyncID, &activeCreatedAt, &activeUpdatedAt)
	require.NoError(t, err)
	assert.NotEmpty(t, activeSyncID)
	assert.Equal(t, activeBegin, activeCreatedAt)
	assert.Equal(t, activeBegin, activeUpdatedAt)
}
