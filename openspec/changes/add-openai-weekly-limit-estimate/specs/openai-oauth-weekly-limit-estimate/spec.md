## ADDED Requirements

### Requirement: Weekly estimate samples platform account cost at percentage transitions

For an OpenAI OAuth account, the system SHALL estimate the implied 7-day
platform account-cost limit only when the displayed upstream 7-day utilization
advances from a previously observed positive integer percentage.

#### Scenario: First observation establishes a baseline

- **WHEN** an account has no estimate snapshot for the current 7-day window
- **THEN** the system SHALL persist the current displayed percentage as the
  baseline
- **THEN** the usage response SHALL NOT contain a weekly-limit estimate

#### Scenario: Percentage advances

- **WHEN** the previous observed percentage is 6
- **WHEN** the next observation is 7 and current 7-day account cost is 160.87
- **THEN** the estimate SHALL be `160.87 / 6 * 100`
- **THEN** the snapshot SHALL record 6 as the basis and 7 as the observed
  percentage

#### Scenario: Percentage remains unchanged

- **WHEN** the displayed percentage has not changed but account cost increases
- **THEN** the previously persisted estimate SHALL remain unchanged

#### Scenario: Overlapping observations hold stale baselines

- **WHEN** two overlapping observations see the same percentage increase with
  different account-cost samples
- **THEN** the first successfully persisted estimate SHALL remain frozen
- **THEN** the later observation SHALL NOT replace it from a stale baseline

#### Scenario: Observation skips an integer percentage

- **WHEN** the previous observed percentage is 6 and the next observation is 8
- **THEN** the system SHALL still use 6 as the estimate basis

### Requirement: Weekly estimate follows the upstream window lifecycle

The system SHALL associate the estimator snapshot with the upstream 7-day reset
anchor and SHALL NOT carry an estimate into a different or corrected window.

#### Scenario: Upstream window changes

- **WHEN** the 7-day reset anchor changes beyond the accepted observation drift
- **THEN** the system SHALL clear the estimate and establish a new baseline

#### Scenario: Utilization moves backwards

- **WHEN** displayed utilization decreases within the observed window
- **THEN** the system SHALL clear the estimate and establish the lower value as
  the new baseline

#### Scenario: Reset countdown drifts slightly

- **WHEN** two reset anchors differ by no more than 15 minutes
- **THEN** the system SHALL treat them as the same 7-day window

### Requirement: Account list shows a platform-only estimate

The OpenAI OAuth account list SHALL show the latest persisted estimate beside
the 7-day row and SHALL identify the sampled account cost and percentage basis
without exposing user-billed cost.

#### Scenario: Estimate exists

- **WHEN** a weekly-limit estimate is present
- **THEN** the 7-day row SHALL show a compact estimated account-cost marker
- **THEN** its accessible description SHALL include the sampled account cost,
  basis percentage, and estimated account cost

#### Scenario: Estimate does not exist

- **WHEN** only a baseline has been established
- **THEN** no estimate marker SHALL be shown
