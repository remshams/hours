package session

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sequencePoller struct {
	mu     sync.Mutex
	states []bool
	index  int
}

func (p *sequencePoller) Locked(context.Context) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.states) == 0 {
		return false, nil
	}

	if p.index >= len(p.states) {
		return p.states[len(p.states)-1], nil
	}

	state := p.states[p.index]
	p.index++
	return state, nil
}

func TestNewMonitorWithNilPollerReturnsNoop(t *testing.T) {
	monitor := newPollingMonitor(context.Background(), nil, time.Millisecond)
	t.Cleanup(func() {
		require.NoError(t, monitor.Close())
	})

	assert.Nil(t, monitor.Events())
}

func TestPollingMonitorEmitsStateTransitions(t *testing.T) {
	monitor := newPollingMonitor(
		context.Background(),
		&sequencePoller{states: []bool{false, false, true, true, false}},
		5*time.Millisecond,
	)
	t.Cleanup(func() {
		require.NoError(t, monitor.Close())
	})

	firstEvent := readEvent(t, monitor.Events())
	assert.Equal(t, EventLocked, firstEvent.Type)

	secondEvent := readEvent(t, monitor.Events())
	assert.Equal(t, EventUnlocked, secondEvent.Type)
}

func readEvent(t *testing.T, events <-chan Event) Event {
	t.Helper()

	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("session event channel closed unexpectedly")
			return Event{}
		}
		return event
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for session event")
		return Event{}
	}
}
