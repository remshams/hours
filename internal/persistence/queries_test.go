package persistence

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/dhth/hours/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite" // sqlite driver
)

const (
	secsInOneHour      = 60 * 60
	taskLogComment     = "a task log outside the time range"
	testComment        = "a test log"
	testCommentUpdated = "a task log, updated"
)

func TestRepository(t *testing.T) {
	testDB, err := sql.Open("sqlite", ":memory:")
	require.NoErrorf(t, err, "error opening DB: %v", err)

	err = InitDB(testDB)
	require.NoErrorf(t, err, "error initializing DB: %v", err)

	err = UpgradeDB(testDB, 1)
	require.NoErrorf(t, err, "error upgrading DB: %v", err)

	t.Run("TestInsertTask", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		// WHEN
		summary := "task 1"
		taskID, err := InsertTask(testDB, summary)

		// THEN
		require.NoError(t, err, "failed to insert task")

		task, fetchErr := fetchTaskByID(testDB, taskID)
		require.NoError(t, fetchErr, "failed to fetch task")

		assert.Equal(t, 3, task.ID)
		assert.Equal(t, summary, task.Summary)
		assert.True(t, task.Active)
		assert.Zero(t, task.SecsSpent)
	})

	t.Run("EditActiveTL", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		numSeconds := 60 * 90
		endTS := time.Now()
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		_, insertErr := InsertNewTL(testDB, taskID, beginTS)
		require.NoError(t, insertErr, "failed to insert task log")

		// WHEN
		updatedBeginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := testComment
		err = EditActiveTL(testDB, updatedBeginTS, &comment)
		activeTaskDetails, err := FetchActiveTaskDetails(testDB)
		require.NoError(t, err, "failed to fetch active task details")

		err = EditActiveTL(testDB, updatedBeginTS, nil)
		require.NoError(t, err, "failed to update active task log the second time")
		activeTaskDetailsTwo, err := FetchActiveTaskDetails(testDB)
		require.NoError(t, err, "failed to fetch active task details the second time")

		// THEN
		assert.Equal(t, taskID, activeTaskDetails.TaskID)
		assert.True(t, updatedBeginTS.Equal(activeTaskDetails.CurrentLogBeginTS))
		require.NotNil(t, activeTaskDetails.CurrentLogComment)
		assert.Equal(t, comment, *activeTaskDetails.CurrentLogComment)

		assert.Equal(t, taskID, activeTaskDetailsTwo.TaskID)
		assert.True(t, updatedBeginTS.Equal(activeTaskDetailsTwo.CurrentLogBeginTS))
		require.Nil(t, activeTaskDetailsTwo.CurrentLogComment)
	})

	t.Run("TestFinishActiveTL", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		numSeconds := 60 * 90
		endTS := time.Now()
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, insertErr := InsertNewTL(testDB, taskID, beginTS)
		require.NoError(t, insertErr, "failed to insert task log")

		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")
		numSecondsBefore := taskBefore.SecsSpent

		// WHEN
		comment := testComment
		err = FinishActiveTL(testDB, tlID, taskID, beginTS, endTS, numSeconds, &comment)

		// THEN
		require.NoError(t, err, "failed to update task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.Equal(t, numSeconds, taskLog.SecsSpent)
		require.NotNil(t, taskLog.Comment)
		assert.Equal(t, comment, *taskLog.Comment)
		assert.Equal(t, numSecondsBefore+numSeconds, taskAfter.SecsSpent)
	})

	t.Run("TestFinishActiveTL can save TL with empty comment", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		numSeconds := 60 * 90
		endTS := time.Now()
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, insertErr := InsertNewTL(testDB, taskID, beginTS)
		require.NoError(t, insertErr, "failed to insert task log")

		// WHEN
		err = FinishActiveTL(testDB, tlID, taskID, beginTS, endTS, numSeconds, nil)

		// THEN
		require.NoError(t, err, "failed to update task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		assert.Equal(t, numSeconds, taskLog.SecsSpent)
		require.Nil(t, taskLog.Comment)
	})

	t.Run("TestQuickSwitchActiveTL", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		secondTaskID := 2
		numSeconds := 60 * 90
		now := time.Now().Truncate(time.Second)
		beginTS := now.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, insertErr := InsertNewTL(testDB, taskID, beginTS)
		require.NoError(t, insertErr, "failed to insert task log")

		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		// WHEN
		result, err := QuickSwitchActiveTL(testDB, secondTaskID, now)

		// THEN
		require.NoError(t, err, "failed to quick switch active task")

		finishedTL, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch last active task log")

		activeTL, err := fetchActiveTLByID(testDB, result.CurrentlyActiveTLID)
		require.NoError(t, err, "failed to fetch active task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.True(t, beginTS.Equal(finishedTL.BeginTS), "finished TL's begin ts is not correct; got=%v, expected=%v", finishedTL.BeginTS, beginTS)
		assert.True(t, now.Equal(finishedTL.EndTS), "finished TL's end ts is not correct; got=%v, expected=%v", finishedTL.EndTS, now)
		assert.Equal(t, numSeconds, finishedTL.SecsSpent)
		require.Nil(t, finishedTL.Comment)
		assert.Equal(t, taskBefore.SecsSpent+numSeconds, taskAfter.SecsSpent)

		assert.True(t, now.Equal(activeTL.BeginTS), "active TL's begin ts is not correct; got=%v, expected=%v", activeTL.BeginTS, now)
		require.Nil(t, activeTL.Comment)
	})

	t.Run("TestQuickSwitchActiveTL works correctly with edited active task log", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		secondTaskID := 2
		numSeconds := 60 * 90
		now := time.Now().Truncate(time.Second)
		beginTS := now.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, insertErr := InsertNewTL(testDB, taskID, beginTS)
		require.NoError(t, insertErr, "failed to insert task log")

		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		updatedBeginTS := now.Add(time.Second * -1 * time.Duration(numSeconds*2))
		comment := testComment
		err = EditActiveTL(testDB, updatedBeginTS, &comment)
		require.NoError(t, err, "failed to update active task log")

		// WHEN
		result, err := QuickSwitchActiveTL(testDB, secondTaskID, now)

		// THEN
		require.NoError(t, err, "failed to quick switch active task")

		finishedTL, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch last active task log")

		activeTL, err := fetchActiveTLByID(testDB, result.CurrentlyActiveTLID)
		require.NoError(t, err, "failed to fetch active task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.True(t, updatedBeginTS.Equal(finishedTL.BeginTS), "finished TL's begin ts is not correct; got=%v, expected=%v", finishedTL.BeginTS, updatedBeginTS)
		assert.True(t, now.Equal(finishedTL.EndTS), "finished TL's end ts is not correct; got=%v, expected=%v", finishedTL.EndTS, now)
		assert.Equal(t, numSeconds*2, finishedTL.SecsSpent)
		require.NotNil(t, finishedTL.Comment)
		require.Equal(t, comment, *finishedTL.Comment)
		assert.Equal(t, taskBefore.SecsSpent+numSeconds*2, taskAfter.SecsSpent)

		assert.True(t, now.Equal(activeTL.BeginTS), "active TL's begin ts is not correct; got=%v, expected=%v", activeTL.BeginTS, now)
		require.Nil(t, activeTL.Comment)
	})

	t.Run("TestQuickSwitchActiveTL returns error if no task is active", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		now := time.Now().Truncate(time.Second)

		// WHEN
		_, err := QuickSwitchActiveTL(testDB, 1, now)

		// THEN
		require.ErrorIs(t, ErrNoTaskActive, err)
	})

	t.Run("TestInsertManualTL", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1

		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")
		numSecondsBefore := taskBefore.SecsSpent

		// WHEN
		comment := testComment
		numSeconds := 60 * 90
		endTS := time.Now()
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, err := InsertManualTL(testDB, taskID, beginTS, endTS, &comment)

		// THEN
		require.NoError(t, err, "failed to insert task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.Equal(t, numSeconds, taskLog.SecsSpent)
		require.NotNil(t, taskLog.Comment)
		assert.Equal(t, comment, *taskLog.Comment)
		assert.Equal(t, numSecondsBefore+numSeconds, taskAfter.SecsSpent)
	})

	t.Run("TestInsertManualTL can insert TL with empty comment", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1

		// WHEN
		numSeconds := 60 * 90
		endTS := time.Now()
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, err := InsertManualTL(testDB, taskID, beginTS, endTS, nil)

		// THEN
		require.NoError(t, err, "failed to insert task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		assert.Equal(t, numSeconds, taskLog.SecsSpent)
		assert.Nil(t, taskLog.Comment)
	})

	t.Run("TestEditSavedTL works when new time spent is larger than the previous one", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1

		comment := testComment
		numSeconds := 60 * 90
		endTS := time.Now().Truncate(time.Second)
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, err := InsertManualTL(testDB, taskID, beginTS, endTS, &comment)
		require.NoError(t, err, "failed to insert task log")
		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task after tl insert")

		// WHEN
		numSecondsDelta := 60
		updatedComment := testCommentUpdated
		newBeginTS := beginTS.Add(time.Second * -1 * time.Duration(numSecondsDelta*2))
		newEndTS := endTS.Add(time.Second * -1 * time.Duration(numSecondsDelta))
		_, err = EditSavedTL(testDB, tlID, newBeginTS, newEndTS, &updatedComment)

		// THEN
		require.NoError(t, err, "failed to edit saved task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.True(t, newBeginTS.Equal(taskLog.BeginTS), "new begin ts is not correct; expected=%v, got=%v", newBeginTS, taskLog.BeginTS)
		assert.True(t, newEndTS.Equal(taskLog.EndTS), "new end ts is not correct; expected=%v, got=%v", newEndTS, taskLog.EndTS)
		assert.Equal(t, numSeconds+numSecondsDelta, taskLog.SecsSpent)
		require.NotNil(t, taskLog.Comment)
		assert.Equal(t, updatedComment, *taskLog.Comment)
		assert.Equal(t, taskBefore.SecsSpent+numSecondsDelta, taskAfter.SecsSpent)
	})

	t.Run("TestEditSavedTL works when new time spent is smaller than the previous one", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1

		comment := testComment
		numSeconds := 60 * 90
		endTS := time.Now().Truncate(time.Second)
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, err := InsertManualTL(testDB, taskID, beginTS, endTS, &comment)
		require.NoError(t, err, "failed to insert task log")
		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task after tl insert")

		// WHEN
		numSecondsDelta := 60
		updatedComment := testCommentUpdated
		newBeginTS := beginTS.Add(time.Second * time.Duration(numSecondsDelta))
		_, err = EditSavedTL(testDB, tlID, newBeginTS, endTS, &updatedComment)

		// THEN
		require.NoError(t, err, "failed to edit saved task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.True(t, newBeginTS.Equal(taskLog.BeginTS), "new begin ts is not correct; expected=%v, got=%v", newBeginTS, taskLog.BeginTS)
		assert.True(t, endTS.Equal(taskLog.EndTS), "new end ts is not correct; expected=%v, got=%v", endTS, taskLog.EndTS)
		assert.Equal(t, numSeconds-numSecondsDelta, taskLog.SecsSpent)
		require.NotNil(t, taskLog.Comment)
		assert.Equal(t, updatedComment, *taskLog.Comment)
		assert.Equal(t, taskBefore.SecsSpent-numSecondsDelta, taskAfter.SecsSpent)
	})

	t.Run("TestEditSavedTL works when time spent is unchanged", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now().Truncate(time.Second)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1

		comment := testComment
		numSeconds := 60 * 90
		endTS := time.Now().Truncate(time.Second)
		beginTS := endTS.Add(time.Second * -1 * time.Duration(numSeconds))
		tlID, err := InsertManualTL(testDB, taskID, beginTS, endTS, &comment)
		require.NoError(t, err, "failed to insert task log")
		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task after tl insert")

		// WHEN
		numSecondsDelta := 60
		updatedComment := testCommentUpdated
		newBeginTS := beginTS.Add(time.Second * -1 * time.Duration(numSecondsDelta))
		newEndTS := endTS.Add(time.Second * -1 * time.Duration(numSecondsDelta))
		_, err = EditSavedTL(testDB, tlID, newBeginTS, newEndTS, &updatedComment)

		// THEN
		require.NoError(t, err, "failed to edit saved task log")

		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.True(t, newBeginTS.Equal(taskLog.BeginTS), "new begin ts is not correct; expected=%v, got=%v", newBeginTS, taskLog.BeginTS)
		assert.True(t, newEndTS.Equal(taskLog.EndTS), "new end ts is not correct; expected=%v, got=%v", newEndTS, taskLog.EndTS)
		assert.Equal(t, numSeconds, taskLog.SecsSpent)
		require.NotNil(t, taskLog.Comment)
		assert.Equal(t, updatedComment, *taskLog.Comment)
		assert.Equal(t, taskBefore.SecsSpent, taskAfter.SecsSpent)
	})

	t.Run("TestDeleteTaskLogEntry", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Now()
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)
		taskID := 1
		tlID := 1
		taskBefore, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")
		numSecondsBefore := taskBefore.SecsSpent
		taskLog, err := fetchTLByID(testDB, tlID)
		require.NoError(t, err, "failed to fetch task log")

		// WHEN
		err = DeleteTL(testDB, &taskLog)

		// THEN
		require.NoError(t, err, "failed to insert task log")

		taskAfter, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")

		assert.Equal(t, numSecondsBefore-taskLog.SecsSpent, taskAfter.SecsSpent)
	})

	t.Run("TestFetchTLEntriesBetweenTS for all tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchTLEntriesBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusAny, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 3)
	})

	t.Run("TestFetchTLEntriesBetweenTS for active tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		err = UpdateTaskActiveStatus(testDB, 1, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 10 * -1)
		entries, err := FetchTLEntriesBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusActive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)
	})

	t.Run("TestFetchTLEntriesBetweenTS for inactive tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		err = UpdateTaskActiveStatus(testDB, 2, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 10 * -1)
		entries, err := FetchTLEntriesBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusInactive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)
	})

	t.Run("TestFetchStats for all tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		comment := "an extra task log"
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		// WHEN
		entries, err := FetchStats(testDB, types.TaskStatusAny, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 2)

		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 3, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour+numSeconds, entries[0].SecsSpent)

		assert.Equal(t, 2, entries[1].TaskID)
		assert.Equal(t, 1, entries[1].NumEntries)
		assert.Equal(t, 4*secsInOneHour, entries[1].SecsSpent)
	})

	t.Run("TestFetchStats for active tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		err = UpdateTaskActiveStatus(testDB, 1, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		entries, err := FetchStats(testDB, types.TaskStatusActive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)

		assert.Equal(t, 2, entries[0].TaskID)
		assert.Equal(t, 1, entries[0].NumEntries)
		assert.Equal(t, 4*secsInOneHour, entries[0].SecsSpent)
	})

	t.Run("TestFetchStats for inactive tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		err = UpdateTaskActiveStatus(testDB, 2, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		entries, err := FetchStats(testDB, types.TaskStatusInactive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)

		assert.Equal(t, 2, entries[0].TaskID)
		assert.Equal(t, 1, entries[0].NumEntries)
		assert.Equal(t, 4*secsInOneHour, entries[0].SecsSpent)
	})

	t.Run("TestFetchStatsBetweenTS for all tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchStatsBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusAny, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 2)

		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 2, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[0].SecsSpent)

		assert.Equal(t, 2, entries[1].TaskID)
		assert.Equal(t, 1, entries[1].NumEntries)
		assert.Equal(t, 4*secsInOneHour, entries[1].SecsSpent)
	})

	t.Run("TestFetchStatsBetweenTS for active tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")
		err = UpdateTaskActiveStatus(testDB, 2, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchStatsBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusActive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)

		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 2, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[0].SecsSpent)
	})

	t.Run("TestFetchStatsBetweenTS for inactive tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")
		err = UpdateTaskActiveStatus(testDB, 1, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchStatsBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusInactive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")
		require.Len(t, entries, 1)

		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 2, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[0].SecsSpent)
	})

	t.Run("TestFetchReportBetweenTS for all tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchReportBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusAny, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")

		require.Len(t, entries, 2)
		assert.Equal(t, 2, entries[0].TaskID)
		assert.Equal(t, 1, entries[0].NumEntries)
		assert.Equal(t, 4*secsInOneHour, entries[0].SecsSpent)

		assert.Equal(t, 1, entries[1].TaskID)
		assert.Equal(t, 2, entries[1].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[1].SecsSpent)
	})

	t.Run("TestFetchReportBetweenTS for active tasks tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		err = UpdateTaskActiveStatus(testDB, 2, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchReportBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusActive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")

		require.Len(t, entries, 1)
		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 2, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[0].SecsSpent)
	})

	t.Run("TestFetchReportBetweenTS for inactive tasks tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN
		referenceTS := time.Date(2024, time.September, 1, 9, 0, 0, 0, time.Local)
		seedData := getTestData(referenceTS)
		seedDB(t, testDB, seedData)

		taskID := 1
		numSeconds := 60 * 90
		tlEndTS := referenceTS.Add(time.Hour * 2)
		tlBeginTS := tlEndTS.Add(time.Second * -1 * time.Duration(numSeconds))
		comment := taskLogComment
		_, err = InsertManualTL(testDB, taskID, tlBeginTS, tlEndTS, &comment)
		require.NoError(t, err, "failed to insert task log")

		err = UpdateTaskActiveStatus(testDB, 1, false)
		require.NoError(t, err, "failed to make task inactive")

		// WHEN
		reportBeginTS := referenceTS.Add(time.Hour * 24 * 7 * -2)
		entries, err := FetchReportBetweenTS(testDB, reportBeginTS, referenceTS, types.TaskStatusInactive, 100)

		// THEN
		require.NoError(t, err, "failed to fetch report entries")

		require.Len(t, entries, 1)
		assert.Equal(t, 1, entries[0].TaskID)
		assert.Equal(t, 2, entries[0].NumEntries)
		assert.Equal(t, 5*secsInOneHour, entries[0].SecsSpent)
	})

	err = testDB.Close()
	require.NoErrorf(t, err, "error closing DB: %v", err)
}

