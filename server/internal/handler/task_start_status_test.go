package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestStartTask_MovesTodoIssueToInProgress locks in the server-side
// todo -> in_progress promotion: when a daemon starts a task for an issue,
// the board must move the card immediately instead of waiting for the model
// to call `multica issue status`.
func TestStartTask_MovesTodoIssueToInProgress(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	var agentID, runtimeID string
	if err := testPool.QueryRow(ctx, `
		SELECT a.id, a.runtime_id
		FROM agent a
		WHERE a.workspace_id = $1 AND a.runtime_id IS NOT NULL
		LIMIT 1
	`, testWorkspaceID).Scan(&agentID, &runtimeID); err != nil {
		t.Fatalf("setup: get agent: %v", err)
	}

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (
			workspace_id, title, status, priority, creator_id, creator_type,
			number, position, assignee_type, assignee_id
		)
		VALUES (
			$1, 'start-task-status', 'todo', 'medium', $2, 'member',
			(SELECT COALESCE(MAX(number), 90000) + 1 FROM issue WHERE workspace_id = $1),
			0, 'agent', $3
		)
		RETURNING id
	`, testWorkspaceID, testUserID, agentID).Scan(&issueID); err != nil {
		t.Fatalf("setup: create issue: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id, status, priority, dispatched_at
		)
		VALUES ($1, $2, $3, 'dispatched', 0, now())
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("setup: create dispatched task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID)
	})

	w := httptest.NewRecorder()
	req := newDaemonTokenRequest(
		"POST",
		"/api/daemon/tasks/"+taskID+"/start",
		nil,
		testWorkspaceID,
		"legit-daemon",
	)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	testHandler.StartTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("StartTask: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status string
	if err := testPool.QueryRow(ctx,
		`SELECT status FROM issue WHERE id = $1`, issueID,
	).Scan(&status); err != nil {
		t.Fatalf("read issue status: %v", err)
	}
	if status != "in_progress" {
		t.Fatalf("issue status after StartTask: expected in_progress, got %q", status)
	}
}

// TestStartTask_MovesInReviewIssueToInProgress locks in the review-resume
// promotion: commenting on an in_review issue wakes the agent, so the card
// must move back to in_progress the moment the run starts.
func TestStartTask_MovesInReviewIssueToInProgress(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	ctx := context.Background()

	var agentID, runtimeID string
	if err := testPool.QueryRow(ctx, `
		SELECT a.id, a.runtime_id
		FROM agent a
		WHERE a.workspace_id = $1 AND a.runtime_id IS NOT NULL
		LIMIT 1
	`, testWorkspaceID).Scan(&agentID, &runtimeID); err != nil {
		t.Fatalf("setup: get agent: %v", err)
	}

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (
			workspace_id, title, status, priority, creator_id, creator_type,
			number, position, assignee_type, assignee_id
		)
		VALUES (
			$1, 'start-task-review-status', 'in_review', 'medium', $2, 'member',
			(SELECT COALESCE(MAX(number), 90000) + 1 FROM issue WHERE workspace_id = $1),
			0, 'agent', $3
		)
		RETURNING id
	`, testWorkspaceID, testUserID, agentID).Scan(&issueID); err != nil {
		t.Fatalf("setup: create issue: %v", err)
	}
	t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (
			agent_id, runtime_id, issue_id, status, priority, dispatched_at
		)
		VALUES ($1, $2, $3, 'dispatched', 0, now())
		RETURNING id
	`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
		t.Fatalf("setup: create dispatched task: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID)
	})

	w := httptest.NewRecorder()
	req := newDaemonTokenRequest(
		"POST",
		"/api/daemon/tasks/"+taskID+"/start",
		nil,
		testWorkspaceID,
		"legit-daemon",
	)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("taskId", taskID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	testHandler.StartTask(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("StartTask: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var status string
	if err := testPool.QueryRow(ctx,
		`SELECT status FROM issue WHERE id = $1`, issueID,
	).Scan(&status); err != nil {
		t.Fatalf("read issue status: %v", err)
	}
	if status != "in_progress" {
		t.Fatalf("issue status after StartTask: expected in_progress, got %q", status)
	}
}

// TestStartTask_MovesDoneAndBlockedIssuesToInProgress locks in the
// comment-reopen promotion: commenting on a done or blocked card re-triggers
// the agent, so the card must move back to in_progress when the run starts.
func TestStartTask_MovesDoneAndBlockedIssuesToInProgress(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}

	for _, initial := range []string{"done", "blocked"} {
		initial := initial
		t.Run(initial, func(t *testing.T) {
			ctx := context.Background()

			var agentID, runtimeID string
			if err := testPool.QueryRow(ctx, `
				SELECT a.id, a.runtime_id
				FROM agent a
				WHERE a.workspace_id = $1 AND a.runtime_id IS NOT NULL
				LIMIT 1
			`, testWorkspaceID).Scan(&agentID, &runtimeID); err != nil {
				t.Fatalf("setup: get agent: %v", err)
			}

			var issueID string
			if err := testPool.QueryRow(ctx, `
				INSERT INTO issue (
					workspace_id, title, status, priority, creator_id, creator_type,
					number, position, assignee_type, assignee_id
				)
				VALUES (
					$1, 'start-task-reopen-status', $4, 'medium', $2, 'member',
					(SELECT COALESCE(MAX(number), 91000) + 1 FROM issue WHERE workspace_id = $1),
					0, 'agent', $3
				)
				RETURNING id
			`, testWorkspaceID, testUserID, agentID, initial).Scan(&issueID); err != nil {
				t.Fatalf("setup: create issue: %v", err)
			}
			t.Cleanup(func() { _, _ = testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })

			var taskID string
			if err := testPool.QueryRow(ctx, `
				INSERT INTO agent_task_queue (
					agent_id, runtime_id, issue_id, status, priority, dispatched_at
				)
				VALUES ($1, $2, $3, 'dispatched', 0, now())
				RETURNING id
			`, agentID, runtimeID, issueID).Scan(&taskID); err != nil {
				t.Fatalf("setup: create dispatched task: %v", err)
			}
			t.Cleanup(func() {
				_, _ = testPool.Exec(context.Background(), `DELETE FROM agent_task_queue WHERE id = $1`, taskID)
			})

			w := httptest.NewRecorder()
			req := newDaemonTokenRequest(
				"POST",
				"/api/daemon/tasks/"+taskID+"/start",
				nil,
				testWorkspaceID,
				"legit-daemon",
			)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("taskId", taskID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			testHandler.StartTask(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("StartTask: expected 200, got %d: %s", w.Code, w.Body.String())
			}

			var status string
			if err := testPool.QueryRow(ctx,
				`SELECT status FROM issue WHERE id = $1`, issueID,
			).Scan(&status); err != nil {
				t.Fatalf("read issue status: %v", err)
			}
			if status != "in_progress" {
				t.Fatalf("issue status after StartTask from %s: expected in_progress, got %q", initial, status)
			}
		})
	}
}
