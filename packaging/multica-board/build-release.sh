#!/usr/bin/env bash
# Builds a macOS Multica Board release bundle.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PACKAGING="$REPO_ROOT/packaging/multica-board"
VERSION="$(cat "$PACKAGING/VERSION")"
ARCH="${MULTICA_BOARD_ARCH:-$(uname -m)}"
case "$ARCH" in
  arm64) GOARCH=arm64 ;;
  x86_64) GOARCH=amd64 ;;
  *) echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
esac
export GOOS=darwin GOARCH="$GOARCH"
OUT="${MULTICA_BOARD_OUT_DIR:-$REPO_ROOT/dist}"
BUNDLE="$(mktemp -d)"
trap 'rm -rf "$BUNDLE"' EXIT

board_info() { printf '==> %s\n' "$*"; }

board_info "Building backend + daemon + migrate"
cd "$REPO_ROOT/server"
go build -ldflags "-s -w -X main.version=${VERSION}" -o "$BUNDLE/bin/server" ./cmd/server
go build -ldflags "-s -w -X main.version=${VERSION}" -o "$BUNDLE/bin/multica" ./cmd/multica
go build -ldflags "-s -w" -o "$BUNDLE/bin/migrate" ./cmd/migrate

board_info "Building Next.js standalone web"
cd "$REPO_ROOT"
STANDALONE=true pnpm --filter @multica/web build

board_info "Assembling web runtime"
mkdir -p "$BUNDLE/web"
cp -R "$REPO_ROOT/apps/web/.next/standalone/." "$BUNDLE/web/"
mkdir -p "$BUNDLE/web/apps/web/.next"
cp -R "$REPO_ROOT/apps/web/.next/static" "$BUNDLE/web/apps/web/.next/static"
if [[ -d "$REPO_ROOT/apps/web/public" ]]; then
  mkdir -p "$BUNDLE/web/apps/web/public"
  cp -R "$REPO_ROOT/apps/web/public/." "$BUNDLE/web/apps/web/public/"
fi

board_info "Bundling control command + patch tool"
cp "$PACKAGING/multica-board" "$BUNDLE/multica-board"
cp -R "$PACKAGING/lib" "$BUNDLE/lib"
cp -R "$PACKAGING/templates" "$BUNDLE/templates"
cp -R "$PACKAGING/patch" "$BUNDLE/patch"
mkdir -p "$BUNDLE/patch/node_modules/.bin" "$BUNDLE/patch/node_modules/@electron/asar/bin"
ASAR_PKG="$REPO_ROOT/node_modules/.pnpm/@electron+asar@3.4.1/node_modules/@electron/asar"
cp -R "$ASAR_PKG/lib" "$BUNDLE/patch/node_modules/@electron/asar/lib"
cp "$ASAR_PKG/bin/asar.js" "$BUNDLE/patch/node_modules/@electron/asar/bin/asar.js"
cp "$ASAR_PKG/package.json" "$BUNDLE/patch/node_modules/@electron/asar/package.json"
cat > "$BUNDLE/patch/node_modules/.bin/asar" <<'WRAPPER'
#!/bin/sh
DIR="$(cd "$(dirname "$0")" && pwd)"
exec node "$DIR/../@electron/asar/bin/asar.js" "$@"
WRAPPER
chmod +x "$BUNDLE/patch/node_modules/.bin/asar"

board_info "Bundling SQL migrations"
mkdir -p "$BUNDLE/migrations"
cp "$REPO_ROOT/server/migrations/"*.sql "$BUNDLE/migrations/"

cp "$PACKAGING/VERSION" "$BUNDLE/VERSION"
cp "$REPO_ROOT/LICENSE" "$BUNDLE/LICENSE"
cp "$REPO_ROOT/NOTICE" "$BUNDLE/NOTICE"
cp "$PACKAGING/README.md" "$BUNDLE/README.md" 2>/dev/null || true

mkdir -p "$OUT"
TAR="$OUT/multica-board-macos-${ARCH}.tar.gz"
tar -C "$BUNDLE" -czf "$TAR" .
(cd "$OUT" && shasum -a 256 "$(basename "$TAR")" > "checksums.txt")
board_info "Built $TAR"
