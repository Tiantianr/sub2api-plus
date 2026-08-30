## ADDED Requirements

### Requirement: Prompt Audit hot-path optimization must preserve semantics

The system SHALL avoid redundant synchronous request-body and rune-chunk copies
while preserving immutable audit input, UTF-8 rune boundaries, priority segment
ordering, complete chunk coverage, and existing side-effect ordering.

#### Scenario: Both synchronous engines inspect one request

- **WHEN** Content Moderation and Prompt Audit run concurrently
- **THEN** they MAY share the frozen request body read-only
- **AND** neither engine SHALL mutate the body

#### Scenario: Asynchronous review outlives the request

- **WHEN** an allowed request schedules asynchronous deep review
- **THEN** the asynchronous boundary SHALL retain its own request-body copy

#### Scenario: Unicode input spans multiple chunks

- **WHEN** Guard input exceeds the configured rune limit
- **THEN** every chunk SHALL end on a valid UTF-8 rune boundary
- **AND** concatenating chunks within each priority segment SHALL reproduce the
  exact original segment

#### Scenario: Optimization cannot reduce audit coverage

- **WHEN** the optimized hot path processes content-bearing input
- **THEN** it MUST preserve canonical extraction, selected modules, required
  chunks, node failover, and fail-closed behavior
