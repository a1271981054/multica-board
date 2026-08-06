#!/usr/bin/env bash
# Shared helpers for the Multica Board installer / CLI.
set -euo pipefail

BOARD_APP_NAME="Multica Board"
BOARD_INSTALL_DIR="${MULTICA_BOARD_INSTALL_DIR:-/Applications/${BOARD_APP_NAME}.app/Contents/Resources}"
BOARD_HOME="${MULTICA_BOARD_HOME:-$HOME/Library/Application Support/${BOARD_APP_NAME}}"
BOARD_LOGS="$HOME/Library/Logs/${BOARD_APP_NAME}"
BOARD_RUNTIME="$BOARD_HOME/runtime"
BOARD_CONFIG="$BOARD_HOME/multica-board.env"
BOARD_LAUNCH_DIR="$BOARD_HOME/launchd"
BOARD_VERSION_FILE="$BOARD_INSTALL_DIR/VERSION"
BOARD_TEMPLATES_DIR="${BOARD_TEMPLATES_DIR:-$BOARD_INSTALL_DIR/templates}"
BOARD_PG_DATA="$BOARD_HOME/pgdata"
BOARD_PG_SOCKET="$BOARD_HOME/pgsocket"
BOARD_REPO="${MULTICA_BOARD_REPO:-a1271981054/multica-board}"
BOARD_RELEASE_BASE="${MULTICA_BOARD_RELEASE_BASE:-https://github.com/${BOARD_REPO}/releases/latest/download}"

BOARD_LABELS=("com.multica-board.backend" "com.multica-board.web" "com.multica-board.daemon")

board_log()  { printf '%s\n' "$*"; }
board_info() { printf '==> %s\n' "$*"; }
board_ok()   { printf '✓ %s\n' "$*"; }
board_warn() { printf '⚠ %s\n' "$*" >&2; }
board_fail() { printf '✗ %s\n' "$*" >&2; exit 1; }

board_load_config() {
  if [[ -f "$BOARD_CONFIG" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$BOARD_CONFIG"
    set +a
  fi
}

board_require_config() {
  board_load_config
  if [[ -z "${MULTICA_BOARD_BACKEND_PORT:-}" ]]; then
    board_fail "Not configured yet. Run: multica-board setup"
  fi
}

board_arch() {
  case "$(uname -m)" in
    x86_64) echo "x86_64" ;;
    arm64) echo "arm64" ;;
    *) board_fail "Unsupported architecture: $(uname -m)" ;;
  esac
}

board_os_check() {
  [[ "$(uname -s)" == "Darwin" ]] || board_fail "Multica Board installer currently supports macOS only."
}

board_free_port() {
  local base="${1:-18080}"
  local port="$base"
  while lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; do
    port=$((port + 1))
  done
  echo "$port"
}

board_codex_path() {
  if [[ -n "${MULTICA_CODEX_PATH:-}" && -x "${MULTICA_CODEX_PATH}" ]]; then
    echo "$MULTICA_CODEX_PATH"
    return
  fi
  if command -v codex >/dev/null 2>&1; then
    command -v codex
    return
  fi
  for candidate in \
    "/Applications/ChatGPT.app/Contents/Resources/codex" \
    "/Applications/Codex.app/Contents/Resources/codex" \
    "$HOME/Applications/ChatGPT.app/Contents/Resources/codex"; do
    if [[ -x "$candidate" ]]; then
      echo "$candidate"
      return
    fi
  done
  echo ""
}

board_codex_home() {
  if [[ -n "${CODEX_HOME:-}" ]]; then
    echo "$CODEX_HOME"
  elif [[ -d "$HOME/.codex" ]]; then
    echo "$HOME/.codex"
  else
    echo ""
  fi
}

board_detect_node() {
  if [[ -x "$BOARD_RUNTIME/node/bin/node" ]]; then
    echo "$BOARD_RUNTIME/node/bin/node"
  elif command -v node >/dev/null 2>&1; then
    command -v node
  else
    echo ""
  fi
}

