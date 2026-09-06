//go:build unit

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type historyAccountRepo struct {
	*oauthSessionPolicyAccountRepo
	mu        sync.Mutex
	bindings  map[string]OpenAIConversationBinding
	lookupErr error
	writeErr  error
}

func historyTestKey(userID, groupID int64, kind, key string) string {
	return fmt.Sprintf("%d:%d:%s:%s", userID, groupID, kind, key)
}

func (r *historyAccountRepo) GetOpenAIConversationBinding(_ context.Context, userID, groupID int64, kind, key string) (*OpenAIConversationBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lookupErr != nil {
		return nil, r.lookupErr
	}
	value, found := r.bindings[historyTestKey(userID, groupID, kind, key)]
	if !found {
		return nil, nil
	}
	value.Valid = openAIHistoryBindingMatchesIdentity(&value, r.accounts[value.CredentialOwnerAccountID])
	return &value, nil
}

func (r *historyAccountRepo) SaveOpenAIConversationBinding(_ context.Context, binding *OpenAIConversationBinding, expectedRevision int64) (*OpenAIConversationBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.writeErr != nil {
		return nil, r.writeErr
	}
	key := historyTestKey(binding.UserID, binding.ScopeGroupID, binding.Type, binding.Key)
	old, exists := r.bindings[key]
	if exists && !(binding.Type == "session" && old.Revision == expectedRevision) &&
		!(old.AccountID == binding.AccountID && old.CredentialOwnerAccountID == binding.CredentialOwnerAccountID &&
			old.OAuthAccountID == binding.OAuthAccountID && old.OAuthUserID == binding.OAuthUserID && old.PolicyScopeVersion == binding.PolicyScopeVersion) {
		return nil, ErrOpenAIHistoryConflict
	}
	saved := *binding
	saved.Revision, saved.Valid = old.Revision+1, true
	r.bindings[key] = saved
	return &saved, nil
}

func newHistoryTestService(t *testing.T, advanced bool) (*OpenAIGatewayService, *historyAccountRepo) {
	t.Helper()
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	openAIAdvancedSchedulerSettingCache.Store(&cachedOpenAIAdvancedSchedulerSetting{
		enabled: advanced, expiresAt: time.Now().Add(time.Hour).UnixNano(),
	})
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
	repo := &historyAccountRepo{
		oauthSessionPolicyAccountRepo: &oauthSessionPolicyAccountRepo{accounts: map[int64]*Account{}},
		bindings:                      map[string]OpenAIConversationBinding{},
	}
	for _, id := range []int64{1, 2} {
		repo.accounts[id] = &Account{ID: id, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: int(id), GroupIDs: []int64{7},
			Credentials: map[string]any{"chatgpt_account_id": fmt.Sprintf("oauth-%d", id)}, Extra: map[string]any{},
		}
	}
	svc := &OpenAIGatewayService{accountRepo: repo, cache: &oauthSessionPolicyCache{}, cfg: &config.Config{RunMode: config.RunModeSimple}}
	return svc, repo
}

func historyTestRequest(t *testing.T, svc *OpenAIGatewayService, userID int64, protocol, session, body string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(body))
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.UserID, userID))
	c.Set("api_key", &APIKey{ID: userID + 100, UserID: userID, GroupID: ptrInt64(7), User: &User{ID: userID}})
	if session != "" {
		c.Request.Header.Set("session_id", session)
	}
	hash := svc.GenerateSessionHash(c, []byte(body))
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, protocol, []byte(body), hash))
	return c
}

func selectHistoryTestAccount(svc *OpenAIGatewayService, c *gin.Context, excluded map[int64]struct{}) (*AccountSelectionResult, error) {
	state := openAIHistoryFromContext(c.Request.Context())
	selection, _, err := svc.SelectAccountWithSchedulerForCapability(c.Request.Context(), ptrInt64(7), state.previousID,
		state.sessionHash, "gpt-5.1", excluded, OpenAIUpstreamTransportAny, OpenAIEndpointCapabilityResponses, false, false, false)
	if selection != nil && selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
	return selection, err
}

