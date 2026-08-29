CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prompt_audit_events_mode_created
    ON prompt_audit_events(execution_mode, created_at DESC, id DESC);