board_detect_postgres() {
  if [[ -x "$BOARD_RUNTIME/postgres/bin/pg_ctl" && -x "$BOARD_RUNTIME/postgres/bin/initdb" ]]; then
    echo "$BOARD_RUNTIME/postgres"
  elif command -v pg_ctl >/dev/null 2>&1 && command -v initdb >/dev/null 2>&1; then
    echo ""
  else
    echo ""
  fi
}

board_make_dirs() {
  mkdir -p "$BOARD_HOME" "$BOARD_LOGS" "$BOARD_RUNTIME" "$BOARD_LAUNCH_DIR" "$BOARD_PG_SOCKET"
}

board_template_plist() {
  local label="$1" out="$2"
  case "$label" in
    backend)
      sed -e "s|{{LABEL}}|com.multica-board.backend|g" \
          -e "s|{{CONFIG}}|$BOARD_CONFIG|g" \
          -e "s|{{STDOUT}}|$BOARD_LOGS/backend.log|g" \
          -e "s|{{STDERR}}|$BOARD_LOGS/backend.err.log|g" \
          -e "s|{{BIN}}|$BOARD_INSTALL_DIR/bin/server|g" \
          "$BOARD_TEMPLATES_DIR/backend.plist" > "$out"
      ;;
    web)
      sed -e "s|{{LABEL}}|com.multica-board.web|g" \
          -e "s|{{CONFIG}}|$BOARD_CONFIG|g" \
          -e "s|{{STDOUT}}|$BOARD_LOGS/web.log|g" \
          -e "s|{{STDERR}}|$BOARD_LOGS/web.err.log|g" \
          -e "s|{{WEB_DIR}}|$BOARD_INSTALL_DIR/web|g" \
          "$BOARD_TEMPLATES_DIR/web.plist" > "$out"
      ;;
    daemon)
      sed -e "s|{{LABEL}}|com.multica-board.daemon|g" \
          -e "s|{{CONFIG}}|$BOARD_CONFIG|g" \
          -e "s|{{STDOUT}}|$BOARD_LOGS/daemon.log|g" \
          -e "s|{{STDERR}}|$BOARD_LOGS/daemon.err.log|g" \
          -e "s|{{BIN}}|$BOARD_INSTALL_DIR/bin/multica|g" \
          "$BOARD_TEMPLATES_DIR/daemon.plist" > "$out"
      ;;
  esac
}

board_write_plists() {
  board_template_plist backend "$BOARD_LAUNCH_DIR/com.multica-board.backend.plist"
  board_template_plist web "$BOARD_LAUNCH_DIR/com.multica-board.web.plist"
  board_template_plist daemon "$BOARD_LAUNCH_DIR/com.multica-board.daemon.plist"
}

board_start() {
  board_require_config
  local uid
  uid="$(id -u)"
  for label in "${BOARD_LABELS[@]}"; do
    local plist="$BOARD_LAUNCH_DIR/$label.plist"
    [[ -f "$plist" ]] || board_fail "Missing plist: $plist"
    launchctl bootout "gui/$uid/$label" >/dev/null 2>&1 || true
    launchctl bootstrap "gui/$uid" "$plist"
    board_ok "started $label"
  done
}

board_stop() {
  local uid
  uid="$(id -u)"
  for label in "${BOARD_LABELS[@]}"; do
    launchctl bootout "gui/$uid/$label" >/dev/null 2>&1 || true
    board_ok "stopped $label"
  done
}

board_status() {
  local uid
  uid="$(id -u)"
  for label in "${BOARD_LABELS[@]}"; do
    if launchctl print "gui/$uid/$label" >/dev/null 2>&1; then
      printf '%-28s running\n' "$label"
    else
      printf '%-28s stopped\n' "$label"
    fi
  done
  board_load_config
  if [[ -n "${MULTICA_BOARD_BACKEND_PORT:-}" ]]; then
    local code
    code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}/api/config" || true)"
    printf 'backend health: %s\n' "$code"
  fi
}
