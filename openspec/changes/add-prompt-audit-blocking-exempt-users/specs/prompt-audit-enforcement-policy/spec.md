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
remain fully reviewed but are not blocked by valid Prompt Audit content
findings. Stored events MUST retain the original Guard decision, risk, action,
and evidence policy. The exemption MUST NOT change Content Moderation or Prompt
Audit dependency-failure behavior.

#### Scenario: Blocking-exempt user produces a Critical finding

- **WHEN** an in-scope blocking-exempt user submits content that Guard classifies as Critical and Block
- **THEN** Prompt Audit SHALL persist the Critical event with its original Block action
- **AND** Prompt Audit SHALL allow the request to continue as a flagged finding
- **AND** synchronous and asynchronous review SHALL NOT create user recovery state

#### Scenario: Blocking-exempt user encounters an audit failure

- **WHEN** an in-scope blocking-exempt user's Prompt Audit review encounters extraction failure, invalid Guard output, or an unavailable dependency
- **THEN** the existing failure-closed or configured failure-allow policy SHALL apply unchanged

#### Scenario: Existing recovery is exempted

- **WHEN** an administrator adds a user with pending recovery to `blocking_exempt_user_ids`
- **THEN** in-scope requests SHALL continue ordinary Prompt Audit review without recovery blocking
- **AND** the existing recovery finding SHALL remain stored
- **AND** removing the exemption SHALL resume required recovery

#### Scenario: Exemption changes during recovery review

- **WHEN** a recovery review starts while its user and group require enforcement
- **AND** the user becomes blocking-exempt or the group leaves scope before exact Allow is committed
- **THEN** the request MAY continue under the newly active policy
- **AND** the existing recovery finding SHALL remain stored

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
