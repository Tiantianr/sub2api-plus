## ADDED Requirements

### Requirement: Keyword detection does not block blocking-exempt requests

The system SHALL treat a configured `blocked_keywords` match as a non-enforcing
shadow finding when a pre-block Content Moderation check receives
`blocking_exempt_at_request = true`. Independent hash and API checks SHALL
continue under their existing policy.

#### Scenario: Blocking-exempt request matches blocked keyword
- **WHEN** an authenticated request marked `blocking_exempt_at_request = true` contains text matching a configured blocked keyword
- **THEN** the keyword finding itself does not set `Blocked: true`
- **AND** a log entry is created with `action = 'shadow'`, `flagged = true`, and `matched_keyword` set to the hit keyword
- **AND** the complete redacted text is retained as AES-256-GCM ciphertext in `input_ciphertext`
- **AND** the log record persists `blocking_exempt_at_request = true`
- **AND** no automatic ban or violation penalty is applied to the user

#### Scenario: Exempt keyword text matches a known risk hash
- **WHEN** a blocking-exempt request matches a configured keyword and its independently checked text hash is known as risky
- **THEN** the keyword finding is recorded as non-enforcing shadow evidence
- **AND** the known hash finding blocks the request under the existing hash policy

#### Scenario: Exempt keyword text receives a blocking API finding
- **WHEN** a blocking-exempt request matches a configured keyword and blocking text API review returns a successful risk finding
- **THEN** the keyword finding is recorded as non-enforcing shadow evidence
- **AND** the API finding blocks the request under the existing text API policy

#### Scenario: Non-exempt request matches blocked keyword
- **WHEN** an ordinary request (`blocking_exempt_at_request = false`) contains text matching a configured blocked keyword
- **THEN** Content Moderation returns `Allowed: false`, `Blocked: true`, and `action = 'keyword_block'`
- **AND** automatic ban and enforcement side effects apply normally

### Requirement: Administrator can inspect decrypted keyword input for exempt hits

The system SHALL allow the administrator endpoint
`GET /api/v1/admin/risk-control/logs/:id/input` to decrypt and return the
complete text for purpose-bound keyword-matched records regardless of whether
the action was `keyword_block` or `shadow`.

#### Scenario: Administrator views exempt keyword finding
- **WHEN** an administrator opens an exempt keyword hit with valid `matched_keyword` and `input_ciphertext`
- **THEN** the response contains decrypted complete text with `complete: true`
