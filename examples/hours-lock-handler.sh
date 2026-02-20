#!/bin/bash
# hours-lock-handler.sh
# Example script for macOS using launchd to monitor screen lock events
#
# This script is designed to work with launchd on macOS to automatically
# stop tracking when the screen locks and resume when it unlocks.
#
# Setup:
#   1. Edit the SSH_HOST variable to point to your hours server
#   2. Copy this script to ~/bin/hours-lock-handler.sh
#   3. Make it executable: chmod +x ~/bin/hours-lock-handler.sh
#   4. Create a launchd plist (see hours-lock-handler.plist.example)

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