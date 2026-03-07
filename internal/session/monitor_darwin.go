//go:build darwin

package session

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
)

var (
	errSessionLockStateNotFound = errors.New("session lock state not found")
	sessionLockStatePattern     = regexp.MustCompile(`(?s)<key>CGSSessionScreenIsLocked</key>\s*<(true|false)/>`)
)

type darwinLockStatePoller struct{}

func newLockStatePoller() lockStatePoller {
	return darwinLockStatePoller{}
}

func (darwinLockStatePoller) Locked(ctx context.Context) (bool, error) {
	output, err := exec.CommandContext(ctx, "ioreg", "-d1", "-a", "-n", "Root").Output()
	if err != nil {
		return false, err
	}

	return parseDarwinSessionLockState(output)
}

func parseDarwinSessionLockState(output []byte) (bool, error) {
	matches := sessionLockStatePattern.FindSubmatch(output)
	if len(matches) != 2 {
		return false, errSessionLockStateNotFound
	}

	return string(matches[1]) == "true", nil
}
