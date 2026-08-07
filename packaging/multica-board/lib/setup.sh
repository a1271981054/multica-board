#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/common.sh"

setup_runtime_node() {
  local node
  node="$(board_detect_node)"
  if [[ -n "$node" ]]; then
    board_info "Node found: $node"
    return
  fi
  local arch url dest
  arch="$(board_arch)"
  url="${MULTICA_BOARD_NODE_URL:-https://nodejs.org/dist/v22.20.0/node-v22.20.0-darwin-${arch}.tar.gz}"
  dest="$BOARD_RUNTIME/node"
  board_info "Downloading portable Node.js from $url"
  local tmp
  tmp="$(mktemp -d)"
  curl -fsSL "$url" -o "$tmp/node.tar.gz"
  mkdir -p "$dest"
  tar -xzf "$tmp/node.tar.gz" -C "$dest" --strip-components=1
  rm -rf "$tmp"
  chmod +x "$dest/bin/node"
}

setup_runtime_postgres() {
  local pg
  pg="$(board_detect_postgres)"
  if [[ -n "$pg" ]]; then
    board_info "PostgreSQL found: $pg"
    return
  fi
  local arch asset url checksums_url dest tmp expected actual brew_pg
  arch="$(board_arch)"
  asset="postgresql-17.10-macos-${arch}.tar.gz"
  url="${MULTICA_BOARD_POSTGRES_URL:-https://github.com/${BOARD_REPO}/releases/latest/download/${asset}}"
  checksums_url="${MULTICA_BOARD_CHECKSUMS_URL:-https://github.com/${BOARD_REPO}/releases/latest/download/checksums.txt}"
  dest="$BOARD_RUNTIME/postgres"
  tmp="$(mktemp -d)"
  board_info "Downloading portable PostgreSQL 17.10 + pgvector from $url"
  if ! curl -fsSL "$url" -o "$tmp/postgres.tar.gz"; then
    board_warn "Portable PostgreSQL download failed; falling back to Homebrew."
    for candidate in postgresql@17 postgresql@16 postgresql; do
      brew_pg="$(brew --prefix "$candidate" 2>/dev/null || true)"
      if [[ -n "$brew_pg" && -x "$brew_pg/bin/pg_ctl" ]]; then
        board_warn "Using Homebrew $candidate at $brew_pg"
        return
      fi
    done
    board_fail "Portable PostgreSQL download failed and no Homebrew PostgreSQL was found."
  fi
  expected="${MULTICA_BOARD_POSTGRES_SHA256:-}"
  if [[ -z "$expected" ]] && curl -fsSL "$checksums_url" -o "$tmp/checksums.txt" 2>/dev/null; then
    expected="$(awk -v f="$asset" '$2 == f {print $1}' "$tmp/checksums.txt" | head -1 || true)"
  fi
  if [[ -n "$expected" ]]; then
    actual="$(shasum -a 256 "$tmp/postgres.tar.gz" | awk '{print $1}')"
    if [[ "$actual" != "$expected" ]]; then
      board_fail "Portable PostgreSQL checksum mismatch (expected $expected, got $actual)."
    fi
    board_ok "PostgreSQL checksum verified"
  else
    board_warn "No checksum entry found for $asset; skipping verification."
  fi
  mkdir -p "$dest"
  tar -xzf "$tmp/postgres.tar.gz" -C "$dest" --strip-components=1
  rm -rf "$tmp"
  if [[ ! -x "$dest/bin/postgres" || ! -f "$dest/share/postgresql/extension/vector.control" ]]; then
    board_fail "Portable PostgreSQL bundle is incomplete (missing postgres binary or pgvector extension)."
  fi
  board_ok "Portable PostgreSQL 17.10 + pgvector installed at $dest"
}

