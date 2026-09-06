//go:build unit

package auditcontent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationHistoryClassificationPreservesCanonicalContent(t *testing.T) {
	for _, tc := range []struct {
		name, protocol, body string
		history              bool
	}{
		{"new_responses", "openai_responses", `{"model":"gpt-5.1","instructions":"policy","tools":[{"type":"function","name":"read","parameters":{"type":"object"}}],"input":"hello"}`, false},
		{"responses_history", "openai_responses", `{"input":[{"role":"assistant","content":"old answer"},{"role":"user","content":"next"}]}`, true},
		{"responses_opaque", "openai_responses", `{"input":[{"type":"compaction","id":"cmp_1","encrypted_content":"opaque-state"},{"role":"user","content":"next"}]}`, true},
		{"responses_reference", "openai_responses", `{"input":[{"type":"item_reference","id":"fc_1"}]}`, true},
		{"responses_empty_output", "openai_responses", `{"input":[{"type":"function_call_output","call_id":"call_1","output":""}]}`, true},
		{"chat_new", "openai_chat_completions", `{"messages":[{"role":"system","content":"instructions"},{"role":"user","content":"new question"}]}`, false},
		{"chat_tool", "openai_chat_completions", `{"messages":[{"role":"tool","tool_call_id":"call_1","content":{"ok":true}}]}`, true},
		{"messages_new", "anthropic_messages", `{"system":"instructions","messages":[{"role":"user","content":"new question"}]}`, false},
		{"messages_tool", "anthropic_messages", `{"messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool_1","content":""}]}]}`, true},
		{"messages_redacted", "anthropic_messages", `{"messages":[{"role":"assistant","content":[{"type":"redacted_thinking","data":"opaque"}]},{"role":"user","content":"next"}]}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := Extract(tc.protocol, []byte(tc.body))
			require.NoError(t, err)
			require.Equal(t, tc.history, document.HistoryBearing)
		})
	}
}
