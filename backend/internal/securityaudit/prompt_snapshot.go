package securityaudit

import (
	"crypto/sha256"
	"encoding/hex"
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

var promptAuditClientWrapperTags = []string{
	"environment_context",
	"permission_profile",
	"system-reminder",
	"filesystem",
}

var promptAuditEnvironmentWrapperTags = []string{
	"environment_context",
	"permission_profile",
	"filesystem",
}

const promptAuditPrioritySeparator = "\x00SUB2API_PROMPT_AUDIT_PRIORITY_END\x00"

type promptSegment struct {
	text    string
	user    bool
	role    string
	current bool
}

type promptSelection string

const (
	promptSelectionUserOnly          promptSelection = "all_user_turns"
	promptSelectionCurrentUser       promptSelection = "current_user_turn"
	promptSelectionBlockingCandidate promptSelection = "blocking_receipt_candidates"
)

func ExtractPromptSnapshot(req Request) (PromptSnapshot, error) {
	snapshot, _, err := extractPromptSnapshotForSelection(req, promptSelectionUserOnly, ReviewModules{})
	return snapshot, err
}

// ExtractBlockingPromptSnapshot retains the latest-user-only compatibility
// helper. Runtime blocking selection uses configured modules through
// extractBlockingPromptSnapshotWithDiagnostics.
func ExtractBlockingPromptSnapshot(req Request, latestTurnOnly bool) (PromptSnapshot, error) {
	_ = latestTurnOnly
	snapshot, _, err := extractPromptSnapshotForSelection(req, promptSelectionCurrentUser, ReviewModules{})
	return snapshot, err
}

func extractBlockingPromptSnapshotWithDiagnostics(req Request, modules ReviewModules) (PromptSnapshot, promptExtractionDiagnostic, error) {
	return extractPromptSnapshotForSelection(req, promptSelectionBlockingCandidate, modules)
}

func extractPromptSnapshotWithDiagnostics(req Request, latestTurnOnly bool) (PromptSnapshot, promptExtractionDiagnostic, error) {
	selection := promptSelectionUserOnly
	modules := ReviewModules{}
	if latestTurnOnly {
		selection = promptSelectionCurrentUser
	}
	return extractPromptSnapshotForSelection(req, selection, modules)
}

func extractDeepPromptSnapshotWithDiagnostics(req Request, modules ReviewModules) (PromptSnapshot, promptExtractionDiagnostic, error) {
	return extractPromptSnapshotForSelection(req, promptSelectionUserOnly, modules)
}

func extractPromptSnapshotForSelection(req Request, selection promptSelection, modules ReviewModules) (PromptSnapshot, promptExtractionDiagnostic, error) {
	document, diagnostic, err := extractPromptDocument(req)
	if err != nil {
		return PromptSnapshot{}, diagnostic, err
	}
	snapshot := buildPromptSnapshot(req, document, selection, modules)
	completeContext, contextHash, contextBytes, segmentCount, contextErr := buildCompletePromptContext(req, document, diagnostic, snapshot, selection, modules)
	if contextErr != nil {
		return PromptSnapshot{}, diagnostic, contextErr
	}
	snapshot.CompleteContext = completeContext
	snapshot.FullContextHash = contextHash
	snapshot.FullContextBytes = contextBytes
	snapshot.FullContextSegmentCount = segmentCount
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

func buildPromptSnapshot(req Request, document auditcontent.Document, selection promptSelection, modules ReviewModules) PromptSnapshot {
	users := promptSegmentsFromAuditContent(document, req.Protocol, modules.System)
	segments := userPromptReviewSegments(users, selection)
	segments = append(segments, configuredPromptModuleSegments(document, modules)...)
	return promptSnapshotFromSegments(req, segments)
}

func promptSnapshotFromSegments(req Request, reviewSegments []PromptReviewSegment) PromptSnapshot {
	if len(reviewSegments) == 0 {
		return PromptSnapshot{}
	}
	texts := make([]string, 0, len(reviewSegments))
	messageCount := 0
	for _, segment := range reviewSegments {
		texts = append(texts, reviewSegmentParts(segment)...)
		messageCount += segment.Count
	}
	scanText, metadataText := buildPrioritizedScanText(texts)
	digest := sha256.Sum256([]byte(metadataText))
	fullPrompt := BuildFullPrompt(metadataText)
	sessionKey := req.SessionKey
	sessionSource := req.SessionSource
	if sessionKey == "" && req.RequestID != "" {
		sessionKey = HashSessionKey(req.UserID, req.Protocol, "request_id_fallback", req.RequestID)
		sessionSource = "request_id_fallback"
	}
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
		PromptLength: utf8.RuneCountInString(metadataText), MessageCount: messageCount, Stage: stage,
		SessionKey: sessionKey, SessionSource: sessionSource,
		ScanText: scanText, ReviewSegments: append([]PromptReviewSegment(nil), reviewSegments...), BodyBytes: len(req.Body),
		AllowReceiptWrite: req.AllowReceiptWrite,
	}
}

func promptSegmentsFromAuditContent(document auditcontent.Document, protocol string, keepSystemReminder bool) []promptSegment {
	allowRolelessMessage := promptAuditAllowsRolelessMessage(protocol)
	segments := make([]promptSegment, 0, len(document.Segments))
	for _, segment := range document.Segments {
		if !isPromptAuditConversationSegment(segment, allowRolelessMessage) {
			continue
		}
		role := segment.Role
		user := role == "user"
		if role == "" && ((segment.Source == auditcontent.SourceMessage && allowRolelessMessage) ||
			segment.Source == auditcontent.SourceSearchQuery ||
			segment.Source == auditcontent.SourceEmbeddingInput ||
			segment.Source == auditcontent.SourceMediaPrompt) {
			user = true
			role = "user"
		}
		segText := segment.Text
		if user {
			if keepSystemReminder {
				segText = stripPromptAuditWrapperBlocks(segText, promptAuditEnvironmentWrapperTags)
			} else {
				segText = stripPromptAuditClientWrapperBlocks(segText)
			}
			if segText == "" {
				continue
			}
		}
		segments = append(segments, promptSegment{
			text: segText, user: user, role: role, current: segment.Current,
		})
	}
	return segments
}

func configuredPromptModuleSegments(document auditcontent.Document, modules ReviewModules) []PromptReviewSegment {
	segments := make(map[string][]PromptReviewSegment)
	for _, segment := range document.Segments {
		if !segment.ClientControlled {
			continue
		}
		module := ""
		switch segment.Source {
		case auditcontent.SourceInstruction:
			if modules.System {
				module = "system"
			}
		case auditcontent.SourceMessage:
			role := strings.ToLower(strings.TrimSpace(segment.Role))
			if modules.Assistant && (role == "assistant" || role == "model") {
				module = "assistant"
			}
		case auditcontent.SourceReasoning:
			if modules.Reasoning {
				module = "reasoning"
			}
		case auditcontent.SourcePromptVariable:
			if modules.PromptVariables {
				module = "prompt_variables"
			}
		case auditcontent.SourceToolDefinition:
			if modules.ToolDefinitions {
				module = "tool_definitions"
			}
		case auditcontent.SourceToolCall:
			if modules.ToolCalls {
				module = "tool_calls"
			}
		case auditcontent.SourceToolOutput:
			if modules.ToolOutputs {
				module = "tool_outputs"
			}
		}
		if module != "" && strings.TrimSpace(segment.Text) != "" {
			text := strings.TrimSpace(segment.Text)
			segments[module] = append(segments[module], PromptReviewSegment{
				Source: module, Text: text, Parts: []string{text}, Count: 1,
			})
		}
	}
	order := []string{"system", "assistant", "reasoning", "prompt_variables", "tool_definitions", "tool_calls", "tool_outputs"}
	result := make([]PromptReviewSegment, 0, len(document.Segments))
	for _, module := range order {
		result = append(result, segments[module]...)
	}
	return result
}

func isPromptAuditConversationSegment(segment auditcontent.Segment, allowRolelessMessage bool) bool {
	switch segment.Source {
	case auditcontent.SourceSearchQuery, auditcontent.SourceEmbeddingInput, auditcontent.SourceMediaPrompt:
		return true
	case auditcontent.SourceMessage:
		// Keep assistant/model messages as turn separators. They are not
		// emitted as user input and optional module selection owns their text.
		return isPromptAuditUserRole(segment.Role, allowRolelessMessage) ||
			strings.EqualFold(segment.Role, "assistant") || strings.EqualFold(segment.Role, "model")
	default:
		return false
	}
}

func isPromptAuditUserRole(role string, allowRoleless bool) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user":
		return true
	case "":
		return allowRoleless
	default:
		return false
	}
}

