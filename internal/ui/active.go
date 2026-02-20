package ui

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	pers "github.com/dhth/hours/internal/persistence"
	"github.com/dhth/hours/internal/types"
)

const (
	ActiveTaskPlaceholder     = "{{task}}"
	ActiveTaskTimePlaceholder = "{{time}}"
	activeSecsThreshold       = 60
	activeSecsThresholdStr    = "<1m"
)

var (
	ErrNoTaskActiveForCLI = errors.New("no task is being actively tracked")
	ErrAlreadyTracking    = errors.New("a task is already being tracked")
	ErrTaskNotFound       = errors.New("task not found")
)

func ShowActiveTask(db *sql.DB, writer io.Writer, template string) error {
	activeTaskDetails, err := pers.FetchActiveTaskDetails(db)
	if err != nil {
		return err
	}

	if activeTaskDetails.TaskID == -1 {
		return nil
	}

	timeSpent := time.Since(activeTaskDetails.CurrentLogBeginTS).Seconds()
	var timeSpentStr string
	if timeSpent <= activeSecsThreshold {
		timeSpentStr = activeSecsThresholdStr
	} else {
		timeSpentStr = types.HumanizeDuration(int(timeSpent))
	}

	activeStr := strings.Replace(template, ActiveTaskPlaceholder, activeTaskDetails.TaskSummary, 1)
	activeStr = strings.Replace(activeStr, ActiveTaskTimePlaceholder, timeSpentStr, 1)
	fmt.Fprint(writer, activeStr)
	return nil
}

// StopTrackingCLI stops the current tracking and outputs the task ID.
// Returns error if nothing is being tracked.
func StopTrackingCLI(db *sql.DB, writer io.Writer, quiet bool) error {
	taskID, taskSummary, err := pers.FinishActiveTLForCLI(db)
	if err != nil {
		return err
	}
	if taskID == -1 {
		return ErrNoTaskActiveForCLI
	}

	if quiet {
		fmt.Fprintf(writer, "%d", taskID)
	} else {
		fmt.Fprintf(writer, "Stopped tracking %q (id: %d)\n", taskSummary, taskID)
	}

	return nil
}

// StartTrackingCLI starts tracking a task by ID.
// Returns error if task doesn't exist or already tracking.
func StartTrackingCLI(db *sql.DB, writer io.Writer, taskID int) error {
	activeTaskDetails, err := pers.FetchActiveTaskDetails(db)
	if err != nil {
		return err
	}
	if activeTaskDetails.TaskID != -1 {
		return ErrAlreadyTracking
	}

	tasks, err := pers.FetchTasks(db, true, 1000)
	if err != nil {
		return err
	}

	var taskSummary string
	var taskFound bool
	for _, task := range tasks {
		if task.ID == taskID {
			taskSummary = task.Summary
			taskFound = true
			break
		}
	}

	inactiveTasks, err := pers.FetchTasks(db, false, 1000)
	if err != nil {
		return err
	}

	for _, task := range inactiveTasks {
		if task.ID == taskID {
			taskSummary = task.Summary
			taskFound = true
			break
		}
	}

	if !taskFound {
		return ErrTaskNotFound
	}

	_, err = pers.InsertNewTL(db, taskID, time.Now())
	if err != nil {
		return err
	}

	fmt.Fprintf(writer, "Started tracking %q\n", taskSummary)
	return nil
}
