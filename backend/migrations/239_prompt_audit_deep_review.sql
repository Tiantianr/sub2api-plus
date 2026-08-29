ALTER TABLE prompt_audit_jobs
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_jobs_execution_mode;

ALTER TABLE prompt_audit_jobs
    ADD CONSTRAINT chk_prompt_audit_jobs_execution_mode
        CHECK (execution_mode IN ('async_audit', 'async_deep', 'blocking'))
        NOT VALID;

ALTER TABLE prompt_audit_jobs
    VALIDATE CONSTRAINT chk_prompt_audit_jobs_execution_mode;

ALTER TABLE prompt_audit_events
    DROP CONSTRAINT IF EXISTS chk_prompt_audit_events_observability;

ALTER TABLE prompt_audit_events
    ADD CONSTRAINT chk_prompt_audit_events_observability
        CHECK (
            prompt_length >= 0 AND message_count >= 0 AND
            execution_mode IN ('async_audit', 'async_deep', 'blocking') AND
            (queue_delay_ms IS NULL OR queue_delay_ms >= 0) AND
            (input_limit IS NULL OR input_limit >= 128) AND
            (matched_chunk_index IS NULL OR (
                matched_chunk_index >= 1 AND matched_chunk_index <= chunk_total
            ))
        ) NOT VALID;

ALTER TABLE prompt_audit_events
    VALIDATE CONSTRAINT chk_prompt_audit_events_observability;
