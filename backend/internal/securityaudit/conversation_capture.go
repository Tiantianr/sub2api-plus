package securityaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
)

const maxConversationOutputBytes = MaxInputLimit*utf8.UTFMax + 64*1024

// ConversationCapture owns one audited turn from Guard allow through the
// final downstream response. It stores only bounded, media-sanitized output;
// any ambiguous lifecycle leaves the already-written FULL_REQUIRED state in
// place.
type ConversationCapture struct {
	store         *RedisConversationStore
	lease         *conversationLease
	protocol      string
	configVersion int64
	contextHash   string
	inputDigest   conversationFingerprint
	encrypt       func(string) (string, error)

	mu       sync.Mutex
	buffer   bytes.Buffer
	framed   bool
	overflow bool
	done     atomic.Bool
}

func newConversationCapture(store *RedisConversationStore, lease *conversationLease, protocol string, configVersion int64, contextHash string, inputDigest conversationFingerprint, encrypt func(string) (string, error)) *ConversationCapture {
	return &ConversationCapture{
		store: store, lease: lease, protocol: strings.TrimSpace(protocol),
		configVersion: configVersion, contextHash: contextHash, inputDigest: inputDigest,
		encrypt: encrypt,
	}
}

// ObserveHTTP records exact bytes after all downstream protocol conversion,
// model rewriting, tool-name restoration, and error sanitization.
func (c *ConversationCapture) ObserveHTTP(payload []byte) {
	c.observe(payload, false)
}

// ObserveFrame records one final WebSocket frame. A newline separates frames;
// JSON strings carry embedded newlines escaped, so framing stays unambiguous.
func (c *ConversationCapture) ObserveFrame(payload []byte) {
	c.observe(payload, true)
}

func (c *ConversationCapture) observe(payload []byte, framed bool) {
	if c == nil || c.done.Load() || len(payload) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done.Load() || c.overflow {
		return
	}
	extra := len(payload)
	if framed && c.buffer.Len() > 0 {
		extra++
	}
	if c.buffer.Len()+extra > maxConversationOutputBytes {
		c.overflow = true
		c.buffer.Reset()
		return
	}
	if framed && c.buffer.Len() > 0 {
		_ = c.buffer.WriteByte('\n')
	}
	_, _ = c.buffer.Write(payload)
	c.framed = c.framed || framed
}

func (c *ConversationCapture) FinishHTTP(status int, contentType string) {
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		c.Abort()
		return
	}
	c.finish(status, contentType, "", false)
}

func (c *ConversationCapture) FinishTurn(responseID string) {
	c.finish(http.StatusOK, "application/json", responseID, true)
}

// CompleteWithoutOutput advances a successfully audited turn whose transport
// has no assistant text, and optionally binds an opaque continuation alias
// such as a Live call id.
func (c *ConversationCapture) CompleteWithoutOutput(parentAlias string) {
	if !c.markDone() {
		return
	}
	ctx, cancel := conversationStoreContext()
	defer cancel()
	if c.encrypt == nil {
		c.failAfterCommitError()
		return
	}
	emptyCiphertext, encryptErr := c.encrypt("")
	if encryptErr != nil {
		c.failAfterCommitError()
		return
	}
	if err := c.store.Commit(ctx, c.lease, c.configVersion, c.contextHash, emptyCiphertext, parentAlias, c.inputDigest, fingerprintConversationTexts(nil)); err != nil {
		c.failAfterCommitError()
	}
}

func (c *ConversationCapture) finish(status int, contentType, fallbackResponseID string, forceFramed bool) {
	if !c.markDone() {
		return
	}
	c.mu.Lock()
	raw := append([]byte(nil), c.buffer.Bytes()...)
	framed := c.framed || forceFramed
	overflow := c.overflow
	c.buffer.Reset()
	c.mu.Unlock()
	if overflow {
		c.failAfterCommitError()
		return
	}
	output, responseID, outputDigest, err := normalizeConversationOutputWithFingerprint(c.protocol, raw, contentType, framed)
	if err != nil || utf8.RuneCountInString(output) > MaxInputLimit {
		c.failAfterCommitError()
		return
	}
	if responseID == "" {
		responseID = strings.TrimSpace(fallbackResponseID)
	}
	if c.encrypt == nil {
		c.failAfterCommitError()
		return
	}
	outputCiphertext, encryptErr := c.encrypt(output)
	if encryptErr != nil {
		c.failAfterCommitError()
		return
	}
	ctx, cancel := conversationStoreContext()
	defer cancel()
	if err := c.store.Commit(ctx, c.lease, c.configVersion, c.contextHash, outputCiphertext, responseID, c.inputDigest, outputDigest); err != nil {
		c.failAfterCommitError()
	}
}

