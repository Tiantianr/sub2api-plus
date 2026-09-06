//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	middleware "github.com/LuckyKuang/sub2api-plus/internal/server/middleware"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type historyHTTPUpstream struct {
	service.HTTPUpstream
	mu         sync.Mutex
	accounts   []int64
	bodies     [][]byte
	stream     bool
	failSecond bool
}

func (u *historyHTTPUpstream) Do(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	u.mu.Lock()
	u.accounts = append(u.accounts, accountID)
	u.bodies = append(u.bodies, body)
	call := len(u.accounts)
	id := fmt.Sprintf("resp_history_%d", call)
	u.mu.Unlock()
	if u.failSecond && call == 2 {
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{
			"Content-Type": []string{"application/json"}, "X-Codex-Primary-Used-Percent": []string{"100"},
			"X-Codex-Primary-Window-Minutes": []string{"300"}, "X-Codex-Primary-Reset-After-Seconds": []string{"3600"},
		}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"rate_limit_error","message":"quota exhausted"}}`))}, nil
	}
	payload := fmt.Sprintf(`{"id":%q,"object":"response","status":"completed","model":"gpt-5.1","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1}}`, id)
	contentType := "application/json"
	if u.stream {
		contentType = "text/event-stream"
		payload = fmt.Sprintf("data: {\"type\":\"response.created\",\"response\":{\"id\":%q,\"status\":\"in_progress\"}}\n\ndata: {\"type\":\"response.completed\",\"response\":%s}\n\n", id, payload)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(strings.NewReader(payload))}, nil
}

func (u *historyHTTPUpstream) calls() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.accounts...)
}

func newHistoryHTTPHandler(t *testing.T, loose bool, accountType string, upstream *historyHTTPUpstream) (*OpenAIGatewayHandler, *openAIHistoryHandlerRepository) {
	t.Helper()
	h := newOpenAIResponsesFailoverTestHandler(t, upstream)
	accounts := []service.Account{
		{ID: 1, Platform: service.PlatformOpenAI, Type: accountType, Status: service.StatusActive, Schedulable: true, Priority: 0, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-a", "api_key": "test-a"}},
		{ID: 2, Platform: service.PlatformOpenAI, Type: accountType, Status: service.StatusActive, Schedulable: true, Priority: 1, GroupIDs: []int64{3131}, Credentials: map[string]any{"access_token": "test-b", "api_key": "test-b"}, Extra: map[string]any{service.OpenAIOAuthRejectExternalHistoryKey: !loose}},
	}
	repo := withOpenAIHistoryTestRepository(openAIImagesFailoverAccountRepo{accounts: accounts}).(*openAIHistoryHandlerRepository)
	h.gatewayService.SetAccountRepoForTest(repo)
	return h, repo
}

func historyHTTPContext(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	c, recorder := newOpenAIResponsesFailoverTestContext(t, nil)
	c.Request.Body = io.NopCloser(strings.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("session_id", "history-http-session")
	return c, recorder
}

func TestOpenAIHistoryHTTPRoutingAndAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, body, accountType string
		loose                   bool
		wantStatus              int
		wantAccounts            []int64
	}{
		{"new", `{"model":"gpt-5.1","input":"hello"}`, service.AccountTypeOAuth, false, 200, []int64{1}},
		{"codex_first_turn", string(codexHistoryFirstTurn(t)), service.AccountTypeOAuth, false, 200, []int64{1}},
		{"strict_external", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, service.AccountTypeOAuth, false, 400, nil},
		{"mixed_pool", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, service.AccountTypeOAuth, true, 200, []int64{2}},
		{"api_key_unchanged", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, service.AccountTypeAPIKey, false, 200, []int64{1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &historyHTTPUpstream{}
			h, repo := newHistoryHTTPHandler(t, tc.loose, tc.accountType, upstream)
			c, recorder := historyHTTPContext(t, tc.body)
			h.Responses(c)
			require.Equal(t, tc.wantStatus, recorder.Code, recorder.Body.String())
			require.Equal(t, tc.wantAccounts, upstream.accounts)
			if tc.wantStatus == 400 {
				require.Equal(t, "external_history_not_allowed", gjson.GetBytes(recorder.Body.Bytes(), "error.code").String())
				require.Zero(t, repo.writes)
			}
			if len(upstream.bodies) > 0 {
				input := gjson.Get(tc.body, "input")
				if input.Type == gjson.String && tc.accountType == service.AccountTypeOAuth {
					require.Equal(t, input.String(), gjson.GetBytes(upstream.bodies[0], "input.0.content").String())
				} else if tc.name == "codex_first_turn" {
					assertCodexHistoryInputPreserved(t, input.Raw, gjson.GetBytes(upstream.bodies[0], "input").Raw)
				} else {
					require.JSONEq(t, input.Raw, gjson.GetBytes(upstream.bodies[0], "input").Raw)
				}
			}
		})
	}
}

