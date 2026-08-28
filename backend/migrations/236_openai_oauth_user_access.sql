CREATE TABLE IF NOT EXISTS openai_oauth_account_access_policies (
    id                     BIGSERIAL PRIMARY KEY,
    account_id             BIGINT NOT NULL UNIQUE REFERENCES accounts(id) ON DELETE CASCADE,
    mode                   VARCHAR(16) NOT NULL DEFAULT 'public'
                           CHECK (mode IN ('public', 'restricted')),
    default_for_new_users  BOOLEAN NOT NULL DEFAULT FALSE,
    revision               BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (mode = 'restricted' OR default_for_new_users = FALSE)
);

CREATE TABLE IF NOT EXISTS openai_oauth_account_user_grants (
    id          BIGSERIAL PRIMARY KEY,
    account_id  BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (account_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_openai_oauth_account_user_grants_user
    ON openai_oauth_account_user_grants (user_id, account_id);

CREATE INDEX IF NOT EXISTS idx_openai_oauth_access_default_new_users
    ON openai_oauth_account_access_policies (account_id)
    WHERE mode = 'restricted' AND default_for_new_users = TRUE;

CREATE OR REPLACE FUNCTION grant_default_openai_oauth_accounts_to_new_user()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    target_account_id BIGINT;
BEGIN
    IF NEW.role <> 'user' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    FOR target_account_id IN
        SELECT p.account_id
        FROM openai_oauth_account_access_policies AS p
        JOIN accounts AS a ON a.id = p.account_id
        WHERE p.mode = 'restricted'
          AND p.default_for_new_users = TRUE
          AND a.platform = 'openai'
          AND a.type = 'oauth'
          AND a.parent_account_id IS NULL
          AND a.deleted_at IS NULL
        ORDER BY p.account_id
    LOOP
        PERFORM a.id
        FROM accounts AS a
        WHERE a.id = target_account_id
          AND a.platform = 'openai'
          AND a.type = 'oauth'
          AND a.parent_account_id IS NULL
          AND a.deleted_at IS NULL
        FOR UPDATE;
        IF NOT FOUND THEN
            CONTINUE;
        END IF;

        PERFORM p.account_id
        FROM openai_oauth_account_access_policies AS p
        WHERE p.account_id = target_account_id
          AND p.mode = 'restricted'
          AND p.default_for_new_users = TRUE
        FOR UPDATE;
        IF NOT FOUND THEN
            CONTINUE;
        END IF;

        INSERT INTO openai_oauth_account_user_grants (account_id, user_id)
        VALUES (target_account_id, NEW.id)
        ON CONFLICT (account_id, user_id) DO NOTHING;

        IF FOUND THEN
            UPDATE openai_oauth_account_access_policies
            SET revision = revision + 1,
                updated_at = NOW()
            WHERE account_id = target_account_id;

            INSERT INTO scheduler_outbox (event_type, account_id)
            SELECT 'account_changed', affected.id
            FROM accounts AS affected
            WHERE affected.deleted_at IS NULL
              AND (affected.id = target_account_id OR affected.parent_account_id = target_account_id);
        END IF;
    END LOOP;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_users_grant_default_openai_oauth_accounts ON users;
CREATE TRIGGER trg_users_grant_default_openai_oauth_accounts
AFTER INSERT ON users
FOR EACH ROW EXECUTE FUNCTION grant_default_openai_oauth_accounts_to_new_user();

CREATE OR REPLACE FUNCTION clear_invalid_openai_oauth_account_access_policy()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.platform <> 'openai'
       OR NEW.type <> 'oauth'
       OR NEW.parent_account_id IS NOT NULL
       OR NEW.deleted_at IS NOT NULL THEN
        DELETE FROM openai_oauth_account_user_grants
        WHERE account_id = NEW.id;
        DELETE FROM openai_oauth_account_access_policies
        WHERE account_id = NEW.id;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_accounts_clear_invalid_openai_oauth_access_policy ON accounts;
CREATE TRIGGER trg_accounts_clear_invalid_openai_oauth_access_policy
AFTER UPDATE OF platform, type, parent_account_id, deleted_at ON accounts
FOR EACH ROW EXECUTE FUNCTION clear_invalid_openai_oauth_account_access_policy();

COMMENT ON TABLE openai_oauth_account_access_policies IS
    'Public/restricted local-user access policy for OpenAI OAuth root accounts';
COMMENT ON TABLE openai_oauth_account_user_grants IS
    'Explicit local-user grants for restricted OpenAI OAuth root accounts';
