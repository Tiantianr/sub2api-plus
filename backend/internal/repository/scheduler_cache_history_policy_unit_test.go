//go:build unit

package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/config"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSchedulerMetadataPreservesOpenAIHistoryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra map[string]any
		want  bool
	}{
		{"disabled", map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: false}, false},
		{"enabled", map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: true}, true},
		{"default", nil, true},
		{"invalid", map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: "false"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			account := service.Account{ID: 1, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
				Extra: tc.extra, Credentials: map[string]any{"access_token": "must-not-enter-metadata"}}
			before, err := json.Marshal(account)
			require.NoError(t, err)
			_, metadata, err := marshalSchedulerCacheAccount(account)
			require.NoError(t, err)
			restored, err := decodeCachedAccount(metadata)
			require.NoError(t, err)
			require.Equal(t, tc.want, restored.Extra[service.OpenAIOAuthRejectExternalHistoryKey])
			require.Equal(t, tc.want, restored.IsOpenAIOAuthRejectExternalHistoryEnabled())
			require.Empty(t, restored.GetCredential("access_token"))
			after, err := json.Marshal(account)
			require.NoError(t, err)
			require.JSONEq(t, string(before), string(after), "materializing cache defaults must not mutate stored settings")
		})
	}
}

type schedulerHistoryPolicyRepo struct {
	service.AccountRepository
	accounts      []service.Account
	listCalls     int
	bindingWrites int
}

func (r *schedulerHistoryPolicyRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for _, account := range r.accounts {
		if account.ID == id {
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r *schedulerHistoryPolicyRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, _ string) ([]service.Account, error) {
	r.listCalls++
	return append([]service.Account(nil), r.accounts...), nil
}

func (r *schedulerHistoryPolicyRepo) ListOpenAISessionPolicyDiagnosticCandidates(context.Context) ([]service.Account, error) {
	return append([]service.Account(nil), r.accounts...), nil
}

func (r *schedulerHistoryPolicyRepo) GetOpenAIConversationBinding(context.Context, int64, int64, string, string) (*service.OpenAIConversationBinding, error) {
	return nil, nil
}

func (r *schedulerHistoryPolicyRepo) SaveOpenAIConversationBinding(_ context.Context, binding *service.OpenAIConversationBinding, _ int64) (*service.OpenAIConversationBinding, error) {
	r.bindingWrites++
	saved := *binding
	saved.Valid, saved.Revision = true, 1
	return &saved, nil
}

func schedulerHistoryPolicyAccount(id int64, extra map[string]any) service.Account {
	return service.Account{ID: id, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth,
		Status: service.StatusActive, Schedulable: true, Concurrency: 1, Priority: int(id), GroupIDs: []int64{7}, Extra: extra}
}

func TestSchedulerHistoryPolicyRebuildsIncompleteOAuthSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra map[string]any
		want  bool
	}{
		{"disabled", map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: false}, false},
		{"default", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cache := newSchedulerCacheUnit(t)
			account := schedulerHistoryPolicyAccount(1, tc.extra)
			bucket := service.SchedulerBucket{GroupID: 7, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
			token, err := cache.CaptureBucketWriteToken(ctx, bucket)
			require.NoError(t, err)
			require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{account}))

			// Reproduce the released cache shape: full account is correct, but
			// the metadata payload has lost the history-policy field.
			oldMetadata := buildSchedulerMetadataAccount(account)
			delete(oldMetadata.Extra, service.OpenAIOAuthRejectExternalHistoryKey)
			payload, err := json.Marshal(oldMetadata)
			require.NoError(t, err)
			require.NoError(t, cache.rdb.Set(ctx, schedulerAccountMetaKey(strconv.FormatInt(account.ID, 10)), payload, 0).Err())
			_, hit, err := cache.GetSnapshot(ctx, bucket)
			require.NoError(t, err)
			require.False(t, hit, "an incomplete projection cannot override the stored policy")

			cfg := &config.Config{}
			cfg.Gateway.Scheduling.DbFallbackEnabled = true
			repo := &schedulerHistoryPolicyRepo{accounts: []service.Account{account}}
			snapshots := service.NewSchedulerSnapshotService(cache, nil, repo, nil, cfg)
			for i := 0; i < 2; i++ {
				accounts, _, err := snapshots.ListSchedulableAccounts(ctx, &bucket.GroupID, bucket.Platform, false)
				require.NoError(t, err)
				require.Len(t, accounts, 1)
				require.Equal(t, tc.want, accounts[0].IsOpenAIOAuthRejectExternalHistoryEnabled())
			}
			require.Equal(t, 1, repo.listCalls, "only the old snapshot should require a rebuild")
			accounts, hit, err := cache.GetSnapshot(ctx, bucket)
			require.NoError(t, err)
			require.True(t, hit)
			require.Equal(t, tc.want, accounts[0].Extra[service.OpenAIOAuthRejectExternalHistoryKey])
		})
	}
}

