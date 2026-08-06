#!/bin/zsh
# Warm the dashboard routes so first clicks after a dev-server restart are
# not blocked by Next.js on-demand compilation. Safe to run repeatedly.

set -u

CONFIG="${MULTICA_CONFIG:-/Users/zlearn/.multica/config.json}"
PORT="${FRONTEND_PORT:-3000}"
BASE="http://localhost:${PORT}/codex-board"
TOKEN=""

if [[ -f "$CONFIG" ]]; then
  TOKEN="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("token", ""))' "$CONFIG" 2>/dev/null || true)"
fi

if [[ -z "$TOKEN" ]]; then
  echo "prewarm: no token in $CONFIG, skipping" >&2
  exit 0
fi

routes=(inbox chat my-issues issues projects autopilots agents squads usage runtimes skills settings)

for route in "${routes[@]}"; do
  printf 'prewarm %-12s ' "$route"
  curl -s -o /dev/null \
    -H "Authorization: Bearer $TOKEN" \
    -w 'http=%{http_code} %{time_total}s\n' \
    "$BASE/$route"
done
