## ADDED Requirements

### Requirement: Pass evidence retention is selected per user

Prompt Audit SHALL retain Pass events and their complete encrypted context only
for explicitly selected authenticated user IDs. An absent, empty, or invalid
retention configuration SHALL select no users. Flag and Critical events SHALL
be retained regardless of the user's Pass-retention selection.

#### Scenario: Unselected user passes review

- **WHEN** Prompt Audit returns Pass for a user absent from the active retention
  list
- **THEN** the review, metrics, and applicable Allow receipt SHALL complete
- **AND** no Pass event or complete event context SHALL be created

#### Scenario: Selected user passes review

- **WHEN** Prompt Audit returns Pass for a user in the active retention list
- **THEN** the system SHALL store the Pass event and encrypted complete context

#### Scenario: Unselected user has a finding

- **WHEN** Prompt Audit returns Flag or Critical for a user absent from the
  active retention list
- **THEN** the event and encrypted complete context SHALL still be stored

### Requirement: Retention changes are independent from Guard policy

The Pass-retention list SHALL have an independent monotonic revision and SHALL
NOT change Prompt Audit's Guard configuration version, invalidate an Allow
receipt, make a queued job stale, or alter blocking and recovery decisions.

#### Scenario: Administrator changes one selected user

- **WHEN** an administrator saves a different Pass-retention user list
- **THEN** the retention revision SHALL advance atomically
- **AND** the active Guard configuration version SHALL remain unchanged
- **AND** other instances SHALL converge through invalidation or bounded reload

#### Scenario: Retention storage is unavailable or corrupt

- **WHEN** the retention snapshot cannot be loaded
- **THEN** no user SHALL retain new Pass evidence
- **AND** Prompt Audit review and blocking SHALL continue with their active
  Guard policy

### Requirement: Normal-event cleanup is previewed and Pass-only

The administration console SHALL provide a cleanup shortcut whose immutable
decision filter is Pass. Cleanup SHALL require an explicit time range and a
displayed server preview before confirmation. The preview SHALL include match
count, context-artifact count, estimated retained-content bytes, snapshot high
water, filter digest, and an administrator-bound expiring confirmation token.

#### Scenario: Administrator previews a user's old normal records

- **WHEN** an administrator selects a user and an explicit cutoff
- **THEN** the preview SHALL count only matching Pass events
- **AND** it SHALL estimate stored encrypted context and full-prompt bytes
  without returning or decrypting content

#### Scenario: Administrator confirms cleanup

- **WHEN** the administrator confirms the unchanged preview within its validity
  window
- **THEN** only matching Pass events at or below the preview high water SHALL
  be deleted in bounded batches
- **AND** complete contexts SHALL be cascade-deleted
- **AND** newer events, Flag/Critical events, and Allow receipts SHALL survive