func TestSchedulerHistoryPolicyDoesNotInvalidateAPIKeySnapshots(t *testing.T) {
	ctx := context.Background()
	cache := newSchedulerCacheUnit(t)
	bucket := service.SchedulerBucket{GroupID: 7, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
	account := schedulerHistoryPolicyAccount(1, nil)
	account.Type = service.AccountTypeAPIKey
	token, err := cache.CaptureBucketWriteToken(ctx, bucket)
	require.NoError(t, err)
	require.NoError(t, cache.SetSnapshot(ctx, bucket, token, []service.Account{account}))
	accounts, hit, err := cache.GetSnapshot(ctx, bucket)
	require.NoError(t, err)
	require.True(t, hit)
	require.NotContains(t, accounts[0].Extra, service.OpenAIOAuthRejectExternalHistoryKey)
}

func TestSchedulerHistoryPolicyWarmCacheAdmission(t *testing.T) {
	for _, tc := range []struct {
		name            string
		loose           bool
		newConversation bool
		wantID          int64
	}{
		{"mixed_pool", true, false, 2},
		{"strict_pool", false, false, 0},
		{"new_conversation", false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			cache := newSchedulerCacheUnit(t)
			bucket := service.SchedulerBucket{GroupID: 7, Platform: service.PlatformOpenAI, Mode: service.SchedulerModeSingle}
			repo := &schedulerHistoryPolicyRepo{accounts: []service.Account{
				schedulerHistoryPolicyAccount(1, nil),
				schedulerHistoryPolicyAccount(2, map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: !tc.loose}),
			}}
			token, err := cache.CaptureBucketWriteToken(ctx, bucket)
			require.NoError(t, err)
			require.NoError(t, cache.SetSnapshot(ctx, bucket, token, repo.accounts))
			cfg := &config.Config{}
			snapshots := service.NewSchedulerSnapshotService(cache, nil, repo, nil, cfg)
			svc := service.NewOpenAIGatewayService(repo, nil, nil, nil, nil, nil, nil, cfg, snapshots,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			body := `{"model":"gpt-5.1","input":[{"role":"assistant","content":"previous reply"},{"role":"user","content":"continue"}]}`
			if tc.newConversation {
				body = `{"model":"gpt-5.1","input":"hello"}`
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
			c.Request.Header.Set("session_id", "cached-history-policy")
			c.Set("api_key", &service.APIKey{ID: 19, UserID: 11, User: &service.User{ID: 11}, GroupID: &bucket.GroupID})
			sessionHash := svc.GenerateSessionHash(c, []byte(body))
			require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, service.ContentModerationProtocolOpenAIResponses, []byte(body), sessionHash))
			selection, _, err := svc.SelectAccountWithSchedulerForCapability(c.Request.Context(), &bucket.GroupID, "", sessionHash,
				"gpt-5.1", nil, service.OpenAIUpstreamTransportAny, service.OpenAIEndpointCapabilityResponses, false, false, false)
			if tc.wantID == 0 {
				require.ErrorIs(t, err, service.ErrOpenAIExternalHistory)
				require.Nil(t, selection)
				require.Zero(t, repo.bindingWrites)
			} else {
				require.NoError(t, err)
				require.NotNil(t, selection)
				if selection.ReleaseFunc != nil {
					selection.ReleaseFunc()
				}
				require.Equal(t, tc.wantID, selection.Account.ID)
				require.Equal(t, 1, repo.bindingWrites)
			}
			require.Zero(t, repo.listCalls, "this must exercise the warm slim snapshot, not the full DB account list")
		})
	}
}
