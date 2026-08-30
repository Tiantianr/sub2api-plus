## ADDED Requirements

### Requirement: Terminal Prompt Audit failures must appear in the event list

The system SHALL create a lightweight failed event for a synchronous Guard
failure and for a terminal asynchronous Prompt Audit failure. The existing
administration event list SHALL show its stable error code and safe generic
reason.

#### Scenario: Synchronous Guard evaluation fails

- **WHEN** a synchronous Guard evaluation ends unavailable or invalid
- **THEN** the system SHALL create one failed event
- **AND** the original failure-allow or fail-closed decision SHALL remain
  unchanged

#### Scenario: Asynchronous audit exhausts retries

- **WHEN** an asynchronous audit job reaches terminal failed status
- **THEN** the system SHALL create one failed event linked to that job

#### Scenario: Asynchronous audit will retry

- **WHEN** an asynchronous audit attempt fails with retries remaining
- **THEN** the system SHALL NOT create a failed list event for that attempt

#### Scenario: Historical terminal failure is migrated

- **WHEN** migration 243 finds a failed Prompt Audit job without an event
- **THEN** it SHALL create one lightweight failed event from the existing
  redacted job metadata and stable stored reason

### Requirement: Failed events must not expose content or dependency secrets

Failed events SHALL contain only the existing redacted request snapshot, stable
error code, and generic safe message.

#### Scenario: Failed event is stored or returned

- **WHEN** a failed event is persisted, listed, or viewed
- **THEN** it MUST NOT contain complete prompts, media, tool payloads,
  credentials, endpoint URLs, raw Guard responses, encrypted context, or the
  underlying dependency error string

#### Scenario: Failure-event persistence fails

- **WHEN** the failed event cannot be persisted
- **THEN** the original audit decision, retry transition, billing, and upstream
  dispatch SHALL remain unchanged
