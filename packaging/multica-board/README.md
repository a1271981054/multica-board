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
- creates `~/Library/Application Support/Multica Board` for data/logs
- creates a local admin + workspace + daemon token automatically
- starts backend / web / daemon with user LaunchAgents
- patches the Codex sidebar (backup is kept; `multica-board patch --undo` restores)

Open `http://127.0.0.1:13000` after install.

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

## Notes

- No Apple notarization is included yet. macOS may ask you to allow the app
  or the `multica-board` binary when first launched.
- The Codex patch is version-sensitive. If the current Codex build does not
  match known anchors, setup aborts and leaves the app untouched.
- Multica Board is distributed under the Multica License. See `LICENSE` and
  `NOTICE`.