// Abort preserves FULL_REQUIRED and releases this turn's lease when possible.
func (c *ConversationCapture) Abort() {
	if !c.markDone() {
		return
	}
	c.failAfterCommitError()
}

func (c *ConversationCapture) markDone() bool {
	if c == nil || !c.done.CompareAndSwap(false, true) {
		return false
	}
	return true
}

func (c *ConversationCapture) failAfterCommitError() {
	if c == nil || c.store == nil || c.lease == nil {
		return
	}
	ctx, cancel := conversationStoreContext()
	defer cancel()
	_ = c.store.Fail(ctx, c.lease)
}

func conversationStoreContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

type capturedOutputFrame struct {
	eventType string
	value     any
	raw       []byte
}

func normalizeConversationOutput(protocol string, raw []byte, contentType string, framed bool) (string, string, error) {
	output, responseID, _, err := normalizeConversationOutputWithFingerprint(protocol, raw, contentType, framed)
	return output, responseID, err
}

func normalizeConversationOutputWithFingerprint(protocol string, raw []byte, contentType string, framed bool) (string, string, conversationFingerprint, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", "", conversationFingerprint{}, errors.New("conversation output is empty")
	}
	stream := framed || strings.Contains(strings.ToLower(contentType), "text/event-stream") || looksLikeSSE(raw)
	if !stream {
		values, err := decodeCapturedJSONValues(raw)
		if err != nil || len(values) != 1 {
			return "", "", conversationFingerprint{}, errors.New("conversation output is not one JSON value")
		}
		if outputFrameFailed("", values[0]) {
			return "", "", conversationFingerprint{}, errors.New("conversation output is an error response")
		}
		text, keep := auditcontent.StructuredText(values[0])
		if !keep {
			text = ""
		}
		texts, ok := capturedCanonicalOutputTexts(protocol, values[0])
		if !ok {
			return "", "", conversationFingerprint{}, errors.New("conversation output shape is not canonical")
		}
		return text, capturedResponseID(values[0]), fingerprintConversationTexts(texts), nil
	}

	frames, doneMarker, err := capturedStreamFrames(raw, framed)
	if err != nil {
		return "", "", conversationFingerprint{}, err
	}
	terminalSuccess := doneMarker
	responseID := ""
	parts := make([]string, 0, len(frames))
	terminalAggregate := ""
	var terminalValue any
	for _, frame := range frames {
		if id := capturedResponseID(frame.value); id != "" {
			responseID = id
		}
		if outputFrameFailed(frame.eventType, frame.value) {
			return "", "", conversationFingerprint{}, errors.New("conversation stream ended with failure")
		}
		if outputFrameSucceeded(frame.eventType, frame.value) {
			terminalSuccess = true
		}
		text, keep := auditcontent.StructuredText(frame.value)
		if !keep {
			continue
		}
		if outputFrameSucceeded(frame.eventType, frame.value) && terminalFrameHasAggregate(frame.value) {
			terminalAggregate = text
			terminalValue = frame.value
			continue
		}
		if outputFrameCarriesContent(frame.eventType, frame.value) {
			parts = append(parts, text)
		}
	}
	if !terminalSuccess {
		return "", "", conversationFingerprint{}, errors.New("conversation stream has no successful terminal event")
	}
	if terminalAggregate != "" {
		texts, ok := capturedCanonicalOutputTexts(protocol, terminalValue)
		if !ok {
			return "", "", conversationFingerprint{}, errors.New("conversation terminal output is not canonical")
		}
		return terminalAggregate, responseID, fingerprintConversationTexts(texts), nil
	}
	texts, ok := capturedDeltaOutputTexts(protocol, frames)
	if !ok {
		return "", "", conversationFingerprint{}, errors.New("conversation stream deltas are not canonical")
	}
	return strings.Join(parts, "\n"), responseID, fingerprintConversationTexts(texts), nil
}

