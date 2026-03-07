//go:build darwin && cgo

package session

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc -fblocks
#cgo darwin LDFLAGS: -framework Foundation

#include <Foundation/Foundation.h>
#include <stdint.h>
#include <stdlib.h>

extern void hoursSessionEventCallback(uintptr_t handle, int eventType, long long seconds, int nanos);

typedef struct {
	NSDistributedNotificationCenter *center;
	NSOperationQueue *queue;
	id lockObserver;
	id unlockObserver;
} HoursEventMonitor;

static void hoursEmitSessionEvent(uintptr_t handle, int eventType) {
	NSDate *now = [NSDate date];
	NSTimeInterval ts = [now timeIntervalSince1970];
	long long seconds = (long long)ts;
	int nanos = (int)((ts - (NSTimeInterval)seconds) * 1000000000.0);
	if (nanos < 0) {
		nanos = 0;
	}
	if (nanos >= 1000000000) {
		seconds += 1;
		nanos = 0;
	}
	hoursSessionEventCallback(handle, eventType, seconds, nanos);
}

static void *hoursStartEventMonitor(uintptr_t handle) {
	@autoreleasepool {
		HoursEventMonitor *monitor = (HoursEventMonitor *)calloc(1, sizeof(HoursEventMonitor));
		if (monitor == NULL) {
			return NULL;
		}

		monitor->center = [NSDistributedNotificationCenter defaultCenter];
		monitor->queue = [[NSOperationQueue alloc] init];
		if (monitor->queue == nil) {
			free(monitor);
			return NULL;
		}
		[monitor->queue setMaxConcurrentOperationCount:1];

		monitor->lockObserver = [monitor->center addObserverForName:@"com.apple.screenIsLocked"
			object:nil
			queue:monitor->queue
			usingBlock:^(NSNotification *note) {
				(void)note;
				hoursEmitSessionEvent(handle, 0);
			}];

		monitor->unlockObserver = [monitor->center addObserverForName:@"com.apple.screenIsUnlocked"
			object:nil
			queue:monitor->queue
			usingBlock:^(NSNotification *note) {
				(void)note;
				hoursEmitSessionEvent(handle, 1);
			}];

		return monitor;
	}
}

static void hoursStopEventMonitor(void *state) {
	@autoreleasepool {
		HoursEventMonitor *monitor = (HoursEventMonitor *)state;
		if (monitor == NULL) {
			return;
		}

		if (monitor->lockObserver != nil) {
			[monitor->center removeObserver:monitor->lockObserver];
			monitor->lockObserver = nil;
		}
		if (monitor->unlockObserver != nil) {
			[monitor->center removeObserver:monitor->unlockObserver];
			monitor->unlockObserver = nil;
		}
		if (monitor->queue != nil) {
			[monitor->queue cancelAllOperations];
			monitor->queue = nil;
		}

		free(monitor);
	}
}
*/
import "C"

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

const darwinEventMonitorBuffer = 8

var (
	darwinEventMonitorRegistry sync.Map
	darwinEventMonitorCounter  atomic.Uint64
)

type darwinEventMonitor struct {
	events    chan Event
	incoming  chan Event
	handle    uint64
	native    unsafe.Pointer
	done      chan struct{}
	closeOnce sync.Once
}

func newEventMonitor(ctx context.Context) Monitor {
	handle := darwinEventMonitorCounter.Add(1)
	monitor := &darwinEventMonitor{
		events:   make(chan Event, darwinEventMonitorBuffer),
		incoming: make(chan Event, darwinEventMonitorBuffer),
		handle:   handle,
		done:     make(chan struct{}),
	}

	darwinEventMonitorRegistry.Store(handle, monitor.incoming)
	monitor.native = C.hoursStartEventMonitor(C.uintptr_t(handle))
	if monitor.native == nil {
		darwinEventMonitorRegistry.Delete(handle)
		close(monitor.done)
		close(monitor.events)
		return nil
	}

	go monitor.run()
	go func() {
		<-ctx.Done()
		_ = monitor.Close()
	}()

	return monitor
}

func (m *darwinEventMonitor) run() {
	defer close(m.events)

	for {
		select {
		case <-m.done:
			return
		case event := <-m.incoming:
			select {
			case <-m.done:
				return
			case m.events <- event:
			}
		}
	}
}

func (m *darwinEventMonitor) Events() <-chan Event { return m.events }

func (m *darwinEventMonitor) Close() error {
	m.closeOnce.Do(func() {
		darwinEventMonitorRegistry.Delete(m.handle)
		close(m.done)
		if m.native != nil {
			C.hoursStopEventMonitor(m.native)
			m.native = nil
		}
	})

	return nil
}
