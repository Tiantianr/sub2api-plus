# OpenAI OAuth Effective Access Specification

## ADDED Requirements

### Requirement: One effective OAuth access rule

Runtime and management paths SHALL treat an OpenAI OAuth account as group-eligible only when the current group binding and session-sharing allowlist both authorize the group. A restricted per-user policy SHALL additionally require a valid user grant.

#### Scenario: Ghost binding is ineffective

- **WHEN** `account_groups` contains a group that is absent from the OAuth session-sharing allowlist
- **THEN** scheduling, sticky reuse, admin matrices, and impact previews all treat the account as unavailable for that group

### Requirement: Group copies do not grant OAuth access

Group duplication and account-copy paths SHALL NOT add an OAuth account binding for a destination group unless the account already explicitly authorizes that group. They SHALL NOT mutate the account allowlist.

#### Scenario: Duplicate an authorized source group

- **WHEN** an administrator duplicates a group containing an OAuth account and the destination group is not allowlisted
- **THEN** the destination group is created without that OAuth account binding

### Requirement: Long-lived access is revalidated

Responses WebSocket and Live calls SHALL revalidate current group state, binding, session-sharing allowlist, and per-user policy before accepting a new user-controlled turn.

#### Scenario: Group authorization is revoked during a connection

- **WHEN** the binding or allowlist authorization is removed after connection establishment
- **THEN** the next user-controlled turn is rejected before upstream write
