## ADDED Requirements

### Requirement: Configured scanners must be a server-enforced category boundary

The system SHALL allow only configured known scanner categories to affect a
normalized Prompt Audit decision or appear as effective event findings. A known
category omitted from the active `scanners` list MUST be discarded before
aggregation, persistence, issue-summary derivation, and administration display.
The system MUST NOT rely on Guard to honor the configured category list.

#### Scenario: Guard reports only a disabled known category

- **WHEN** Guard returns Controversial or Unsafe with only a known category that is not enabled
- **THEN** Prompt Audit SHALL normalize the result as Pass and Allow
- **AND** effective categories, matched scanners, scores, evidence, and issue summaries SHALL be empty

#### Scenario: Guard reports enabled and disabled known categories

- **WHEN** Guard returns a result containing both an enabled known category and a disabled known category
- **THEN** only the enabled category SHALL participate in the decision
- **AND** only the enabled category SHALL appear in the event and administration views

#### Scenario: Guard reports an unknown Unsafe category

- **WHEN** Guard returns Unsafe with a category that is not in the stable scanner catalog
- **THEN** Prompt Audit SHALL retain the existing Critical and Block behavior
- **AND** the unknown identifier SHALL remain value-free and non-reversible

#### Scenario: Historical event contains a disabled observed category

- **WHEN** an existing event row contains a known category absent from its persisted `matched_scanners`
- **THEN** event API reads and derived issue summaries SHALL expose only `matched_scanners` as effective categories
- **AND** the historical database row SHALL NOT be rewritten
