package securityaudit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeConversationOutputUsesFinalResponsesAggregate(t *testing.T) {
	raw := []byte(
		`{"type":"response.output_text.delta","delta":"partial"}` + "\n" +
			`{"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"final answer"}]},{"type":"reasoning","encrypted_content":"secret"}]}}`,
	)

	output, responseID, err := normalizeConversationOutput("openai_responses", raw, "application/json", true)
	require.NoError(t, err)
	require.Equal(t, "resp_1", responseID)
	require.Contains(t, output, "final answer")
	require.NotContains(t, output, "partial")
	require.NotContains(t, output, "secret")
}

func TestNormalizeConversationOutputCollectsChatSSEDeltas(t *testing.T) {
	raw := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n" +
		"data: [DONE]\n\n")

	output, _, err := normalizeConversationOutput("openai_chat_completions", raw, "text/event-stream", false)
	require.NoError(t, err)
	require.Contains(t, output, "hello ")
	require.Contains(t, output, "world")
}

func TestNormalizeConversationOutputRejectsFailedOrIncompleteStream(t *testing.T) {
	_, _, err := normalizeConversationOutput("openai_responses", []byte(`{"type":"response.failed","response":{"status":"failed"}}`), "application/json", true)
	require.Error(t, err)

	_, _, err = normalizeConversationOutput("openai_responses", []byte(`{"type":"response.output_text.delta","delta":"partial"}`), "application/json", true)
	require.Error(t, err)

	_, _, err = normalizeConversationOutput("openai_responses", nil, "application/json", false)
	require.Error(t, err)

	_, _, err = normalizeConversationOutput("openai_responses", []byte(`{"status":"completed","future_output":"ambiguous"}`), "application/json", false)
	require.Error(t, err)
}
