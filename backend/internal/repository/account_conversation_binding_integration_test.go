//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAIConversationBindingPersistence(t *testing.T) {
	ctx := context.Background()
	user := mustCreateUser(t, integrationEntClient, &service.User{})
	a := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "history-a", Platform: service.PlatformOpenAI,
		Credentials: map[string]any{"chatgpt_account_id": "owner-a", "chatgpt_user_id": "user-a"}})
	b := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "history-b", Platform: service.PlatformOpenAI,
		Credentials: map[string]any{"chatgpt_account_id": "owner-b"}})
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id IN ($1, $2)", a.ID, b.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, err)
	})
	repo := NewAccountRepository(integrationEntClient, integrationDB, nil).(service.OpenAIConversationBindingRepository)
	makeBinding := func(kind, id string, account *service.Account) *service.OpenAIConversationBinding {
		return &service.OpenAIConversationBinding{UserID: user.ID, ScopeGroupID: 7, APIKeyID: 17, Type: kind,
			Key: fmt.Sprintf("%x", sha256.Sum256([]byte(id))), AccountID: account.ID, CredentialOwnerAccountID: account.ID,
			OAuthAccountID: account.GetCredential("chatgpt_account_id"), OAuthUserID: account.GetCredential("chatgpt_user_id")}
	}
	response := makeBinding("response", "resp_persistent", a)
	saved, err := repo.SaveOpenAIConversationBinding(ctx, response, 0)
	require.NoError(t, err)
	require.True(t, saved.Valid)
	require.Equal(t, int64(1), saved.Revision)
	_, err = integrationDB.ExecContext(ctx, "UPDATE openai_conversation_bindings SET created_at = NOW() - INTERVAL '30 days', updated_at = NOW() - INTERVAL '30 days' WHERE user_id = $1", user.ID)
	require.NoError(t, err)
	restarted := NewAccountRepository(integrationEntClient, integrationDB, nil).(service.OpenAIConversationBindingRepository)
	loaded, err := restarted.GetOpenAIConversationBinding(ctx, user.ID, 7, response.Type, response.Key)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.True(t, loaded.Valid)
	require.Equal(t, a.ID, loaded.AccountID)
	missing, err := restarted.GetOpenAIConversationBinding(ctx, user.ID+1, 7, response.Type, response.Key)
	require.NoError(t, err)
	require.Nil(t, missing)
	missing, err = restarted.GetOpenAIConversationBinding(ctx, user.ID, 8, response.Type, response.Key)
	require.NoError(t, err)
	require.Nil(t, missing)
	_, err = repo.SaveOpenAIConversationBinding(ctx, makeBinding("response", "resp_persistent", b), saved.Revision)
	require.ErrorIs(t, err, service.ErrOpenAIHistoryConflict, "responses cannot move to a different account")

	session := makeBinding("session", "session_persistent", a)
	saved, err = repo.SaveOpenAIConversationBinding(ctx, session, 0)
	require.NoError(t, err)
	_, err = repo.SaveOpenAIConversationBinding(ctx, makeBinding("session", "session_persistent", b), 0)
	require.ErrorIs(t, err, service.ErrOpenAIHistoryConflict)
	moved, err := repo.SaveOpenAIConversationBinding(ctx, makeBinding("session", "session_persistent", b), saved.Revision)
	require.NoError(t, err)
	require.Equal(t, b.ID, moved.AccountID)

	_, err = integrationDB.ExecContext(ctx, "UPDATE accounts SET credentials = credentials || '{\"access_token\":\"refreshed\"}'::jsonb, rate_limit_reset_at = NOW() + INTERVAL '1 hour' WHERE id = $1", a.ID)
	require.NoError(t, err)
	loaded, err = repo.GetOpenAIConversationBinding(ctx, user.ID, 7, response.Type, response.Key)
	require.NoError(t, err)
	require.True(t, loaded.Valid, "token refresh and cooldown preserve ownership")
	_, err = integrationDB.ExecContext(ctx, "UPDATE accounts SET credentials = credentials || '{\"chatgpt_account_id\":\"different-owner\"}'::jsonb WHERE id = $1", a.ID)
	require.NoError(t, err)
	loaded, err = repo.GetOpenAIConversationBinding(ctx, user.ID, 7, response.Type, response.Key)
	require.NoError(t, err)
	require.NotNil(t, loaded, "retain an invalid tombstone so cache cannot revive the old identity")
	require.False(t, loaded.Valid)
	_, err = repo.SaveOpenAIConversationBinding(ctx, response, 0)
	require.ErrorIs(t, err, service.ErrOpenAIHistoryConflict, "stale identity cannot create new records")
	_, err = integrationDB.ExecContext(ctx, "UPDATE accounts SET deleted_at = NOW() WHERE id = $1", a.ID)
	require.NoError(t, err)
	loaded, err = repo.GetOpenAIConversationBinding(ctx, user.ID, 7, response.Type, response.Key)
	require.NoError(t, err)
	require.Nil(t, loaded)
	_, err = integrationDB.ExecContext(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = $1", user.ID)
	require.NoError(t, err)
	loaded, err = repo.GetOpenAIConversationBinding(ctx, user.ID, 7, session.Type, session.Key)
	require.NoError(t, err)
	require.Nil(t, loaded)
}

func TestOpenAIConversationBindingConcurrentInsert(t *testing.T) {
	ctx := context.Background()
	user := mustCreateUser(t, integrationEntClient, &service.User{})
	accounts := []*service.Account{
		mustCreateAccount(t, integrationEntClient, &service.Account{Name: "concurrent-a", Platform: service.PlatformOpenAI}),
		mustCreateAccount(t, integrationEntClient, &service.Account{Name: "concurrent-b", Platform: service.PlatformOpenAI}),
	}
	t.Cleanup(func() {
		_, err := integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id IN ($1, $2)", accounts[0].ID, accounts[1].ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id = $1", user.ID)
		require.NoError(t, err)
	})
	repo := NewAccountRepository(integrationEntClient, integrationDB, nil).(service.OpenAIConversationBindingRepository)
	start := make(chan struct{})
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for _, account := range accounts {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			<-start
			_, err := repo.SaveOpenAIConversationBinding(ctx, &service.OpenAIConversationBinding{
				UserID: user.ID, ScopeGroupID: 7, APIKeyID: 17, Type: "session", Key: fmt.Sprintf("%x", sha256.Sum256([]byte("concurrent"))),
				AccountID: id, CredentialOwnerAccountID: id,
			}, 0)
			errors <- err
		}(account.ID)
	}
	close(start)
	wg.Wait()
	close(errors)
	successes := 0
	for err := range errors {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, service.ErrOpenAIHistoryConflict)
		}
	}
	require.Equal(t, 1, successes)
}
