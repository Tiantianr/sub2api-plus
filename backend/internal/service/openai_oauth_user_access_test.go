package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	infraerrors "github.com/LuckyKuang/sub2api-plus/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestOpenAIOAuthUserAccessIsFailClosedAcrossSchedulers(t *testing.T) {
	account := &Account{
		ID:          11,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		OpenAIOAuthUserAccess: &OpenAIOAuthUserAccessSnapshot{
			Mode:           OpenAIOAuthUserAccessModeRestricted,
			Revision:       3,
			GrantedUserIDs: []int64{7, 9},
		},
	}
	allowedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(7))
	deniedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(8))

	require.Empty(t, openAIOAuthUserAccessFailureReason(allowedCtx, account))
	require.Equal(t, openAIOAuthUserAccessDeniedReason, openAIOAuthUserAccessFailureReason(deniedCtx, account))
	require.Equal(t, openAIOAuthUserAccessDeniedReason, openAIOAuthUserAccessFailureReason(context.Background(), account))
	require.Empty(t, openAIOAuthUserAccessFailureReason(context.Background(), &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}))

	require.Empty(t, openAICompatibleAccountEligibilityFailureReasonBeforeProfit(
		allowedCtx, account, PlatformOpenAI, "", false, "",
	))
	require.Equal(t, openAIOAuthUserAccessDeniedReason, openAICompatibleAccountEligibilityFailureReasonBeforeProfit(
		deniedCtx, account, PlatformOpenAI, "", false, "",
	))

	scheduler := &defaultOpenAIAccountScheduler{}
	compatible, reason := scheduler.isAccountRequestCompatibleReason(allowedCtx, account, OpenAIAccountScheduleRequest{})
	require.True(t, compatible)
	require.Empty(t, reason)
	compatible, reason = scheduler.isAccountRequestCompatibleReason(deniedCtx, account, OpenAIAccountScheduleRequest{})
	require.False(t, compatible)
	require.Equal(t, openAIOAuthUserAccessDeniedReason, reason)
}

func TestOpenAIAPIKeyUserAccessUsesTheSameRestrictedGate(t *testing.T) {
	account := &Account{
		ID:       12,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		OpenAIOAuthUserAccess: &OpenAIOAuthUserAccessSnapshot{
			Mode:           OpenAIOAuthUserAccessModeRestricted,
			GrantedUserIDs: []int64{7},
		},
	}
	allowedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(7))
	deniedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(8))

	require.Empty(t, openAIOAuthUserAccessFailureReason(allowedCtx, account))
	require.Equal(t, openAIOAuthUserAccessDeniedReason, openAIOAuthUserAccessFailureReason(deniedCtx, account))
}

func TestOpenAIOAuthUserAccessSnapshotCompatibility(t *testing.T) {
	var missing *OpenAIOAuthUserAccessSnapshot
	require.True(t, missing.AllowsUser(0), "missing policy remains public after migration")
	require.True(t, (&OpenAIOAuthUserAccessSnapshot{Mode: OpenAIOAuthUserAccessModePublic}).AllowsUser(0))
	require.False(t, (&OpenAIOAuthUserAccessSnapshot{Mode: OpenAIOAuthUserAccessModeRestricted}).AllowsUser(1))
	require.False(t, (&OpenAIOAuthUserAccessSnapshot{Mode: "unexpected", GrantedUserIDs: []int64{1}}).AllowsUser(1))

	original := &OpenAIOAuthUserAccessSnapshot{
		Mode:           OpenAIOAuthUserAccessModeRestricted,
		GrantedUserIDs: []int64{2, 4},
	}
	clone := original.Clone()
	clone.GrantedUserIDs[0] = 99
	require.Equal(t, []int64{2, 4}, original.GrantedUserIDs)
}

type openAIOAuthAccessRepoFake struct {
	accounts   []OpenAIOAuthAccessAccount
	users      []OpenAIOAuthAccessUser
	applied    []OpenAIOAuthAccessPolicyChange
	refreshErr error
}

func (r *openAIOAuthAccessRepoFake) ListAccounts(context.Context) ([]OpenAIOAuthAccessAccount, error) {
	if len(r.applied) > 0 && r.refreshErr != nil {
		return nil, r.refreshErr
	}
	return append([]OpenAIOAuthAccessAccount(nil), r.accounts...), nil
}

func (r *openAIOAuthAccessRepoFake) ListUsers(context.Context, string, string) ([]OpenAIOAuthAccessUser, error) {
	return append([]OpenAIOAuthAccessUser(nil), r.users...), nil
}

