## MODIFIED Requirements

### Requirement: Selected users may be exempt from Prompt Audit finding blocks

The system SHALL accept a bounded versioned list of authenticated user IDs that
are admitted to full asynchronous Prompt Audit without waiting for synchronous
Guard. Stored events MUST retain the original Guard decision, risk, action, and
evidence policy and MUST identify that the request was admitted as
blocking-exempt. The exemption MUST NOT weaken Content Moderation. Incomplete
content extraction, encryption failure, and configuration-version mismatch
MUST remain fail closed; dependencies whose final stable result is
`prompt_guard_unavailable` MUST NOT block the current request.

#### Scenario: Blocking-exempt user produces a Critical finding

- **WHEN** an in-scope blocking-exempt user submits content that Guard classifies as Critical and Block
- **THEN** the request SHALL NOT wait for synchronous Guard
- **AND** the request SHALL continue when Content Moderation permits it
- **AND** a successfully admitted asynchronous review SHALL persist the Critical event with its original Block action and request-time exemption marker
- **AND** asynchronous review SHALL NOT create Allow receipts or user recovery state

#### Scenario: Blocking-exempt request has a local security failure

- **WHEN** an in-scope blocking-exempt request encounters incomplete extraction, encryption failure, or a configuration-version mismatch
- **THEN** Prompt Audit SHALL fail closed before account selection, billing, concurrency acquisition, or upstream dispatch
- **AND** synchronous Guard SHALL NOT be called

#### Scenario: Blocking-exempt asynchronous admission is unavailable

- **WHEN** an in-scope blocking-exempt request encounters queue capacity exhaustion, database unavailability, payload-storage failure, queue-publication failure, or another admission dependency ending as `prompt_guard_unavailable`
- **THEN** the current request SHALL continue when Content Moderation permits it
- **AND** synchronous Guard SHALL NOT be called
- **AND** any staging failure created before admission ended SHALL best-effort persist the existing safe failed event with the request-time exemption marker
- **AND** the request SHALL NOT create an Allow receipt

#### Scenario: Existing recovery is unavailable

- **WHEN** a user has a pending recovery finding and the required Guard review ends as `prompt_guard_unavailable`
- **THEN** the current request SHALL continue without a Prompt Audit unavailable 503
- **AND** the existing recovery finding SHALL remain stored
- **AND** any temporary recovery claim SHALL be released
