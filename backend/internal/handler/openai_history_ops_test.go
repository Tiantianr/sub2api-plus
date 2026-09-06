//go:build unit

package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func takeHistoryOpsLog(t *testing.T) *service.OpsInsertErrorLogInput {
	t.Helper()
	select {
	case job := <-opsErrorLogQueue:
		return job.entry
	case <-time.After(3 * time.Second):
		t.Fatal("history rejection was not queued for the failed-request list")
		return nil
	}
}

func assertHistoryOpsRejection(t *testing.T, entry *service.OpsInsertErrorLogInput, keyID, groupID int64, requestID string) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, entry.StatusCode)
	require.Equal(t, "routing", entry.ErrorPhase)
	require.Equal(t, "platform", entry.ErrorOwner)
	require.Equal(t, "gateway", entry.ErrorSource)
	require.True(t, entry.IsBusinessLimited)
	require.Equal(t, int64(100), *entry.UserID)
	require.Equal(t, keyID, *entry.APIKeyID)
	require.Equal(t, groupID, *entry.GroupID)
	require.Equal(t, requestID, entry.RequestID)
	require.Equal(t, "gpt-5.1", entry.Model)
	require.Nil(t, entry.AccountID)
	require.Nil(t, entry.UpstreamStatusCode)
	require.Empty(t, entry.UpstreamErrors)
	require.Empty(t, entry.UpstreamErrorMessage)
	require.Empty(t, entry.UpstreamErrorDetail)
	require.Empty(t, entry.UpstreamEndpoint)
	require.Empty(t, entry.UpstreamModel)

	visible := service.ToUserErrorRequest(&service.OpsErrorLog{
		StatusCode: entry.StatusCode, Phase: entry.ErrorPhase, Type: entry.ErrorType,
		Message: entry.ErrorMessage, RequestedModel: entry.RequestedModel,
	})
	require.Equal(t, http.StatusBadRequest, visible.StatusCode)
	require.Equal(t, entry.ErrorMessage, visible.Message)
	require.Equal(t, "service_unavailable", visible.Category)
	require.Empty(t, opsErrorLogQueue, "a rejected request should not be logged twice")
}

func TestOpenAIHistoryHTTPRejectionIsLoggedForUsage(t *testing.T) {
	for _, tc := range []struct {
		path   string
		body   string
		handle func(*OpenAIGatewayHandler, *gin.Context)
	}{
		{"/v1/responses", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"private-old-content"},{"role":"user","content":"continue"}]}`, (*OpenAIGatewayHandler).Responses},
		{"/v1/responses/compact", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"private-old-content"},{"role":"user","content":"continue"}]}`, (*OpenAIGatewayHandler).Responses},
		{"/v1/chat/completions", `{"model":"gpt-5.1","messages":[{"role":"assistant","content":"private-old-content"},{"role":"user","content":"continue"}]}`, (*OpenAIGatewayHandler).ChatCompletions},
		{"/v1/messages", `{"model":"gpt-5.1","max_tokens":32,"messages":[{"role":"assistant","content":"private-old-content"},{"role":"user","content":"continue"}]}`, (*OpenAIGatewayHandler).Messages},
	} {
		t.Run(tc.path, func(t *testing.T) {
			setupOpsErrorLogTestQueue(t, 2)
			upstream := &historyHTTPUpstream{}
			h, repo := newHistoryHTTPHandler(t, false, service.AccountTypeOAuth, upstream)
			seed, _ := historyHTTPContext(t, tc.body)
			key, _ := middleware.GetAPIKeyFromContext(seed)
			key.Group.AllowMessagesDispatch = true
			ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			router := gin.New()
			router.Use(OpsErrorLoggerMiddleware(ops))
			router.POST(tc.path, func(c *gin.Context) {
				for key, value := range seed.Keys {
					c.Set(key, value)
				}
				c.Header("X-Request-Id", "history-http-request")
				tc.handle(h, c)
			})
			request := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			request.Header.Set("session_id", "history-usage")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Empty(t, upstream.calls())
			require.Zero(t, repo.writes)
			entry := takeHistoryOpsLog(t)
			assertHistoryOpsRejection(t, entry, 99, 3131, "history-http-request")
			require.Equal(t, service.ErrOpenAIExternalHistory.Error(), entry.ErrorMessage)
			require.NotContains(t, entry.ErrorBody, "private-old-content")
		})
	}
}

func TestOpenAIHistorySSERejectionRetainsLogicalStatus(t *testing.T) {
	for _, path := range []string{"/v1/responses", "/v1/chat/completions", "/v1/messages"} {
		for _, staleUpstream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stale_%t", path, staleUpstream), func(t *testing.T) {
				setupOpsErrorLogTestQueue(t, 2)
				seed, _ := historyHTTPContext(t, "")
				ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				router := gin.New()
				router.Use(OpsErrorLoggerMiddleware(ops))
				router.POST(path, func(c *gin.Context) {
					for key, value := range seed.Keys {
						c.Set(key, value)
					}
					c.Header("X-Request-Id", "history-stream-request")
					setOpsRequestContext(c, "gpt-5.1", true)
					if staleUpstream {
						c.Set(opsAccountIDKey, int64(45))
						c.Set(service.OpsSkipPassthroughKey, true)
						c.Set(service.OpsUpstreamStatusCodeKey, http.StatusUnauthorized)
						c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{
							Stage: "account_auth", UpstreamStatusCode: http.StatusUnauthorized, SkipMonitoring: true,
						}})
					}
					c.Header("Content-Type", "text/event-stream")
					c.Status(http.StatusOK)
					_, _ = c.Writer.Write([]byte(": ping\n\n"))
					c.Writer.Flush()
					h := new(OpenAIGatewayHandler)
					require.True(t, h.handleOpenAIHistoryError(c, service.ErrOpenAIExternalHistory, path == "/v1/messages", true))
				})
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Contains(t, recorder.Body.String(), service.ErrOpenAIExternalHistory.Error())
				entry := takeHistoryOpsLog(t)
				assertHistoryOpsRejection(t, entry, 99, 3131, "history-stream-request")
				require.Contains(t, entry.ErrorBody, "external_history_not_allowed")
			})
		}
	}
}

func TestOpenAIHistoryOpsClassificationRequiresLocalMarker(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	service.SetOpsUpstreamError(c, http.StatusBadRequest, service.ErrOpenAIExternalHistory.Error(), "")
	phase, limited, owner, source := classifyOpsErrorLog(c, "invalid_request_error", service.ErrOpenAIExternalHistory.Error(), "external_history_not_allowed", http.StatusBadRequest)
	require.Equal(t, "upstream", phase)
	require.False(t, limited)
	require.Equal(t, "provider", owner)
	require.Equal(t, "upstream_http", source)
}
