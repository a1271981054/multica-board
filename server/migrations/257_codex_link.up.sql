ALTER TABLE issue ADD COLUMN codex_thread_id TEXT;
CREATE UNIQUE INDEX idx_issue_codex_thread_id ON issue(codex_thread_id) WHERE codex_thread_id IS NOT NULL;

ALTER TABLE project ADD COLUMN codex_workspace_path TEXT;
CREATE UNIQUE INDEX idx_project_codex_workspace_path ON project(codex_workspace_path) WHERE codex_workspace_path IS NOT NULL;
