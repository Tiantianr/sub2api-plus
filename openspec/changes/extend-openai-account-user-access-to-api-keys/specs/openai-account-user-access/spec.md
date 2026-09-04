## ADDED Requirements

### Requirement: OpenAI API-key root accounts must be manageable
The administrator OpenAI account access matrix SHALL list non-deleted root
accounts on the OpenAI platform with type `oauth` or `apikey`. Setup-token,
other-platform, and shadow accounts SHALL remain excluded from direct policy
management.

#### Scenario: An API-key account is listed
- **WHEN** an active or inactive OpenAI API-key root account exists
- **THEN** the account SHALL appear in the matrix with `type: apikey`
- **THEN** its existing policy, grants, groups, status, and revision SHALL be returned

### Requirement: OpenAI API-key accounts must obey the same access policy
The system SHALL apply public/restricted mode, explicit user grants, API-key
group intersection, revision checks, atomic updates, scheduler refresh, and
long-lived connection rechecks to OpenAI API-key root accounts using the same
semantics as OpenAI OAuth accounts.

#### Scenario: A granted user uses a restricted API-key account
- **WHEN** the user's API-key groups intersect the account groups and the user has a grant
- **THEN** the account MAY be selected and used

#### Scenario: An ungranted user uses a restricted API-key account
- **WHEN** the user's API-key groups intersect the account groups but no grant exists
- **THEN** the account SHALL be excluded before upstream use
- **THEN** no ungranted API-key account fallback SHALL be allowed

### Requirement: Existing OpenAI behavior remains compatible
Accounts without a policy row SHALL remain public. Existing OAuth policies,
grants, revision behavior, error reason codes, endpoint paths, and future-user
semantics SHALL remain compatible. API-key accounts SHALL not be included in
the future-user default grant unless the policy is restricted and the default
is enabled, and administrator identities SHALL not be granted automatically.

#### Scenario: A legacy OpenAI API-key account has no policy
- **WHEN** the new migration and application are deployed
- **THEN** the account SHALL retain its existing unrestricted behavior

#### Scenario: A future ordinary user is created
- **WHEN** a restricted OpenAI OAuth or API-key root has the future-user default enabled
- **THEN** an explicit grant SHALL be inserted in the same user transaction
- **THEN** no administrator grant SHALL be inserted by that trigger
