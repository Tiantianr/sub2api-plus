## ADDED Requirements

### Requirement: Image moderation follows the frozen Prompt Audit blocking exemption

The system SHALL resolve Prompt Audit blocking exemption before starting the
concurrent Prompt Guard and Content Moderation checks and SHALL apply that same
request-scoped decision to image enforcement.

#### Scenario: Ordinary image request remains synchronous

- **WHEN** Content Moderation is in pre-block mode
- **WHEN** the request contains an image and is not blocking-exempt
- **THEN** the system SHALL wait for the image Moderation API result
- **THEN** a valid configured risk finding SHALL block the current request

#### Scenario: Blocking-exempt image request is admitted immediately

- **WHEN** Prompt Audit scope applies and the user is blocking-exempt at request time
- **WHEN** the request contains an image
- **THEN** Content Moderation SHALL NOT wait for image API completion
- **THEN** Content Moderation SHALL enqueue the image for asynchronous review
- **THEN** the current request SHALL NOT be blocked by that image review

#### Scenario: Exempt image finding remains non-enforcing

- **WHEN** asynchronous review later finds risk in an exempt request image
- **THEN** the system SHALL persist a non-enforcing shadow observation
- **THEN** it SHALL NOT create a flagged hash, increment enforcement counters,
  send enforcement email, or disable the user

#### Scenario: Local text policy remains authoritative

- **WHEN** a blocking-exempt image request also contains text matching a local
  blocking keyword or a known text-only flagged hash
- **THEN** the local text policy MAY still block the current request
- **THEN** the image exemption SHALL NOT disable unrelated text controls

### Requirement: Moderation API availability never blocks conversation admission

The system SHALL treat Moderation API availability failures as observable
non-findings and SHALL allow the current request.

#### Scenario: Synchronous image API returns an error

- **WHEN** an ordinary pre-block image request reaches the Moderation API
- **WHEN** the API times out, returns non-2xx, returns malformed output, or
  returns no result
- **THEN** the system SHALL allow the current request
- **THEN** it SHALL record a stable dependency failure without raw upstream body

#### Scenario: No Moderation API key is usable

- **WHEN** image or blocking text API moderation is required
- **WHEN** no configured Moderation API key is currently usable
- **THEN** the system SHALL allow the current request
- **THEN** it SHALL expose the failure through safe operational telemetry

#### Scenario: API failure is not a safe result

- **WHEN** a Moderation API dependency failure is allowed
- **THEN** the system SHALL NOT create a safe moderation result, flagged hash,
  allow receipt, recovery state, or enforcement side effect

#### Scenario: Explicit security boundaries remain fail closed

- **WHEN** canonical extraction, active configuration, or required hash state
  fails in pre-block mode
- **THEN** the existing explicit security failure behavior SHALL remain unchanged
- **THEN** a known keyword, known hash, or successful API risk finding SHALL
  retain its configured blocking authority for non-exempt requests

### Requirement: Transport surfaces share image exemption semantics

HTTP and WebSocket request paths SHALL apply the same frozen image exemption
and API-failure allowance without bypassing canonical extraction.

#### Scenario: WebSocket turn belongs to an exempt user

- **WHEN** a first or subsequent WebSocket turn contains an image
- **WHEN** Prompt Audit marks the user blocking-exempt for that turn
- **THEN** image moderation SHALL be asynchronous and non-enforcing for that turn

#### Scenario: HTTP media request belongs to an ordinary user

- **WHEN** an HTTP media or model request contains an image and is not exempt
- **THEN** successful image findings SHALL retain pre-block authority
- **THEN** Moderation API availability failure SHALL still allow the request
