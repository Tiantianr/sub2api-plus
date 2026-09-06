//go:build unit

package handler

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func codexHistoryFirstTurn(t *testing.T) []byte {
	t.Helper()
	body, err := os.ReadFile("../auditcontent/testdata/codex_first_turn.json")
	require.NoError(t, err)
	return body
}

func TestOpenAIHistoryCodexFirstTurnPreservesBothAuditContracts(t *testing.T) {
	body := codexHistoryFirstTurn(t)
	original := bytes.Clone(body)
	c, _ := historyHTTPContext(t, string(body))
	svc := new(service.OpenAIGatewayService)
	require.NoError(t, svc.PrepareOpenAIHistoryRequest(c, service.ContentModerationProtocolOpenAIResponses, body, "codex-first"))
	require.Equal(t, original, body)
	document, err := auditcontent.Extract(service.ContentModerationProtocolOpenAIResponses, body)
	require.NoError(t, err)
	require.False(t, document.HistoryBearing)
	require.False(t, document.Incomplete)
	require.Equal(t, "How many candies guarantee an apple and a peach of different shapes?",
		service.ExtractContentModerationText(service.ContentModerationProtocolOpenAIResponses, body))
	snapshot, err := securityaudit.ExtractPromptSnapshot(securityaudit.Request{Protocol: service.ContentModerationProtocolOpenAIResponses, Body: body})
	require.NoError(t, err)
	require.Contains(t, snapshot.ScanText, "# AGENTS.md")
	require.Contains(t, snapshot.ScanText, "Run the focused Go tests.")
	require.Contains(t, snapshot.ScanText, "How many candies")
	require.NotContains(t, snapshot.ScanText, "<environment_context>")
}

type historyFirstTurnModerationBlock struct {
	calls int
	body  []byte
}

func (e *historyFirstTurnModerationBlock) Check(_ context.Context, req securityaudit.Request) (*securityaudit.LegacyDecision, error) {
	e.calls++
	e.body = bytes.Clone(req.Body)
	return &securityaudit.LegacyDecision{Blocked: true, StatusCode: http.StatusForbidden,
		ErrorCode: "content_policy_violation", Message: "blocked by moderation"}, nil
}

func TestOpenAIHistoryCodexFirstTurnAuditsBeforeSideEffects(t *testing.T) {
	body := codexHistoryFirstTurn(t)
	for _, accountType := range []string{service.AccountTypeOAuth, service.AccountTypeAPIKey} {
		for _, engineName := range []string{"content_moderation", "prompt_audit"} {
			t.Run(accountType+"/"+engineName, func(t *testing.T) {
				upstream := &historyHTTPUpstream{}
				h, repo := newHistoryHTTPHandler(t, false, accountType, upstream)
				legacy := &historyFirstTurnModerationBlock{}
				prompt := &turnCountingEngine{mode: securityaudit.ModeBlocking,
					decisions: []*securityaudit.PromptDecision{{Kind: securityaudit.DecisionBlock, AllowNextStage: false}}}
				if engineName == "content_moderation" {
					h.securityAuditCoordinator = securityaudit.NewCoordinator(legacy, nil)
				} else {
					h.securityAuditCoordinator = securityaudit.NewCoordinator(nil, prompt)
				}
				c, recorder := historyHTTPContext(t, string(body))
				h.Responses(c)
				require.Equal(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
				require.Zero(t, repo.lookups)
				require.Zero(t, repo.writes)
				require.Empty(t, upstream.calls())
				if engineName == "content_moderation" {
					require.Equal(t, 1, legacy.calls)
					require.JSONEq(t, string(body), string(legacy.body))
				} else {
					require.Equal(t, int64(1), prompt.evaluates.Load())
				}
			})
		}
	}
}
