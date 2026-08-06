#!/usr/bin/env bash
# Multica Board installer (macOS).
#
# Usage:
#   curl -fsSL https://github.com/<owner>/multica-board/releases/latest/download/install.sh | sudo bash
#   sudo ./install.sh --source ./multica-board-macos-arm64.tar.gz
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/lib/common.sh" ]]; then
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/lib/common.sh"
else
  BOARD_APP_NAME="Multica Board"
  BOARD_INSTALL_DIR="/Applications/${BOARD_APP_NAME}.app/Contents/Resources"
  if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
    USER_HOME="$(sudo -u "$SUDO_USER" sh -c 'echo "$HOME"')"
  else
    USER_HOME="$HOME"
  fi
  BOARD_HOME="${MULTICA_BOARD_HOME:-$USER_HOME/Library/Application Support/${BOARD_APP_NAME}}"
  BOARD_REPO="${MULTICA_BOARD_REPO:-a1271981054/multica-board}"
  BOARD_RELEASE_BASE="${MULTICA_BOARD_RELEASE_BASE:-https://github.com/${BOARD_REPO}/releases/latest/download}"
fi

SOURCE="${MULTICA_BOARD_SOURCE:-}"
SKIP_PATCH="${MULTICA_BOARD_SKIP_PATCH:-false}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source) SOURCE="$2"; shift 2 ;;
    --skip-patch) SKIP_PATCH=true; shift ;;
    --version) VERSION="$2"; shift 2 ;;
    *) echo "Unknown arg: $1" >&2; exit 1 ;;
  esac
done

board_os_check
[[ "$(id -u)" -eq 0 ]] || board_fail "Install to /Applications requires sudo. Re-run with sudo."

ARCH="$(board_arch)"
VERSION="${VERSION:-latest}"
if [[ -z "$SOURCE" ]]; then
  SOURCE="$BOARD_RELEASE_BASE/multica-board-macos-${ARCH}.tar.gz"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

board_info "Downloading release: $SOURCE"
curl -fsSL "$SOURCE" -o "$TMP/multica-board.tar.gz"
mkdir -p "$TMP/bundle"
tar -xzf "$TMP/multica-board.tar.gz" -C "$TMP/bundle"

APP_DIR="/Applications/${BOARD_APP_NAME}.app"
RESOURCES="$APP_DIR/Contents/Resources"
mkdir -p "$RESOURCES/bin" "$RESOURCES/web" "$RESOURCES/lib" "$RESOURCES/patch"

board_info "Installing to $APP_DIR"
cp -R "$TMP/bundle/"* "$RESOURCES/"
chmod +x "$RESOURCES/bin/server" "$RESOURCES/bin/multica" "$RESOURCES/bin/migrate" "$RESOURCES/multica-board" 2>/dev/null || true
ln -sf "$RESOURCES/multica-board" /usr/local/bin/multica-board

board_ok "Installed Multica Board $VERSION"
board_info "Configuring local services (as $SUDO_USER)"
if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
  sudo -u "$SUDO_USER" env \
    MULTICA_BOARD_HOME="$BOARD_HOME" \
    MULTICA_BOARD_INSTALL_DIR="$RESOURCES" \
    "$RESOURCES/multica-board" setup
else
  "$RESOURCES/multica-board" setup
fi

if [[ "$SKIP_PATCH" != "true" ]]; then
  if [[ -n "${SUDO_USER:-}" && "${SUDO_USER}" != "root" ]]; then
    sudo -u "$SUDO_USER" "$RESOURCES/multica-board" patch || board_fail "Codex sidebar patch failed. Restore with: sudo -u $SUDO_USER $RESOURCES/multica-board patch --undo"
  else
    "$RESOURCES/multica-board" patch || board_fail "Codex sidebar patch failed. Restore with: $RESOURCES/multica-board patch --undo"
  fi
fi

board_ok "Install complete. Open http://127.0.0.1:${MULTICA_BOARD_WEB_PORT:-13000} or run: multica-board status"
