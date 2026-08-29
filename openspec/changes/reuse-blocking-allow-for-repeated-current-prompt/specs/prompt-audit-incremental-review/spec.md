## RENAMED Requirements

- FROM: `### Requirement: A new current user turn receives its first review`
- TO: `### Requirement: Current user review may reuse exact blocking receipts`

## MODIFIED Requirements

### Requirement: Current user review may reuse exact blocking receipts

Blocking Prompt Audit SHALL omit direct current-user content only when it has
an unexpired stored Allow receipt for the same authenticated user, active
Prompt Audit configuration and Guard policy, source class, and
receipt-equivalent canonical text. For user segments, receipt equivalence MAY
replace only strict `[images:<hex>]` media markers with a stable placeholder;
all other text SHALL remain exact. Every historical user turn SHALL retain the same receipt
requirement. An asynchronous-only current user turn SHALL ignore stored
receipts. After an exact complete synchronous Allow, asynchronous deep review
MAY reuse the same request's trusted in-process receipt handoff.

#### Scenario: User repeats receipt-equivalent current text

- **WHEN** blocking Prompt Audit previously completed exact Allow for one
  current user segment and its combined request was permitted
- **AND** the same user submits receipt-equivalent canonical text under the
  same active policy before the receipt expires
- **THEN** blocking Prompt Audit SHALL omit that segment from Guard input
- **AND** receipt hit observability SHALL increase

#### Scenario: Repeated request changes only an opaque media marker

- **WHEN** the selected Prompt Audit text has a valid blocking Allow receipt
- **AND** only a strict 32-, 40-, or 64-character hexadecimal identifier in an
  `[images:<hex>]` user-text marker changes
- **THEN** Prompt Audit MAY omit the certified canonical text
- **AND** Content Moderation SHALL continue evaluating applicable current text
  and media independently

#### Scenario: Current text or policy changes

- **WHEN** receipt-normalized current canonical text, authenticated user,
  configuration version, Guard policy, source class, or receipt-schema version
  differs
- **THEN** the stored receipt SHALL miss
- **AND** the current segment SHALL run normal Guard review

#### Scenario: Async-only request repeats current text

- **WHEN** Prompt Audit is async-only and current canonical text has a stored
  receipt
- **THEN** the async job SHALL review that current segment again

#### Scenario: Tool continuation repeats the conversation

- **WHEN** a request ends with a current tool output and carries a historical
  user turn with a valid receipt
- **THEN** synchronous Prompt Audit SHALL NOT resubmit that historical user
  turn
- **AND** asynchronous review SHALL consider only receipt misses

### Requirement: Dual-lane and recovery behavior remains intact

Content Moderation and synchronous Prompt Audit SHALL continue concurrently.
Only combined Allow SHALL commit pending synchronous receipts and enqueue
asynchronous deep review. Normal upstream processing and asynchronous deep
review SHALL then continue concurrently. A late deep Block SHALL retain
existing next-request recovery behavior. Forced recovery SHALL bypass all
receipts and only a complete synchronous deep Allow SHALL clear recovery state.

#### Scenario: Required recovery repeats certified text

- **WHEN** a user has pending deep recovery and submits text with valid Allow
  receipts
- **THEN** recovery SHALL review every selected segment without receipt reuse
- **AND** only complete exact deep Allow MAY clear the recovery state

#### Scenario: Deep review blocks a new tool output

- **WHEN** incremental asynchronous deep review Blocks a new tool output
- **THEN** the already allowed response SHALL NOT be retroactively cancelled
- **AND** the user's next request SHALL require full synchronous deep recovery
