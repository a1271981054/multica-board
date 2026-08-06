package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// acquireSharedCodexLock is retained for compatibility with older daemons /
// tests that exercised the old serialization. Current shared-mode execution no
// longer takes this lock: Codex supports concurrent threads against one
// CODEX_HOME, and shared mode must still allow parallel tasks.
func acquireSharedCodexLock(codexHome string) (func(), error) {
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		return nil, fmt.Errorf("create shared codex home: %w", err)
	}
	lockPath := filepath.Join(codexHome, ".multica-shared.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open shared codex lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("lock shared codex home: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

// backupSharedCodexHome snapshots the small but critical Codex state files
// before a task runs against the shared home.
func backupSharedCodexHome(codexHome, taskID string) error {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	backupDir := filepath.Join(
		codexHome,
		".taskboard",
		"backups",
		fmt.Sprintf("%s-%s", taskID, timestamp),
	)
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	for _, name := range []string{
		"auth.json",
		"config.toml",
		"config.json",
		"state_5.sqlite",
		"session_index.jsonl",
	} {
		src := filepath.Join(codexHome, name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(backupDir, name), data, 0o600); err != nil {
			return fmt.Errorf("backup %s: %w", name, err)
		}
	}
	return nil
}
