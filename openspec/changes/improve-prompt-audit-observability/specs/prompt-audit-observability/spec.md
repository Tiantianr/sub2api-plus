## ADDED Requirements

### Requirement: Prompt Audit events must retain complete and attributable review content
The system SHALL retain the complete extracted text submitted for each newly stored Prompt Audit event, together with prompt length, message count, execution mode, and trusted client IP. Full text MUST remain absent from list queries, logs, errors, and non-admin responses.
This requirement supersedes the original Prompt Audit change's prohibition on admin-only event-content persistence; jobs and operational logs remain redacted.

#### Scenario: An administrator opens a new event
- **WHEN** an authorized administrator requests one Prompt Audit event detail
- **THEN** the response MUST contain the complete extracted text used for that audit
- **THEN** the event MUST identify its client IP, prompt size, message count, and execution mode
- **THEN** the sensitive read MUST be recorded by the existing administration audit log without the prompt content

#### Scenario: Historical content was already truncated
- **WHEN** a stored event contains fewer characters than its recorded prompt length
- **THEN** the API and UI MUST identify the retained content as incomplete
- **THEN** download and review actions MUST NOT describe the retained content as complete

### Requirement: Prompt Audit events must expose queue and chunk diagnostics
The system SHALL store the async queue delay, effective input limit, and first highest-severity matched chunk index for each new event. Blocking events SHALL report zero queue delay; historical events whose delay cannot be reconstructed SHALL report it as unknown.

#### Scenario: An asynchronous event completes after waiting
- **WHEN** a queued or retried job produces an event
- **THEN** queue delay MUST measure from job creation to the processing claim that produced the event
- **THEN** Guard latency MUST remain a separate field

#### Scenario: A chunk blocks a request
- **WHEN** one chunk determines the highest-severity aggregate result
- **THEN** the event MUST report that one-based chunk index and the effective per-chunk character limit

### Requirement: Administrators must be able to filter events by client IP
The event API and console SHALL support exact normalized client-IP filtering. The event list SHALL render the IP as an accessible action that applies that exact filter.

#### Scenario: An administrator selects an event IP
- **WHEN** the administrator activates a non-empty IP value in an event row
- **THEN** the event list MUST immediately query using that exact IP
- **THEN** all other active filters MUST remain unchanged

### Requirement: Endpoint limits must be visible and support bounded long-context audits
The endpoint input limit SHALL accept 128 through 500,000 Unicode characters. The console MUST display the accepted timeout and input ranges next to their controls and explain that oversized input is split and that the minimum enabled-node limit applies.

#### Scenario: An administrator configures a 500,000-character limit
- **WHEN** the endpoint configuration is otherwise valid and one audited segment contains no more than 500,000 Unicode characters
- **THEN** the backend MUST accept and activate the value
- **THEN** that segment MUST be submitted as one chunk

### Requirement: Runtime success and error times must be distinct
Runtime status SHALL expose the last successful processing time and the last worker-error time independently.

#### Scenario: Processing succeeds after an earlier error
- **WHEN** a later job succeeds without a new worker error
- **THEN** the success timestamp MUST advance
- **THEN** the older error MUST retain its original timestamp and MUST NOT appear to have occurred at the success time
