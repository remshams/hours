//go:build !darwin

package session

func newLockStatePoller() lockStatePoller {
	return nil
}
