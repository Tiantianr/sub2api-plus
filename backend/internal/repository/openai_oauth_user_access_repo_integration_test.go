//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthDefaultGrantTriggerAndShadowHydration(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	rootID := insertOpenAIOAuthAccessTestAccount(t, fmt.Sprintf("oauth-root-%d", suffix), nil)
	shadowID := insertOpenAIOAuthAccessTestAccount(t, fmt.Sprintf("oauth-shadow-%d", suffix), &rootID)
	var groupID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO groups (name) VALUES ($1) RETURNING id
	`, fmt.Sprintf("oauth-shadow-group-%d", suffix)).Scan(&groupID))
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO account_groups (account_id, group_id) VALUES ($1, $2)
	`, shadowID, groupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id IN ($1, $2)", shadowID, rootID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id = $1", groupID)
	})

	_, err = integrationDB.ExecContext(ctx, `
		INSERT INTO openai_oauth_account_access_policies
			(account_id, mode, default_for_new_users, revision)
		VALUES ($1, 'restricted', TRUE, 1)
	`, rootID)
	require.NoError(t, err)

	var outboxStart int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM scheduler_outbox").Scan(&outboxStart))
	userID := insertOpenAIOAuthAccessTestUser(t, fmt.Sprintf("oauth-default-%d@example.com", suffix), "user")
	adminID := insertOpenAIOAuthAccessTestUser(t, fmt.Sprintf("oauth-admin-%d@example.com", suffix), "admin")
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id IN ($1, $2)", userID, adminID)
	})

	var grantCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM openai_oauth_account_user_grants
		WHERE account_id = $1 AND user_id = $2
	`, rootID, userID).Scan(&grantCount))
	require.Equal(t, 1, grantCount)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM openai_oauth_account_user_grants
		WHERE account_id = $1 AND user_id = $2
	`, rootID, adminID).Scan(&grantCount))
	require.Zero(t, grantCount)

	var revision int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT revision FROM openai_oauth_account_access_policies WHERE account_id = $1
	`, rootID).Scan(&revision))
	require.Equal(t, int64(2), revision)

	var outboxCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM scheduler_outbox
		WHERE id > $1 AND event_type = 'account_changed' AND account_id IN ($2, $3)
	`, outboxStart, rootID, shadowID).Scan(&outboxCount))
	require.Equal(t, 2, outboxCount)

	accountRepo := newAccountRepositoryWithSQL(integrationEntClient, integrationDB, nil)
	shadow, err := accountRepo.GetByID(ctx, shadowID)
	require.NoError(t, err)
	require.NotNil(t, shadow.OpenAIOAuthUserAccess)
	require.Equal(t, []int64{userID}, shadow.OpenAIOAuthUserAccess.GrantedUserIDs)

	accessRepo := &openAIOAuthUserAccessRepository{db: integrationDB}
	accounts, err := accessRepo.ListAccounts(ctx)
	require.NoError(t, err)
	var root *service.OpenAIOAuthAccessAccount
	for i := range accounts {
		if accounts[i].ID == rootID {
			root = &accounts[i]
			break
		}
	}
	require.NotNil(t, root)
	require.Contains(t, root.GroupIDs, groupID, "root policy preview must include shadow scheduling groups")

	_, err = integrationDB.ExecContext(ctx, "UPDATE accounts SET deleted_at = NOW() WHERE id = $1", rootID)
	require.NoError(t, err)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM openai_oauth_account_access_policies WHERE account_id = $1
	`, rootID).Scan(&grantCount))
	require.Zero(t, grantCount)
}

func TestOpenAIOAuthUserAccessRepositoryAppliesRevisionCheckedPolicies(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	rootID := insertOpenAIOAuthAccessTestAccount(t, fmt.Sprintf("oauth-apply-%d", suffix), nil)
	userOne := insertOpenAIOAuthAccessTestUser(t, fmt.Sprintf("oauth-one-%d@example.com", suffix), "user")
	userTwo := insertOpenAIOAuthAccessTestUser(t, fmt.Sprintf("oauth-two-%d@example.com", suffix), "user")
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id IN ($1, $2)", userOne, userTwo)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id = $1", rootID)
	})

	repo := &openAIOAuthUserAccessRepository{db: integrationDB}
	err := repo.ApplyPolicies(ctx, []service.OpenAIOAuthAccessPolicyChange{{
		AccountID:          rootID,
		ExpectedRevision:   0,
		Mode:               service.OpenAIOAuthUserAccessModeRestricted,
		DefaultForNewUsers: true,
		GrantedUserIDs:     []int64{userOne},
	}})
	require.NoError(t, err)

	accounts, err := repo.ListAccounts(ctx)
	require.NoError(t, err)
	var applied *service.OpenAIOAuthAccessAccount
	for i := range accounts {
		if accounts[i].ID == rootID {
			applied = &accounts[i]
			break
		}
	}
	require.NotNil(t, applied)
	require.Equal(t, int64(1), applied.Revision)
	require.Equal(t, []int64{userOne}, applied.GrantedUserIDs)

	err = repo.ApplyPolicies(ctx, []service.OpenAIOAuthAccessPolicyChange{{
		AccountID: rootID, ExpectedRevision: 0, Mode: service.OpenAIOAuthUserAccessModeRestricted,
		GrantedUserIDs: []int64{userTwo},
	}})
	require.Error(t, err)
	require.Equal(t, "OPENAI_OAUTH_ACCESS_REVISION_CONFLICT", infraerrors.Reason(err))

	err = repo.ApplyPolicies(ctx, []service.OpenAIOAuthAccessPolicyChange{{
		AccountID: rootID, ExpectedRevision: 1, Mode: service.OpenAIOAuthUserAccessModePublic,
	}})
	require.NoError(t, err)
	var mode string
	var revision int64
	var defaultForNewUsers bool
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT mode, default_for_new_users, revision
		FROM openai_oauth_account_access_policies WHERE account_id = $1
	`, rootID).Scan(&mode, &defaultForNewUsers, &revision))
	require.Equal(t, service.OpenAIOAuthUserAccessModePublic, mode)
	require.False(t, defaultForNewUsers)
	require.Equal(t, int64(2), revision)
	var grants int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM openai_oauth_account_user_grants WHERE account_id = $1
	`, rootID).Scan(&grants))
	require.Zero(t, grants)
}

func TestOpenAIOAuthUserAccessRepositoryListsUsersByIDDescending(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	search := fmt.Sprintf("oauth-order-%d", suffix)
	olderID := insertOpenAIOAuthAccessTestUser(t, "z-"+search+"@example.com", "user")
	newerID := insertOpenAIOAuthAccessTestUser(t, "a-"+search+"@example.com", "user")
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE id IN ($1, $2)", olderID, newerID)
	})

	repo := &openAIOAuthUserAccessRepository{db: integrationDB}
	users, err := repo.ListUsers(ctx, search, "")
	require.NoError(t, err)
	require.Len(t, users, 2)
	require.Greater(t, newerID, olderID)
	require.Equal(t, []int64{newerID, olderID}, []int64{users[0].ID, users[1].ID})
}

func TestOpenAIOAuthUserAccessRepositoryExcludesGhostGroupBindings(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	rootID := insertOpenAIOAuthAccessTestAccount(t, fmt.Sprintf("oauth-ghost-%d", suffix), nil)
	var boundGroupID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO groups (name) VALUES ($1) RETURNING id`, fmt.Sprintf("oauth-ghost-group-%d", suffix)).Scan(&boundGroupID))
	_, err := integrationDB.ExecContext(ctx, `INSERT INTO account_groups (account_id, group_id) VALUES ($1,$2)`, rootID, boundGroupID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id=$1", rootID)
		_, _ = integrationDB.ExecContext(ctx, "DELETE FROM groups WHERE id=$1", boundGroupID)
	})

	setAllowedGroup := func(groupID int64) {
		_, updateErr := integrationDB.ExecContext(ctx, `
			UPDATE accounts SET extra=jsonb_build_object(
				'openai_oauth_session_policy',jsonb_build_object(
					'enabled',true,'allowed_group_ids',jsonb_build_array($2::bigint),'scope_version','scope-a'
				)
			) WHERE id=$1`, rootID, groupID)
		require.NoError(t, updateErr)
	}
	findAccount := func(accounts []service.OpenAIOAuthAccessAccount) *service.OpenAIOAuthAccessAccount {
		for i := range accounts {
			if accounts[i].ID == rootID {
				return &accounts[i]
			}
		}
		return nil
	}

	repo := &openAIOAuthUserAccessRepository{db: integrationDB}
	setAllowedGroup(boundGroupID + 1000000)
	accounts, err := repo.ListAccounts(ctx)
	require.NoError(t, err)
	require.NotNil(t, findAccount(accounts))
	require.NotContains(t, findAccount(accounts).GroupIDs, boundGroupID)

	setAllowedGroup(boundGroupID)
	accounts, err = repo.ListAccounts(ctx)
	require.NoError(t, err)
	require.Contains(t, findAccount(accounts).GroupIDs, boundGroupID)
}

