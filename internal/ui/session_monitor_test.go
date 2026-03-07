package ui

import (
	"testing"
	"time"

	"github.com/dhth/hours/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubSessionMonitor struct {
	events chan session.Event
}

func (m stubSessionMonitor) Events() <-chan session.Event { return m.events }

func (stubSessionMonitor) Close() error { return nil }

func TestWaitForSessionEventWithoutMonitorReturnsNil(t *testing.T) {
	assert.Nil(t, waitForSessionEvent(nil))
}

func TestWaitForSessionEventReturnsSessionStateChangedMsg(t *testing.T) {
	events := make(chan session.Event, 1)
	expectedAt := referenceTime.Add(15 * time.Minute)
	events <- session.Event{Type: session.EventLocked, At: expectedAt}

	cmd := waitForSessionEvent(stubSessionMonitor{events: events})
	require.NotNil(t, cmd)

	msg := cmd()
	changedMsg, ok := msg.(sessionStateChangedMsg)
	require.True(t, ok)
	assert.Equal(t, session.EventLocked, changedMsg.event.Type)
	assert.Equal(t, expectedAt, changedMsg.event.At)
}

func TestHandleMsgSessionStateChangedMsgUpdatesSessionLockStateAndRearms(t *testing.T) {
	events := make(chan session.Event, 1)
	m := createTestModel()
	m.sessionMonitor = stubSessionMonitor{events: events}

	cmds := m.handleMsg(sessionStateChangedMsg{event: session.Event{Type: session.EventLocked}})
	assert.True(t, m.sessionLocked)
	assert.False(t, m.trackingActive)
	assert.Equal(t, -1, m.autoStopTaskID)
	assert.Equal(t, -1, m.autoResumeTaskID)
	assert.False(t, m.changesLocked)
	require.Len(t, cmds, 1)

	events <- session.Event{Type: session.EventUnlocked}
	msg := cmds[0]()
	changedMsg, ok := msg.(sessionStateChangedMsg)
	require.True(t, ok)

	cmds = m.handleMsg(changedMsg)
	assert.False(t, m.sessionLocked)
	assert.False(t, m.trackingActive)
	assert.Equal(t, -1, m.autoStopTaskID)
	assert.Equal(t, -1, m.autoResumeTaskID)
	assert.False(t, m.changesLocked)
	require.Len(t, cmds, 1)
}

func TestHandleMsgSessionStateChangedMsgAutoStopsTrackedTask(t *testing.T) {
	events := make(chan session.Event, 1)
	m := createTestModel()
	m.sessionMonitor = stubSessionMonitor{events: events}
	task := createTestTask(1, "Tracked task", true, true, m.timeProvider)
	m.taskMap[1] = task
	m.trackingActive = true
	m.activeTaskID = 1
	m.activeTLBeginTS = referenceTime.Add(-1 * time.Hour)
	lockAt := referenceTime.Add(2*time.Hour + 750*time.Millisecond)

	cmds := m.handleMsg(sessionStateChangedMsg{event: session.Event{Type: session.EventLocked, At: lockAt}})

	assert.True(t, m.sessionLocked)
	assert.Equal(t, 1, m.autoStopTaskID)
	assert.Equal(t, -1, m.autoResumeTaskID)
	assert.Equal(t, lockAt.Truncate(time.Second), m.activeTLEndTS)
	require.Len(t, cmds, 2)
}

func TestHandleMsgSessionStateChangedMsgUnlockResumesOnlyAutoStoppedTask(t *testing.T) {
	t.Run("uses unlock event timestamp for eligible auto-resume", func(t *testing.T) {
		events := make(chan session.Event, 1)
		m := createTestModel()
		m.sessionMonitor = stubSessionMonitor{events: events}
		task := createTestTask(1, "Tracked task", true, false, m.timeProvider)
		m.taskMap[1] = task
		m.autoResumeTaskID = 1
		unlockAt := referenceTime.Add(45*time.Minute + 900*time.Millisecond)

		cmds := m.handleMsg(sessionStateChangedMsg{event: session.Event{Type: session.EventUnlocked, At: unlockAt}})

		assert.False(t, m.sessionLocked)
		assert.Equal(t, -1, m.autoResumeTaskID)
		assert.True(t, m.changesLocked)
		assert.Equal(t, unlockAt.Truncate(time.Second), m.activeTLBeginTS)
		require.Len(t, cmds, 2)
	})

	t.Run("manual stop before lock keeps unlock as a no-op", func(t *testing.T) {
		events := make(chan session.Event, 1)
		m := createTestModel()
		m.sessionMonitor = stubSessionMonitor{events: events}
		task := createTestTask(1, "Tracked task", true, false, m.timeProvider)
		m.taskMap[1] = task
		m.autoResumeNoticePending = true
		m.autoResumePauseDuration = 20 * time.Minute

		cmds := m.handleMsg(sessionStateChangedMsg{event: session.Event{Type: session.EventUnlocked, At: referenceTime.Add(time.Hour)}})

		assert.False(t, m.sessionLocked)
		assert.Equal(t, -1, m.autoResumeTaskID)
		assert.False(t, m.changesLocked)
		assert.False(t, m.trackingActive)
		assert.Empty(t, m.message.value)
		assert.False(t, m.autoResumeNoticePending)
		assert.Zero(t, m.autoResumePauseDuration)
		require.Len(t, cmds, 1)
	})

	t.Run("idle unlock stays silent and clears stale notice state", func(t *testing.T) {
		events := make(chan session.Event, 1)
		m := createTestModel()
		m.sessionMonitor = stubSessionMonitor{events: events}
		m.autoResumeNoticePending = true
		m.autoResumePauseDuration = 20 * time.Minute

		cmds := m.handleMsg(sessionStateChangedMsg{event: session.Event{Type: session.EventUnlocked, At: referenceTime.Add(2 * time.Hour)}})

		assert.False(t, m.sessionLocked)
		assert.Equal(t, -1, m.autoResumeTaskID)
		assert.False(t, m.changesLocked)
		assert.False(t, m.trackingActive)
		assert.Empty(t, m.message.value)
		assert.False(t, m.autoResumeNoticePending)
		assert.Zero(t, m.autoResumePauseDuration)
		require.Len(t, cmds, 1)
	})
}

func TestModelUpdateDispatchesSessionStateChangedMsgWhenFormViewActive(t *testing.T) {
	m := createTestModel()
	m.activeView = finishActiveTLView

	newModel, cmd := m.Update(sessionStateChangedMsg{event: session.Event{Type: session.EventLocked}})
	updated := newModel.(Model)

	assert.True(t, updated.sessionLocked)
	assert.Nil(t, cmd)
}
