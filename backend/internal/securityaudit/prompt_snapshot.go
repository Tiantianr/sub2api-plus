package securityaudit

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/LuckyKuang/sub2api-plus/internal/auditcontent"
)

var (
	ErrNoPromptText = errors.New("prompt audit request contains no user text")
)

type promptExtractionDiagnostic struct {
	Failed    bool
	ErrorCode string
	Reasons   []auditcontent.IncompleteReason
}

var (
	bearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+\-/]+=*`)
	apiKeyPattern = regexp.MustCompile(`(?i)\b(sk|rk|pk|api[_-]?key|token|secret|password)[-_:=\s]+[A-Za-z0-9._~+\-/]{8,}`)
	canaryPattern = regexp.MustCompile(`(?i)([A-Z]+_CANARY_)[A-Za-z0-9_-]+`)
	emailPattern  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phonePattern  = regexp.MustCompile(`(?:\+?\d[\d\s().-]{8,}\d)`)
)

const promptAuditPrioritySeparator = "\x00SUB2API_PROMPT_AUDIT_PRIORITY_END\x00"

type promptSegment struct {
	text string
	user bool
	role string
}

type conversationFingerprint struct {
	Hash  string
	Count int
}

func ExtractPromptSnapshot(req Request) (PromptSnapshot, error) {
	snapshot, _, err := extractPromptSnapshotWithDiagnostics(req, false)
	return snapshot, err
}

// ExtractBlockingPromptSnapshot builds the synchronous guard input. The caller
// chooses between the latest user turn and all canonical conversation text.
func ExtractBlockingPromptSnapshot(req Request, latestTurnOnly bool) (PromptSnapshot, error) {
	snapshot, _, err := extractPromptSnapshotWithDiagnostics(req, latestTurnOnly)
	return snapshot, err
}

func extractPromptSnapshotWithDiagnostics(req Request, latestTurnOnly bool) (PromptSnapshot, promptExtractionDiagnostic, error) {
	document, diagnostic, err := extractPromptDocument(req)
	if err != nil {
		return PromptSnapshot{}, diagnostic, err
	}
	snapshot := buildPromptSnapshot(req, document, latestTurnOnly)
	if strings.TrimSpace(snapshot.ScanText) == "" {
		return PromptSnapshot{}, diagnostic, ErrNoPromptText
	}
	return snapshot, diagnostic, nil
}

func extractPromptDocument(req Request) (auditcontent.Document, promptExtractionDiagnostic, error) {
	document, err := auditcontent.Extract(req.Protocol, req.Body)
	if err != nil {
		return auditcontent.Document{}, promptExtractionDiagnostic{Failed: true, ErrorCode: "invalid_json"}, errors.New("prompt audit request JSON is invalid")
	}
	diagnostic := promptExtractionDiagnostic{}
	if document.Incomplete {
		diagnostic = promptExtractionDiagnostic{
			Failed: true, ErrorCode: "incomplete_content",
			Reasons: auditcontent.SanitizeIncompleteReasons(document.IncompleteReasons),
		}
	}
	return document, diagnostic, nil
}

func buildPromptSnapshot(req Request, document auditcontent.Document, latestTurnOnly bool) PromptSnapshot {
	extracted := promptSegmentsFromAuditContent(document, latestTurnOnly)
	var segments []string
	if latestTurnOnly {
		segments = blockingSegmentsLatestUser(extracted)
	} else {
		segments = normalizeSegmentsLatestUserFirst(extracted)
	}
	return promptSnapshotFromSegments(req, segments)
}

func buildConversationIncrementalSnapshot(req Request, document auditcontent.Document, previousOutput string) PromptSnapshot {
	segments := normalizeSegmentsLatestUserFirst(currentPromptSegmentsFromAuditContent(document))
	if previousOutput = strings.TrimSpace(previousOutput); previousOutput != "" {
		segments = append(segments, previousOutput)
	}
	return promptSnapshotFromSegments(req, segments)
}

func promptSnapshotFromSegments(req Request, segments []string) PromptSnapshot {
	if len(segments) == 0 {
		return PromptSnapshot{}
	}
	scanText, metadataText := buildPrioritizedScanText(segments)
	digest := sha256.Sum256([]byte(metadataText))
	fullPrompt := BuildFullPrompt(metadataText)
	stage := strings.TrimSpace(req.Stage)
	if stage == "" {
		stage = "http"
	}
	return PromptSnapshot{
		RequestID: req.RequestID, ClientIP: normalizePromptClientIP(req.ClientIP), UserID: req.UserID, UsernameSnapshot: req.Username,
		UserEmailSnapshot: req.UserEmail, APIKeyID: req.APIKeyID, APIKeyNameSnapshot: req.APIKeyName,
		GroupID: cloneInt64Ptr(req.GroupID), GroupName: req.GroupName, Provider: req.Provider,
		Endpoint: req.Endpoint, Protocol: req.Protocol, Model: req.Model,
		PromptHash: hex.EncodeToString(digest[:]), RedactedPreview: BuildPromptPreview(metadataText, DefaultPromptPreviewMaxRunes),
		FullPrompt: fullPrompt, FullPromptTruncated: utf8.RuneCountInString(fullPrompt) < utf8.RuneCountInString(metadataText),
		PromptLength: utf8.RuneCountInString(metadataText), MessageCount: len(segments), Stage: stage,
		ScanText: scanText, BodyBytes: len(req.Body),
	}
}

func promptSegmentsFromAuditContent(document auditcontent.Document, latestTurnOnly bool) []promptSegment {
	segments := make([]promptSegment, 0, len(document.Segments))
	for _, segment := range document.Segments {
		if !isPromptAuditConversationSegment(segment, latestTurnOnly) {
			continue
		}
		role := segment.Role
		user := role == "user"
		if role == "" && (segment.Source == auditcontent.SourceMessage ||
			segment.Source == auditcontent.SourceSearchQuery ||
			segment.Source == auditcontent.SourceEmbeddingInput ||
			segment.Source == auditcontent.SourceMediaPrompt) {
			user = true
			role = "user"
		}
		segments = append(segments, promptSegment{
			text: segment.Text,
			user: user,
			role: role,
		})
	}
	return segments
}

func currentPromptSegmentsFromAuditContent(document auditcontent.Document) []promptSegment {
	segments := make([]promptSegment, 0, len(document.Segments))
	for _, segment := range document.Segments {
		if !segment.Current || !segment.ClientControlled {
			continue
		}
		switch segment.Source {
		case auditcontent.SourceInstruction, auditcontent.SourceToolDefinition:
			continue
		}
		role := strings.ToLower(strings.TrimSpace(segment.Role))
		segments = append(segments, promptSegment{
			text: segment.Text,
			user: role == "user" || role == "",
			role: role,
		})
	}
	return segments
}

func conversationContextHash(req Request, document auditcontent.Document) (string, bool) {
	var value bytes.Buffer
	// A request without a continuation alias is a complete client replay, so
	// absence of system/tool context is itself meaningful. Server-side
	// continuations may omit unchanged context and inherit the prior hash.
	hasContext := strings.TrimSpace(req.ParentID) == ""
	_, _ = value.WriteString("protocol=")
	_, _ = value.WriteString(strings.ToLower(strings.TrimSpace(req.Protocol)))
	for _, segment := range document.Segments {
		if segment.Source != auditcontent.SourceInstruction && segment.Source != auditcontent.SourceToolDefinition {
			continue
		}
		hasContext = true
		_, _ = value.WriteString("\nsource=")
		_, _ = value.WriteString(string(segment.Source))
		_, _ = value.WriteString("\nrole=")
		_, _ = value.WriteString(strings.ToLower(strings.TrimSpace(segment.Role)))
		_, _ = value.WriteString("\ntext=")
		_, _ = value.WriteString(segment.Text)
	}
	digest := sha256.Sum256(value.Bytes())
	return hex.EncodeToString(digest[:]), hasContext
}

func conversationInputFingerprint(document auditcontent.Document) conversationFingerprint {
	return fingerprintConversationTexts(conversationDocumentTexts(document, false))
}

func conversationHistoryMatches(checkpoint conversationCheckpoint, document auditcontent.Document) bool {
	if checkpoint.Input.Hash == "" || checkpoint.Input.Count < 0 || checkpoint.OutputDigest.Hash == "" || checkpoint.OutputDigest.Count < 0 {
		return false
	}
	history := conversationDocumentTexts(document, true)
	if checkpoint.Input.Count > len(history) || checkpoint.OutputDigest.Count > len(history)-checkpoint.Input.Count ||
		len(history) != checkpoint.Input.Count+checkpoint.OutputDigest.Count {
		return false
	}
	input := fingerprintConversationTexts(history[:checkpoint.Input.Count])
	output := fingerprintConversationTexts(history[checkpoint.Input.Count:])
	return input == checkpoint.Input && output == checkpoint.OutputDigest
}

func conversationDocumentTexts(document auditcontent.Document, historicalOnly bool) []string {
	texts := make([]string, 0, len(document.Segments))
	for _, segment := range document.Segments {
		if historicalOnly && segment.Current {
			continue
		}
		if segment.Source == auditcontent.SourceInstruction || segment.Source == auditcontent.SourceToolDefinition {
			continue
		}
		if !isPromptAuditConversationSegment(segment, false) {
			continue
		}
		if text := strings.TrimSpace(segment.Text); text != "" {
			texts = append(texts, text)
		}
	}
	return texts
}

func fingerprintConversationTexts(texts []string) conversationFingerprint {
	raw, _ := json.Marshal(texts)
	digest := sha256.Sum256(raw)
	return conversationFingerprint{Hash: hex.EncodeToString(digest[:]), Count: len(texts)}
}

func isPromptAuditConversationSegment(segment auditcontent.Segment, latestTurnOnly bool) bool {
	if latestTurnOnly {
		switch segment.Source {
		case auditcontent.SourceMessage:
			// Keep assistant/model messages as turn separators so older user
			// text is not joined with the latest user turn. They are not emitted.
			return true
		case auditcontent.SourceSearchQuery, auditcontent.SourceEmbeddingInput, auditcontent.SourceMediaPrompt:
			return true
		default:
			return false
		}
	}
	switch segment.Source {
	case auditcontent.SourceMessage, auditcontent.SourceInstruction,
		auditcontent.SourceSearchQuery, auditcontent.SourceEmbeddingInput,
		auditcontent.SourceMediaPrompt, auditcontent.SourcePromptVariable,
		auditcontent.SourceReasoning, auditcontent.SourceToolDefinition,
		auditcontent.SourceToolCall, auditcontent.SourceToolOutput:
		return true
	default:
		return false
	}
}

// DefaultPromptPreviewMaxRunes caps how much sanitized prompt text may be
// considered before BuildPromptPreview withholds the majority for storage/UI.
const DefaultPromptPreviewMaxRunes = 96

func normalizeSegmentsLatestUserFirst(values []promptSegment) []string {
	normalized := normalizedPromptSegments(values)
	if len(normalized) == 0 {
		return nil
	}
	priorityIndex := len(normalized) - 1
	for index := len(normalized) - 1; index >= 0; index-- {
		if isUserSegment(normalized[index]) {
			priorityIndex = index
			break
		}
	}
	result := make([]string, 0, len(normalized))
	result = append(result, normalized[priorityIndex].text)
	for index, segment := range normalized {
		if index != priorityIndex {
			result = append(result, segment.text)
		}
	}
	return result
}

// blockingSegmentsLatestUser limits synchronous guard input to the current
// user turn. Instructions, previous assistant/model output, and older user
// messages stay out of blocking so client harness text cannot trip the guard.
func blockingSegmentsLatestUser(values []promptSegment) []string {
	normalized := normalizedPromptSegments(values)
	latestUserStart := latestUserSegmentStart(normalized)
	if latestUserStart < 0 {
		return nil
	}
	latestUserEnd := latestUserStart
	for latestUserEnd < len(normalized) && isUserSegment(normalized[latestUserEnd]) {
		latestUserEnd++
	}
	currentUserText := make([]string, 0, latestUserEnd-latestUserStart)
	for _, segment := range normalized[latestUserStart:latestUserEnd] {
		currentUserText = append(currentUserText, segment.text)
	}
	return []string{strings.Join(currentUserText, "\n\n")}
}

func normalizedPromptSegments(values []promptSegment) []promptSegment {
	normalized := make([]promptSegment, 0, len(values))
	for _, value := range values {
		value.text = strings.TrimSpace(value.text)
		if value.text != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func latestUserSegmentStart(values []promptSegment) int {
	latest := -1
	for index := len(values) - 1; index >= 0; index-- {
		if isUserSegment(values[index]) {
			latest = index
			break
		}
	}
	for latest > 0 && isUserSegment(values[latest-1]) {
		latest--
	}
	return latest
}

func isUserSegment(segment promptSegment) bool {
	return segment.user || segment.role == "user"
}

func buildPrioritizedScanText(segments []string) (scanText string, metadataText string) {
	metadataText = strings.Join(segments, "\n\n")
	if len(segments) <= 1 {
		return metadataText, metadataText
	}
	return segments[0] + promptAuditPrioritySeparator + strings.Join(segments[1:], "\n\n"), metadataText
}

func RedactPreview(value string, maxRunes int) string {
	value = bearerPattern.ReplaceAllString(value, "Bearer ***")
	value = apiKeyPattern.ReplaceAllStringFunc(value, func(match string) string {
		if index := strings.IndexAny(match, ":= \t"); index >= 0 {
			return match[:index+1] + "***"
		}
		return "***"
	})
	value = canaryPattern.ReplaceAllString(value, "${1}***")
	value = emailPattern.ReplaceAllString(value, "***@***")
	value = phonePattern.ReplaceAllString(value, "***PHONE***")
	return TrimRunes(value, maxRunes)
}

// BuildPromptPreview stores only a short, non-recoverable head of sanitized
// input. Ordinary confidential prompts must not land nearly intact in PostgreSQL
// or the admin UI merely because no secret regex matched.
func BuildPromptPreview(value string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = DefaultPromptPreviewMaxRunes
	}
	redacted := strings.TrimSpace(RedactPreview(value, maxRunes))
	if redacted == "" {
		return ""
	}
	runes := []rune(redacted)
	hadTruncation := strings.HasSuffix(redacted, "…")
	if hadTruncation && len(runes) > 0 {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return "***…"
	}
	const minLengthForPartialPreview = 32
	if len(runes) < minLengthForPartialPreview {
		if hadTruncation {
			return "***…"
		}
		return "***"
	}
	keep := len(runes) / 4
	if keep > 24 {
		keep = 24
	}
	preview := string(runes[:keep]) + "***"
	if hadTruncation || keep < len(runes) {
		preview += "…"
	}
	return preview
}

// BuildFullPrompt returns the complete prompt text for admin-only event review.
// NUL bytes are stripped because PostgreSQL TEXT rejects them.
func BuildFullPrompt(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	return strings.TrimSpace(value)
}

// FullPromptFromScanText reconstructs display text from the worker payload.
func FullPromptFromScanText(scanText string) string {
	return BuildFullPrompt(strings.ReplaceAll(scanText, promptAuditPrioritySeparator, "\n\n"))
}

func normalizePromptClientIP(value string) string {
	parsed := net.ParseIP(strings.TrimSpace(value))
	if parsed == nil {
		return ""
	}
	return parsed.String()
}

func TrimRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
