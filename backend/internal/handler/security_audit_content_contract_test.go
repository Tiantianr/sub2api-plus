package handler

import (
	"testing"

	"github.com/LuckyKuang/sub2api-plus/internal/securityaudit"
	"github.com/LuckyKuang/sub2api-plus/internal/service"
	"github.com/stretchr/testify/require"
)

func TestContentModerationUsesLatestUserTextWithoutInstructionContext(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		body     string
		want     string
		images   []string
	}{
		{
			name: "responses latest user", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"instructions":"audit instruction","tools":[{"type":"function","name":"lookup","description":"audit tool definition"}],` +
				`"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"message","role":"user","content":"你好"}]}`,
			want: "你好",
		},
		{
			name: "responses legacy messages", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"model":"gpt-test","messages":[{"role":"system","content":"legacy system"},{"role":"user","content":"legacy user"}]}`,
			want: "legacy user",
		},
		{
			name: "responses legacy prompt", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"model":"gpt-test","prompt":"legacy prompt"}`,
			want: "legacy prompt",
		},
		{
			name: "responses tool loop skipped", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"function_call_output","call_id":"call_1","output":"current tool result"}]}`,
		},
		{
			name: "responses assistant item skipped", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"message","role":"assistant","content":"current assistant payload"}]}`,
		},
		{
			name: "responses prompt variables skipped", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"prompt":{"id":"pmpt_1","variables":{"plain":"reusable variable","image":{"type":"input_image","image_url":"https://example.test/prompt.png"}}}}`,
		},
		{
			name: "nested websocket user", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"type":"response.create","input":[],"response":{"input":[{"type":"message","role":"user","content":"nested content cannot be shadowed"}]}}`,
			want: "nested content cannot be shadowed",
		},
		{
			name: "chat latest user", protocol: service.ContentModerationProtocolOpenAIChat,
			body: `{"messages":[{"role":"system","content":"chat system context"},{"role":"user","content":"你好"}]}`,
			want: "你好",
		},
		{
			name: "chat reminder markup is untrusted user text", protocol: service.ContentModerationProtocolOpenAIChat,
			body: `{"messages":[{"role":"user","content":"<system-reminder>context</system-reminder> real question"}]}`,
			want: "<system-reminder>context</system-reminder> real question",
		},
		{
			name: "chat user reasoning remains direct user text", protocol: service.ContentModerationProtocolOpenAIChat,
			body: `{"messages":[{"role":"assistant","reasoning_content":"assistant private"},{"role":"user","content":"user content","reasoning_content":"user reasoning"}]}`,
			want: "user content user reasoning",
		},
		{
			name: "chat tool result skipped", protocol: service.ContentModerationProtocolOpenAIChat,
			body: `{"messages":[{"role":"user","content":"older"},{"role":"tool","content":"chat tool result"}]}`,
		},
		{
			name: "anthropic tool result skipped", protocol: service.ContentModerationProtocolAnthropicMessages,
			body: `{"system":"anthropic instruction","messages":[{"role":"user","content":[{"type":"tool_result","content":{"safe":false}}]}]}`,
		},
		{
			name: "gemini function response skipped", protocol: service.ContentModerationProtocolGemini,
			body: `{"systemInstruction":{"parts":[{"text":"gemini instruction"}]},"contents":[{"role":"user","parts":[{"functionResponse":{"response":{"safe":false}}}]}]}`,
		},
		{
			name: "live user image preserved", protocol: service.ContentModerationProtocolOpenAILive,
			body:   `{"type":"conversation.item.create","item":{"type":"message","role":"user","content":[{"type":"input_image","image_url":"https://example.test/live.png"}]}}`,
			images: []string{"https://example.test/live.png"},
		},
		{
			name: "live session instructions skipped", protocol: service.ContentModerationProtocolOpenAILive,
			body: `{"model":"gpt-live-test","instructions":"live instructions"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			legacy := service.ExtractContentModerationInput(test.protocol, []byte(test.body))
			require.Equal(t, test.want, legacy.Text)
			if len(test.images) == 0 {
				require.Empty(t, legacy.Images)
			} else {
				require.Equal(t, test.images, legacy.Images)
			}
			require.NotContains(t, legacy.Text, "instruction")
			require.NotContains(t, legacy.Text, "tool definition")
			require.NotContains(t, legacy.Text, "system context")
		})
	}
}

func TestPromptAuditUsesCompleteConversationTextWhileLatestHelperStaysNarrow(t *testing.T) {
	tests := []struct {
		name       string
		protocol   string
		body       string
		full       []string
		latest     string
		omitFull   []string
		omitLatest []string
		noLatest   bool
	}{
		{
			name: "assistant text and tool schema are scanned in complete context", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"instructions":"audit instruction","tools":[{"type":"function","name":"lookup","description":"audit tool definition"}],` +
				`"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"message","role":"assistant","content":"current assistant payload"}]}`,
			full:       []string{"current assistant payload", "audit instruction", "audit tool definition", "older prompt"},
			latest:     "older prompt",
			omitLatest: []string{"current assistant payload", "audit instruction", "audit tool definition"},
		},
		{
			name: "responses function result remains complete prompt-audit context", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:       `{"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"function_call_output","call_id":"call_1","output":"current tool result"}]}`,
			full:       []string{"older prompt", "current tool result"},
			latest:     "older prompt",
			omitLatest: []string{"current tool result"},
		},
		{
			name: "responses reusable prompt variables remain conversation text", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:     `{"prompt":{"id":"pmpt_1","variables":{"plain":"reusable variable","typed":{"type":"input_text","text":"typed variable"}}}}`,
			full:     []string{"reusable variable", "typed variable"},
			noLatest: true,
		},
		{
			name: "responses legacy messages follow input conversion", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:   `{"model":"gpt-test","messages":[{"role":"system","content":"legacy system"},{"role":"user","content":"legacy user"}]}`,
			full:   []string{"legacy system", "legacy user"},
			latest: "legacy user",
		},
		{
			name: "chat reasoning remains full-session context", protocol: service.ContentModerationProtocolOpenAIChat,
			body:   `{"messages":[{"role":"assistant","reasoning_content":"visible reasoning"},{"role":"user","content":"continue"}]}`,
			full:   []string{"visible reasoning", "continue"},
			latest: "continue",
		},
		{
			name: "chat user reasoning is current input", protocol: service.ContentModerationProtocolOpenAIChat,
			body:   `{"messages":[{"role":"assistant","reasoning_content":"assistant reasoning"},{"role":"user","content":"continue","reasoning_content":"user reasoning"}]}`,
			full:   []string{"assistant reasoning", "continue", "user reasoning"},
			latest: "continue",
		},
		{
			name: "alpha search covers PAT fallback fields", protocol: "openai_alpha_search",
			body:   `{"commands":{"search_query":[{"q":"security query"}],"mode":"deep"},"settings":{"region":"global"},"input":[{"type":"message","role":"user","content":"recent context"}]}`,
			full:   []string{`"security query"`, `"mode":"deep"`, `"region":"global"`, `"recent context"`},
			latest: "security query",
		},
		{
			name: "responses compaction controls are opaque", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:       `{"input":[{"type":"compaction","encrypted_content":"opaque-one"},{"type":"compaction_summary","encrypted_content":"opaque-two","summary":[{"type":"summary_text","text":"visible summary"}]},{"type":"compaction_trigger","future_visible":"visible trigger field"},{"type":"message","role":"user","content":"visible sibling"}]}`,
			full:       []string{"visible summary", "visible trigger field", "visible sibling"},
			latest:     "visible sibling",
			omitFull:   []string{"opaque-one", "opaque-two"},
			omitLatest: []string{"opaque-one", "opaque-two"},
		},
		{
			name: "live initial session instructions remain conversation text", protocol: service.ContentModerationProtocolOpenAILive,
			body: `{"model":"gpt-live-test","instructions":"live instructions",` +
				`"input_audio_transcription":{"model":"gpt-4o-transcribe","prompt":"legacy transcription context"},` +
				`"audio":{"input":{"transcription":{"model":"gpt-live-transcribe","prompt":"current transcription context","keywords":["premium plan","AC-42"]}}}}`,
			full:     []string{"live instructions", "legacy transcription context", "current transcription context", "premium plan", "AC-42"},
			noLatest: true,
		},
		{
			name: "chat tool-role and tool-call arguments remain complete context", protocol: service.ContentModerationProtocolOpenAIChat,
			body:       `{"messages":[{"role":"system","content":"chat system context"},{"role":"user","content":"older"},{"role":"assistant","tool_calls":[{"function":{"arguments":"{\"secret\":true}"}}]},{"role":"tool","content":{"first":true}},{"role":"function","content":{"second":false}}]}`,
			full:       []string{"chat system context", "older", `"secret":true`, `{"first":true}`, `{"second":false}`},
			latest:     "older",
			omitLatest: []string{`"secret":true`, `{"first":true}`, `{"second":false}`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			full, err := securityaudit.ExtractPromptSnapshot(securityaudit.Request{
				Protocol: test.protocol,
				Body:     []byte(test.body),
			})
			require.NoError(t, err)
			latest, err := securityaudit.ExtractBlockingPromptSnapshot(securityaudit.Request{
				Protocol: test.protocol,
				Body:     []byte(test.body),
			}, true)
			if test.noLatest {
				require.ErrorIs(t, err, securityaudit.ErrNoPromptText)
			} else {
				require.NoError(t, err)
				require.Contains(t, latest.ScanText, test.latest)
				for _, omitted := range test.omitLatest {
					require.NotContains(t, latest.ScanText, omitted)
				}
			}

			for _, expected := range test.full {
				require.Contains(t, full.ScanText, expected)
			}
			for _, omitted := range test.omitFull {
				require.NotContains(t, full.ScanText, omitted)
			}
		})
	}
}
