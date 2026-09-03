## Context

Prompt Audit and Content Moderation share the unified security audit coordinator
at gateway ingress. When a request belongs to an authenticated user configured in
`blocking_exempt_user_ids`, the coordinator resolves and freezes
`blocking_exempt_at_request = true`.

Previously, Content Moderation exempted direct-user images by sending them to
asynchronous shadow review while keeping local text keywords strictly
authoritative and blocking. This change adjusts local keyword enforcement so
that blocking-exempt requests are allowed while continuing to capture and
persist complete keyword hit evidence for administration review.

## Decisions

### 1. Keyword check branching for blocking-exempt requests

In `ContentModerationService.Check`, when `cfg.Mode == ContentModerationModePreBlock`
and `matchBlockedKeyword(content.Text)` finds a match:

- **For ordinary requests (`!input.BlockingExemptAtRequest`)**:
  - Behavior is unchanged: returns `ContentModerationDecision{Blocked: true, Action: ContentModerationActionKeywordBlock}`, emits `keyword_block` metric, builds log, and applies enforcement side effects (violation counters, auto-ban, notification email).
- **For exempt requests (`input.BlockingExemptAtRequest == true`)**:
  - Emits `content_moderation.keyword_exempt_shadow` structured log.
  - Builds log with:
    - `Action = ContentModerationActionShadow`
    - `Flagged = true`
    - `HighestCategory = "keyword"`
    - `HighestScore = 1.0`
    - `MatchedKeyword = keyword`
    - `InputCiphertext = encryptedAuditedText`
    - `BlockingExemptAtRequest = true`
  - Enqueues record with `recordHash = false` and `applySideEffects = false`.
  - Continues through the existing pre-hash and external API pipeline. The
    keyword finding itself cannot block, while an independent known hash or
    successful blocking API risk finding retains its existing authority.

### 2. Snapshot column in content moderation logs

Add `blocking_exempt_at_request BOOLEAN NOT NULL DEFAULT FALSE` to
`content_moderation_logs`.

- Captured at insert time from `ContentModerationCheckInput.BlockingExemptAtRequest`.
- Persisted permanently so list views do not need to recompute current policy.
- Excluded from list-payload ciphertext leaks (retains only `input_excerpt` in list).

### 3. Record detail decrypt authorization

Update `GetLogInput` so that any record with a valid `MatchedKeyword` and
`InputCiphertext` (whether stored as `keyword_block` or `shadow` with exemption)
can be decrypted by authenticated administrators via `GET /api/v1/admin/risk-control/logs/:id/input`.

### 4. Risk Control UI representation

- In `RiskControlView.vue`, render an "免阻塞" / "Non-blocking" badge with tooltip
  when `row.blocking_exempt_at_request` is true.
- Display the matched keyword and status accurately.