const historyFreshBody = `{"model":"gpt-5.1","input":"hello"}`
const historyReplayBody = `{"model":"gpt-5.1","input":[{"role":"assistant","content":"previous reply"},{"role":"user","content":"continue"}]}`

func TestOpenAIHistoryAdmissionCandidateMatrix(t *testing.T) {
	codexFirstTurn, err := os.ReadFile("../auditcontent/testdata/codex_first_turn.json")
	require.NoError(t, err)
	for _, advanced := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			body   string
			bind   bool
			loose  bool
			want   int64
			denied bool
		}{
			{"new_session", historyFreshBody, false, false, 1, false},
			{"codex_first_turn", string(codexFirstTurn), false, false, 1, false},
			{"own_history", historyReplayBody, true, false, 1, false},
			{"external_history", historyReplayBody, false, false, 0, true},
			{"mixed_pool", historyReplayBody, false, true, 2, false},
		} {
			t.Run(fmt.Sprintf("advanced_%t/%s", advanced, tc.name), func(t *testing.T) {
				svc, repo := newHistoryTestService(t, advanced)
				if tc.loose {
					repo.accounts[2].Extra[OpenAIOAuthRejectExternalHistoryKey] = false
				}
				if tc.bind {
					first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "session-a", historyFreshBody)
					_, err := selectHistoryTestAccount(svc, first, nil)
					require.NoError(t, err)
				}
				c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "session-a", tc.body)
				selection, err := selectHistoryTestAccount(svc, c, nil)
				if tc.denied {
					require.ErrorIs(t, err, ErrOpenAIExternalHistory)
					require.Nil(t, selection)
					require.Empty(t, repo.bindings)
				} else {
					require.NoError(t, err)
					require.NotNil(t, selection)
					require.Equal(t, tc.want, selection.Account.ID)
				}
			})
		}
	}
}

func TestOpenAIHistoryCodexContextDoesNotBypassContinuationChecks(t *testing.T) {
	firstTurn, err := os.ReadFile("../auditcontent/testdata/codex_first_turn.json")
	require.NoError(t, err)
	for _, advanced := range []bool{false, true} {
		for _, tc := range []struct {
			name, extraItem, previousID string
			bound                       bool
		}{
			{name: "existing_owner", bound: true},
			{name: "unknown_response", previousID: "resp_unknown"},
			{name: "assistant", extraItem: `{"type":"message","role":"assistant","content":[]}`},
			{name: "tool_output", extraItem: `{"type":"function_call_output","call_id":"call_1","output":""}`},
			{name: "incomplete_sibling", extraItem: `{"role":"user","content":[{"type":"future_content","text":"unknown content"}]}`},
			{name: "incomplete_tool_declaration", extraItem: `{"type":"additional_tools","role":"developer"}`},
		} {
			t.Run(fmt.Sprintf("advanced_%t/%s", advanced, tc.name), func(t *testing.T) {
				svc, repo := newHistoryTestService(t, advanced)
				var excluded map[int64]struct{}
				if tc.bound {
					first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "codex-session", string(firstTurn))
					selected, err := selectHistoryTestAccount(svc, first, nil)
					require.NoError(t, err)
					require.Equal(t, int64(1), selected.Account.ID)
					excluded = map[int64]struct{}{1: {}}
				}
				var payload map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(firstTurn, &payload))
				if tc.extraItem != "" {
					var input []json.RawMessage
					require.NoError(t, json.Unmarshal(payload["input"], &input))
					input = append(input, json.RawMessage(tc.extraItem))
					payload["input"], err = json.Marshal(input)
					require.NoError(t, err)
				}
				if tc.previousID != "" {
					payload["previous_response_id"], err = json.Marshal(tc.previousID)
					require.NoError(t, err)
				}
				body, err := json.Marshal(payload)
				require.NoError(t, err)
				bindingCount := len(repo.bindings)
				request := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "codex-session", string(body))
				selected, err := selectHistoryTestAccount(svc, request, excluded)
				require.ErrorIs(t, err, ErrOpenAIExternalHistory)
				require.Nil(t, selected)
				require.Len(t, repo.bindings, bindingCount)
			})
		}
	}
}

