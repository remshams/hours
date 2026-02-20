# Client Automation Examples

This folder contains example scripts and configurations for automating `hours stop` and `hours start` commands based on system events like screen lock/unlock.

## Use Case

Running `hours` on a remote machine (e.g., Raspberry Pi), accessed via SSH. When the client machine's screen locks, tracking should stop. When unlocked, tracking should resume on the same task.

```
[Client Machine]                          [Remote Machine]
Screen lock detected  ───SSH command──▶  hours stop
Screen unlock detected ───SSH command──▶ hours start <task_id>
```

## Files

### macOS

| File | Description |
|------|-------------|
| `hours-lock-handler.sh` | Bash script for handling lock/unlock events |
| `hours-lock-handler.plist.example` | macOS launchd plist for lock events |
| `hours-unlock-handler.plist.example` | macOS launchd plist for unlock events |

### Linux

| File | Description |
|------|-------------|
| `hours-lock-handler-linux.sh` | Script for GNOME (uses `org.gnome.ScreenSaver`) |
| `hours-lock-handler-kde.sh` | Script for KDE Plasma (uses `org.freedesktop.ScreenSaver`) |
| `hours-lock-handler-logind.sh` | Script for systemd-logind (desktop-agnostic) |
| `hours-lock-handler.service.example` | systemd user service file |

## macOS Setup

1. Edit `hours-lock-handler.sh` and set `SSH_HOST` to your remote machine
2. Copy the script to `~/bin/hours-lock-handler.sh`
3. Make it executable:
   ```bash
   chmod +x ~/bin/hours-lock-handler.sh
   ```
4. Copy the plist files to `~/Library/LaunchAgents/`:
   ```bash
   cp hours-lock-handler.plist.example ~/Library/LaunchAgents/com.user.hours-lock-handler.plist
   cp hours-unlock-handler.plist.example ~/Library/LaunchAgents/com.user.hours-unlock-handler.plist
   ```
5. Update the username in the plist files:
   ```bash
   sed -i '' 's/YOUR_USERNAME/your_actual_username/g' ~/Library/LaunchAgents/com.user.hours-*.plist
   ```
6. Load the launch agents:
   ```bash
   launchctl load ~/Library/LaunchAgents/com.user.hours-lock-handler.plist
   launchctl load ~/Library/LaunchAgents/com.user.hours-unlock-handler.plist
   ```

## Platform-Specific Lock Detection

| Platform | Method | Script |
|----------|--------|--------|
| **macOS** | `launchd` event listener | `hours-lock-handler.sh` |
| **Linux (GNOME)** | `org.gnome.ScreenSaver` dbus signal | `hours-lock-handler-linux.sh` |
| **Linux (KDE)** | `org.freedesktop.ScreenSaver` dbus signal | `hours-lock-handler-kde.sh` |
| **Linux (systemd)** | `loginctl` / logind dbus signals | `hours-lock-handler-logind.sh` |

## Linux Setup (systemd)

1. Edit the appropriate script and set `SSH_HOST` to your remote machine
2. Copy the script to `~/bin/`:
   ```bash
   mkdir -p ~/bin
   cp hours-lock-handler-linux.sh ~/bin/
   ```
3. Make it executable:
   ```bash
   chmod +x ~/bin/hours-lock-handler-linux.sh
   ```
4. Copy the systemd service file:
   ```bash
   cp hours-lock-handler.service.example ~/.config/systemd/user/hours-lock-handler.service
   ```
5. Enable and start the service:
   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now hours-lock-handler.service
   ```

### Choosing the Right Linux Script

- **GNOME users**: Use `hours-lock-handler-linux.sh`
- **KDE Plasma users**: Use `hours-lock-handler-kde.sh`
- **Other desktops / desktop-agnostic**: Use `hours-lock-handler-logind.sh`

## Commands Reference

### `hours stop`

Stops the currently active time tracking.

```bash
# Normal output
hours stop
# Output: Stopped tracking "Task Summary" (id: 5)

# Quiet mode (for scripting)
hours stop --quiet
# Output: 5
```

**Exit codes:**
- `0`: Successfully stopped tracking
- `1`: Nothing was being tracked (or error)

### `hours start <task_id>`

Starts tracking a task by its ID.

```bash
hours start 5
# Output: Started tracking "Task Summary"
```

**Exit codes:**
- `0`: Successfully started tracking
- `1`: Error (task doesn't exist, already tracking, etc.)