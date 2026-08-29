# Prompt Audit Incremental Review Specification

## ADDED Requirements

### Requirement: A new current user turn receives its first review

Blocking Prompt Audit SHALL synchronously review direct user content marked
current by the canonical extractor. Every historical user turn SHALL require a
valid receipt; historical receipt misses SHALL be reviewed synchronously even
when another user turn is marked current.
An asynchronous-only current user turn SHALL ignore old receipts. After an
exact complete synchronous Allow, asynchronous deep review MAY reuse the same
request's trusted in-process receipt handoff.

#### Scenario: Tool continuation repeats the conversation

- **WHEN** a request ends with a current tool output and carries a historical
  user turn
- **AND** that user turn has a valid receipt
- **THEN** synchronous Prompt Audit SHALL NOT resubmit that historical user
  turn
- **AND** asynchronous review SHALL consider only receipt misses

#### Scenario: Client appends a claimed assistant continuation

- **WHEN** a client submits unreviewed user text followed by an assistant or
  tool item
- **THEN** absence of a current-user marker SHALL NOT exempt the user text
- **AND** its receipt miss SHALL be reviewed synchronously

#### Scenario: Client places unreviewed text before a benign current user

- **WHEN** an earlier user turn has no valid receipt
- **AND** a later user turn is marked current
- **THEN** synchronous Guard input SHALL include both receipt misses

#### Scenario: User intentionally repeats identical text

- **WHEN** identical text is marked as a new current user turn
- **THEN** its first active lane SHALL call Guard regardless of an old receipt

### Requirement: Review receipts are canonical-segment scoped

The system SHALL derive receipts from canonical user turns and individually
classified system, assistant, reasoning, prompt-variable, tool-definition,
tool-call, and tool-output segments. Adding one segment SHALL NOT invalidate
unchanged sibling segments. All misses SHALL be submitted in one prioritized
Guard input.

#### Scenario: A tool loop adds one output

- **WHEN** prior user, assistant, and tool segments have valid receipts
- **AND** one new tool output is appended
- **THEN** Guard SHALL receive the new tool output
- **AND** SHALL NOT receive the unchanged historical segments

### Requirement: Only an exact complete Allow creates receipts

The system SHALL write receipts only after every submitted chunk completes and
the aggregate action is Allow, and only when Content Moderation also permits
the original request. Warn, Block, invalid, timeout, extraction failure,
partial results, and combined gateway rejection SHALL NOT create receipts.

#### Scenario: One attachment chunk times out

- **WHEN** any chunk in a cache miss times out
- **THEN** no submitted segment receipt is stored
- **AND** blocking behavior remains fail closed

### Requirement: Receipts are isolated and bounded

An Allow receipt SHALL bind the user, Prompt Audit config version, Guard
endpoint and scanner policy, source class, and exact segment content. Entries
SHALL expire after the configured TTL, whose compatibility default is one hour.
The key SHALL also bind a receipt-schema revision so implementation changes do
not reuse older extraction or selection semantics.

#### Scenario: Same segment belongs to another user

- **WHEN** another user submits byte-identical content
- **THEN** the first user's Allow SHALL NOT be reused

#### Scenario: Configuration changes while a job is queued

- **WHEN** a worker receives a job whose configuration version differs from
  the active version
- **THEN** it SHALL fail the stale job without calling Guard
- **AND** SHALL NOT write any receipt

### Requirement: Receipt dependencies fail open to review, not to upstream

A receipt miss, timeout, or Redis error SHALL run the normal Guard review. A
receipt write failure SHALL preserve the completed Guard decision but SHALL NOT
create an Allow receipt.

#### Scenario: Redis lookup is unavailable

- **WHEN** receipt lookup fails
- **THEN** Guard receives every selected segment not covered by the trusted
  same-request handoff
- **AND** normal blocking or asynchronous decision semantics continue

### Requirement: Dual-lane and recovery behavior remains intact

Content Moderation and synchronous Prompt Audit SHALL continue concurrently.
Only combined Allow SHALL enqueue asynchronous deep review. Normal upstream
processing and asynchronous deep review SHALL then continue concurrently. A
late deep Block SHALL retain existing next-request recovery behavior. Forced
recovery SHALL bypass all receipts and only a complete synchronous deep Allow
SHALL clear recovery state.

#### Scenario: Deep review blocks a new tool output

- **WHEN** incremental asynchronous deep review Blocks a new tool output
- **THEN** the already allowed response SHALL NOT be retroactively cancelled
- **AND** the user's next request SHALL require full synchronous deep recovery