func promptAuditAllowsRolelessMessage(protocol string) bool {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "openai_responses", "openai_live", "gemini":
		return true
	default:
		return false
	}
}

// DefaultPromptPreviewMaxRunes caps how much sanitized prompt text may be
// considered before BuildPromptPreview withholds the majority for storage/UI.
const DefaultPromptPreviewMaxRunes = 96

func userPromptReviewSegments(values []promptSegment, selection promptSelection) []PromptReviewSegment {
	normalized := normalizedPromptSegments(values)
	if len(normalized) == 0 {
		return nil
	}
	turns := make([]PromptReviewSegment, 0)
	for index := 0; index < len(normalized); {
		if !isUserSegment(normalized[index]) {
			index++
			continue
		}
		end := index
		texts := make([]string, 0, 1)
		current := normalized[index].current
		for end < len(normalized) && isUserSegment(normalized[end]) && normalized[end].current == current {
			texts = append(texts, normalized[end].text)
			end++
		}
		turns = append(turns, PromptReviewSegment{
			Source: "user", Text: strings.Join(texts, "\n\n"), Parts: append([]string(nil), texts...), CurrentUser: current, Count: end - index,
		})
		index = end
	}
	if len(turns) == 0 {
		return nil
	}
	priority := len(turns) - 1
	for index := len(turns) - 1; index >= 0; index-- {
		if turns[index].CurrentUser {
			priority = index
			break
		}
	}
	if selection == promptSelectionCurrentUser {
		if turns[priority].CurrentUser {
			current := turns[priority]
			current.Count = 1
			current.CombineParts = true
			return []PromptReviewSegment{current}
		}
		return nil
	}
	if selection == promptSelectionUserOnly && len(turns[priority].Parts) > 1 {
		parts := turns[priority].Parts
		turns[priority].Parts = append([]string{parts[len(parts)-1]}, parts[:len(parts)-1]...)
	}
	result := make([]PromptReviewSegment, 0, len(turns))
	result = append(result, turns[priority])
	result = append(result, turns[:priority]...)
	result = append(result, turns[priority+1:]...)
	if selection == promptSelectionBlockingCandidate && result[0].CurrentUser {
		result[0].Count = 1
		result[0].CombineParts = true
	}
	return result
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

func isUserSegment(segment promptSegment) bool {
	return segment.user || segment.role == "user"
}

func reviewSegmentParts(segment PromptReviewSegment) []string {
	if len(segment.Parts) > 0 {
		if segment.CombineParts {
			return []string{strings.Join(segment.Parts, "\n\n")}
		}
		return segment.Parts
	}
	if strings.TrimSpace(segment.Text) == "" {
		return nil
	}
	return []string{segment.Text}
}

// stripPromptAuditClientWrapperBlocks removes client harness XML from user
// text while keeping the surrounding user-authored sentences. Whole blocks
// are dropped; leftover wrapper-only segments become empty and are omitted.
func stripPromptAuditClientWrapperBlocks(text string) string {
	return stripPromptAuditWrapperBlocks(text, promptAuditClientWrapperTags)
}

func stripPromptAuditWrapperBlocks(text string, tags []string) string {
	text = strings.TrimSpace(text)
	if text == "" || !strings.Contains(text, "<") {
		return text
	}
	stripped := text
	for {
		next := stripOnePromptAuditClientWrapperBlock(stripped, tags)
		if next == stripped {
			break
		}
		stripped = next
	}
	stripped = strings.ReplaceAll(stripped, "\r\n", "\n")
	for strings.Contains(stripped, "\n\n\n") {
		stripped = strings.ReplaceAll(stripped, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(stripped)
}

func stripOnePromptAuditClientWrapperBlock(text string, tags []string) string {
	lower := strings.ToLower(text)
	bestStart, bestEnd := -1, -1
	for _, name := range tags {
		openToken := "<" + name
		searchFrom := 0
		for {
			openRel := strings.Index(lower[searchFrom:], openToken)
			if openRel < 0 {
				break
			}
			openAt := searchFrom + openRel
			afterName := openAt + len(openToken)
			if afterName < len(lower) {
				next := lower[afterName]
				if next != '>' && next != '/' && next != ' ' && next != '\t' && next != '\n' && next != '\r' {
					searchFrom = afterName
					continue
				}
			}
			gt := strings.Index(text[openAt:], ">")
			if gt < 0 {
				break
			}
			tagEnd := openAt + gt + 1
			rawTag := text[openAt:tagEnd]
			if strings.HasSuffix(strings.TrimSpace(rawTag), "/>") {
				if bestStart < 0 || openAt < bestStart {
					bestStart, bestEnd = openAt, tagEnd
				}
				break
			}
			closeToken := "</" + name
			closeRel := strings.Index(lower[tagEnd:], closeToken)
			if closeRel < 0 {
				if bestStart < 0 || openAt < bestStart {
					bestStart, bestEnd = openAt, len(text)
				}
				break
			}
			closeAt := tagEnd + closeRel
			closeGt := strings.Index(text[closeAt:], ">")
			if closeGt < 0 {
				break
			}
			end := closeAt + closeGt + 1
			if bestStart < 0 || openAt < bestStart {
				bestStart, bestEnd = openAt, end
			}
			break
		}
	}
	if bestStart < 0 {
		return text
	}
	return strings.TrimSpace(text[:bestStart]) + "\n\n" + strings.TrimSpace(text[bestEnd:])
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
