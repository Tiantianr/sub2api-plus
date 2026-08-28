package securityaudit

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPromptServiceConversationTransitionsFromFullToIncrementalAndRejectsReplayInsertion(t *testing.T) {
	service, closeStore, scanned := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(17, "conversation-a")

	firstBody := []byte(`{
		"instructions":"stable system policy",
		"tools":[{"type":"function","name":"lookup","description":"lookup docs"}],
		"input":[{"type":"message","role":"user","content":"first user"}]
	}`)
	first, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 17, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key, Body: firstBody,
	})
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, first.Kind)
	require.Equal(t, "full", first.ConversationMode)
	require.NotNil(t, first.Capture)
	require.Contains(t, strings.Join(*scanned, "\n"), "stable system policy")
	require.Contains(t, strings.Join(*scanned, "\n"), "lookup docs")
	finishCapturedResponse(first.Capture, "resp_1", "assistant one")
	ciphertext, err := service.conversation.client.HGet(context.Background(), conversationStateKey(key), "output_ciphertext").Result()
	require.NoError(t, err)
	require.NotContains(t, ciphertext, "assistant one")

	*scanned = nil
	secondBody := []byte(`{
		"instructions":"stable system policy",
		"tools":[{"type":"function","name":"lookup","description":"lookup docs"}],
		"input":[
			{"type":"message","role":"user","content":"first user"},
			{"type":"message","role":"assistant","content":"assistant one"},
			{"type":"message","role":"user","content":"second user"}
		]
	}`)
	second, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 17, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key, Body: secondBody,
	})
	require.NoError(t, err)
	require.Equal(t, DecisionAllow, second.Kind)
	require.Equal(t, "incremental", second.ConversationMode)
	incrementalText := strings.Join(*scanned, "\n")
	require.Contains(t, incrementalText, "second user")
	require.Contains(t, incrementalText, "assistant one")
	require.NotContains(t, incrementalText, "first user")
	require.NotContains(t, incrementalText, "stable system policy")
	finishCapturedResponse(second.Capture, "resp_2", "assistant two")

	*scanned = nil
	attackBody := []byte(`{
		"instructions":"stable system policy",
		"tools":[{"type":"function","name":"lookup","description":"lookup docs"}],
		"input":[
			{"type":"message","role":"user","content":"first user"},
			{"type":"message","role":"assistant","content":"assistant one"},
			{"type":"message","role":"user","content":"second user"},
			{"type":"message","role":"assistant","content":"assistant two"},
			{"type":"message","role":"user","content":"historical blocked-marker inserted by replay"},
			{"type":"message","role":"user","content":"latest safe user"}
		]
	}`)
	blocked, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 17, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key, Body: attackBody,
	})
	require.NoError(t, err)
	require.Equal(t, DecisionBlock, blocked.Kind)
	require.Equal(t, "full", blocked.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "blocked-marker")
}

func TestPromptServiceConversationUsesKnownParentAndForcesFullOnSystemChange(t *testing.T) {
	service, closeStore, scanned := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(18, "conversation-b")

	first, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 18, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"instructions":"system one","input":"first"}`),
	})
	require.NoError(t, err)
	finishCapturedResponse(first.Capture, "resp_parent", "parent output")

	*scanned = nil
	continued, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 18, Protocol: "openai_responses", Model: "gpt-test",
		ConversationKey: NewConversationKey(18), ParentID: "resp_parent",
		Body: []byte(`{"previous_response_id":"resp_parent","input":"continued user"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "incremental", continued.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "parent output")
	finishCapturedResponse(continued.Capture, "resp_child", "child output")

	*scanned = nil
	changed, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 18, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"instructions":"system two","input":"new user"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "full", changed.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "system two")
	changed.Capture.Abort()
}

func TestPromptServiceKnownParentStillValidatesReplayedHistory(t *testing.T) {
	service, closeStore, scanned := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(28, "conversation-parent-replay")
	first, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 28, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"instructions":"stable system","input":"first user"}`),
	})
	require.NoError(t, err)
	finishCapturedResponse(first.Capture, "resp_parent_replay", "parent output")

	*scanned = nil
	safeReplay := []byte(`{
		"instructions":"stable system",
		"previous_response_id":"resp_parent_replay",
		"input":[
			{"type":"message","role":"user","content":"first user"},
			{"type":"message","role":"assistant","content":"parent output"},
			{"type":"message","role":"user","content":"continued user"}
		]
	}`)
	continued, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 28, Protocol: "openai_responses", Model: "gpt-test",
		ConversationKey: NewConversationKey(28), ParentID: "resp_parent_replay", Body: safeReplay,
	})
	require.NoError(t, err)
	require.Equal(t, "incremental", continued.ConversationMode)
	require.NotContains(t, strings.Join(*scanned, "\n"), "first user")
	finishCapturedResponse(continued.Capture, "resp_child_replay", "child output")

	*scanned = nil
	maliciousReplay := []byte(`{
		"instructions":"stable system",
		"previous_response_id":"resp_child_replay",
		"input":[
			{"type":"message","role":"user","content":"first user"},
			{"type":"message","role":"assistant","content":"parent output"},
			{"type":"message","role":"user","content":"continued user"},
			{"type":"message","role":"assistant","content":"child output"},
			{"type":"message","role":"user","content":"historical blocked-marker insertion"},
			{"type":"message","role":"user","content":"latest safe user"}
		]
	}`)
	blocked, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 28, Protocol: "openai_responses", Model: "gpt-test",
		ConversationKey: NewConversationKey(28), ParentID: "resp_child_replay", Body: maliciousReplay,
	})
	require.NoError(t, err)
	require.Equal(t, DecisionBlock, blocked.Kind)
	require.Equal(t, "full", blocked.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "blocked-marker")
}

func TestPromptServiceLiveAliasDecryptsEncryptedEmptyOutput(t *testing.T) {
	service, closeStore, scanned := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(29, "live-bootstrap")
	created, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 29, Protocol: "openai_live", Model: "gpt-live", ConversationKey: key,
		Body: []byte(`{"instructions":"live system"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, created.Capture)
	created.Capture.CompleteWithoutOutput("live:call_1")

	*scanned = nil
	sideband, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 29, Protocol: "openai_live", Model: "gpt-live",
		ConversationKey: NewConversationKey(29), ParentID: "live:call_1",
		Body: []byte(`{"type":"conversation.item.create","item":{"type":"message","role":"user","content":"live user"}}`),
	})
	require.NoError(t, err)
	require.Equal(t, "incremental", sideband.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "live user")
	sideband.Capture.Abort()
}