func TestOpenAIHistorySurvivesCacheLossAndServiceRestart(t *testing.T) {
	svc, repo := newHistoryTestService(t, true)
	first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "long-lived", historyFreshBody)
	selection, err := selectHistoryTestAccount(svc, first, nil)
	require.NoError(t, err)
	require.NoError(t, svc.persistOpenAIHistoryResponse(first.Request.Context(), selection.Account, "resp_durable"))

	restarted := &OpenAIGatewayService{accountRepo: repo, cache: &oauthSessionPolicyCache{}, cfg: svc.cfg}
	repo.accounts[2].Priority = 0
	repo.accounts[2].Extra[OpenAIOAuthRejectExternalHistoryKey] = false
	for _, body := range []string{historyReplayBody, `{"model":"gpt-5.1","previous_response_id":"resp_durable","input":"next"}`} {
		c := historyTestRequest(t, restarted, 11, ContentModerationProtocolOpenAIResponses, "long-lived", body)
		selected, err := selectHistoryTestAccount(restarted, c, nil)
		require.NoError(t, err)
		require.Equal(t, int64(1), selected.Account.ID)
	}
	owned, err := restarted.ValidateOpenAIHTTPResponseOwner(context.Background(), 7, "resp_durable", 11, 999)
	require.NoError(t, err)
	require.True(t, owned, "same user's different API key remains interoperable")
	owned, err = restarted.ValidateOpenAIHTTPResponseOwner(context.Background(), 7, "resp_durable", 12, 112)
	require.NoError(t, err)
	require.False(t, owned)
}

func TestOpenAIHistoryDoesNotTrustNewStickyWritesOrUnknownResponse(t *testing.T) {
	svc, _ := newHistoryTestService(t, false)
	first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "same-session", historyFreshBody)
	_, err := selectHistoryTestAccount(svc, first, nil)
	require.NoError(t, err)

	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "same-session", historyReplayBody)
	selected, err := selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), selected.Account.ID)
	require.NoError(t, svc.setStickySessionAccountID(c.Request.Context(), ptrInt64(7), openAIHistoryFromContext(c.Request.Context()).sessionHash, 2, time.Hour))
	selected, err = selectHistoryTestAccount(svc, c, map[int64]struct{}{1: {}})
	require.ErrorIs(t, err, ErrOpenAIExternalHistory)
	require.Nil(t, selected)

	unknown := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "same-session", `{"model":"gpt-5.1","previous_response_id":"resp_unknown","input":"next"}`)
	_, err = selectHistoryTestAccount(svc, unknown, nil)
	require.ErrorIs(t, err, ErrOpenAIExternalHistory)
}

