package session

import (
	"context"
	"time"
)

type EventType uint

const (
	EventLocked EventType = iota
	EventUnlocked
)

type Event struct {
	Type EventType
	At   time.Time
}

type Monitor interface {
	Events() <-chan Event
	Close() error
}

const defaultPollInterval = 2 * time.Second

type lockStatePoller interface {
	Locked(context.Context) (bool, error)
}

type noopMonitor struct{}

func (noopMonitor) Events() <-chan Event { return nil }

func (noopMonitor) Close() error { return nil }

type pollingMonitor struct {
	events chan Event
	cancel context.CancelFunc
	done   chan struct{}
}

func NewMonitor(ctx context.Context) Monitor {
	if eventMonitor := newEventMonitor(ctx); eventMonitor != nil {
		return eventMonitor
	}

	return newMonitor(ctx, newLockStatePoller(), defaultPollInterval)
}

func newMonitor(ctx context.Context, poller lockStatePoller, interval time.Duration) Monitor {
	if poller == nil {
		return noopMonitor{}
	}

	if interval <= 0 {
		interval = defaultPollInterval
	}

	childCtx, cancel := context.WithCancel(ctx)
	monitor := &pollingMonitor{
		events: make(chan Event, 1),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	go monitor.run(childCtx, poller, interval)

	return monitor
}

func (m *pollingMonitor) Events() <-chan Event { return m.events }

func (m *pollingMonitor) Close() error {
	m.cancel()
	<-m.done
	return nil
}

func (m *pollingMonitor) run(ctx context.Context, poller lockStatePoller, interval time.Duration) {
	defer close(m.done)
	defer close(m.events)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var stateKnown bool
	var lastLocked bool

	poll := func() bool {
		locked, err := poller.Locked(ctx)
		if err != nil {
			return false
		}

		if !stateKnown {
			lastLocked = locked
			stateKnown = true
			return false
		}

		if locked == lastLocked {
			return false
		}

		lastLocked = locked
		eventType := EventUnlocked
		if locked {
			eventType = EventLocked
		}

		select {
		case m.events <- Event{Type: eventType, At: time.Now().Truncate(time.Second)}:
			return false
		case <-ctx.Done():
			return true
		}
	}

	if poll() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if poll() {
				return
			}
		}
	}
}
