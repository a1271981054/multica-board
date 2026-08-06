package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TestClaimTasksForRuntimes_ConcurrentQuickCreates locks in that two
// quick-create tasks for the same agent can be claimed in one batch. Quick
// create completion is keyed by origin_id (the task itself), so the old
// all-FK-null serialization is no longer needed.
func TestClaimTasksForRuntimes_ConcurrentQuickCreates(t *testing.T) {
	ctx := context.Background()
	pool := newTaskClaimRacePool(t)
	svc := NewTaskService(db.New(pool), pool, nil, events.New())

	suffix := time.Now().UnixNano()
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO "user" (name, email) VALUES ($1,$2) RETURNING id`,
		"Quick Parallel Test", fmt.Sprintf("quick-parallel-%d@multica.ai", suffix)).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}
	var workspaceID string
	if err := pool.QueryRow(ctx, `INSERT INTO workspace (name, slug, description, issue_prefix) VALUES ($1,$2,$3,$4) RETURNING id`,
		"Quick Parallel Test", fmt.Sprintf("quick-parallel-%d", suffix), "temp", "QPT").Scan(&workspaceID); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1,$2,'owner')`, workspaceID, userID); err != nil {
		t.Fatalf("create member: %v", err)
	}
	var runtimeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_runtime (workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at, visibility, owner_id)
		VALUES ($1, 'daemon-quick', $2, 'cloud', 'batch_provider', 'online', 'test', '{}'::jsonb, now(), 'private', $3)
		RETURNING id`, workspaceID, "quick-rt", userID).Scan(&runtimeID); err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	var agentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent (workspace_id, name, description, runtime_mode, runtime_config, runtime_id, visibility, max_concurrent_tasks, owner_id)
		VALUES ($1, 'quick-agent', '', 'cloud', '{}'::jsonb, $2, 'private', 5, $3)
		RETURNING id`, workspaceID, runtimeID, userID).Scan(&agentID); err != nil {
		t.Fatalf("create agent: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		pool.Exec(c, `DELETE FROM agent_task_queue WHERE agent_id = $1`, agentID)
		pool.Exec(c, `DELETE FROM agent WHERE id = $1`, agentID)
		pool.Exec(c, `DELETE FROM agent_runtime WHERE id = $1`, runtimeID)
		pool.Exec(c, `DELETE FROM member WHERE workspace_id = $1`, workspaceID)
		pool.Exec(c, `DELETE FROM workspace WHERE id = $1`, workspaceID)
		pool.Exec(c, `DELETE FROM "user" WHERE id = $1`, userID)
	})

	ctxJSON, _ := json.Marshal(map[string]any{
		"type":         "quick_create",
		"prompt":       "create a test issue",
		"requester_id": userID,
		"workspace_id": workspaceID,
	})
	for i := 0; i < 2; i++ {
		if _, err := pool.Exec(ctx, `
			INSERT INTO agent_task_queue (agent_id, runtime_id, issue_id, status, priority, context)
			VALUES ($1, $2, NULL, 'queued', 0, $3)
		`, agentID, runtimeID, ctxJSON); err != nil {
			t.Fatalf("create quick-create task %d: %v", i, err)
		}
	}

	ids := []pgtype.UUID{util.MustParseUUID(runtimeID)}
	claimed, err := svc.ClaimTasksForRuntimes(ctx, ids, 5)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed %d quick-create tasks, want 2 (parallel)", len(claimed))
	}
	seen := map[string]bool{}
	for _, task := range claimed {
		key := util.UUIDToString(task.ID)
		if seen[key] {
			t.Fatalf("duplicate task claimed: %s", key)
		}
		seen[key] = true
	}
}
