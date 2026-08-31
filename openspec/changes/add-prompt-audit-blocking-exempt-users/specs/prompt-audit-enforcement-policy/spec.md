## ADDED Requirements

### Requirement: Configured group scope must bound every Prompt Audit lane

The system SHALL apply active Prompt Audit group scope before synchronous
review, asynchronous enqueue, recovery claiming, and final recovery fencing.
Pending user recovery MUST remain stored while a request is outside scope and
MUST resume on a later in-scope request.

#### Scenario: Pending recovery user changes to an unselected group

- **WHEN** a user has pending Prompt Audit recovery
- **AND** the user submits a request through a group not selected by the active Prompt Audit policy
- **THEN** Prompt Audit SHALL NOT review, enqueue, or create an event for that request
- **AND** the request SHALL NOT be blocked by Prompt Audit recovery
- **AND** the pending recovery finding SHALL remain stored

#### Scenario: User returns to a selected group

- **WHEN** a user has pending Prompt Audit recovery retained outside scope
- **AND** the user later submits a request through a selected group
- **THEN** the existing required recovery review SHALL resume before upstream side effects

#### Scenario: Queued job becomes out of scope

- **WHEN** an asynchronous job was admitted under an older selected-group configuration
- **AND** the active configuration removes that group before processing starts
- **THEN** the worker SHALL terminate the job without calling Guard or creating an event

### Requirement: Selected users may be exempt from Prompt Audit finding blocks

The system SHALL accept a bounded versioned list of authenticated user IDs that
are reliably admitted to full asynchronous Prompt Audit without waiting for
synchronous Guard. Stored events MUST retain the original Guard decision, risk,
action, and evidence policy and MUST identify that the request was admitted as
blocking-exempt. The exemption MUST NOT weaken Content Moderation or allow a
request to continue when content extraction, encryption, database admission,
payload storage, or queue publication fails.

#### Scenario: Blocking-exempt user produces a Critical finding

- **WHEN** an in-scope blocking-exempt user submits content that Guard classifies as Critical and Block
- **THEN** Prompt Audit SHALL reliably queue the complete deep review before upstream side effects
- **AND** the request SHALL NOT wait for synchronous Guard
- **AND** the request SHALL continue when Content Moderation permits it
- **AND** Prompt Audit SHALL persist the Critical event with its original Block action
- **AND** the event SHALL identify that the request was blocking-exempt at admission
- **AND** asynchronous review SHALL NOT create Allow receipts or user recovery state

#### Scenario: Blocking-exempt request cannot be reliably queued

- **WHEN** an in-scope blocking-exempt request encounters incomplete extraction, encryption failure, queue capacity exhaustion, or an unavailable admission dependency
- **THEN** Prompt Audit SHALL fail closed before account selection, billing, concurrency acquisition, or upstream dispatch
- **AND** synchronous Guard SHALL NOT be called
- **AND** a payload-storage or queue-publication failure after job creation SHALL best-effort persist the existing safe failed event with the request-time exemption marker

#### Scenario: Blocking-exempt asynchronous Guard fails

- **WHEN** a reliably queued blocking-exempt job exhausts retries because Guard is unavailable or returns invalid output
- **THEN** Prompt Audit SHALL persist the existing safe terminal failure event with the request-time exemption marker
- **AND** the already admitted request SHALL NOT be retroactively blocked

#### Scenario: Content Moderation blocks a blocking-exempt user

- **WHEN** Content Moderation blocks a request whose user is blocking-exempt only from Prompt Audit
- **THEN** the request SHALL remain blocked by Content Moderation
- **AND** the queued Prompt Audit job SHALL NOT create an Allow receipt

#### Scenario: Existing recovery is exempted

- **WHEN** an administrator adds a user with pending recovery to `blocking_exempt_user_ids`
- **THEN** an in-scope request SHALL skip recovery claiming and synchronously admit full asynchronous review
- **AND** the existing recovery finding SHALL remain stored
- **AND** removing the exemption SHALL resume required recovery for later requests

#### Scenario: Exemption is removed after job admission

- **WHEN** a blocking-exempt job is admitted and the administrator later removes the user from `blocking_exempt_user_ids`
- **THEN** that job SHALL retain its request-time exemption marker
- **AND** its Guard finding SHALL NOT create recovery state
- **AND** later in-scope requests SHALL use the new non-exempt policy

### Requirement: Blocking exemption history must be immutable and visible

The system SHALL persist a request-time blocking-exemption snapshot on Prompt
Audit jobs and events. The existing event list SHALL display that snapshot in
the queue/audit column and MUST NOT infer historical status from the current
configuration.

#### Scenario: Administrator views an exempt audit event

- **WHEN** an event was created from a request admitted while its user was blocking-exempt
- **THEN** the queue/audit column SHALL display both asynchronous execution and blocking-exempt status
- **AND** changing the current exemption list SHALL NOT change the historical marker

### Requirement: Blocking exemption configuration must be bounded and auditable

The system SHALL canonicalize blocking-exempt user IDs as a sorted unique list
of positive IDs and SHALL reject more than 100 submitted IDs. Configuration
updates SHALL use the existing Prompt Audit version conflict protection, and
bounded change summaries SHALL contain only the selected count and set hash.

#### Scenario: Administrator saves duplicate user IDs

- **WHEN** an administrator saves repeated valid blocking-exempt user IDs
- **THEN** the active and public configuration SHALL contain one sorted occurrence of each ID

#### Scenario: Older client omits the additive field

- **WHEN** a valid Prompt Audit configuration update omits `blocking_exempt_user_ids`
- **THEN** the currently stored blocking-exempt user list SHALL be preserved

#### Scenario: Administrator exceeds the exemption limit

- **WHEN** an administrator submits more than 100 blocking-exempt user IDs
- **THEN** the update SHALL be rejected with a stable validation error
