## ADDED Requirements

### Requirement: Prompt Audit events identify their deciding Guard node

Every persisted Prompt Audit event SHALL snapshot the ID, configured name, and
model of the Guard node whose successful result determines the aggregate event
decision. The administration event list and detail view SHALL display the node
and model without consulting mutable current configuration.

#### Scenario: Priority node completes the audit

- **WHEN** a configured Guard node successfully produces the deciding result
- **THEN** the event SHALL store that node's ID, configured name, and model
- **AND** the list SHALL display its name and model

#### Scenario: Guard fails over to another node

- **WHEN** an earlier node is unavailable and a later node produces the
  deciding result
- **THEN** the event SHALL identify the later node actually used
- **AND** it SHALL NOT attribute the event to the failed node

#### Scenario: Historical event lacks a node name snapshot

- **WHEN** an event predates node-name snapshots
- **THEN** the administration UI SHALL fall back to its stored endpoint ID
- **AND** it SHALL display the existing Qwen3Guard scanner version as the model
  fallback without rewriting the historical event

### Requirement: Node attribution excludes sensitive configuration

Event attribution SHALL NOT persist or return the Guard Base URL, token,
decrypted credential state, request content, or raw Guard response.

#### Scenario: Event is serialized for the administration API

- **WHEN** the event is listed or retrieved
- **THEN** attribution SHALL contain only node ID, node name, and model
