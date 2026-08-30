## ADDED Requirements

### Requirement: Synchronous Prompt Audit Blocks must persist user recovery

The system SHALL persist user-scoped recovery state when synchronous Prompt
Audit Blocks an authenticated non-exempt request. API-key and client-controlled
session identity changes MUST NOT avoid the next recovery review. Enforcement
MAY pause outside configured Prompt Audit group scope or while the user is
blocking-exempt, without clearing the state. Exact Allow SHALL clear the state
immediately when recovery enforcement applies.

#### Scenario: Blocked user changes request identity

- **WHEN** synchronous Prompt Audit Blocks an authenticated user's request
- **AND** the user's next in-scope request changes API key or session identity
- **THEN** the next request SHALL remain subject to recovery

#### Scenario: First recovery review allows

- **WHEN** a user has pending synchronous Block recovery
- **AND** the next required recovery review returns exact Allow
- **THEN** the system SHALL clear the user-scoped recovery state
- **AND** later requests SHALL resume ordinary Prompt Audit selection

### Requirement: Recovery must perform uncached active deep review

Recovery SHALL synchronously review every selected user turn and the optional
sources enabled in the active `deep_review_modules`. It MUST bypass every
trusted or stored Allow receipt and MUST use the active configuration at the
time of the recovery request.

#### Scenario: Previously allowed history enters recovery

- **WHEN** a user with pending recovery submits a request whose historical
  segments have valid Allow receipts
- **THEN** every user turn and every active deep-review module segment SHALL be
  sent to Guard
- **THEN** no receipt hit SHALL omit a selected segment

#### Scenario: Deep module configuration changes before recovery

- **WHEN** an administrator changes `deep_review_modules` after the Block and
  before the user's recovery request
- **THEN** recovery SHALL use the newly active deep-review module selection

### Requirement: Only complete exact Allow may clear recovery

The system SHALL clear recovery for any content-bearing request type when its
currently submitted canonical context completes uncached deep review with exact
Allow. No other review result may clear the recovery state. Configured group
scope and blocking exemption MAY pause enforcement without clearing it. Removed prior
content SHALL NOT be reattached to the recovery input. The non-expiring finding
token MUST remain separate from the bounded in-progress recovery claim.

#### Scenario: Automatic tool continuation recovers

- **WHEN** a user with pending recovery submits a content-bearing tool
  continuation without a new direct user turn
- **AND** complete deep review returns exact Allow
- **THEN** the system SHALL clear recovery and permit upstream processing

#### Scenario: Recovery is not exact Allow

- **WHEN** recovery produces Warn, Block, invalid response, timeout, extraction
  failure, empty selection, or recovery-state failure
- **THEN** upstream processing SHALL remain blocked
- **THEN** recovery SHALL remain required for the user

### Requirement: All Prompt Audit Blocks must share versioned recovery state

Synchronous Prompt Audit Block and asynchronous deep Block SHALL write the same
user-level versioned recovery state. A recovery request MUST clear only the
exact state version it claimed.

#### Scenario: Concurrent recovery requests

- **WHEN** one request holds the bounded per-user recovery claim
- **AND** another request observes the same pending finding
- **THEN** the later request MUST NOT replace the first request's claim or the
  pending finding token
- **AND** an exact Allow from the claim owner MAY clear the unchanged finding

#### Scenario: Recovery process stops before completion

- **WHEN** a process stops while holding the bounded recovery claim
- **THEN** the claim lease MAY expire
- **AND** the underlying non-expiring finding MUST remain required
- **AND** a later request MAY acquire a new claim and perform complete recovery

#### Scenario: Synchronous Block creates recovery

- **WHEN** synchronous Prompt Audit returns Block for an authenticated user
- **THEN** the system SHALL write a new synchronous-Block recovery token before
  returning the blocked response

#### Scenario: New finding races with recovery Allow

