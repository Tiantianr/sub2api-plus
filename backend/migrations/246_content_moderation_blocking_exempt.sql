ALTER TABLE content_moderation_logs
    ADD COLUMN IF NOT EXISTS blocking_exempt_at_request BOOLEAN NOT NULL DEFAULT FALSE;