func TestOpenAIHistoryStorageFailuresAndRoutingConflicts(t *testing.T) {
	t.Run("lookup_error", func(t *testing.T) {
		svc, repo := newHistoryTestService(t, false)
		repo.lookupErr = errors.New("database offline")
		c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "offline", historyReplayBody)
		_, err := selectHistoryTestAccount(svc, c, nil)
		require.ErrorIs(t, err, ErrOpenAIHistoryUnavailable)
		require.NotErrorIs(t, err, ErrOpenAIExternalHistory)
	})
	t.Run("write_error", func(t *testing.T) {
		svc, repo := newHistoryTestService(t, false)
		repo.writeErr = errors.New("database read only")
		c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "read-only", historyFreshBody)
		_, err := selectHistoryTestAccount(svc, c, nil)
		require.ErrorIs(t, err, ErrOpenAIHistoryUnavailable)
	})
	t.Run("concurrent_new_session", func(t *testing.T) {
		svc, repo := newHistoryTestService(t, false)
		first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "concurrent", historyFreshBody)
		second := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "concurrent", historyFreshBody)
		require.Empty(t, openAIHistoryCandidateFailureReason(first.Request.Context(), repo.accounts[1]))
		require.Empty(t, openAIHistoryCandidateFailureReason(second.Request.Context(), repo.accounts[2]))
		require.NoError(t, svc.CommitOpenAIHistoryRoute(first.Request.Context(), repo.accounts[1]))
		require.ErrorIs(t, svc.CommitOpenAIHistoryRoute(second.Request.Context(), repo.accounts[2]), ErrOpenAIHistoryConflict)
	})
}

func TestOpenAIHistoryIdentityRefreshAndShadowPolicy(t *testing.T) {
	svc, repo := newHistoryTestService(t, false)
	first := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "identity", historyFreshBody)
	_, err := selectHistoryTestAccount(svc, first, nil)
	require.NoError(t, err)
	repo.accounts[1].Credentials["access_token"] = "refreshed-token"
	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "identity", historyReplayBody)
	require.NoError(t, svc.ValidateOpenAIHistoryTurn(c.Request.Context(), repo.accounts[1]))
	repo.accounts[1].Credentials["chatgpt_account_id"] = "different-upstream-owner"
	require.ErrorIs(t, svc.ValidateOpenAIHistoryTurn(c.Request.Context(), repo.accounts[1]), ErrOpenAIExternalHistory)

	shadow := &Account{ID: 3, Platform: PlatformOpenAI, Type: AccountTypeOAuth, ParentAccountID: ptrInt64(1)}
	repo.accounts[3] = shadow
	repo.accounts[1].Extra[OpenAIOAuthRejectExternalHistoryKey] = false
	foreign := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "foreign", historyReplayBody)
	require.Empty(t, openAIHistoryCandidateFailureReason(foreign.Request.Context(), shadow))
	repo.accounts[1].Extra[OpenAIOAuthRejectExternalHistoryKey] = true
	require.ErrorIs(t, svc.ValidateOpenAIHistoryTurn(foreign.Request.Context(), shadow), ErrOpenAIExternalHistory)
}

func TestOpenAIHistoryOptInAfterLooseAccountAndUserIsolation(t *testing.T) {
	svc, repo := newHistoryTestService(t, false)
	repo.accounts[2].Extra[OpenAIOAuthRejectExternalHistoryKey] = false
	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "imported", historyReplayBody)
	selected, err := selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), selected.Account.ID)
	repo.accounts[2].Extra[OpenAIOAuthRejectExternalHistoryKey] = true
	c = historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "imported", historyReplayBody)
	selected, err = selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.Equal(t, int64(2), selected.Account.ID)
	foreignUser := historyTestRequest(t, svc, 12, ContentModerationProtocolOpenAIResponses, "imported", historyReplayBody)
	_, err = selectHistoryTestAccount(svc, foreignUser, nil)
	require.ErrorIs(t, err, ErrOpenAIExternalHistory)
}

func TestOpenAIHistoryConfiguration(t *testing.T) {
	for _, raw := range []any{nil, true, false, "false", 0} {
		account := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{OpenAIOAuthRejectExternalHistoryKey: raw}}
		require.Equal(t, raw != false, account.IsOpenAIOAuthRejectExternalHistoryEnabled())
	}
	current := &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{OpenAIOAuthRejectExternalHistoryKey: false}}
	extra, err := normalizeOpenAIHistoryExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{"unrelated": true}, current)
	require.NoError(t, err)
	require.Equal(t, false, extra[OpenAIOAuthRejectExternalHistoryKey])
	_, err = normalizeOpenAIHistoryExtra(PlatformOpenAI, AccountTypeOAuth, map[string]any{OpenAIOAuthRejectExternalHistoryKey: "false"}, nil)
	require.Error(t, err)
	extra, err = normalizeOpenAIHistoryExtra(PlatformOpenAI, AccountTypeOAuth, nil, nil)
	require.NoError(t, err)
	require.Equal(t, true, extra[OpenAIOAuthRejectExternalHistoryKey])
}

