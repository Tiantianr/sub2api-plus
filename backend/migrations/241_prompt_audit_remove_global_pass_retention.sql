-- Pass evidence retention is now selected by user in an independently
-- versioned setting. Removing the legacy field makes older binaries default to
-- false after reload or rollback instead of resuming global Pass persistence.
UPDATE settings
SET value = (value::jsonb - 'store_pass_events')::text,
    updated_at = NOW()
WHERE key = 'prompt_audit_config'
  AND value::jsonb ? 'store_pass_events';