func (r *openAIOAuthAccessRepoFake) ApplyPolicies(_ context.Context, changes []OpenAIOAuthAccessPolicyChange) error {
	r.applied = append([]OpenAIOAuthAccessPolicyChange(nil), changes...)
	for _, change := range changes {
		for i := range r.accounts {
			if r.accounts[i].ID != change.AccountID {
				continue
			}
			r.accounts[i].Mode = change.Mode
			r.accounts[i].DefaultForNewUsers = change.DefaultForNewUsers
			r.accounts[i].GrantedUserIDs = append([]int64(nil), change.GrantedUserIDs...)
			r.accounts[i].Revision++
		}
	}
	return nil
}

func TestOpenAIOAuthUserAccessPreviewFindsUsersLosingAllCandidates(t *testing.T) {
	repo := &openAIOAuthAccessRepoFake{
		accounts: []OpenAIOAuthAccessAccount{{
			ID: 1, Name: "OAuth A", GroupIDs: []int64{10}, Mode: OpenAIOAuthUserAccessModePublic,
		}},
		users: []OpenAIOAuthAccessUser{
			{ID: 101, Email: "allowed@example.com", APIKeyGroupIDs: []int64{10}},
			{ID: 102, Email: "blocked@example.com", APIKeyGroupIDs: []int64{10}},
		},
	}
	svc := NewOpenAIOAuthUserAccessService(repo)
	batch := OpenAIOAuthAccessPolicyBatch{Changes: []OpenAIOAuthAccessPolicyChange{{
		AccountID: 1, ExpectedRevision: 0, Mode: OpenAIOAuthUserAccessModeRestricted,
		DefaultForNewUsers: true, GrantedUserIDs: []int64{101},
	}}}

	preview, err := svc.Preview(context.Background(), batch)
	require.NoError(t, err)
	require.Equal(t, 1, preview.GrantAddedCount)
	require.Equal(t, 1, preview.UsersLosingAllAccessCount)
	require.Equal(t, int64(102), preview.UsersLosingAllAccess[0].ID)

	result, err := svc.Apply(context.Background(), batch)
	require.NoError(t, err)
	require.Len(t, repo.applied, 1)
	require.Equal(t, int64(1), result.Accounts[0].Revision)
}

func TestOpenAIOAuthUserAccessPreviewRejectsStaleRevision(t *testing.T) {
	repo := &openAIOAuthAccessRepoFake{accounts: []OpenAIOAuthAccessAccount{{
		ID: 5, Name: "OAuth", Mode: OpenAIOAuthUserAccessModeRestricted, Revision: 8,
	}}}
	svc := NewOpenAIOAuthUserAccessService(repo)
	_, err := svc.Preview(context.Background(), OpenAIOAuthAccessPolicyBatch{Changes: []OpenAIOAuthAccessPolicyChange{{
		AccountID: 5, ExpectedRevision: 7, Mode: OpenAIOAuthUserAccessModeRestricted,
	}}})
	require.Error(t, err)
	require.Equal(t, "OPENAI_OAUTH_ACCESS_REVISION_CONFLICT", infraerrors.Reason(err))
}

