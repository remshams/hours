# CLI Start/Stop Commands

## Overview

Add `hours stop` and `hours start <task_id>` CLI commands to enable remote control of time tracking via SSH. This allows automation scripts to pause/resume tracking based on external events (e.g., screen lock).

## Use Case

Running `hours` on a Raspberry Pi, accessed via SSH. When the client machine's screen locks, tracking should stop. When unlocked, tracking should resume on the same task.

```
[Client Machine]                          [Raspberry Pi]
Screen lock detected  ───SSH command──▶  hours stop
Screen unlock detected ───SSH command──▶ hours start <task_id>
```

## Commands

### `hours stop`

Stops the currently active time tracking.

| Flag | Description |
|------|-------------|
| `--quiet` / `-q` | Output only the task ID (for scripting) |
| `--dbpath` | Database path (standard flag) |

**Output:**
```
# Normal output
Stopped tracking "Task Summary" (id: 5)

# Quiet mode
5
```

**Exit codes:**
- `0`: Successfully stopped tracking
- `1`: Nothing was being tracked (or error)

### `hours start <task_id>`

Starts tracking a task by its ID.

| Argument | Description |
|----------|-------------|
| `<task_id>` | Numeric ID of the task to track |

| Flag | Description |
|------|-------------|
| `--dbpath` | Database path (standard flag) |

**Output:**
```
Started tracking "Task Summary"
```

**Exit codes:**
- `0`: Successfully started tracking
- `1`: Error (task doesn't exist, already tracking, etc.)

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| `stop` when nothing tracking | Error message, exit 1 |
| `start` when already tracking | Error message, exit 1 |
| `start` with non-existent task ID | Error message, exit 1 |
| `start` with inactive (deactivated) task | Works - allows tracking deactivated tasks |

## Implementation

### No Database Changes Required

Existing functions handle all requirements:
- `FetchActiveTaskDetails()` - get current tracking state
- `FinishActiveTL()` - stop tracking
- `InsertNewTL()` - start tracking

### Files to Modify

| File | Changes |
|------|---------|
| `internal/persistence/queries.go` | Add `FinishActiveTLForCLI()` - wrapper that fetches active details and finishes tracking |
| `internal/ui/active.go` | Add `StopTrackingCLI()` and `StartTrackingCLI()` functions |
| `cmd/root.go` | Add `stopCmd` and `startCmd` subcommands |

### New Functions

#### `internal/persistence/queries.go`

```go
// FinishActiveTLForCLI stops the currently active task log and returns its task ID.
// Returns -1 if nothing was being tracked.
func FinishActiveTLForCLI(db *sql.DB) (int, error)
```

#### `internal/ui/active.go`

```go
// StopTrackingCLI stops the current tracking and outputs the task ID.
// Returns error if nothing is being tracked.
func StopTrackingCLI(db *sql.DB, quiet bool) error

// StartTrackingCLI starts tracking a task by ID.
// Returns error if task doesn't exist or already tracking.
func StartTrackingCLI(db *sql.DB, taskID int) error
```

## Client Automation Reference

Example script for macOS using launchd to monitor screen lock events:

```bash
#!/bin/bash
# ~/bin/hours-lock-handler.sh

SSH_HOST="pi@raspberry-pi"
LAST_TASK_FILE="$HOME/.hours_last_task"

case "$1" in
  lock)
    TASK_ID=$(ssh "$SSH_HOST" "hours stop --quiet" 2>/dev/null)
    if [ -n "$TASK_ID" ] && [ "$TASK_ID" -gt 0 ] 2>/dev/null; then
      echo "$TASK_ID" > "$LAST_TASK_FILE"
    fi
    ;;
  unlock)
    if [ -f "$LAST_TASK_FILE" ]; then
      TASK_ID=$(cat "$LAST_TASK_FILE")
      ssh "$SSH_HOST" "hours start $TASK_ID" 2>/dev/null
    fi
    ;;
esac
```

### Platform-Specific Lock Detection

| Platform | Method |
|----------|--------|
| **macOS** | `launchd` event listener for `com.apple.screenIsLocked` / `com.apple.screenIsUnlocked` |
| **Linux (systemd)** | `loginctl session-status` or dbus monitoring |
| **Linux (GNOME)** | `org.gnome.ScreenSaver` dbus signal `ActiveChanged` |

## Testing

1. Test `hours stop` when nothing is tracking
2. Test `hours stop` when tracking is active
3. Test `hours start` with valid task ID
4. Test `hours start` with invalid task ID
5. Test `hours start` when already tracking
6. Test `hours start` with inactive task
7. Test `hours stop --quiet` output format