setup_postgres() {
  local pgroot pg_ctl initdb psql createdb
  pgroot="$(board_detect_postgres)"
  if [[ -z "$pgroot" ]]; then
    pgroot="$(brew --prefix postgresql@17 2>/dev/null || brew --prefix postgresql@16 2>/dev/null || brew --prefix postgresql 2>/dev/null || true)"
  fi
  pg_ctl="${pgroot:+$pgroot/bin/pg_ctl}"
  initdb="${pgroot:+$pgroot/bin/initdb}"
  psql="${pgroot:+$pgroot/bin/psql}"
  createdb="${pgroot:+$pgroot/bin/createdb}"
  [[ -x "$pg_ctl" ]] || pg_ctl="$BOARD_RUNTIME/postgres/bin/pg_ctl"
  [[ -x "$initdb" ]] || initdb="$BOARD_RUNTIME/postgres/bin/initdb"
  [[ -x "$psql" ]] || psql="$BOARD_RUNTIME/postgres/bin/psql"
  [[ -x "$createdb" ]] || createdb="$BOARD_RUNTIME/postgres/bin/createdb"
  [[ -x "$pg_ctl" && -x "$initdb" ]] || board_fail "PostgreSQL binaries not found."

  if [[ ! -f "$BOARD_PG_DATA/PG_VERSION" ]]; then
    board_info "Initializing PostgreSQL data directory"
    mkdir -p "$BOARD_PG_DATA"
    "$initdb" -D "$BOARD_PG_DATA" -U "$MULTICA_BOARD_PG_USER" --auth=trust >/dev/null
  fi

  if ! "$pg_ctl" -D "$BOARD_PG_DATA" status >/dev/null 2>&1; then
    board_info "Starting PostgreSQL on 127.0.0.1:${MULTICA_BOARD_PG_PORT}"
    "$pg_ctl" -D "$BOARD_PG_DATA" -l "$BOARD_LOGS/postgres.log" \
      -o "-p ${MULTICA_BOARD_PG_PORT} -k '${BOARD_PG_SOCKET}' -h 127.0.0.1" start >/dev/null
  fi

  if ! "$psql" -h 127.0.0.1 -p "$MULTICA_BOARD_PG_PORT" -U "$MULTICA_BOARD_PG_USER" -lqt 2>/dev/null | cut -d'|' -f1 | grep -qw "$MULTICA_BOARD_PG_DB"; then
    "$createdb" -h 127.0.0.1 -p "$MULTICA_BOARD_PG_PORT" -U "$MULTICA_BOARD_PG_USER" "$MULTICA_BOARD_PG_DB"
  fi

  if "$psql" -h 127.0.0.1 -p "$MULTICA_BOARD_PG_PORT" -U "$MULTICA_BOARD_PG_USER" -d "$MULTICA_BOARD_PG_DB" \
    -v ON_ERROR_STOP=1 -c 'CREATE EXTENSION IF NOT EXISTS vector' >/dev/null 2>&1; then
    board_info "pgvector extension ready"
  else
    board_warn "pgvector extension unavailable; embedding features will be limited."
  fi
}

setup_write_env() {
  board_make_dirs
  local jwt vcs secret code
  jwt="$(openssl rand -hex 32)"
  vcs="$(openssl rand -base64 32 | tr -d '\n')"
  code="$(openssl rand -hex 3)"
  codex="$(board_codex_path)"
  codex_home="$(board_codex_home)"
  [[ -n "$codex" ]] || board_warn "Codex CLI not found; daemon will not run until MULTICA_BOARD_CODEX_PATH is set."
  cat > "$BOARD_CONFIG" <<CFG
# Generated by Multica Board setup. Do not commit or publish this file.
APP_ENV=development
MULTICA_BOARD_AUTO_LOGIN=true
MULTICA_BOARD_AUTO_LOGIN_EMAIL=${MULTICA_BOARD_EMAIL:-local@multica.local}
MULTICA_DEV_VERIFICATION_CODE=${code}
ALLOW_SIGNUP=false
DISABLE_WORKSPACE_CREATION=true
JWT_SECRET=${jwt}
MULTICA_VCS_SECRET_KEY=${vcs}
POSTGRES_DB=${MULTICA_BOARD_PG_DB:-multica}
POSTGRES_USER=${MULTICA_BOARD_PG_USER:-multica}
POSTGRES_PASSWORD=${MULTICA_BOARD_PG_PASSWORD:-multica}
POSTGRES_PORT=${MULTICA_BOARD_PG_PORT:-15432}
DATABASE_URL=postgres://${MULTICA_BOARD_PG_USER:-multica}:${MULTICA_BOARD_PG_PASSWORD:-multica}@127.0.0.1:${MULTICA_BOARD_PG_PORT:-15432}/${MULTICA_BOARD_PG_DB:-multica}?sslmode=disable
PORT=${MULTICA_BOARD_BACKEND_PORT}
FRONTEND_PORT=${MULTICA_BOARD_WEB_PORT}
FRONTEND_ORIGIN=http://127.0.0.1:${MULTICA_BOARD_WEB_PORT}
MULTICA_APP_URL=http://127.0.0.1:${MULTICA_BOARD_WEB_PORT}
MULTICA_PUBLIC_URL=http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}
NEXT_PUBLIC_API_URL=http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}
NEXT_PUBLIC_WS_URL=ws://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}/ws
LOCAL_UPLOAD_BASE_URL=http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}
LOCAL_UPLOAD_DIR=${BOARD_HOME}/uploads
MULTICA_BOARD_HOME=${BOARD_HOME}
MULTICA_BOARD_BACKEND_PORT=${MULTICA_BOARD_BACKEND_PORT}
MULTICA_BOARD_WEB_PORT=${MULTICA_BOARD_WEB_PORT}
MULTICA_BOARD_CODEX_PATH=${codex}
MULTICA_BOARD_CODEX_HOME=${codex_home}
MULTICA_BOARD_CODEX_SHARED_HOME=false
MULTICA_BOARD_CODEX_ISOLATED_HOME=true
MULTICA_BOARD_ALLOW_PARALLEL_LOCAL_DIRECTORY=true
MULTICA_BOARD_DEVICE_NAME=Multica Board
MULTICA_BOARD_INSTALL_DIR=${BOARD_INSTALL_DIR}
MULTICA_BOARD_REPO=${BOARD_REPO}
MULTICA_BOARD_NODE=$(board_detect_node)
CFG
  board_write_plists
}

