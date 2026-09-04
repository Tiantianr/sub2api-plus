-- Extend the existing OpenAI OAuth user-access policy to OpenAI API-key roots.
-- The table and endpoint names remain compatible with the original rollout.

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
          AND a.type IN ('oauth', 'apikey')
          AND a.parent_account_id IS NULL
          AND a.deleted_at IS NULL
        ORDER BY p.account_id
    LOOP
        PERFORM a.id
        FROM accounts AS a
        WHERE a.id = target_account_id
          AND a.platform = 'openai'
          AND a.type IN ('oauth', 'apikey')
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

CREATE OR REPLACE FUNCTION clear_invalid_openai_oauth_account_access_policy()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.platform <> 'openai'
       OR NEW.type NOT IN ('oauth', 'apikey')
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
