ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS guard_endpoint_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS guard_model TEXT NOT NULL DEFAULT '';
