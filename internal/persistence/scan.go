package persistence

import (
	"database/sql"

	"github.com/dhth/hours/internal/types"
)

// scanTask scans a single task row into a types.Task value.
// It also converts time fields to local timezone.
func scanTask(row *sql.Rows) (types.Task, error) {
	var entry types.Task
	err := row.Scan(
		&entry.ID,
		&entry.Summary,
		&entry.SecsSpent,
		&entry.CreatedAt,
		&entry.UpdatedAt,
		&entry.Active,
	)
	if err != nil {
		return types.Task{}, err
	}
	entry.CreatedAt = entry.CreatedAt.Local()
	entry.UpdatedAt = entry.UpdatedAt.Local()
	return entry, nil
}

// scanTaskLogEntry scans a single task log row into a types.TaskLogEntry value.
// It also converts time fields to local timezone.
func scanTaskLogEntry(row *sql.Rows) (types.TaskLogEntry, error) {
	var entry types.TaskLogEntry
	err := row.Scan(
		&entry.ID,
		&entry.TaskID,
		&entry.TaskSummary,
		&entry.BeginTS,
		&entry.EndTS,
		&entry.SecsSpent,
		&entry.Comment,
	)
	if err != nil {
		return types.TaskLogEntry{}, err
	}
	entry.BeginTS = entry.BeginTS.Local()
	entry.EndTS = entry.EndTS.Local()
	return entry, nil
}

// scanTaskReportEntry scans a single task report row into a types.TaskReportEntry value.
func scanTaskReportEntry(row *sql.Rows) (types.TaskReportEntry, error) {
	var entry types.TaskReportEntry
	err := row.Scan(
		&entry.TaskID,
		&entry.TaskSummary,
		&entry.NumEntries,
		&entry.SecsSpent,
	)
	if err != nil {
		return types.TaskReportEntry{}, err
	}
	return entry, nil
}

// collectTasks iterates over rows and collects them into a slice of types.Task.
// It is the caller's responsibility to close rows.
func collectTasks(rows *sql.Rows) ([]types.Task, error) {
	var tasks []types.Task
	for rows.Next() {
		entry, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// collectTaskLogEntries iterates over rows and collects them into a slice of
// types.TaskLogEntry. It is the caller's responsibility to close rows.
func collectTaskLogEntries(rows *sql.Rows) ([]types.TaskLogEntry, error) {
	var logEntries []types.TaskLogEntry
	for rows.Next() {
		entry, err := scanTaskLogEntry(rows)
		if err != nil {
			return nil, err
		}
		logEntries = append(logEntries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logEntries, nil
}

// collectTaskReportEntries iterates over rows and collects them into a slice of
// types.TaskReportEntry. It is the caller's responsibility to close rows.
func collectTaskReportEntries(rows *sql.Rows) ([]types.TaskReportEntry, error) {
	var entries []types.TaskReportEntry
	for rows.Next() {
		entry, err := scanTaskReportEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
