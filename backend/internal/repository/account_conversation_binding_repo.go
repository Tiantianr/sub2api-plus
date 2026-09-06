package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
)

var _ service.OpenAIConversationBindingRepository = (*accountRepository)(nil)

func (r *accountRepository) GetOpenAIConversationBinding(ctx context.Context, userID, scopeGroupID int64, kind, key string) (*service.OpenAIConversationBinding, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT b.user_id, b.scope_group_id, b.binding_type, b.binding_key,
		       b.account_id, b.credential_owner_account_id, b.api_key_id,
		       b.oauth_account_id, b.oauth_user_id, b.policy_scope_version, b.revision,
		       (a.deleted_at IS NULL AND owner.deleted_at IS NULL AND u.deleted_at IS NULL
		        AND a.platform = 'openai' AND a.type = 'oauth'
		        AND owner.platform = 'openai' AND owner.type = 'oauth'
		        AND COALESCE(a.parent_account_id, a.id) = b.credential_owner_account_id
		        AND COALESCE(owner.credentials->>'chatgpt_account_id', '') = b.oauth_account_id
		        AND COALESCE(owner.credentials->>'chatgpt_user_id', '') = b.oauth_user_id
		        AND CASE WHEN owner.extra->'openai_oauth_session_policy'->>'enabled' = 'true'
		                 THEN COALESCE(owner.extra->'openai_oauth_session_policy'->>'scope_version', '') ELSE '' END = b.policy_scope_version)
		FROM openai_conversation_bindings b
		JOIN accounts a ON a.id = b.account_id
		JOIN accounts owner ON owner.id = b.credential_owner_account_id
		JOIN users u ON u.id = b.user_id
		WHERE b.user_id = $1 AND b.scope_group_id = $2 AND b.binding_type = $3 AND b.binding_key = $4
	`, userID, scopeGroupID, kind, key)
	if err != nil {
		return nil, fmt.Errorf("lookup OpenAI conversation binding: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var binding service.OpenAIConversationBinding
	if err := rows.Scan(&binding.UserID, &binding.ScopeGroupID, &binding.Type, &binding.Key,
		&binding.AccountID, &binding.CredentialOwnerAccountID, &binding.APIKeyID,
		&binding.OAuthAccountID, &binding.OAuthUserID, &binding.PolicyScopeVersion, &binding.Revision, &binding.Valid); err != nil {
		return nil, err
	}
	return &binding, rows.Err()
}

func (r *accountRepository) SaveOpenAIConversationBinding(ctx context.Context, binding *service.OpenAIConversationBinding, expectedRevision int64) (*service.OpenAIConversationBinding, error) {
	if binding == nil || binding.UserID <= 0 || binding.APIKeyID <= 0 || binding.AccountID <= 0 ||
		binding.CredentialOwnerAccountID <= 0 || len(binding.Key) != 64 ||
		(binding.Type != "session" && binding.Type != "response") {
		return nil, errors.New("invalid OpenAI conversation binding")
	}
	rows, err := r.sql.QueryContext(ctx, `
		INSERT INTO openai_conversation_bindings AS existing
		    (user_id, scope_group_id, binding_type, binding_key, account_id,
		     credential_owner_account_id, api_key_id, oauth_account_id, oauth_user_id, policy_scope_version)
		SELECT $1, $2, $3, $4, a.id, owner.id, $7, $8, $9, $10
		FROM accounts a
		JOIN accounts owner ON owner.id = COALESCE(a.parent_account_id, a.id)
		JOIN users u ON u.id = $1 AND u.deleted_at IS NULL
		WHERE a.id = $5 AND owner.id = $6 AND a.deleted_at IS NULL AND owner.deleted_at IS NULL
		  AND a.platform = 'openai' AND a.type = 'oauth' AND owner.platform = 'openai' AND owner.type = 'oauth'
		  AND COALESCE(owner.credentials->>'chatgpt_account_id', '') = $8
		  AND COALESCE(owner.credentials->>'chatgpt_user_id', '') = $9
		  AND CASE WHEN owner.extra->'openai_oauth_session_policy'->>'enabled' = 'true'
		           THEN COALESCE(owner.extra->'openai_oauth_session_policy'->>'scope_version', '') ELSE '' END = $10
		ON CONFLICT (user_id, scope_group_id, binding_type, binding_key) DO UPDATE SET
		    account_id = EXCLUDED.account_id,
		    credential_owner_account_id = EXCLUDED.credential_owner_account_id,
		    api_key_id = EXCLUDED.api_key_id,
		    oauth_account_id = EXCLUDED.oauth_account_id,
		    oauth_user_id = EXCLUDED.oauth_user_id,
		    policy_scope_version = EXCLUDED.policy_scope_version,
		    revision = existing.revision + 1,
		    updated_at = NOW()
		WHERE (existing.binding_type = 'session' AND existing.revision = $11)
		   OR (existing.account_id = EXCLUDED.account_id
		       AND existing.credential_owner_account_id = EXCLUDED.credential_owner_account_id
		       AND existing.oauth_account_id = EXCLUDED.oauth_account_id
		       AND existing.oauth_user_id = EXCLUDED.oauth_user_id
		       AND existing.policy_scope_version = EXCLUDED.policy_scope_version)
		RETURNING revision
	`, binding.UserID, binding.ScopeGroupID, binding.Type, binding.Key, binding.AccountID,
		binding.CredentialOwnerAccountID, binding.APIKeyID, binding.OAuthAccountID,
		binding.OAuthUserID, binding.PolicyScopeVersion, expectedRevision)
	if err != nil {
		return nil, fmt.Errorf("save OpenAI conversation binding: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, service.ErrOpenAIHistoryConflict
	}
	result := *binding
	if err := rows.Scan(&result.Revision); err != nil {
		return nil, err
	}
	result.Valid = true
	return &result, rows.Err()
}