func TestOpenAIOAuthUserAccessEmptyCollectionsEncodeAsArrays(t *testing.T) {
	repo := &openAIOAuthAccessRepoFake{
		accounts: []OpenAIOAuthAccessAccount{{ID: 7, Name: "New OAuth", Mode: OpenAIOAuthUserAccessModePublic}},
	}
	svc := NewOpenAIOAuthUserAccessService(repo)
	accounts, err := svc.ListAccounts(context.Background())
	require.NoError(t, err)
	require.NotNil(t, accounts[0].GroupIDs)
	require.NotNil(t, accounts[0].GrantedUserIDs)

	preview, err := svc.Preview(context.Background(), OpenAIOAuthAccessPolicyBatch{Changes: []OpenAIOAuthAccessPolicyChange{{
		AccountID: 7, ExpectedRevision: 0, Mode: OpenAIOAuthUserAccessModeRestricted,
	}}})
	require.NoError(t, err)
	require.NotNil(t, preview.Accounts)
	require.NotNil(t, preview.UsersLosingAllAccess)
	encoded, err := json.Marshal(preview)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"users_losing_all_access":[]`)

	page, err := svc.ListUsers(context.Background(), OpenAIOAuthAccessUserFilter{})
	require.NoError(t, err)
	require.NotNil(t, page.Items)
}

func TestOpenAIOAuthUserAccessPublicModeRejectsHiddenGrants(t *testing.T) {
	_, err := normalizeOpenAIOAuthAccessChanges([]OpenAIOAuthAccessPolicyChange{{
		AccountID: 1, Mode: OpenAIOAuthUserAccessModePublic, GrantedUserIDs: []int64{9},
	}})
	require.Error(t, err)
	require.Equal(t, "OPENAI_OAUTH_ACCESS_PUBLIC_WITH_GRANTS", infraerrors.Reason(err))
}

func TestOpenAIOAuthUserAccessApplyDoesNotReportCommittedChangeAsFailed(t *testing.T) {
	repo := &openAIOAuthAccessRepoFake{
		accounts:   []OpenAIOAuthAccessAccount{{ID: 6, Name: "OAuth", Mode: OpenAIOAuthUserAccessModePublic}},
		refreshErr: errors.New("refresh unavailable"),
	}
	result, err := NewOpenAIOAuthUserAccessService(repo).Apply(context.Background(), OpenAIOAuthAccessPolicyBatch{
		Changes: []OpenAIOAuthAccessPolicyChange{{
			AccountID: 6, ExpectedRevision: 0, Mode: OpenAIOAuthUserAccessModeRestricted,
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result.Accounts)
	require.Equal(t, "6", result.AuditAccountIDs)
	require.Equal(t, "0>1", result.AuditRevisions)
	require.Equal(t, "public>restricted", result.AuditModes)
}

type openAIOAuthAccessAccountRepoFake struct {
	AccountRepository
	account *Account
}

func (r *openAIOAuthAccessAccountRepoFake) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

func TestOpenAIOAuthUserAccessRevalidatesWebSocketAndLiveTurns(t *testing.T) {
	account := &Account{
		ID:          20,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		OpenAIOAuthUserAccess: &OpenAIOAuthUserAccessSnapshot{
			Mode:           OpenAIOAuthUserAccessModeRestricted,
			GrantedUserIDs: []int64{7},
		},
	}
	svc := &OpenAIGatewayService{accountRepo: &openAIOAuthAccessAccountRepoFake{account: account}}
	allowedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(7))
	deniedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(8))

	revalidated, err := svc.RevalidateOpenAIAccountForWebSocketTurn(
		allowedCtx, account, nil, PlatformOpenAI, "", "", "",
	)
	require.NoError(t, err)
	require.NotNil(t, revalidated)
	revalidated, err = svc.RevalidateOpenAIAccountForWebSocketTurn(
		deniedCtx, account, nil, PlatformOpenAI, "", "", "",
	)
	require.NoError(t, err)
	require.Nil(t, revalidated)

	record := &LiveCallRecord{AccountID: account.ID, UserID: 7}
	require.NoError(t, svc.RevalidateLiveCallUserAccess(allowedCtx, record))
	require.ErrorIs(t, svc.RevalidateLiveCallUserAccess(deniedCtx, record), ErrOpenAIOAuthUserAccessDenied)

	staleSelection := *account
	staleSelection.OpenAIOAuthUserAccess = &OpenAIOAuthUserAccessSnapshot{
		Mode:           OpenAIOAuthUserAccessModeRestricted,
		GrantedUserIDs: []int64{8},
	}
	latest, vetoed, reason := svc.ProfitControlVetoLatest(deniedCtx, &staleSelection)
	require.True(t, vetoed, "revocation while waiting for a slot must block terminal admission")
	require.Equal(t, openAIOAuthUserAccessDeniedReason, reason)
	require.Same(t, account, latest)
}

func TestOpenAIOAuthEffectiveAccessRequiresBindingAllowlistAndGrant(t *testing.T) {
	groupID := int64(14)
	account := &Account{
		ID: 40, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true,
		Extra: map[string]any{OpenAIOAuthSessionPolicyExtraKey: map[string]any{
			"enabled": true, "allowed_group_ids": []int64{groupID}, "scope_version": "scope-a",
		}},
		OpenAIOAuthUserAccess: &OpenAIOAuthUserAccessSnapshot{
			Mode: OpenAIOAuthUserAccessModeRestricted, GrantedUserIDs: []int64{7},
		},
	}
	allowedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(7))
	svc := &OpenAIGatewayService{accountRepo: &openAIOAuthAccessAccountRepoFake{account: account}}

	revalidated, err := svc.RevalidateOpenAIAccountForWebSocketTurn(allowedCtx, account, &groupID, PlatformOpenAI, "", "", "")
	require.NoError(t, err)
	require.Nil(t, revalidated, "allowlist-only ghost binding must be ineffective")
	require.ErrorIs(t, svc.RevalidateLiveCallUserAccess(allowedCtx, &LiveCallRecord{AccountID: account.ID, UserID: 7, GroupID: groupID}), ErrOpenAIOAuthUserAccessDenied)

	account.GroupIDs = []int64{groupID}
	revalidated, err = svc.RevalidateOpenAIAccountForWebSocketTurn(allowedCtx, account, &groupID, PlatformOpenAI, "", "", "")
	require.NoError(t, err)
	require.NotNil(t, revalidated)
	require.NoError(t, svc.RevalidateLiveCallUserAccess(allowedCtx, &LiveCallRecord{AccountID: account.ID, UserID: 7, GroupID: groupID}))
	require.Equal(t, []int64{groupID}, EffectiveOpenAIAccountGroupIDs(account))

	account.Groups = []*Group{{ID: groupID, Status: "disabled"}}
	revalidated, err = svc.RevalidateOpenAIAccountForWebSocketTurn(allowedCtx, account, &groupID, PlatformOpenAI, "", "", "")
	require.NoError(t, err)
	require.Nil(t, revalidated)
	require.ErrorIs(t, svc.RevalidateLiveCallUserAccess(allowedCtx, &LiveCallRecord{AccountID: account.ID, UserID: 7, GroupID: groupID}), ErrOpenAIOAuthUserAccessDenied)
	account.Groups = nil

	account.OpenAIOAuthUserAccess.GrantedUserIDs = nil
	revalidated, err = svc.RevalidateOpenAIAccountForWebSocketTurn(allowedCtx, account, &groupID, PlatformOpenAI, "", "", "")
	require.NoError(t, err)
	require.Nil(t, revalidated)
}

type groupCopyAccessAccountRepo struct {
	AccountRepository
	accounts []*Account
}

func (r *groupCopyAccessAccountRepo) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return r.accounts, nil
}

func TestGroupCopySkipsOAuthAccountsOutsideDestinationAllowlist(t *testing.T) {
	destinationGroupID := int64(14)
	repo := &groupCopyAccessAccountRepo{accounts: []*Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeOAuth},
		{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{
			OpenAIOAuthSessionPolicyExtraKey: map[string]any{"enabled": true, "allowed_group_ids": []int64{5}, "scope_version": "scope-a"},
		}},
		{ID: 4, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{
			OpenAIOAuthSessionPolicyExtraKey: map[string]any{"enabled": true, "allowed_group_ids": []int64{destinationGroupID}, "scope_version": "scope-b"},
		}},
	}}
	svc := &adminServiceImpl{accountRepo: repo}
	filtered, err := svc.filterGroupCopyAccountIDs(context.Background(), []int64{1, 2, 3, 4}, destinationGroupID)
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 4}, filtered)
}

type openAIOAuthAccessRefreshFailureRepo struct{ AccountRepository }

func (*openAIOAuthAccessRefreshFailureRepo) GetByID(context.Context, int64) (*Account, error) {
	return nil, nil
}

func TestOpenAIOAuthUserAccessTerminalRecheckFailsClosed(t *testing.T) {
	svc := &OpenAIGatewayService{accountRepo: &openAIOAuthAccessRefreshFailureRepo{}}
	for _, accType := range []string{AccountTypeOAuth, AccountTypeAPIKey} {
		account := &Account{ID: 30, Platform: PlatformOpenAI, Type: accType}
		_, vetoed, reason := svc.ProfitControlVetoLatest(context.Background(), account)
		require.True(t, vetoed)
		require.Equal(t, openAIOAuthUserAccessRecheckReason, reason)
	}
}

func TestOpenAIAPIKeyUserAccessRevalidatesWebSocketAndLiveTurns(t *testing.T) {
	account := &Account{
		ID:          21,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		OpenAIOAuthUserAccess: &OpenAIOAuthUserAccessSnapshot{
			Mode:           OpenAIOAuthUserAccessModeRestricted,
			GrantedUserIDs: []int64{7},
		},
	}
	svc := &OpenAIGatewayService{accountRepo: &openAIOAuthAccessAccountRepoFake{account: account}}
	allowedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(7))
	deniedCtx := context.WithValue(context.Background(), ctxkey.UserID, int64(8))

	revalidated, err := svc.RevalidateOpenAIAccountForWebSocketTurn(
		allowedCtx, account, nil, PlatformOpenAI, "", "", "",
	)
	require.NoError(t, err)
	require.NotNil(t, revalidated)
	revalidated, err = svc.RevalidateOpenAIAccountForWebSocketTurn(
		deniedCtx, account, nil, PlatformOpenAI, "", "", "",
	)
	require.NoError(t, err)
	require.Nil(t, revalidated)

	record := &LiveCallRecord{AccountID: account.ID, UserID: 7}
	require.ErrorIs(t, svc.RevalidateLiveCallUserAccess(allowedCtx, record), ErrOpenAIOAuthUserAccessDenied)
	require.ErrorIs(t, svc.RevalidateLiveCallUserAccess(deniedCtx, record), ErrOpenAIOAuthUserAccessDenied)

	staleSelection := *account
	staleSelection.OpenAIOAuthUserAccess = &OpenAIOAuthUserAccessSnapshot{
		Mode:           OpenAIOAuthUserAccessModeRestricted,
		GrantedUserIDs: []int64{8},
	}
	latest, vetoed, reason := svc.ProfitControlVetoLatest(deniedCtx, &staleSelection)
	require.True(t, vetoed, "revocation while waiting for a slot must block terminal admission for apikey accounts")
	require.Equal(t, openAIOAuthUserAccessDeniedReason, reason)
	require.Same(t, account, latest)
}

