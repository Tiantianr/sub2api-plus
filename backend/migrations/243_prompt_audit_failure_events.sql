ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS error_code VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS error_message VARCHAR(160) NOT NULL DEFAULT '';

ALTER TABLE prompt_audit_events
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_events_decision;
ALTER TABLE prompt_audit_events
    ADD CONSTRAINT chk_prompt_audit_events_decision
        CHECK (decision IN ('pass', 'flag', 'critical', 'failed')) NOT VALID;
ALTER TABLE prompt_audit_events
    VALIDATE CONSTRAINT chk_prompt_audit_events_decision;

ALTER TABLE prompt_audit_events
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_events_risk_level;
ALTER TABLE prompt_audit_events
    ADD CONSTRAINT chk_prompt_audit_events_risk_level
        CHECK (risk_level IN ('low', 'medium', 'high', 'critical', 'unknown')) NOT VALID;
ALTER TABLE prompt_audit_events
    VALIDATE CONSTRAINT chk_prompt_audit_events_risk_level;

ALTER TABLE prompt_audit_events
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_events_action;
ALTER TABLE prompt_audit_events
    ADD CONSTRAINT chk_prompt_audit_events_action
        CHECK (action IN ('Allow', 'Warn', 'Block', 'Error')) NOT VALID;
ALTER TABLE prompt_audit_events
    VALIDATE CONSTRAINT chk_prompt_audit_events_action;

ALTER TABLE prompt_audit_events
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_events_failure_reason;
ALTER TABLE prompt_audit_events
    ADD CONSTRAINT chk_prompt_audit_events_failure_reason
        CHECK (
            (decision = 'failed' AND error_code <> '' AND error_message <> '') OR
            (decision <> 'failed' AND error_code = '' AND error_message = '')
        ) NOT VALID;
ALTER TABLE prompt_audit_events
    VALIDATE CONSTRAINT chk_prompt_audit_events_failure_reason;

INSERT INTO prompt_audit_events (
    job_id, request_id, user_id, username_snapshot, user_email_snapshot, api_key_id,
    api_key_name_snapshot, group_id, group_name, provider, endpoint, protocol, model,
    prompt_hash, redacted_preview, stage, decision, risk_level, action, categories,
    matched_scanners, scanner_scores, scanner_evidence, scanner_backend, scanner_version,
    guard_endpoint_id, guard_endpoint_name, guard_model, policy_id, policy_version,
    config_version, chunk_total, latency_ms, full_prompt, client_ip, prompt_length,
    message_count, execution_mode, queue_delay_ms, input_limit, matched_chunk_index,
    full_prompt_truncated, error_code, error_message, created_at
)
SELECT
    j.id, j.request_id, j.user_id, j.username_snapshot, j.user_email_snapshot, j.api_key_id,
    j.api_key_name_snapshot, j.group_id, j.group_name, j.provider, j.endpoint, j.protocol, j.model,
    j.prompt_hash, j.redacted_preview, j.stage, 'failed', 'unknown', 'Error', '[]'::jsonb,
    '[]'::jsonb, '{}'::jsonb, '{}'::jsonb, 'qwen3guard-openai', '',
    '', '', '', 'priority', 0,
    j.config_version, 0, 0, '', j.client_ip, j.prompt_length,
    j.message_count, j.execution_mode, NULL, NULL, NULL,
    TRUE,
    CASE
        WHEN LOWER(j.last_error_code) ~ '^[a-z0-9_.-]{1,64}$' THEN LOWER(j.last_error_code)
        ELSE 'prompt_guard_unavailable'
    END,
    CASE j.last_error_code
        WHEN 'prompt_guard_blocked' THEN 'Prompt Guard blocked the request'
        WHEN 'prompt_guard_unavailable' THEN 'Prompt Audit dependency is unavailable'
        WHEN 'payload_store_unavailable' THEN 'Prompt Audit dependency is unavailable'
        WHEN 'payload_missing' THEN 'Prompt Audit dependency is unavailable'
        WHEN 'prompt_guard_invalid_response' THEN 'Prompt Guard returned an invalid response'
        WHEN 'queue_full' THEN 'Prompt Audit queue is unavailable'
        WHEN 'queue_admission_busy' THEN 'Prompt Audit queue is unavailable'
        WHEN 'worker_panic' THEN 'Prompt Audit worker failed'
        WHEN 'config_load_failed' THEN 'Prompt Audit configuration could not be loaded'
        WHEN 'config_ttl_reload_failed' THEN 'Prompt Audit configuration could not be loaded'
        WHEN 'config_invalidation_reload_failed' THEN 'Prompt Audit configuration could not be loaded'
        ELSE 'Prompt Audit operation failed'
    END,
    COALESCE(j.processed_at, j.updated_at, j.created_at)
FROM prompt_audit_jobs AS j
WHERE j.status = 'failed'
  AND NOT EXISTS (SELECT 1 FROM prompt_audit_events AS e WHERE e.job_id = j.id);
