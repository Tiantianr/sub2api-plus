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

func TestPromptAuditUsesUserAuthoredConversationText(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		body     string
		full     []string
		latest   string
		omit     []string
		noFull   bool
		noLatest bool
	}{
		{
			name: "assistant text and tool schema are excluded", protocol: service.ContentModerationProtocolOpenAIResponses,
			body: `{"instructions":"audit instruction","tools":[{"type":"function","name":"lookup","description":"audit tool definition"}],` +
				`"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"message","role":"assistant","content":"current assistant payload"}]}`,
			full:   []string{"older prompt"},
			latest: "older prompt",
			omit:   []string{"audit tool definition", "current assistant payload", "audit instruction"},
		},
		{
			name: "responses function result is excluded", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:   `{"input":[{"type":"message","role":"user","content":"older prompt"},{"type":"function_call_output","call_id":"call_1","output":"current tool result"}]}`,
			full:   []string{"older prompt"},
			latest: "older prompt",
			omit:   []string{"current tool result"},
		},
		{
			name: "responses reusable prompt variables are excluded", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:     `{"prompt":{"id":"pmpt_1","variables":{"plain":"reusable variable","typed":{"type":"input_text","text":"typed variable"}}}}`,
			omit:     []string{"reusable variable", "typed variable"},
			noFull:   true,
			noLatest: true,
		},
		{
			name: "responses legacy messages follow input conversion", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:   `{"model":"gpt-test","messages":[{"role":"system","content":"legacy system"},{"role":"user","content":"legacy user"}]}`,
			full:   []string{"legacy user"},
			latest: "legacy user",
			omit:   []string{"legacy system"},
		},
		{
			name: "chat reasoning is excluded", protocol: service.ContentModerationProtocolOpenAIChat,
			body:   `{"messages":[{"role":"assistant","reasoning_content":"visible reasoning"},{"role":"user","content":"continue"}]}`,
			full:   []string{"continue"},
			latest: "continue",
			omit:   []string{"visible reasoning"},
		},
		{
			name: "chat user reasoning remains user-authored text", protocol: service.ContentModerationProtocolOpenAIChat,
			body:   `{"messages":[{"role":"assistant","reasoning_content":"assistant reasoning"},{"role":"user","content":"continue","reasoning_content":"user reasoning"}]}`,
			full:   []string{"continue", "user reasoning"},
			latest: "continue\n\nuser reasoning",
			omit:   []string{"assistant reasoning"},
		},
		{
			name: "live session instructions and transcription context are excluded", protocol: service.ContentModerationProtocolOpenAILive,
			body: `{"model":"gpt-live-test","instructions":"live instructions",` +
				`"input_audio_transcription":{"model":"gpt-4o-transcribe","prompt":"legacy transcription context"},` +
				`"audio":{"input":{"transcription":{"model":"gpt-live-transcribe","prompt":"current transcription context","keywords":["premium plan","AC-42"]}}}}`,
			omit:     []string{"live instructions", "legacy transcription context", "current transcription context", "premium plan", "AC-42"},
			noFull:   true,
			noLatest: true,
		},
		{
			name: "client environment xml is stripped from user text", protocol: service.ContentModerationProtocolOpenAIResponses,
			body:   `{"input":[{"type":"message","role":"user","content":"继续"},{"type":"message","role":"user","content":"<environment_context><cwd>/Users/pontus</cwd><permission_profile type=\"managed\"></permission_profile></environment_context>"},{"type":"message","role":"user","content":"你能做什么？"}]}`,
			full:   []string{"继续", "你能做什么？"},
			latest: "继续\n\n你能做什么？",
			omit:   []string{"environment_context", "permission_profile", "/Users/pontus"},
		},
		{
			name: "chat tool-role and tool-call arguments are omitted", protocol: service.ContentModerationProtocolOpenAIChat,
			body:   `{"messages":[{"role":"system","content":"chat system context"},{"role":"user","content":"older"},{"role":"assistant","tool_calls":[{"function":{"arguments":"{\"secret\":true}"}}]},{"role":"tool","content":{"first":true}},{"role":"function","content":{"second":false}}]}`,
			full:   []string{"older"},
			latest: "older",
			omit:   []string{`"secret":true`, `{"first":true}`, `{"second":false}`, "chat system context"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			full, err := securityaudit.ExtractPromptSnapshot(securityaudit.Request{
				Protocol: test.protocol,
				Body:     []byte(test.body),
			})
			if test.noFull {
				require.ErrorIs(t, err, securityaudit.ErrNoPromptText)
			} else {
				require.NoError(t, err)
				for _, expected := range test.full {
					require.Contains(t, full.ScanText, expected)
				}
				for _, omitted := range test.omit {
					require.NotContains(t, full.ScanText, omitted)
				}
			}
			latest, err := securityaudit.ExtractBlockingPromptSnapshot(securityaudit.Request{
				Protocol: test.protocol,
				Body:     []byte(test.body),
			}, true)
			if test.noLatest {
				require.ErrorIs(t, err, securityaudit.ErrNoPromptText)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.latest, latest.ScanText)
				for _, omitted := range test.omit {
					require.NotContains(t, latest.ScanText, omitted)
				}
			}
		})
	}
}