func capturedCanonicalOutputTexts(protocol string, value any) ([]string, bool) {
	root, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if response, exists := root["response"].(map[string]any); exists {
		root = response
	}
	if output, exists := root["output"]; exists {
		body, err := json.Marshal(map[string]any{"input": output})
		if err != nil {
			return nil, false
		}
		document, err := auditcontent.Extract("openai_responses", body)
		if err != nil || document.Incomplete {
			return nil, false
		}
		return conversationDocumentTexts(document, false), true
	}
	if choices, exists := root["choices"].([]any); exists {
		messages := make([]any, 0, len(choices))
		for _, rawChoice := range choices {
			choice, _ := rawChoice.(map[string]any)
			message, _ := choice["message"].(map[string]any)
			if message == nil {
				return nil, false
			}
			messages = append(messages, message)
		}
		body, err := json.Marshal(map[string]any{"messages": messages})
		if err != nil {
			return nil, false
		}
		document, err := auditcontent.Extract("openai_chat_completions", body)
		if err != nil || document.Incomplete {
			return nil, false
		}
		return conversationDocumentTexts(document, false), true
	}
	if content, exists := root["content"]; exists && (protocol == "anthropic_messages" || protocol == "messages") {
		body, err := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "assistant", "content": content}}})
		if err != nil {
			return nil, false
		}
		document, err := auditcontent.Extract("anthropic_messages", body)
		if err != nil || document.Incomplete {
			return nil, false
		}
		return conversationDocumentTexts(document, false), true
	}
	return nil, false
}

func capturedDeltaOutputTexts(protocol string, frames []capturedOutputFrame) ([]string, bool) {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	switch protocol {
	case "openai_chat_completions", "openai_chat", "chat_completions":
		var text strings.Builder
		for _, frame := range frames {
			root, _ := frame.value.(map[string]any)
			choices, _ := root["choices"].([]any)
			for _, rawChoice := range choices {
				choice, _ := rawChoice.(map[string]any)
				delta, _ := choice["delta"].(map[string]any)
				if hasNonEmptyCapture(delta["tool_calls"]) || hasNonEmptyCapture(delta["function_call"]) || hasNonEmptyCapture(delta["reasoning_content"]) {
					return nil, false
				}
				if value, ok := delta["content"].(string); ok {
					_, _ = text.WriteString(value)
				}
			}
		}
		if text.Len() == 0 {
			return nil, true
		}
		return []string{text.String()}, true
	case "anthropic_messages", "messages":
		var text strings.Builder
		for _, frame := range frames {
			root, _ := frame.value.(map[string]any)
			delta, _ := root["delta"].(map[string]any)
			if hasNonEmptyCapture(delta["thinking"]) || hasNonEmptyCapture(delta["partial_json"]) {
				return nil, false
			}
			if value, ok := delta["text"].(string); ok {
				_, _ = text.WriteString(value)
			}
		}
		if text.Len() == 0 {
			return nil, true
		}
		return []string{text.String()}, true
	default:
		return nil, false
	}
}