func TestOpenAIHistoryHTTPStorageFailureDoesNotExposeResponse(t *testing.T) {
	for _, kind := range []string{"session", "response"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream_%t", kind, stream), func(t *testing.T) {
				upstream := &historyHTTPUpstream{stream: stream}
				h, repo := newHistoryHTTPHandler(t, false, service.AccountTypeOAuth, upstream)
				repo.failWrites = kind
				c, recorder := historyHTTPContext(t, fmt.Sprintf(`{"model":"gpt-5.1","input":"hello","stream":%t}`, stream))
				h.Responses(c)
				require.NotContains(t, recorder.Body.String(), "resp_history_1")
				require.Contains(t, recorder.Body.String(), "conversation_ownership_unavailable")
				if kind == "session" {
					require.Empty(t, upstream.accounts)
				} else {
					require.Equal(t, []int64{1}, upstream.accounts)
				}
			})
		}
	}
}

func TestOpenAIHistoryCompatibilityAndCountingRejectBeforeUpstream(t *testing.T) {
	for _, tc := range []struct {
		path   string
		body   string
		handle func(*OpenAIGatewayHandler, *gin.Context)
	}{
		{"/v1/responses/compact", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, (*OpenAIGatewayHandler).Responses},
		{"/v1/chat/completions", `{"model":"gpt-5.1","messages":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, (*OpenAIGatewayHandler).ChatCompletions},
		{"/v1/messages", `{"model":"gpt-5.1","max_tokens":64,"messages":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, (*OpenAIGatewayHandler).Messages},
		{"/v1/responses/input_tokens", `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, (*OpenAIGatewayHandler).ResponsesInputTokens},
		{"/v1/messages/count_tokens", `{"model":"gpt-5.1","messages":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`, (*OpenAIGatewayHandler).CountTokens},
	} {
		t.Run(tc.path, func(t *testing.T) {
			upstream := &historyHTTPUpstream{}
			h, repo := newHistoryHTTPHandler(t, false, service.AccountTypeOAuth, upstream)
			c, recorder := historyHTTPContext(t, tc.body)
			c.Request.URL.Path = tc.path
			key, _ := middleware.GetAPIKeyFromContext(c)
			key.Group.AllowMessagesDispatch = true
			tc.handle(h, c)
			require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
			require.Contains(t, gjson.GetBytes(recorder.Body.Bytes(), "error.message").String(), service.ErrOpenAIExternalHistory.Error())
			require.Empty(t, upstream.calls())
			require.Zero(t, repo.writes)
		})
	}
}

func TestOpenAIHistoryAuditBeforeOwnershipAndUpstream(t *testing.T) {
	for _, accountType := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		t.Run(accountType, func(t *testing.T) {
			upstream := &historyHTTPUpstream{}
			h, repo := newHistoryHTTPHandler(t, true, accountType, upstream)
			engine := &turnCountingEngine{mode: securityaudit.ModeBlocking, decisions: []*securityaudit.PromptDecision{{Kind: securityaudit.DecisionBlock, AllowNextStage: false}}}
			h.securityAuditCoordinator = securityaudit.NewCoordinator(nil, engine)
			c, recorder := historyHTTPContext(t, `{"model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"blocked"}]}`)
			h.Responses(c)
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, int64(1), engine.evaluates.Load())
			require.Zero(t, repo.lookups)
			require.Zero(t, repo.writes)
			require.Empty(t, upstream.accounts)
		})
	}
}

func TestOpenAIHistoryCanonicalAuditContract(t *testing.T) {
	for _, tc := range []struct{ protocol, body string }{
		{service.ContentModerationProtocolOpenAIResponses, `{"input":[{"type":"function_call","call_id":"c1","name":"read","arguments":"{\"path\":\"old-file\"}"},{"type":"function_call_output","call_id":"c1","output":"old result"},{"role":"user","content":"next question"}]}`},
		{service.ContentModerationProtocolOpenAIChat, `{"messages":[{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"read","arguments":"{\"path\":\"old-file\"}"}}]},{"role":"tool","tool_call_id":"c1","content":"old result"},{"role":"user","content":"next question"}]}`},
		{service.ContentModerationProtocolAnthropicMessages, `{"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"c1","name":"read","input":{"path":"old-file"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"c1","content":"old result"},{"type":"text","text":"next question"}]}]}`},
	} {
		t.Run(tc.protocol, func(t *testing.T) {
			body := []byte(tc.body)
			original := bytes.Clone(body)
			c, _ := historyHTTPContext(t, tc.body)
			svc := new(service.OpenAIGatewayService)
			require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, tc.protocol, body, "canonical"))
			require.Equal(t, original, body)
			document, err := auditcontent.Extract(tc.protocol, body)
			require.NoError(t, err)
			require.True(t, document.HistoryBearing)
			require.False(t, document.Incomplete)
			require.Equal(t, "next question", service.ExtractContentModerationText(tc.protocol, body))
			snapshot, err := securityaudit.ExtractPromptSnapshot(securityaudit.Request{Protocol: tc.protocol, Body: body})
			require.NoError(t, err)
			require.Contains(t, snapshot.ScanText, "old-file")
			require.Contains(t, snapshot.ScanText, "old result")
			require.Contains(t, snapshot.ScanText, "next question")
		})
	}
}

