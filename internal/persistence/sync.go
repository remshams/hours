package persistence

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dhth/hours/internal/types"
)

var ErrConflictingActiveSyncTaskLog = errors.New("sync would create multiple active task logs")

type sqlScanner interface {
	Scan(dest ...any) error
}

func newSyncID() (string, error) {
	var raw [16]byte
	_, err := rand.Read(raw[:])
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(raw[:]), nil
}

func FetchSyncTasks(db *sql.DB) ([]types.SyncTaskRecord, error) {
	rows, err := db.Query(`
SELECT id, sync_id, summary, secs_spent, active, created_at, updated_at
FROM task
ORDER BY updated_at ASC, id ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []types.SyncTaskRecord
	for rows.Next() {
		record, scanErr := scanSyncTaskRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func FetchSyncTaskByID(db *sql.DB, id int) (types.SyncTaskRecord, error) {
	row := db.QueryRow(`
SELECT id, sync_id, summary, secs_spent, active, created_at, updated_at
FROM task
WHERE id = ?;
	`, id)

	return scanSyncTaskRecord(row)
}

func FetchSyncTaskLogs(db *sql.DB) ([]types.SyncTaskLogRecord, error) {
	rows, err := db.Query(`
SELECT tl.id, tl.sync_id, tl.task_id, t.sync_id, tl.begin_ts, tl.end_ts,
	   tl.secs_spent, tl.comment, tl.active, tl.created_at, tl.updated_at
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
ORDER BY tl.updated_at ASC, tl.id ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []types.SyncTaskLogRecord
	for rows.Next() {
		record, scanErr := scanSyncTaskLogRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func FetchSyncTaskLogByID(db *sql.DB, id int) (types.SyncTaskLogRecord, error) {
	row := db.QueryRow(`
SELECT tl.id, tl.sync_id, tl.task_id, t.sync_id, tl.begin_ts, tl.end_ts,
	   tl.secs_spent, tl.comment, tl.active, tl.created_at, tl.updated_at
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.id = ?;
	`, id)

	return scanSyncTaskLogRecord(row)
}

func ApplySyncBundle(db *sql.DB, tasks []types.SyncTaskRecord, taskLogs []types.SyncTaskLogRecord) error {
	return runInTx(db, func(tx *sql.Tx) error {
		for _, task := range tasks {
			if err := applySyncTask(tx, task); err != nil {
				return err
			}
		}

		for _, taskLog := range taskLogs {
			if err := applySyncTaskLog(tx, taskLog); err != nil {
				return err
			}
		}

		_, err := tx.Exec(`
UPDATE task
SET secs_spent = COALESCE((
	SELECT SUM(secs_spent)
	FROM task_log
	WHERE task_id = task.id
), 0);
		`)
		return err
	})
}

func applySyncTask(tx *sql.Tx, incoming types.SyncTaskRecord) error {
	current, err := fetchSyncTaskBySyncID(tx, incoming.SyncID)
	if errors.Is(err, sql.ErrNoRows) {
		_, execErr := tx.Exec(`
INSERT INTO task (sync_id, summary, secs_spent, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);
		`, incoming.SyncID, incoming.Summary, incoming.SecsSpent, incoming.Active, incoming.CreatedAt.UTC(), incoming.UpdatedAt.UTC())
		return execErr
	}
	if err != nil {
		return err
	}

	if !shouldReplaceTask(current, incoming) {
		return nil
	}

	_, err = tx.Exec(`
UPDATE task
SET summary = ?, secs_spent = ?, active = ?, created_at = ?, updated_at = ?
WHERE sync_id = ?;
	`, incoming.Summary, incoming.SecsSpent, incoming.Active, incoming.CreatedAt.UTC(), incoming.UpdatedAt.UTC(), incoming.SyncID)
	return err
}

func applySyncTaskLog(tx *sql.Tx, incoming types.SyncTaskLogRecord) error {
	taskLocalID, err := fetchTaskLocalIDBySyncID(tx, incoming.TaskSyncID)
	if err != nil {
		return err
	}

	current, err := fetchSyncTaskLogBySyncID(tx, incoming.SyncID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := ensureNoConflictingActiveTaskLog(tx, incoming.SyncID, incoming.Active); err != nil {
			return err
		}

		_, execErr := tx.Exec(`
INSERT INTO task_log (sync_id, task_id, begin_ts, end_ts, secs_spent, comment, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, incoming.SyncID, taskLocalID, incoming.BeginTS.UTC(), nullableTime(incoming.EndTS), incoming.SecsSpent, incoming.Comment, incoming.Active, incoming.CreatedAt.UTC(), incoming.UpdatedAt.UTC())
		return execErr
	}
	if err != nil {
		return err
	}

	if !shouldReplaceTaskLog(current, incoming) {
		return nil
	}

	if err := ensureNoConflictingActiveTaskLog(tx, incoming.SyncID, incoming.Active); err != nil {
		return err
	}

	_, err = tx.Exec(`
UPDATE task_log
SET task_id = ?, begin_ts = ?, end_ts = ?, secs_spent = ?, comment = ?, active = ?, created_at = ?, updated_at = ?
WHERE sync_id = ?;
	`, taskLocalID, incoming.BeginTS.UTC(), nullableTime(incoming.EndTS), incoming.SecsSpent, incoming.Comment, incoming.Active, incoming.CreatedAt.UTC(), incoming.UpdatedAt.UTC(), incoming.SyncID)
	return err
}

func fetchSyncTaskBySyncID(tx *sql.Tx, syncID string) (types.SyncTaskRecord, error) {
	row := tx.QueryRow(`
SELECT id, sync_id, summary, secs_spent, active, created_at, updated_at
FROM task
WHERE sync_id = ?;
	`, syncID)

	return scanSyncTaskRecord(row)
}

func fetchTaskLocalIDBySyncID(tx *sql.Tx, syncID string) (int, error) {
	row := tx.QueryRow(`
SELECT id
FROM task
WHERE sync_id = ?;
	`, syncID)

	var localID int
	if err := row.Scan(&localID); err != nil {
		return -1, err
	}

	return localID, nil
}

func fetchSyncTaskLogBySyncID(tx *sql.Tx, syncID string) (types.SyncTaskLogRecord, error) {
	row := tx.QueryRow(`
SELECT tl.id, tl.sync_id, tl.task_id, t.sync_id, tl.begin_ts, tl.end_ts,
	   tl.secs_spent, tl.comment, tl.active, tl.created_at, tl.updated_at
FROM task_log tl
LEFT JOIN task t ON tl.task_id = t.id
WHERE tl.sync_id = ?;
	`, syncID)

	return scanSyncTaskLogRecord(row)
}

func ensureNoConflictingActiveTaskLog(tx *sql.Tx, syncID string, active bool) error {
	if !active {
		return nil
	}

	row := tx.QueryRow(`
SELECT sync_id
FROM task_log
WHERE active = 1 AND sync_id != ?
LIMIT 1;
	`, syncID)

	var conflictingSyncID string
	err := row.Scan(&conflictingSyncID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return fmt.Errorf("%w: %s conflicts with %s", ErrConflictingActiveSyncTaskLog, syncID, conflictingSyncID)
}

func shouldReplaceTask(current types.SyncTaskRecord, incoming types.SyncTaskRecord) bool {
	return shouldReplace(current.UpdatedAt, incoming.UpdatedAt, taskConflictKey(current), taskConflictKey(incoming))
}

func shouldReplaceTaskLog(current types.SyncTaskLogRecord, incoming types.SyncTaskLogRecord) bool {
	return shouldReplace(current.UpdatedAt, incoming.UpdatedAt, taskLogConflictKey(current), taskLogConflictKey(incoming))
}

func shouldReplace(currentUpdatedAt time.Time, incomingUpdatedAt time.Time, currentKey string, incomingKey string) bool {
	currentUTC := currentUpdatedAt.UTC()
	incomingUTC := incomingUpdatedAt.UTC()

	if incomingUTC.After(currentUTC) {
		return true
	}
	if incomingUTC.Before(currentUTC) {
		return false
	}

	return incomingKey > currentKey
}

func taskConflictKey(record types.SyncTaskRecord) string {
	return fmt.Sprintf("%s|%t|%d|%s", record.Summary, record.Active, record.SecsSpent, record.CreatedAt.UTC().Format(time.RFC3339Nano))
}

func taskLogConflictKey(record types.SyncTaskLogRecord) string {
	return fmt.Sprintf(
		"%s|%s|%s|%d|%s|%t|%s|%s",
		record.TaskSyncID,
		record.BeginTS.UTC().Format(time.RFC3339Nano),
		formatTimePtr(record.EndTS),
		record.SecsSpent,
		normalizeStringPtr(record.Comment),
		record.Active,
		record.CreatedAt.UTC().Format(time.RFC3339Nano),
		record.SyncID,
	)
}

func normalizeStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func scanSyncTaskRecord(scanner sqlScanner) (types.SyncTaskRecord, error) {
	var record types.SyncTaskRecord
	err := scanner.Scan(
		&record.LocalID,
		&record.SyncID,
		&record.Summary,
		&record.SecsSpent,
		&record.Active,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	return record, err
}

func scanSyncTaskLogRecord(scanner sqlScanner) (types.SyncTaskLogRecord, error) {
	var record types.SyncTaskLogRecord
	var endTS sql.NullTime
	err := scanner.Scan(
		&record.LocalID,
		&record.SyncID,
		&record.TaskLocalID,
		&record.TaskSyncID,
		&record.BeginTS,
		&endTS,
		&record.SecsSpent,
		&record.Comment,
		&record.Active,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return record, err
	}

	if endTS.Valid {
		end := endTS.Time
		record.EndTS = &end
	}

	return record, nil
}