func capturedStreamFrames(raw []byte, framed bool) ([]capturedOutputFrame, bool, error) {
	if framed {
		lines := bytes.Split(raw, []byte{'\n'})
		frames := make([]capturedOutputFrame, 0, len(lines))
		for _, line := range lines {
			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			values, err := decodeCapturedJSONValues(line)
			if err != nil || len(values) != 1 {
				return nil, false, errors.New("conversation websocket frame is invalid JSON")
			}
			frames = append(frames, capturedOutputFrame{eventType: capturedEventType(values[0]), value: values[0], raw: append([]byte(nil), line...)})
		}
		return frames, false, nil
	}

	frames := make([]capturedOutputFrame, 0, 16)
	done := false
	var eventType string
	var dataLines []string
	flush := func() error {
		if len(dataLines) == 0 {
			eventType = ""
			return nil
		}
		data := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if data == "[DONE]" {
			done = true
			eventType = ""
			return nil
		}
		values, err := decodeCapturedJSONValues([]byte(data))
		if err != nil {
			return err
		}
		for _, value := range values {
			effectiveType := strings.TrimSpace(eventType)
			if effectiveType == "" {
				effectiveType = capturedEventType(value)
			}
			frames = append(frames, capturedOutputFrame{eventType: effectiveType, value: value, raw: []byte(data)})
		}
		eventType = ""
		return nil
	}
	for _, rawLine := range strings.Split(string(raw), "\n") {
		line := strings.TrimRight(rawLine, "\r")
		if line == "" {
			if err := flush(); err != nil {
				return nil, false, err
			}
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := flush(); err != nil {
		return nil, false, err
	}
	return frames, done, nil
}

func decodeCapturedJSONValues(raw []byte) ([]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	values := make([]any, 0, 1)
	for {
		var value any
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
}

func looksLikeSSE(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return bytes.HasPrefix(trimmed, []byte("data:")) || bytes.HasPrefix(trimmed, []byte("event:"))
}

func capturedEventType(value any) string {
	root, _ := value.(map[string]any)
	typeName, _ := root["type"].(string)
	return strings.ToLower(strings.TrimSpace(typeName))
}

func outputFrameSucceeded(eventType string, value any) bool {
	eventType = strings.ToLower(strings.TrimSpace(firstNonEmptyCapture(eventType, capturedEventType(value))))
	switch eventType {
	case "response.completed", "response.done", "message_stop":
		return true
	}
	root, _ := value.(map[string]any)
	if status, _ := root["status"].(string); strings.EqualFold(strings.TrimSpace(status), "completed") {
		return true
	}
	response, _ := root["response"].(map[string]any)
	status, _ := response["status"].(string)
	return strings.EqualFold(strings.TrimSpace(status), "completed")
}

func outputFrameFailed(eventType string, value any) bool {
	eventType = strings.ToLower(strings.TrimSpace(firstNonEmptyCapture(eventType, capturedEventType(value))))
	switch eventType {
	case "error", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	}
	root, _ := value.(map[string]any)
	if hasNonEmptyCapture(root["error"]) {
		return true
	}
	if status, _ := root["status"].(string); status != "" {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "failed", "incomplete", "cancelled", "canceled":
			return true
		}
	}
	response, _ := root["response"].(map[string]any)
	status, _ := response["status"].(string)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "incomplete", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func outputFrameCarriesContent(eventType string, value any) bool {
	eventType = strings.ToLower(strings.TrimSpace(firstNonEmptyCapture(eventType, capturedEventType(value))))
	for _, marker := range []string{"output_text", "refusal", "reasoning", "function_call", "tool_call", "content_block", "transcript"} {
		if strings.Contains(eventType, marker) {
			return true
		}
	}
	root, _ := value.(map[string]any)
	for _, key := range []string{"choices", "content", "output", "candidates", "item", "part", "delta"} {
		if hasNonEmptyCapture(root[key]) {
			return true
		}
	}
	response, _ := root["response"].(map[string]any)
	return hasNonEmptyCapture(response["output"]) || hasNonEmptyCapture(response["content"])
}

func terminalFrameHasAggregate(value any) bool {
	root, _ := value.(map[string]any)
	if hasNonEmptyCapture(root["output"]) || hasNonEmptyCapture(root["content"]) || hasNonEmptyCapture(root["choices"]) || hasNonEmptyCapture(root["candidates"]) {
		return true
	}
	response, _ := root["response"].(map[string]any)
	return hasNonEmptyCapture(response["output"]) || hasNonEmptyCapture(response["content"])
}

func capturedResponseID(value any) string {
	root, _ := value.(map[string]any)
	if response, ok := root["response"].(map[string]any); ok {
		if id, ok := response["id"].(string); ok && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	if id, ok := root["id"].(string); ok {
		return strings.TrimSpace(id)
	}
	return ""
}

func hasNonEmptyCapture(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func firstNonEmptyCapture(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
