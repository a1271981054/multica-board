# Multica Board (macOS Distribution)

Multica Board is a local, single-user task board that runs Multica's backend,
web UI, and Codex agent daemon on your own Mac and embeds the board into the
Codex sidebar.

## Install

Prerequisite: macOS with Codex installed (ChatGPT.app or Codex.app).

```bash
curl -fsSL https://github.com/a1271981054/multica-board/releases/latest/download/install.sh | sudo bash
```

For a local build:

```bash
./packaging/multica-board/build-release.sh
sudo ./packaging/multica-board/install.sh --source ./dist/multica-board-macos-$(uname -m).tar.gz
```

The installer:

- installs to `/Applications/Multica Board.app`
- downloads portable Node.js and PostgreSQL on first run
- verifies the portable PostgreSQL bundle against `checksums.txt`
- supports Apple Silicon (`arm64`) and Intel (`x86_64`) macOS bundles
- creates `~/Library/Application Support/Multica Board` for data/logs
- creates a local admin + workspace + daemon token automatically
- starts backend / web / daemon with user LaunchAgents
- patches the Codex sidebar (backup is kept; `multica-board patch --undo` restores)

Open `http://127.0.0.1:13000` after install.

## After install

- Quit Codex / ChatGPT before installing so the sidebar patch can run; the installer backs up the original app and `multica-board patch --undo` restores it.
- Reopen Codex after install: the left sidebar now has a "任务看板" entry that opens the board webview at `http://127.0.0.1:13000`.
- No email login: setup creates `local@multica.local` and enables auto-login, so the web app auto-submits the local verification code and lands directly on the board.

## Codex model configuration

The Board follows the Codex configuration that is active when a task starts.
That means the same installation supports both the native OpenAI/ChatGPT Codex
configuration and a CC Switch provider. Leave an agent's model and thinking
level at their runtime defaults when switching providers; if an older saved
model is not present in the active provider's catalog, the daemon drops that
one-run override and lets the active Codex runtime choose its default instead
of sending an incompatible model to the API.

## Commands

```bash
multica-board status
multica-board start
multica-board stop
multica-board patch
multica-board patch --undo
multica-board update
multica-board uninstall
```

## Runtime ports

| Service | Default port |
|---|---|
| Backend | 18080 |
| Web | 13000 |
| PostgreSQL | 15432 |

Override with `MULTICA_BOARD_BACKEND_PORT`, `MULTICA_BOARD_WEB_PORT`,
`MULTICA_BOARD_PG_PORT`.

## Portable PostgreSQL

The first run downloads `postgresql-17.10-macos-<arch>.tar.gz` from the latest
GitHub release, verifies its SHA-256 against `checksums.txt`, and unpacks
PostgreSQL 17.10 with the `pgvector` extension into the user data directory.
Both `arm64` and `x86_64` bundles are published; the installer picks the one
matching the machine.
To pin a custom mirror use `MULTICA_BOARD_POSTGRES_URL`, or pin a checksum with
`MULTICA_BOARD_POSTGRES_SHA256`. A mirror that also hosts `checksums.txt` can
point at it with `MULTICA_BOARD_CHECKSUMS_URL`. If the download is unavailable,
setup falls back to a Homebrew PostgreSQL 17/16 install when one exists.

## Notes

- No Apple notarization is included yet. macOS may ask you to allow the app
  or the `multica-board` binary when first launched.
- The Codex patch is version-sensitive. If the current Codex build does not
  match known anchors, setup aborts and leaves the app untouched.
- Multica Board is distributed under the Multica License. See `LICENSE` and
  `NOTICE`.
