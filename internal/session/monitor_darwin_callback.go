//go:build darwin && cgo

package session

/*
#include <stdint.h>
*/
import "C"

import "time"

//export hoursSessionEventCallback
func hoursSessionEventCallback(handle C.uintptr_t, eventType C.int, seconds C.longlong, nanos C.int) {
	value, ok := darwinEventMonitorRegistry.Load(uint64(handle))
	if !ok {
		return
	}

	incoming := value.(chan Event)
	event := Event{
		Type: EventLocked,
		At:   time.Unix(int64(seconds), int64(nanos)).In(time.Local).Truncate(time.Second),
	}
	if EventType(eventType) == EventUnlocked {
		event.Type = EventUnlocked
	}

	incoming <- event
}
