## ADDED Requirements

### Requirement: Keyword findings retain encrypted audited text

For a new `keyword_block` finding, the system SHALL retain the normalized text
that participated in keyword matching after applying the existing Content
Moderation secret redactor and SHALL persist only AES-256-GCM ciphertext for
that complete retained value.

The authenticated plaintext SHALL carry the fixed
`sub2api:content-moderation-keyword-input:v1:` purpose/version prefix. The
detail path SHALL reject successfully decrypted values without that prefix.

The retained text SHALL respect the existing Content Moderation input bound and
SHALL NOT include image data URLs.

#### Scenario: Long keyword-hit text is retained

- **WHEN** a normalized text input longer than the list excerpt limit matches a configured blocked keyword
- **THEN** the request is blocked under the existing keyword policy
- **AND** `input_excerpt` remains bounded to the existing list-safe limit
- **AND** the complete redacted audited text is stored as ciphertext
- **AND** plaintext complete text is absent from the persisted row

#### Scenario: Encryption is unavailable

- **WHEN** a known keyword finding cannot encrypt its complete text
- **THEN** the request remains blocked
- **AND** the ordinary excerpt record is still eligible for persistence
- **AND** no plaintext fallback is persisted
- **AND** the failure log contains no prompt or ciphertext

#### Scenario: Ciphertext belongs to another application purpose

- **WHEN** the detail column contains valid shared-key ciphertext whose plaintext does not carry the keyword-input purpose/version prefix
- **THEN** the detail endpoint returns a generic internal error
- **AND** no decrypted value is returned

### Requirement: Complete text is excluded from log lists

The paginated Content Moderation log API SHALL NOT select or return retained
ciphertext or decrypted complete text.

#### Scenario: Administrator lists keyword findings

- **WHEN** an administrator lists Content Moderation logs containing keyword findings
- **THEN** each row contains only the existing bounded `input_excerpt`
- **AND** neither ciphertext nor complete text appears in the response

### Requirement: Complete text is loaded through record-scoped administration

The system SHALL provide an authenticated administrator-only endpoint that
loads one Content Moderation record by positive numeric ID and decrypts retained
content only for a valid keyword finding row.

#### Scenario: New keyword finding is opened

- **WHEN** an administrator opens a keyword finding with valid retained ciphertext
- **THEN** the response contains the complete redacted audited text
- **AND** `complete` is true

#### Scenario: Historical keyword finding is opened

- **WHEN** an administrator opens a historical keyword finding without retained ciphertext
- **THEN** the response contains its existing excerpt
- **AND** `complete` is false
- **AND** the UI identifies it as an incomplete historical record

#### Scenario: Missing record is requested

- **WHEN** an administrator requests an unknown record ID
- **THEN** the endpoint returns a generic not-found response
- **AND** it discloses no stored content or database details

#### Scenario: Ciphertext cannot be decrypted

- **WHEN** retained ciphertext fails authenticated decryption
- **THEN** the endpoint returns a generic internal error
- **AND** neither ciphertext nor cryptographic details appear in the response

### Requirement: Detail loading does not expand other disclosure paths

Complete retained input SHALL NOT be included in table cells, HTML title
attributes, searches, notifications, application logs, or Content Moderation
failure records.

#### Scenario: Administrator has not opened a row

- **WHEN** the Content Moderation log table renders
- **THEN** no complete retained input has been requested
- **AND** the DOM contains only the list-safe excerpt
