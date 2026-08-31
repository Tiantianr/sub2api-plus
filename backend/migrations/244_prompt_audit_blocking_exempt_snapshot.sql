ALTER TABLE prompt_audit_jobs
    ADD COLUMN IF NOT EXISTS blocking_exempt_at_request BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS blocking_exempt_at_request BOOLEAN NOT NULL DEFAULT FALSE;