func TestArchiveStaleTasks(t *testing.T) {
	testDB, err := sql.Open("sqlite", ":memory:")
	require.NoErrorf(t, err, "error opening DB: %v", err)

	err = InitDB(testDB)
	require.NoErrorf(t, err, "error initializing DB: %v", err)

	err = UpgradeDB(testDB, 1)
	require.NoErrorf(t, err, "error upgrading DB: %v", err)

	t.Run("TestArchiveStaleTasks archives tasks with no recent log entries", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN - reference time is now, we have tasks with old logs (>2 weeks ago)
		referenceTS := time.Now()
		seedData := getTestData(referenceTS.Add(time.Hour * 24 * 21 * -1)) // 3 weeks ago
		seedDB(t, testDB, seedData)

		// Add a recent log entry for task 1 only
		recentLogEndTS := referenceTS.Add(time.Hour * -2)
		recentLogBeginTS := recentLogEndTS.Add(time.Hour * -1)
		recentComment := "recent log entry"
		_, err = InsertManualTL(testDB, 1, recentLogBeginTS, recentLogEndTS, &recentComment)
		require.NoError(t, err, "failed to insert recent task log")

		twoWeeksAgo := referenceTS.AddDate(0, 0, -14)

		// WHEN
		archivedCount, err := ArchiveStaleTasks(testDB, twoWeeksAgo)

		// THEN
		require.NoError(t, err, "failed to archive stale tasks")
		assert.Equal(t, 1, archivedCount, "expected 1 stale task to be archived")

		// Verify task 1 is still active (has recent log)
		task1, err := fetchTaskByID(testDB, 1)
		require.NoError(t, err, "failed to fetch task 1")
		assert.True(t, task1.Active, "task 1 should still be active")

		// Verify task 2 is now inactive (no recent logs)
		task2, err := fetchTaskByID(testDB, 2)
		require.NoError(t, err, "failed to fetch task 2")
		assert.False(t, task2.Active, "task 2 should be archived (inactive)")
	})

	t.Run("TestArchiveStaleTasks archives tasks with no log entries at all", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN - create a task with no log entries
		referenceTS := time.Now()
		taskID, err := InsertTask(testDB, "task with no logs")
		require.NoError(t, err, "failed to insert task")

		twoWeeksAgo := referenceTS.AddDate(0, 0, -14)

		// WHEN
		archivedCount, err := ArchiveStaleTasks(testDB, twoWeeksAgo)

		// THEN
		require.NoError(t, err, "failed to archive stale tasks")
		assert.Equal(t, 1, archivedCount, "expected 1 stale task to be archived")

		// Verify the task is now inactive
		task, err := fetchTaskByID(testDB, taskID)
		require.NoError(t, err, "failed to fetch task")
		assert.False(t, task.Active, "task with no logs should be archived")
	})

	t.Run("TestArchiveStaleTasks does not archive tasks with recent log entries", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN - tasks with recent logs (within last 2 weeks)
		referenceTS := time.Now()
		seedData := getTestData(referenceTS.Add(time.Hour * 24 * 3 * -1)) // 3 days ago
		seedDB(t, testDB, seedData)

		twoWeeksAgo := referenceTS.AddDate(0, 0, -14)

		// WHEN
		archivedCount, err := ArchiveStaleTasks(testDB, twoWeeksAgo)

		// THEN
		require.NoError(t, err, "failed to archive stale tasks")
		assert.Equal(t, 0, archivedCount, "expected 0 stale tasks to be archived")

		// Verify all tasks are still active
		task1, err := fetchTaskByID(testDB, 1)
		require.NoError(t, err, "failed to fetch task 1")
		assert.True(t, task1.Active, "task 1 should still be active")

		task2, err := fetchTaskByID(testDB, 2)
		require.NoError(t, err, "failed to fetch task 2")
		assert.True(t, task2.Active, "task 2 should still be active")
	})

	t.Run("TestArchiveStaleTasks does not archive already inactive tasks", func(t *testing.T) {
		t.Cleanup(func() { cleanupDB(t, testDB) })

		// GIVEN - inactive task with old logs
		referenceTS := time.Now()
		seedData := getTestData(referenceTS.Add(time.Hour * 24 * 21 * -1)) // 3 weeks ago
		seedDB(t, testDB, seedData)

		// Make task 2 inactive
		err = UpdateTaskActiveStatus(testDB, 2, false)
		require.NoError(t, err, "failed to make task inactive")

		twoWeeksAgo := referenceTS.AddDate(0, 0, -14)

		// WHEN
		archivedCount, err := ArchiveStaleTasks(testDB, twoWeeksAgo)

		// THEN
		require.NoError(t, err, "failed to archive stale tasks")
		assert.Equal(t, 1, archivedCount, "expected 1 stale task to be archived (only task 1)")

		// Verify task 2 is still inactive
		task2, err := fetchTaskByID(testDB, 2)
		require.NoError(t, err, "failed to fetch task 2")
		assert.False(t, task2.Active, "task 2 should still be inactive")
	})

	err = testDB.Close()
	require.NoErrorf(t, err, "error closing DB: %v", err)
}

