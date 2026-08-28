package securityaudit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompletePromptContextRetainsContentExcludedFromGuard(t *testing.T) {
	body := []byte(`{
		"instructions":"system harness",
		"tools":[{"type":"function","name":"exec","description":"tool schema"}],
		"input":[
			{"type":"message","role":"assistant","content":"assistant output"},
			{"type":"function_call_output","call_id":"c1","output":"tool result"},
			{"type":"message","role":"user","content":"<system-reminder>client wrapper</system-reminder> actual question"}
		]
	}`)
	snapshot, diagnostic, err := extractPromptSnapshotWithDiagnostics(Request{
		RequestID: "req-context", Protocol: "openai_responses", Body: body,
	}, false)
	require.NoError(t, err)
	require.False(t, diagnostic.Failed)
	require.Equal(t, "actual question", snapshot.ScanText)
	require.NotContains(t, snapshot.ScanText, "system harness")

	for _, expected := range []string{"system harness", "tool schema", "assistant output", "tool result", "client wrapper", "actual question"} {
		require.Contains(t, snapshot.CompleteContext, expected)
	}

	config := &fakeConfigStore{}
	ciphertext, err := encryptCompletePromptContext(config, snapshot.CompleteContext)
	require.NoError(t, err)
	raw, err := decryptCompletePromptContext(config, ciphertext)
	require.NoError(t, err)
	var contextPayload map[string]any
	require.NoError(t, json.Unmarshal(raw, &contextPayload))
	require.Equal(t, "all_user_turns", contextPayload["guard_mode"])
	require.Equal(t, "actual question", contextPayload["guard_input"])
	require.Len(t, contextPayload["segments"], 5)
}

func TestTransientPromptPayloadCarriesEncryptedContextAndReadsLegacyPayload(t *testing.T) {
	snapshot := PromptSnapshot{
		ScanText: "selected user text", FullContextCiphertext: "ciphertext",
		FullContextHash: "hash", FullContextBytes: 42, FullContextSegmentCount: 3,
	}
	encoded, err := encodeTransientPromptPayload(snapshot)
	require.NoError(t, err)
	decoded := decodeTransientPromptPayload(encoded)
	require.Equal(t, snapshot.ScanText, decoded.ScanText)
	require.Equal(t, snapshot.FullContextCiphertext, decoded.ContextCiphertext)
	require.Equal(t, snapshot.FullContextSegmentCount, decoded.ContextSegmentCount)

	legacy := decodeTransientPromptPayload("legacy plain scan text")
	require.Equal(t, "legacy plain scan text", legacy.ScanText)
	require.Empty(t, legacy.ContextCiphertext)
}