func TestOpenAIHistoryReadOnlyCountDoesNotCreateOwnership(t *testing.T) {
	svc, repo := newHistoryTestService(t, false)
	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "count-only", historyFreshBody)
	hash := svc.GenerateSessionHash(c, []byte(historyFreshBody))
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, ContentModerationProtocolOpenAIResponses, []byte(historyFreshBody), hash, true))
	selected, err := selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.NoError(t, svc.ValidateOpenAIHistoryTurn(c.Request.Context(), selected.Account))
	require.Empty(t, repo.bindings)
	c = historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "count-only", historyReplayBody)
	_, err = selectHistoryTestAccount(svc, c, nil)
	require.ErrorIs(t, err, ErrOpenAIExternalHistory, "counting cannot authorize a subsequent history import")
}

func TestOpenAIHistorySharedScopeSurvivesCacheLoss(t *testing.T) {
	svc, repo := newHistoryTestService(t, false)
	owner := newOpenAIOAuthSessionPolicyAccount(1, 7, 8)
	repo.accounts[1] = &owner
	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "shared-history", historyFreshBody)
	_, err := selectHistoryTestAccount(svc, c, map[int64]struct{}{2: {}})
	require.NoError(t, err)
	svc = &OpenAIGatewayService{accountRepo: repo, cfg: svc.cfg}
	c = historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "shared-history", historyReplayBody)
	key, _ := c.Get("api_key")
	key.(*APIKey).GroupID = ptrInt64(8)
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, ContentModerationProtocolOpenAIResponses, []byte(historyReplayBody), svc.GenerateSessionHash(c, []byte(historyReplayBody))))
	require.Empty(t, openAIHistoryCandidateFailureReason(c.Request.Context(), &owner))
	key.(*APIKey).GroupID = ptrInt64(9)
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, ContentModerationProtocolOpenAIResponses, []byte(historyReplayBody), svc.GenerateSessionHash(c, []byte(historyReplayBody))))
	require.Equal(t, "history_owner_missing", openAIHistoryCandidateFailureReason(c.Request.Context(), &owner))
}

func TestOpenAIHistoryLegacyResponseBackfillHonorsReadOnlyAndUser(t *testing.T) {
	svc, repo := newHistoryTestService(t, false)
	ctx := context.Background()
	require.NoError(t, svc.BindOpenAIHTTPResponseOwner(ctx, 7, "resp_legacy", 11, 111))
	require.NoError(t, svc.getOpenAIWSStateStore().BindResponseAccount(ctx, 7, "resp_legacy", 1, time.Hour))
	body := `{"model":"gpt-5.1","previous_response_id":"resp_legacy","input":"next"}`
	c := historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "legacy", body)
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, ContentModerationProtocolOpenAIResponses, []byte(body), svc.GenerateSessionHash(c, []byte(body)), true))
	selected, err := selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), selected.Account.ID)
	require.Empty(t, repo.bindings)

	foreign := historyTestRequest(t, svc, 12, ContentModerationProtocolOpenAIResponses, "legacy", body)
	_, err = selectHistoryTestAccount(svc, foreign, nil)
	require.ErrorIs(t, err, ErrOpenAIExternalHistory)
	require.Empty(t, repo.bindings)

	c = historyTestRequest(t, svc, 11, ContentModerationProtocolOpenAIResponses, "legacy", body)
	selected, err = selectHistoryTestAccount(svc, c, nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), selected.Account.ID)
	require.Len(t, repo.bindings, 2)
}
