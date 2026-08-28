CREATE TABLE IF NOT EXISTS prompt_audit_event_contexts (
    event_id BIGINT PRIMARY KEY REFERENCES prompt_audit_events(id) ON DELETE CASCADE,
    context_ciphertext TEXT NOT NULL,
    context_sha256 VARCHAR(64) NOT NULL,
    context_bytes BIGINT NOT NULL CHECK (context_bytes >= 0),
    segment_count INTEGER NOT NULL CHECK (segment_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE prompt_audit_event_contexts IS 'Application-encrypted complete canonical context for authorized Prompt Audit event download';
