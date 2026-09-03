## ADDED Requirements

### Requirement: Prompt Audit content is grouped by user session

The system SHALL associate each retained Prompt Audit content record with an
existing user and an opaque, user-scoped session key. A client-provided stable
session identifier SHALL be preferred when available; a request-scoped fallback
MUST be used when no stable identifier exists, and the system MUST NOT infer a
session by comparing prompt text.

#### Scenario: Stable session identifier is present

- **WHEN** two accepted requests from the same user carry the same supported
  session identifier
- **THEN** they SHALL resolve to the same Prompt Audit session

#### Scenario: Session identifier is absent

- **WHEN** an accepted request has no supported stable session identifier
- **THEN** the request ID SHALL identify a fallback session and it MUST NOT be
  merged with another request solely because their content is equal

#### Scenario: Cache routing key is reused

- **WHEN** independent requests reuse the same `prompt_cache_key` without a
  stable conversation or thread identifier
- **THEN** they MUST NOT be grouped into one Prompt Audit session

### Requirement: Session content is deduplicated without changing evidence

The system SHALL store complete Prompt Audit content outside event metadata.
Within one user session, equal complete prompt content and equal complete
context content SHALL be stored once, and each matching event SHALL reference
that content record. An event MUST continue to resolve the content that was
audited at that event, rather than silently displaying a later session state.

#### Scenario: Repeated session content

- **WHEN** the same complete prompt and complete context are recorded twice in
  one user session
- **THEN** the events SHALL reference one chat content record

#### Scenario: Different session versions

- **WHEN** a later request changes the complete prompt or complete context
- **THEN** it SHALL receive a distinct content record and the earlier event
  SHALL continue to resolve its original content

### Requirement: Administrator can analyze the selected session

The administration console SHALL provide an analysis action on each detection
event that has a user identity. The analysis endpoint SHALL load content only
from the session attached to the selected event and SHALL use the configured
Prompt Audit endpoint pool with normal endpoint priority and failover.

The analysis result SHALL be returned in the response and MUST NOT be persisted
as a chat record or included in application logs. Sensitive analysis responses
MUST use `Cache-Control: no-store`.

#### Scenario: Analyze selected session

- **WHEN** an administrator analyzes an event with available session content
- **THEN** the model SHALL receive only that session's bounded content and the
  response SHALL identify the selected session and the model used

#### Scenario: No retained session content

- **WHEN** the selected event has no available content because it expired or
  was never retained
- **THEN** the endpoint SHALL return a generic no-content error and MUST NOT
  invoke an audit model

### Requirement: Chat content retention and backup exclusion

Unselected Pass chat content SHALL expire seven days after its latest retained
occurrence. The configured Pass evidence retention exception and existing risk
evidence policy SHALL remain eligible for indefinite retention. Risk decisions
and event metadata remain searchable. Expired content
MUST be deleted without deleting its event metadata, and the existing event
deletion and user-scoped cleanup actions SHALL continue to delete matching
events and orphaned content.

Logical PostgreSQL backups MUST exclude chat content data, including complete
context ciphertext. Backup restore MUST remain valid when event metadata refers
to omitted chat content.

#### Scenario: Automatic content expiry

- **WHEN** an unselected Pass chat content record is older than seven days
- **THEN** the cleanup loop SHALL delete the content while retaining the
  detection event metadata

#### Scenario: Backup excludes chat content

- **WHEN** a PostgreSQL logical backup is created
- **THEN** it MUST omit chat content rows and legacy complete-context rows while
  retaining restorable audit metadata