- **WHEN** recovery claims one state version
- **AND** a newer synchronous or asynchronous Block writes another version
- **THEN** the older recovery Allow MUST NOT clear the newer state
- **THEN** the recovery request SHALL fail closed before upstream side effects

#### Scenario: New recovery state appears during ordinary review

- **WHEN** an ordinary blocking request initially observes no recovery state
- **AND** another request writes recovery before the ordinary request completes
  its final gateway audit gate
- **THEN** the final recovery fence SHALL prevent receipt persistence, deep-job
  enqueue, account selection, billing, and upstream writes for that request

### Requirement: Administrative mode changes must not clear recovery state

The system SHALL retain user-level recovery state when an administrator
disables risk control, Prompt Audit, or synchronous blocking, changes group
scope, or makes the user blocking-exempt. Enforcement MAY pause while blocking
does not apply and MUST resume when it applies again.

#### Scenario: Blocking is re-enabled

- **WHEN** a user has pending recovery while blocking Prompt Audit is disabled
- **AND** an administrator enables blocking Prompt Audit again
- **THEN** the user's next request SHALL require complete uncached deep review

### Requirement: Recovery must preserve gateway error compatibility

The system SHALL preserve the existing protocol-specific error mappings for
initial synchronous Block, required recovery, and unavailable recovery
dependencies.

#### Scenario: Initial synchronous Block is returned

- **WHEN** synchronous Prompt Audit Blocks an ordinary request
- **THEN** HTTP ingress SHALL return 403 with `prompt_guard_blocked`
- **THEN** WebSocket ingress SHALL use the existing 4403 policy close mapping

#### Scenario: Required recovery does not Allow

- **WHEN** required recovery returns Warn or Block
- **THEN** HTTP ingress SHALL return 403 with
  `prompt_guard_deep_review_required`
- **THEN** WebSocket ingress SHALL use the existing 4403 policy close mapping

#### Scenario: Recovery dependency is unavailable

- **WHEN** recovery cannot complete because extraction, Guard, or recovery
  state is unavailable
- **THEN** HTTP ingress SHALL return 503 with the existing stable error code
- **THEN** WebSocket ingress SHALL use the existing 1013 unavailable mapping
- **AND** a recovery-state failure message SHALL be distinguishable from
  ordinary Guard unavailability

### Requirement: Every selected synchronous Block must trigger recovery

A synchronous aggregate Block for an in-scope, non-exempt user SHALL write
user-level recovery regardless of which source selected by the active blocking
policy caused the result. Prompt Audit recovery MUST NOT create Content
Moderation punishment state.

#### Scenario: Selected non-user source Blocks

- **WHEN** synchronous Prompt Audit selects an enabled system, assistant, or
  tool source
- **AND** the aggregate Guard result is Block
- **THEN** the system SHALL write user-level recovery before returning 403
- **THEN** it MUST NOT increment a user violation count or automatically ban
  the user because of that Prompt Audit finding

### Requirement: Recovery state must not expire automatically

User-level Prompt Audit recovery state SHALL be stored without a TTL. Elapsed
time and service restart MUST NOT satisfy or clear required recovery.

#### Scenario: Blocked user returns later

- **WHEN** a user has pending recovery and submits no requests for an arbitrary
  period
- **THEN** the next request SHALL still require complete uncached deep review

### Requirement: Recovery transitions must be observable without content

The system SHALL expose bounded structured logs and cumulative runtime counters
for recovery creation by source, successful clearing, retained recovery, and
state errors. Recovery observability MUST NOT expose raw state tokens, prompts,
tool values, media, or client session identifiers.

#### Scenario: Synchronous Block writes recovery

- **WHEN** synchronous Prompt Audit writes user-level recovery
- **THEN** a structured event and synchronous-source counter SHALL record the
  transition using bounded identifiers only

#### Scenario: Recovery remains required

- **WHEN** recovery cannot clear because of a non-Allow result, concurrent
  state version, or dependency error
- **THEN** the appropriate retained or error event and counter SHALL be updated
- **THEN** no audited content or raw recovery token SHALL enter observability