func TestOpenAIOAuthDefaultGrantTriggerLocksAccountBeforePolicy(t *testing.T) {
	ctx := context.Background()
	suffix := time.Now().UnixNano()
	rootID := insertOpenAIOAuthAccessTestAccount(t, fmt.Sprintf("oauth-lock-order-%d", suffix), nil)
	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO openai_oauth_account_access_policies
			(account_id, mode, default_for_new_users, revision)
		VALUES ($1, 'restricted', TRUE, 1)
	`, rootID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, "DELETE FROM accounts WHERE id = $1", rootID) })

	adminTx, err := integrationDB.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)
	defer func() { _ = adminTx.Rollback() }()
	_, err = adminTx.ExecContext(ctx, "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", rootID)
	require.NoError(t, err)

	userEmail := fmt.Sprintf("oauth-lock-order-%d@example.com", suffix)
	insertDone := make(chan error, 1)
	go func() {
		_, insertErr := integrationDB.ExecContext(context.Background(), `
			INSERT INTO users (email, password_hash, role)
			VALUES ($1, 'test-password-hash', 'user')
		`, userEmail)
		insertDone <- insertErr
	}()

	select {
	case insertErr := <-insertDone:
		require.NoError(t, insertErr)
		t.Fatal("user insert should wait for the account lock")
	case <-time.After(150 * time.Millisecond):
	}

	_, err = adminTx.ExecContext(ctx, "SET LOCAL lock_timeout = '500ms'")
	require.NoError(t, err)
	_, err = adminTx.ExecContext(ctx, `
		SELECT account_id
		FROM openai_oauth_account_access_policies
		WHERE account_id = $1
		FOR UPDATE
	`, rootID)
	require.NoError(t, err, "trigger must not hold policy while waiting for account")
	require.NoError(t, adminTx.Commit())

	select {
	case insertErr := <-insertDone:
		require.NoError(t, insertErr)
	case <-time.After(3 * time.Second):
		t.Fatal("user insert did not complete after releasing account lock")
	}
	t.Cleanup(func() { _, _ = integrationDB.ExecContext(ctx, "DELETE FROM users WHERE email = $1", userEmail) })
}

func insertOpenAIOAuthAccessTestAccount(t *testing.T, name string, parentID *int64) int64 {
	t.Helper()
	quotaDimension := "global"
	if parentID != nil {
		quotaDimension = "spark"
	}
	query := `
		INSERT INTO accounts (name, platform, type, credentials, extra, parent_account_id, quota_dimension)
		VALUES ($1, 'openai', 'oauth', '{}'::jsonb, '{}'::jsonb, $2, $3)
		RETURNING id
	`
	var id int64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), query, name, parentID, quotaDimension).Scan(&id))
	return id
}

func insertOpenAIOAuthAccessTestUser(t *testing.T, email, role string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, integrationDB.QueryRowContext(context.Background(), `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, 'test-password-hash', $2)
		RETURNING id
	`, email, role).Scan(&id))
	return id
}
