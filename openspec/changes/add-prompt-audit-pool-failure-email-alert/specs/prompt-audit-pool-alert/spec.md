## ADDED Requirements

### Requirement: Consecutive Prompt Audit pool failures must notify operations

The system SHALL count final Prompt Audit Guard pool evaluations that return
unavailable or invalid across synchronous and asynchronous audit lanes. The
system SHALL send one critical email to the configured Ops alert recipients when
the count reaches five consecutive failures. Any complete Allow, Flag, or Block
result SHALL reset the count and permit a later failure streak to notify again.

#### Scenario: Fifth consecutive pool evaluation fails

- **WHEN** five consecutive final Guard pool evaluations return unavailable or
  invalid
- **THEN** the system SHALL asynchronously send one critical Ops alert email
- **AND** later failures in the same streak SHALL NOT send duplicate emails

#### Scenario: Pool returns a complete decision

- **WHEN** a Guard pool evaluation returns Allow, Flag, or Block
- **THEN** the consecutive failure count SHALL reset
- **AND** a later sequence of five failures SHALL be eligible for a new email

#### Scenario: Failover succeeds

- **WHEN** one Guard node attempt fails but an enabled fallback node returns a
  complete result
- **THEN** the evaluation SHALL count as a successful pool result

#### Scenario: Failure occurs outside the Guard pool

- **WHEN** content extraction, configuration, encryption, or recovery state is
  unavailable before a final Guard pool outcome
- **THEN** the event SHALL NOT increment the Guard pool failure streak

#### Scenario: Caller cancels an in-flight evaluation

- **WHEN** the parent request context is canceled before a final Guard pool
  outcome
- **THEN** the cancellation SHALL NOT increment or reset the failure streak

#### Scenario: Alert delivery fails

- **WHEN** alert configuration lookup or email delivery fails
- **THEN** the Prompt Audit decision and request enforcement SHALL remain
  unchanged

#### Scenario: Alert content is rendered

- **WHEN** the failure alert email is created
- **THEN** it SHALL contain only bounded operational metadata
- **AND** it MUST NOT contain prompts, request bodies, user data, credentials,
  endpoint URLs, or raw Guard responses
