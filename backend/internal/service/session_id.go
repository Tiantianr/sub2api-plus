package service

import (
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// maxPersistedSessionIDLength bounds the persisted client session identifier to the
// usage_logs.session_id column width (VARCHAR(255)). Longer values are rejected so
// distinct identifiers can never alias through truncation.
const (
	maxPersistedSessionIDLength = 255
	codexSessionIDHeader        = "session-id"
)

// clientSessionIDHeaders extends the OpenAI-compatible sticky-session signals with
// native protocol identifiers that are safe to persist but must not alter OpenAI
// scheduling behavior.
var clientSessionIDHeaders = append(
	append([]string(nil), explicitOpenAIHeaderSessionNames...),
	claudeCodeSessionHeader,
)

// ClaudeCodeSessionIDFromHeader returns the stable Claude Code conversation
// identifier carried by X-Claude-Code-Session-Id. It is intentionally exposed
// separately from ExtractClientSessionID: callers that use it for routing must
// make that scope explicit rather than accidentally changing every protocol's
// session semantics.
func ClaudeCodeSessionIDFromHeader(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	return sanitizeSessionID(c.GetHeader(claudeCodeSessionHeader))
}

// ExtractClientSessionID resolves the explicit client-provided session identifier from
// request headers for usage-log correlation and returns it sanitized. It is
// protocol-agnostic and shared by every gateway handler so all supported protocols
// record session_id through one seam. Returns "" when no valid identifier is present.
//
// This value feeds only usage_logs.session_id persistence. It does NOT affect sticky
// routing, account selection, request_id semantics, or upstream prompt caching, which
// keep their own (intentionally broader) session-signal resolution.
func ExtractClientSessionID(c *gin.Context) string {
	value, _ := ExtractClientSessionIdentityWithBody(c, nil)
	return value
}

// ExtractClientSessionIDWithBody resolves the stable client session identity
// from headers and, for OpenAI-compatible JSON requests, conversation_id or
// thread_id. prompt_cache_key is deliberately excluded because it may be
// shared by unrelated conversations.
// The body fallback is used only for audit grouping; request-scoped IDs are
// deliberately excluded so every turn does not become a new session.
func ExtractClientSessionIDWithBody(c *gin.Context, body []byte) string {
	value, _ := ExtractClientSessionIdentityWithBody(c, body)
	return value
}

// ExtractClientSessionIdentityWithBody also returns a stable source namespace
// so equal values from unrelated header/body fields cannot collapse together.
func ExtractClientSessionIdentityWithBody(c *gin.Context, body []byte) (string, string) {
	if c == nil || c.Request == nil {
		return "", ""
	}
	for _, header := range clientSessionIDHeaders {
		if sessionID := sanitizeSessionID(c.GetHeader(header)); sessionID != "" {
			return sessionID, "header:" + strings.ToLower(header)
		}
	}
	if sessionID := sanitizeSessionID(openAICodexTurnMetadataSessionID(c.GetHeader("X-Codex-Turn-Metadata"))); sessionID != "" {
		return sessionID, "codex_turn_metadata"
	}
	if isGrokRequestContext(c) {
		if sessionID := sanitizeSessionID(c.GetHeader(grokConversationIDHeader)); sessionID != "" {
			return sessionID, "grok_conversation_header"
		}
	}
	if sessionID, source := extractBodySessionID(body); sessionID != "" {
		return sessionID, source
	}
	return "", ""
}

func extractBodySessionID(body []byte) (string, string) {
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return "", ""
	}
	root := gjson.ParseBytes(body)
	for _, path := range []string{"conversation_id", "thread_id"} {
		if key := bodySessionValue(root.Get(path)); key != "" {
			return key, "body:" + path
		}
	}
	if eventType := strings.ToLower(strings.TrimSpace(root.Get("type").String())); strings.HasPrefix(eventType, "response.") {
		if response := root.Get("response"); response.Exists() && response.IsObject() {
			for _, path := range []string{"conversation_id", "thread_id"} {
				if key := bodySessionValue(response.Get(path)); key != "" {
					return key, "response:" + path
				}
			}
		}
	}
	return "", ""
}

func bodySessionValue(value gjson.Result) string {
	if value.Type != gjson.String {
		return ""
	}
	return sanitizeSessionID(value.String())
}

// sanitizeSessionID normalizes a raw client-supplied session identifier for safe
// persistence: it trims surrounding whitespace, rejects the value outright if it
// contains any control character (CR/LF/tab/NUL/…) so a log- or header-injection style
// payload cannot slip into stored correlation data, and rejects values longer than
// the DB column bound. Absent or invalid input yields "".
func sanitizeSessionID(raw string) string {
	if !utf8.ValidString(raw) {
		return ""
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	count := 0
	for _, r := range trimmed {
		if r < 0x20 || r == 0x7f {
			// An explicit correlation id never legitimately contains control
			// characters; drop the whole value rather than persist a mangled or
			// partially-injected identifier.
			return ""
		}
		count++
		if count > maxPersistedSessionIDLength {
			return ""
		}
	}
	return trimmed
}