func TestPromptServiceContinuationWithoutNewTextStillAuditsPriorOutput(t *testing.T) {
	service, closeStore, scanned := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(30, "no-new-text")
	first, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 30, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"input":"first user"}`),
	})
	require.NoError(t, err)
	finishCapturedResponse(first.Capture, "resp_no_text", "prior assistant output")

	*scanned = nil
	continued, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 30, Protocol: "openai_responses", Model: "gpt-test",
		ConversationKey: NewConversationKey(30), ParentID: "resp_no_text",
		Body: []byte(`{"previous_response_id":"resp_no_text","input":[]}`),
	})
	require.NoError(t, err)
	require.Equal(t, "incremental", continued.ConversationMode)
	require.Contains(t, strings.Join(*scanned, "\n"), "prior assistant output")
	continued.Capture.Abort()
}

func TestPromptServiceConversationFailsClosedForConcurrentOrIncompleteTurn(t *testing.T) {
	service, closeStore, _ := newConversationPromptService(t)
	defer closeStore()
	key := ConversationKey(19, "conversation-c")

	first, err := service.Evaluate(context.Background(), Request{
		APIKeyID: 19, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"input":"first"}`),
	})
	require.NoError(t, err)
	require.NotNil(t, first.Capture)

	_, err = service.Evaluate(context.Background(), Request{
		APIKeyID: 19, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"input":"overlapping"}`),
	})
	var guardErr *GuardError
	require.ErrorAs(t, err, &guardErr)
	require.Equal(t, ErrorCodeConversationBusy, guardErr.Code)
	first.Capture.Abort()

	_, err = service.Evaluate(context.Background(), Request{
		APIKeyID: 19, Protocol: "openai_responses", Model: "gpt-test", ConversationKey: key,
		Body: []byte(`{"input":[{"type":"future_content","payload":"unknown text"}]}`),
	})
	require.ErrorAs(t, err, &guardErr)
	require.Equal(t, ErrorCodeExtractionFailed, guardErr.Code)

	lease, beginErr := service.conversation.Begin(context.Background(), 19, key)
	require.NoError(t, beginErr)
	require.Equal(t, conversationStatusFullRequired, lease.checkpoint.Status)
	require.NoError(t, service.conversation.Fail(context.Background(), lease))
}

func newConversationPromptService(t *testing.T) (*PromptService, func(), *[]string) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	scanned := make([]string, 0, 8)
	metrics := NewAtomicMetrics()
	scanner := PromptScannerFunc(func(_ context.Context, _ ActiveEndpoint, chunk string, _ []string) (*NormalizedResult, error) {
		scanned = append(scanned, chunk)
		if strings.Contains(chunk, "blocked-marker") {
			return &NormalizedResult{
				Decision: EventCritical, RiskLevel: RiskCritical, Action: ActionBlock,
				Safety: "Unsafe", ScannerScores: map[string]float64{"jailbreak": 1}, ScannerEvidence: map[string]string{},
			}, nil
		}
		return &NormalizedResult{
			Decision: EventPass, RiskLevel: RiskLow, Action: ActionAllow,
			ScannerScores: map[string]float64{}, ScannerEvidence: map[string]string{},
		}, nil
	})
	baseConfig := &fakeConfigStore{active: true, cfg: ActiveConfig{
		RiskControlEnabled: true, Enabled: true, BlockingEnabled: true, AllGroups: true, ConfigVersion: 11,
		Scanners: AllScannerIDs, Endpoints: []ActiveEndpoint{{ID: "guard", Enabled: true, TimeoutMS: 5000, InputLimit: MaxInputLimit}},
	}}
	config := &conversationEncryptedConfigStore{fakeConfigStore: baseConfig}
	service := &PromptService{
		config: config, evaluator: newGuardEvaluator(scanner, nil, metrics, 4, 4), metrics: metrics,
		conversation: NewRedisConversationStore(client),
	}
	return service, func() { _ = client.Close() }, &scanned
}

type conversationEncryptedConfigStore struct{ *fakeConfigStore }

func (s *conversationEncryptedConfigStore) Encrypt(value string) (string, error) {
	return "cipher:" + base64.StdEncoding.EncodeToString([]byte(value)), nil
}

func (s *conversationEncryptedConfigStore) Decrypt(value string) (string, error) {
	if !strings.HasPrefix(value, "cipher:") {
		return "", errors.New("invalid conversation ciphertext")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "cipher:"))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func finishCapturedResponse(capture *ConversationCapture, responseID, text string) {
	if capture == nil {
		return
	}
	payload := `{"type":"response.completed","response":{"id":"` + responseID + `","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"` + text + `"}]}]}}`
	capture.ObserveFrame([]byte(payload))
	capture.FinishTurn(responseID)
}