bootstrap_http() {
  local base="$1" email="$2" code="$3"
  curl -fsS -X POST "$base/auth/send-code" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\"}" >/dev/null
  curl -fsS -X POST "$base/auth/verify-code" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"code\":\"$code\"}"
}

bootstrap() {
  board_load_config
  local base="http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}"
  local email="${MULTICA_BOARD_AUTO_LOGIN_EMAIL}"
  local code="${MULTICA_DEV_VERIFICATION_CODE}"
  local login
  login="$(bootstrap_http "$base" "$email" "$code")"
  local token
  token="$(printf '%s' "$login" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
  local slug="multica-board-$(date +%s | tail -c 5)"
  local ws_json
  ws_json="$(curl -fsS -X POST "$base/api/workspaces" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
    -d "{\"name\":\"Multica Board\",\"slug\":\"$slug\",\"issue_prefix\":\"MB\"}")"
  local ws_id
  ws_id="$(printf '%s' "$ws_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
  local pat_json
  pat_json="$(curl -fsS -X POST "$base/api/tokens" -H "Authorization: Bearer $token" -H 'Content-Type: application/json' \
    -d '{"name":"multica-board-daemon","expires_in_days":3650}')"
  local pat
  pat="$(printf '%s' "$pat_json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
  printf '\nMULTICA_BOARD_WORKSPACE_ID=%s\nMULTICA_TOKEN=%s\n' "$ws_id" "$pat" >> "$BOARD_CONFIG"
  printf 'MULTICA_BOARD_WORKSPACE_SLUG=%s\n' "$slug" >> "$BOARD_CONFIG"
  printf 'MULTICA_BOARD_AGENT_NAME=Codex\n' >> "$BOARD_CONFIG"
  board_ok "workspace $slug ($ws_id) ready"
}

create_codex_agent() {
  board_load_config
  local base="http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}"
  local slug="${MULTICA_BOARD_WORKSPACE_SLUG}"
  local login
  login="$(bootstrap_http "$base" "${MULTICA_BOARD_AUTO_LOGIN_EMAIL}" "${MULTICA_DEV_VERIFICATION_CODE}")"
  local token
  token="$(printf '%s' "$login" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
  local runtime_id=""
  for _ in $(seq 1 30); do
    local runtimes
    runtimes="$(curl -fsS -H "Authorization: Bearer $token" -H "X-Workspace-Slug: $slug" "$base/api/runtimes" || true)"
    runtime_id="$(printf '%s' "$runtimes" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(next((r["id"] for r in (d if isinstance(d,list) else d.get("runtimes",[])) if r.get("provider")=="codex"), ""))' 2>/dev/null || true)"
    [[ -n "$runtime_id" ]] && break
    sleep 2
  done
  [[ -n "$runtime_id" ]] || board_warn "Codex runtime not registered yet; create an agent in the UI after daemon connects."
  if [[ -n "$runtime_id" ]]; then
    curl -fsS -X POST "$base/api/agents" -H "Authorization: Bearer $token" -H "X-Workspace-Slug: $slug" -H 'Content-Type: application/json' \
      -d "{\"name\":\"Codex\",\"runtime_id\":\"$runtime_id\",\"visibility\":\"private\",\"permission_mode\":\"private\",\"max_concurrent_tasks\":3}" >/dev/null \
      || board_warn "Codex agent creation failed; create one in the UI."
    board_ok "Codex agent created"
  fi
}

cmd_setup() {
  board_os_check
  board_make_dirs
  MULTICA_BOARD_BACKEND_PORT="${MULTICA_BOARD_BACKEND_PORT:-$(board_free_port 18080)}"
  MULTICA_BOARD_WEB_PORT="${MULTICA_BOARD_WEB_PORT:-$(board_free_port 13000)}"
  MULTICA_BOARD_PG_PORT="${MULTICA_BOARD_PG_PORT:-15432}"
  setup_runtime_node
  setup_runtime_postgres
  setup_write_env
  setup_postgres
  board_load_config
  "$BOARD_INSTALL_DIR/bin/migrate" up
  local uid
  uid="$(id -u)"
  launchctl bootout "gui/$uid/com.multica-board.backend" >/dev/null 2>&1 || true
  launchctl bootstrap "gui/$uid" "$BOARD_LAUNCH_DIR/com.multica-board.backend.plist"
  for _ in $(seq 1 60); do
    [[ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${MULTICA_BOARD_BACKEND_PORT}/api/config" || true)" == "200" ]] && break
    sleep 1
  done
  bootstrap
  launchctl bootstrap "gui/$uid" "$BOARD_LAUNCH_DIR/com.multica-board.web.plist" || true
  launchctl bootstrap "gui/$uid" "$BOARD_LAUNCH_DIR/com.multica-board.daemon.plist" || true
  create_codex_agent
  board_ok "Setup complete: http://127.0.0.1:${MULTICA_BOARD_WEB_PORT}"
}
