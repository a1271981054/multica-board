# Multica Board

Multica Board is a local, single-user task board that runs Multica's backend,
web UI, and Codex agent daemon on your own Mac and embeds the board into the
Codex sidebar.

## Install

Prerequisite: macOS with Codex installed (ChatGPT.app or Codex.app), or CC
Switch configured.

```bash
curl -fsSL https://github.com/a1271981054/multica-board/releases/latest/download/install.sh | sudo bash
```

The installer:

- installs to `/Applications/Multica Board.app`
- downloads portable Node.js and PostgreSQL 17.10 + pgvector on first run
- verifies downloads against `checksums.txt`
- supports Apple Silicon (`arm64`) and Intel (`x86_64`) macOS
- creates data and logs under `~/Library/Application Support/Multica Board`
- creates a local admin, workspace, and daemon token automatically
- starts backend / web / daemon with user LaunchAgents
- patches the Codex sidebar (a backup is kept; `multica-board patch --undo` restores)

After install, open `http://127.0.0.1:13000`.

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

## Releases

Download assets from the [Releases](https://github.com/a1271981054/multica-board/releases)
page: `multica-board-macos-<arch>.tar.gz`, `postgresql-17.10-macos-<arch>.tar.gz`,
`checksums.txt`, and `install.sh`.

## Notes

- Not Apple-signed or notarized yet; macOS may ask you to allow the app.
- Multica Board is based on [Multica](https://github.com/multica-ai/multica).
  See `LICENSE` and `NOTICE` for attribution.
