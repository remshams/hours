#!/bin/bash
# hours-lock-handler-kde.sh
# Example script for Linux using dbus to monitor screen lock events (KDE Plasma)
#
# This script monitors KDE ScreenSaver signals to detect lock/unlock events.
#
# Setup:
#   1. Edit the SSH_HOST variable to point to your hours server
#   2. Make this script executable: chmod +x hours-lock-handler-kde.sh
#   3. Run it at startup (e.g., via systemd user service or ~/.bash_profile)

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

dbus-monitor --session "type=signal,interface=org.freedesktop.ScreenSaver" 2>/dev/null |
while read -r line; do
    case "$line" in
        *"boolean true"*)
            handle_lock
            ;;
        *"boolean false"*)
            handle_unlock
            ;;
    esac
done