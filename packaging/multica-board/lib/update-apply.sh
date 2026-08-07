#!/usr/bin/env bash
# Applies a downloaded Multica Board release. Runs detached from the old
# bundle so replacing the app resources cannot corrupt the running updater.
set -euo pipefail
SRC_BUNDLE="$1"
INSTALL_DIR="$2"
BOARD_HOME="$3"
STATUS_FILE="$BOARD_HOME/updates/status.json"
LOG_DIR="$BOARD_HOME/updates"

write_status() {
  local state="$1" message="${2:-}"
  local latest
  latest="$(cat "$SRC_BUNDLE/VERSION" 2>/dev/null || echo "")"
  printf '{"status":"%s","latest":"%s","message":"%s","updated_at":"%s"}\n' \
    "$state" "$latest" "$message" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" > "$STATUS_FILE"
}

write_status applying "正在替换应用文件"

# Stop services before swapping files; the web/backend processes may be
# executing from the bundle being replaced.
for label in com.multica-board.backend com.multica-board.web com.multica-board.daemon; do
  launchctl bootout "gui/$(id -u)/$label" >/dev/null 2>&1 || true
done

mkdir -p "$INSTALL_DIR"
if [[ -w "$INSTALL_DIR" ]]; then
  cp -R "$SRC_BUNDLE/." "$INSTALL_DIR/"
else
  sudo cp -R "$SRC_BUNDLE/." "$INSTALL_DIR/"
fi
chmod +x "$INSTALL_DIR/bin/server" "$INSTALL_DIR/bin/multica" "$INSTALL_DIR/bin/migrate" "$INSTALL_DIR/multica-board" 2>/dev/null || true
chown -R "$USER:admin" "$INSTALL_DIR" 2>/dev/null || true

# Run any new DB migrations before the backend starts against the updated
# schema.
"$INSTALL_DIR/bin/migrate" up

for label in com.multica-board.backend com.multica-board.web com.multica-board.daemon; do
  plist="$BOARD_HOME/launchd/$label.plist"
  if [[ -f "$plist" ]]; then
    launchctl bootstrap "gui/$(id -u)" "$plist" || true
  fi
done

write_status done "更新完成"
rm -rf "$SRC_BUNDLE"
