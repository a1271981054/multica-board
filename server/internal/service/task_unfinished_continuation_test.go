package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// TestCompleteTask_ContinuesUnfinishedInProgressIssue locks in the "agent
// stopped but the card is still in_progress" fix: completing an issue task
// while the issue is still in_progress must re-activate the same agent with a
// handoff note, and keep doing so (within the cap) until the agent moves the
// issue to in_review.
func TestCompleteTask_ContinuesUnfinishedInProgressIssue(t *testing.T) {
	ctx := context.Background()
	pool := newTaskClaimRacePool(t)
	svc := NewTaskService(db.New(pool), pool, nil, events.New())

	suffix := time.Now().UnixNano()
	var userID string
	if err := pool.QueryRow(ctx, `INSERT INTO "user" (name, email) VALUES ($1,$2) RETURNING id`,
		"Unfinished Continuation", fmt.Sprintf("unfinished-%d@multica.ai", suffix)).Scan(&userID); err != nil {
		t.Fatalf("create user: %v", err)
	}
	var workspaceID string
	if err := pool.QueryRow(ctx, `INSERT INTO workspace (name, slug, description, issue_prefix) VALUES ($1,$2,$3,$4) RETURNING id`,
		"Unfinished Continuation", fmt.Sprintf("unfinished-%d", suffix), "temp", "UFC").Scan(&workspaceID); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO member (workspace_id, user_id, role) VALUES ($1,$2,'owner')`, workspaceID, userID); err != nil {
		t.Fatalf("create member: %v", err)
	}
	var runtimeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_runtime (workspace_id, daemon_id, name, runtime_mode, provider, status, device_info, metadata, last_seen_at, visibility, owner_id)
		VALUES ($1, 'daemon-unfinished', $2, 'cloud', 'codex', 'online', 'test', '{}'::jsonb, now(), 'private', $3)
		RETURNING id`, workspaceID, "unfinished-rt", userID).Scan(&runtimeID); err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	var agentID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent (workspace_id, name, description, runtime_mode, runtime_config, runtime_id, visibility, max_concurrent_tasks, owner_id)
		VALUES ($1, 'unfinished-agent', '', 'cloud', '{}'::jsonb, $2, 'private', 5, $3)
		RETURNING id`, workspaceID, runtimeID, userID).Scan(&agentID); err != nil {
		t.Fatalf("create agent: %v", err)
	}
	var issueID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO issue (
			workspace_id, title, status, priority, creator_id, creator_type,
			number, position, assignee_type, assignee_id
		)
		VALUES (
			$1, 'unfinished issue', 'in_progress', 'medium', $2, 'member',
			(SELECT COALESCE(MAX(number), 90000) + 1 FROM issue WHERE workspace_id = $1),
			0, 'agent', $3
		)
		RETURNING id`, workspaceID, userID, agentID).Scan(&issueID); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	var taskID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id, status, priority,
			started_at, attempt, max_attempts, context
		)
		VALUES ($1, $2, $3, 'running', 0, now(), 1, 2, '{}'::jsonb)
		RETURNING id`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("create running task: %v", err)
	}

	t.Cleanup(func() {
		c := context.Background()
		pool.Exec(c, `DELETE FROM agent_task_queue WHERE agent_id = $1`, agentID)
		pool.Exec(c, `DELETE FROM issue WHERE id = $1`, issueID)
		pool.Exec(c, `DELETE FROM agent WHERE id = $1`, agentID)
		pool.Exec(c, `DELETE FROM agent_runtime WHERE id = $1`, runtimeID)
		pool.Exec(c, `DELETE FROM member WHERE workspace_id = $1`, workspaceID)
		pool.Exec(c, `DELETE FROM workspace WHERE id = $1`, workspaceID)
		pool.Exec(c, `DELETE FROM "user" WHERE id = $1`, userID)
	})

	result, _ := json.Marshal(map[string]any{"output": "修改已完成，正在做类型检查验证"})
	if _, err := svc.CompleteTask(ctx, util.MustParseUUID(taskID), result, "session-1", "/work", false, ""); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	var childID, handoff string
	var attempt, maxAttempts int32
	err := pool.QueryRow(ctx, `
		SELECT id::text, COALESCE(handoff_note, ''), attempt, max_attempts
		FROM agent_task_queue
		WHERE parent_task_id = $1 AND status = 'queued'
		ORDER BY created_at DESC
		LIMIT 1`, taskID).Scan(&childID, &handoff, &attempt, &maxAttempts)
	if err != nil {
		t.Fatalf("read continuation task: %v", err)
	}
	if attempt != 2 {
		t.Errorf("continuation attempt = %d, want 2", attempt)
	}
	if maxAttempts != maxUnfinishedContinuations {
		t.Errorf("continuation max_attempts = %d, want %d", maxAttempts, maxUnfinishedContinuations)
	}
	if handoff == "" {
		t.Error("continuation handoff note is empty")
	}
}
