# Prompt Audit Evidence Specification

## ADDED Requirements

### Requirement: Guard scans user-authored prompt text

Synchronous blocking SHALL scan only the latest user-authored input. Asynchronous Prompt Audit SHALL scan all user-authored turns. Both modes SHALL exclude non-user sources and strip supported client harness XML from selected user text.

#### Scenario: Harness does not block a normal user prompt

- **WHEN** a request contains Codex or Claude harness content and a normal user prompt
- **THEN** Guard receives the applicable user prompt without the harness block

### Requirement: Blocking extraction failures fail closed

A content-bearing malformed or incomplete canonical extraction SHALL prevent account selection, billing, concurrency acquisition, and upstream writes when blocking Prompt Audit applies.

#### Scenario: Recognized sibling cannot hide unsupported content

- **WHEN** a blocking request contains selected user text and an unsupported content-bearing sibling
- **THEN** the request fails with the extraction dependency error before upstream side effects

### Requirement: Stored events retain downloadable complete context

Every newly stored Prompt Audit event SHALL retain an application-encrypted complete canonical context artifact containing all extracted text segments, including content excluded from Guard selection. Ordinary event APIs and logs SHALL NOT expose the artifact or ciphertext.

#### Scenario: Administrator downloads a blocked event context

- **WHEN** an authenticated administrator downloads context for a newly stored blocked event
- **THEN** the response contains the complete canonical segments and exact Guard selection with `no-store` caching

#### Scenario: Event deletion removes context

- **WHEN** an event is deleted individually or in bulk
- **THEN** its complete-context artifact is deleted by the same transaction boundary

## MODIFIED Requirements

### Requirement: Security audit input must cover effective upstream text semantics

The canonical extractor and downloadable event evidence SHALL retain every supported client-controlled text value that an accepted request can cause the gateway to send as upstream prompt or conversation text. Each audit engine MAY apply its documented attribution selection to that canonical document. Prompt Audit Guard SHALL select user-authored text only.

#### Scenario: Excluded upstream context remains reviewable

- **WHEN** an accepted request includes instructions, assistant text, reasoning, prompt variables, or tool content that Prompt Audit excludes from Guard input
- **THEN** the canonical document SHALL retain that content
- **THEN** a newly stored Prompt Audit event SHALL make it available through the encrypted admin context download

### Requirement: Administration controls must reflect effective blocking behavior

The administration console SHALL state that synchronous blocking always scans the latest user input and SHALL NOT present a mutable latest/full selection control. The compatibility field MAY remain in storage and API schemas.

#### Scenario: Administrator edits blocking configuration

- **WHEN** the administrator saves Prompt Audit configuration
- **THEN** the console SHALL describe the fixed latest-user blocking policy
- **THEN** changing the compatibility field SHALL NOT change runtime selection

## REMOVED Requirements

### Requirement: Blocking Prompt Audit must maintain fail-closed conversation checkpoints

**Reason**: Full conversation blocking created unacceptable latency and false positives from non-user context.

**Migration**: Blocking now scans the latest user input; event evidence retains complete canonical context.

#### Scenario: Blocking request no longer creates a checkpoint

- **WHEN** blocking Prompt Audit allows a request
- **THEN** it SHALL NOT create or advance a conversation checkpoint

### Requirement: Incremental audit must use trusted prior output and current input

**Reason**: Prompt Audit no longer reviews assistant output.

**Migration**: Async audit scans all user-authored turns in each accepted request.

#### Scenario: Stable continuation uses user-authored selection

- **WHEN** an async continuation is audited
- **THEN** all user-authored turns in that request SHALL be selected without checkpoint state

### Requirement: Full replay must not bypass historical blocking

**Reason**: Historical user turns are assigned to async review rather than synchronous blocking.

**Migration**: Synchronous blocking scans the latest user turn; async review scans historical user turns.

#### Scenario: Historical text is not synchronously replayed

- **WHEN** a blocking request includes older user turns and a latest user turn
- **THEN** only the latest user turn SHALL enter synchronous Guard input

### Requirement: Context and parent changes must invalidate incremental eligibility

**Reason**: Incremental checkpoint eligibility no longer exists.

**Migration**: Every request applies the fixed blocking or async user selection independently.

#### Scenario: Parent changes do not consult a checkpoint

- **WHEN** a request changes or omits a parent response identifier
- **THEN** Prompt Audit selection SHALL remain based on the request's user-authored text

### Requirement: Downstream output capture must be complete, bounded, and protocol-aware

**Reason**: Assistant output is excluded from Prompt Audit Guard input.

**Migration**: Complete event evidence captures inbound canonical context only.

#### Scenario: Allowed output is not retained as a checkpoint

- **WHEN** an allowed response completes over HTTP, SSE, Responses WebSocket, or Live
- **THEN** Prompt Audit SHALL NOT retain that output for a later Guard request

### Requirement: Legacy latest-turn configuration must not misrepresent runtime behavior

**Reason**: Latest-user selection is now the fixed runtime policy rather than a legacy option.

**Migration**: The field remains accepted only for rolling-upgrade compatibility.

#### Scenario: Compatibility value is ignored

- **WHEN** stored `blocking_latest_turn_only` is true or false
- **THEN** synchronous Prompt Audit SHALL scan only the latest user input
