## ADDED Requirements

### Requirement: Concurrent recovery claim contention must wait

The system SHALL wait while the current request context remains active when a
user has a pending Prompt Audit recovery finding and another request owns the
bounded recovery claim. The waiter MUST NOT replace the owner claim or finding,
and claim contention MUST NOT be reported as unavailable recovery state.

#### Scenario: Claim owner clears recovery

- **WHEN** a concurrent request waits behind the recovery claim owner
- **AND** the owner completes exact Allow and clears the finding
- **THEN** the waiting request SHALL resume ordinary blocking Prompt Audit
- **AND** it SHALL NOT receive a claim-contention 503

#### Scenario: Claim owner retains recovery

- **WHEN** a concurrent request waits behind the recovery claim owner
- **AND** the owner releases its claim without clearing the finding
- **THEN** one waiter MAY acquire the released claim
- **AND** that waiter SHALL perform complete uncached recovery review

#### Scenario: Waiting request ends

- **WHEN** the waiting request context is canceled or reaches its deadline
- **THEN** waiting SHALL stop promptly
- **AND** the owner claim and pending finding SHALL remain unchanged

#### Scenario: Recovery storage fails while waiting

- **WHEN** Redis cannot read the finding or acquire the recovery claim
- **THEN** recovery SHALL remain fail closed with
  `prompt_guard_deep_review_state_unavailable`

#### Scenario: An old claim owner finishes after replacement

- **WHEN** a recovery claim expires or is replaced before its original owner
  finishes
- **AND** the original owner later produces Allow
- **THEN** the original owner MUST NOT clear the finding
- **AND** it SHALL fail closed before upstream side effects