func cleanupDB(t *testing.T, testDB *sql.DB) {
	t.Helper()

	var err error
	for _, tbl := range []string{"task_log", "task"} {
		_, err = testDB.Exec(fmt.Sprintf("DELETE FROM %s", tbl))
		require.NoErrorf(t, err, "failed to clean up table %q: %v", tbl, err)

		_, err := testDB.Exec("DELETE FROM sqlite_sequence WHERE name=?;", tbl)
		require.NoErrorf(t, err, "failed to reset auto increment for table %q: %v", tbl, err)
	}
}

type testData struct {
	tasks    []types.Task
	taskLogs []types.TaskLogEntry
}

func getTestData(referenceTS time.Time) testData {
	ua := referenceTS.UTC()
	ca := ua.Add(time.Hour * 24 * 7 * -1)
	tasks := []types.Task{
		{
			ID:        1,
			Summary:   "seeded task 1",
			Active:    true,
			CreatedAt: ca,
			UpdatedAt: ca.Add(time.Hour * 9),
			SecsSpent: 5 * secsInOneHour,
		},
		{
			ID:        2,
			Summary:   "seeded task 2",
			Active:    true,
			CreatedAt: ca,
			UpdatedAt: ca.Add(time.Hour * 6),
			SecsSpent: 4 * secsInOneHour,
		},
	}

	commentTask1TL1 := "task 1 tl 1"
	commentTask1TL2 := "task 1 tl 2"
	commentTask2TL1 := "task 2 tl 1"
	taskLogs := []types.TaskLogEntry{
		{
			ID:        1,
			TaskID:    1,
			BeginTS:   ca.Add(time.Hour * 2),
			EndTS:     ca.Add(time.Hour * 4),
			SecsSpent: 2 * secsInOneHour,
			Comment:   &commentTask1TL1,
		},
		{
			ID:        2,
			TaskID:    1,
			BeginTS:   ca.Add(time.Hour * 6),
			EndTS:     ca.Add(time.Hour * 9),
			SecsSpent: 3 * secsInOneHour,
			Comment:   &commentTask1TL2,
		},
		{
			ID:        3,
			TaskID:    2,
			BeginTS:   ca.Add(time.Hour * 2),
			EndTS:     ca.Add(time.Hour * 6),
			SecsSpent: 4 * secsInOneHour,
			Comment:   &commentTask2TL1,
		},
	}

	return testData{tasks, taskLogs}
}

func seedDB(t *testing.T, db *sql.DB, data testData) {
	t.Helper()

	for _, task := range data.tasks {
		_, err := db.Exec(`
INSERT INTO task (id, summary, secs_spent, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`, task.ID, task.Summary, task.SecsSpent, task.Active, task.CreatedAt, task.UpdatedAt)
		require.NoError(t, err, "failed to insert data into table \"task\": %v", err)
	}

	for _, taskLog := range data.taskLogs {
		_, err := db.Exec(`
INSERT INTO task_log (id, task_id, begin_ts, end_ts, secs_spent, comment, active)
VALUES (?, ?, ?, ?, ?, ?, ?)`, taskLog.ID, taskLog.TaskID, taskLog.BeginTS, taskLog.EndTS, taskLog.SecsSpent, taskLog.Comment, false)
		require.NoError(t, err, "failed to insert data into table \"task_log\": %v", err)
	}
}