func TestOpenAIHistoryWebSocketTurns(t *testing.T) {
	for _, tc := range []struct {
		name            string
		firstHistory    bool
		codexFirstTurn  bool
		unknownPrevious bool
		blockAudit      bool
		failSecond      bool
	}{
		{name: "reject_external_first_turn", firstHistory: true},
		{name: "reject_unknown_response_on_later_turn", unknownPrevious: true},
		{name: "continue_owned_response"},
		{name: "codex_first_turn_then_owned_response", codexFirstTurn: true},
		{name: "codex_first_turn_audit_block", codexFirstTurn: true, blockAudit: true},
		{name: "audit_before_history", firstHistory: true, blockAudit: true},
		{name: "current_turn_failover_keeps_original_owner", failSecond: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOpsErrorLogTestQueue(t, 4)
			ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			upstream := &historyHTTPUpstream{stream: true, failSecond: tc.failSecond}
			h, repo := newHistoryHTTPHandler(t, false, service.AccountTypeOAuth, upstream)
			h.cfg.Gateway.OpenAIWS.Enabled = true
			h.cfg.Gateway.OpenAIWS.OAuthEnabled = true
			h.cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
			h.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
			h.cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = true
			accounts := repo.AccountRepository.(openAIImagesFailoverAccountRepo)
			for i := range accounts.accounts {
				if accounts.accounts[i].Extra == nil {
					accounts.accounts[i].Extra = map[string]any{}
				}
				accounts.accounts[i].Extra["openai_oauth_responses_websockets_v2_mode"] = service.OpenAIWSIngressModeHTTPBridge
				accounts.accounts[i].Extra["openai_oauth_responses_websockets_v2_enabled"] = true
			}
			repo.AccountRepository = accounts
			engine := &turnCountingEngine{mode: securityaudit.ModeBlocking}
			if tc.blockAudit {
				engine.decisions = []*securityaudit.PromptDecision{{Kind: securityaudit.DecisionBlock, AllowNextStage: false}}
			}
			h.securityAuditCoordinator = securityaudit.NewCoordinator(nil, engine)
			server := newOpenAIWSHandlerTestServer(t, h, middleware.AuthSubject{UserID: 100}, func(c *gin.Context) {
				c.Header("X-Request-Id", "history-ws-request")
				c.Next()
			}, OpsErrorLoggerMiddleware(ops))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/openai/v1/responses", &coderws.DialOptions{HTTPHeader: http.Header{"session_id": []string{"ws-history"}}})
			require.NoError(t, err)
			defer func() { _ = conn.CloseNow() }()
			first := `{"type":"response.create","model":"gpt-5.1","input":"hello"}`
			if tc.codexFirstTurn {
				var event map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(codexHistoryFirstTurn(t), &event))
				event["type"] = json.RawMessage(`"response.create"`)
				event["stream"] = json.RawMessage(`true`)
				encoded, err := json.Marshal(event)
				require.NoError(t, err)
				first = string(encoded)
			}
			if tc.firstHistory {
				first = `{"type":"response.create","model":"gpt-5.1","input":[{"role":"assistant","content":"old"},{"role":"user","content":"next"}]}`
			}
			require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(first)))
			readTerminal := func() []byte {
				for {
					_, frame, err := conn.Read(ctx)
					require.NoError(t, err)
					kind := gjson.GetBytes(frame, "type").String()
					if kind == "response.completed" || kind == "error" {
						return frame
					}
				}
			}
			frame := readTerminal()
			if tc.firstHistory || tc.blockAudit {
				if !tc.blockAudit {
					require.Equal(t, "external_history_not_allowed", gjson.GetBytes(frame, "error.code").String())
					_ = conn.CloseNow()
					entry := takeHistoryOpsLog(t)
					assertHistoryOpsRejection(t, entry, 101, 2, "history-ws-request")
					require.Contains(t, entry.ErrorBody, "external_history_not_allowed")
				}
				require.Empty(t, upstream.calls())
				repo.mu.Lock()
				lookups, writes := repo.lookups, repo.writes
				repo.mu.Unlock()
				require.Zero(t, writes)
				if tc.blockAudit {
					require.Zero(t, lookups)
					require.Equal(t, int64(1), engine.evaluates.Load())
					require.Equal(t, "error", gjson.GetBytes(frame, "type").String())
				}
				return
			}
			require.Equal(t, "resp_history_1", gjson.GetBytes(frame, "response.id").String())
			if tc.codexFirstTurn {
				upstream.mu.Lock()
				forwarded := bytes.Clone(upstream.bodies[0])
				upstream.mu.Unlock()
				assertCodexHistoryInputPreserved(t, gjson.Get(first, "input").Raw, gjson.GetBytes(forwarded, "input").Raw)
			}
			previous := "resp_history_1"
			if tc.unknownPrevious {
				previous = "resp_unknown"
			}
			second := fmt.Sprintf(`{"type":"response.create","model":"gpt-5.1","previous_response_id":%q,"input":"continue"}`, previous)
			require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(second)))
			frame = readTerminal()
			if tc.failSecond {
				require.Equal(t, "external_history_not_allowed", gjson.GetBytes(frame, "error.code").String())
				require.Equal(t, []int64{1, 1}, upstream.calls(), "the replacement strict account must never receive another owner's history")
			} else if tc.unknownPrevious {
				require.Equal(t, "conversation_reconnect_required", gjson.GetBytes(frame, "error.code").String())
				require.Equal(t, []int64{1}, upstream.calls())
			} else {
				require.Equal(t, "resp_history_2", gjson.GetBytes(frame, "response.id").String())
				require.Equal(t, []int64{1, 1}, upstream.calls())
			}
			require.Equal(t, int64(2), engine.evaluates.Load())
			if tc.unknownPrevious || tc.failSecond {
				_ = conn.CloseNow()
				entry := takeHistoryOpsLog(t)
				assertHistoryOpsRejection(t, entry, 101, 2, "history-ws-request")
				require.Contains(t, entry.ErrorBody, gjson.GetBytes(frame, "error.code").String())
			}
		})
	}
}
