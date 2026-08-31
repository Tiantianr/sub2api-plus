## ADDED Requirements

### Requirement: Prompt Audit external Guard calls must redact direct sensitive identifiers

The system SHALL replace recognizable credentials, email addresses, telephone
numbers, checksum-valid Chinese identity numbers and bank cards, and valid IP
addresses with fixed typed placeholders after canonical extraction but before
constructing every external Guard request. This boundary MUST apply to
synchronous review, asynchronous workers, retries, and failover. The external
request MUST NOT contain the matched plaintext value or a plaintext-derived
hash, suffix, or domain.

#### Scenario: A synchronous chunk contains credentials and contact details

- **WHEN** a synchronous Prompt Audit chunk contains a recognizable credential, email address, or telephone number
- **THEN** the Guard request SHALL contain only the corresponding typed placeholders
- **AND** the Guard request SHALL NOT contain any matched plaintext value
- **AND** the canonical prompt hash and encrypted evidence SHALL remain derived from the original content

#### Scenario: An asynchronous job retries another Guard node

- **WHEN** an asynchronous chunk containing sensitive identifiers is retried or failed over
- **THEN** every Guard node attempt SHALL receive the redacted outbound copy
- **AND** no attempt SHALL receive the matched plaintext value

#### Scenario: Numeric candidates are not valid identifiers

- **WHEN** a Chinese identity candidate fails its checksum, a bank-card candidate fails Luhn, or an IP candidate fails parsing
- **THEN** the system SHALL NOT classify that candidate as the corresponding identifier type

### Requirement: Outbound redaction must preserve configured PII decision semantics

The system SHALL retain a value-free local PII signal for email, telephone,
identity, bank-card, and IP replacements. The signal MUST affect a Guard result
only when the configured scanner allowlist contains `pii` and the strictly
parsed Guard safety is non-Safe. Credentials alone MUST NOT create a PII signal.

#### Scenario: Guard returns Safe for redacted PII

- **WHEN** local outbound redaction finds PII
- **AND** Guard returns `Safety: Safe`
- **THEN** the final result SHALL remain Pass / Allow

#### Scenario: PII is enabled and Guard returns Controversial

- **WHEN** local outbound redaction finds PII
- **AND** `pii` is enabled
- **AND** Guard returns `Safety: Controversial`
- **THEN** the effective result SHALL include `pii`
- **AND** the existing elevated Controversial PII rule SHALL produce Critical / Block

#### Scenario: PII is disabled

- **WHEN** local outbound redaction finds PII
- **AND** `pii` is not enabled
- **THEN** the local signal SHALL NOT add PII categories, scores, evidence, or a PII decision

### Requirement: Outbound redaction must not widen persistent sensitive-data exposure

The system MUST NOT log, persist, cache, or globally map values matched by the
outbound redactor. Requests without any match SHOULD reuse the original string,
and matched requests SHOULD construct no more than one redacted content string
before the existing JSON serialization.

#### Scenario: No outbound-sensitive pattern matches

- **WHEN** a Guard chunk contains no recognized outbound-sensitive value
- **THEN** the redactor SHALL return content identical to the input
- **AND** it SHALL report no local PII signal

#### Scenario: A matched request completes

- **WHEN** a Guard request containing typed placeholders completes
- **THEN** logs and event diagnostics SHALL contain no matched plaintext value
- **AND** no redaction mapping SHALL be retained after the scanner call
