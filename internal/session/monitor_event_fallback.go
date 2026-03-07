//go:build !darwin || !cgo

package session

import "context"

func newEventMonitor(context.Context) Monitor { return nil }
