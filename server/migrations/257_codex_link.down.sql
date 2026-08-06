DROP INDEX IF EXISTS idx_project_codex_workspace_path;
ALTER TABLE project DROP COLUMN IF EXISTS codex_workspace_path;

DROP INDEX IF EXISTS idx_issue_codex_thread_id;
ALTER TABLE issue DROP COLUMN IF EXISTS codex_thread_id;
