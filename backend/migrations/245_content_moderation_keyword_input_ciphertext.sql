-- Retain complete secret-redacted keyword-hit input as AES-256-GCM ciphertext.
-- Existing rows remain excerpt-only and cannot be backfilled.
ALTER TABLE content_moderation_logs
    ADD COLUMN IF NOT EXISTS input_ciphertext TEXT NOT NULL DEFAULT '';
