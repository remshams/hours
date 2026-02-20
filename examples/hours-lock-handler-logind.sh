#!/bin/bash
# hours-lock-handler-logind.sh
# Example script for Linux using logind to monitor session lock events
#
# This script uses loginctl to monitor session lock/unlock events.
# Works with any desktop environment that uses systemd-logind.
#
# Setup:
#   1. Edit the SSH_HOST variable to point to your hours server
#   2. Make this script executable: chmod +x hours-lock-handler-logind.sh
#   3. Run it at startup (e.g., via systemd user service or ~/.bash_profile)
#
# Note: This requires systemd and dbus.

SSH_HOST="pi@raspberry-pi"
LAST_TASK_FILE="$HOME/.hours_last_task"

handle_lock() {
    TASK_ID=$(ssh "$SSH_HOST" "hours stop --quiet" 2>/dev/null)
    if [ -n "$TASK_ID" ] && [ "$TASK_ID" -gt 0 ] 2>/dev/null; then
        echo "$TASK_ID" > "$LAST_TASK_FILE"
        echo "Locked: stopped task $TASK_ID"
    fi
}

handle_unlock() {
    if [ -f "$LAST_TASK_FILE" ]; then
        TASK_ID=$(cat "$LAST_TASK_FILE")
        ssh "$SSH_HOST" "hours start $TASK_ID" 2>/dev/null
        echo "Unlocked: started task $TASK_ID"
    fi
}

dbus-monitor --system "type=signal,interface=org.freedesktop.login1.Manager" 2>/dev/null |
while read -r line; do
    case "$line" in
        *"LockSession"*)
            handle_lock
            ;;
        *"UnlockSession"*)
            handle_unlock
            ;;
    esac
done