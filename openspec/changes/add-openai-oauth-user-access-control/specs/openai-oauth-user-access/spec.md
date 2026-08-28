## ADDED Requirements

### Requirement: Restricted OpenAI OAuth accounts must require an explicit user grant
The system SHALL apply OpenAI OAuth user access after ordinary API-key group eligibility. Public accounts SHALL preserve existing scheduling behavior. Restricted accounts MUST require a grant for the trusted authenticated local user and MUST fail closed when that identity or grant is absent.

#### Scenario: An authorized user sends a request
- **WHEN** an API key's group contains a restricted OpenAI OAuth root account and its owner has a grant for that account
- **THEN** the account MAY participate in ordinary scheduling
- **THEN** the user MUST continue using the existing API key without receiving upstream account details

#### Scenario: An unauthorized account has higher priority
- **WHEN** an ungranted restricted account would otherwise outrank a granted or public account
- **THEN** the ungranted account MUST be excluded before priority or load selection
- **THEN** the scheduler MAY choose only another ordinarily eligible account that the user may access

#### Scenario: No permitted candidate remains
- **WHEN** all ordinarily eligible OpenAI OAuth candidates are restricted and ungranted
- **THEN** the request MUST fail without falling back to an ungranted account
- **THEN** the client response MUST NOT reveal account identity or grant details

### Requirement: Access policies must cover every credential reuse path
The system SHALL enforce the current grant for new selections, sticky selections, response continuations, failover, and long-lived connection turns. Spark shadows SHALL inherit the policy of their OpenAI OAuth root account.

#### Scenario: A sticky or previous-response account was revoked
- **WHEN** a user no longer has a grant for an account referenced by sticky state or response continuation
- **THEN** that state MUST NOT authorize reuse of the account
- **THEN** any fallback MUST remain within the user's permitted candidates

#### Scenario: Authorization changes during a long connection
- **WHEN** an administrator revokes the selected account after an HTTP or SSE request has started
- **THEN** the in-flight request MAY finish naturally
- **THEN** the next HTTP request or WebSocket/Live control turn MUST revalidate and reject reuse of the revoked account

#### Scenario: A Spark shadow is selected
- **WHEN** the scheduler considers a shadow of a restricted OpenAI OAuth account
- **THEN** access MUST be decided from the root account's policy and grants
- **THEN** the shadow MUST NOT have an independent policy or grant set

### Requirement: Administrators must manage policies atomically
The system SHALL provide an administrator-only user/account matrix for OpenAI OAuth root accounts. It SHALL support public/restricted mode, future-user defaults, batch grant and revoke, impact preview, and optimistic revision checks.

#### Scenario: An administrator restricts several accounts
- **WHEN** the administrator previews and confirms a valid batch of policy and grant changes
- **THEN** every submitted account change MUST commit in one transaction or none may commit
- **THEN** the audit record MUST describe identities, revisions, modes, and grant counts without credentials or tokens

#### Scenario: Two administrators edit the same account
- **WHEN** the submitted expected revision differs from durable policy state
- **THEN** the update MUST fail with a conflict response
- **THEN** no submitted policy or grant change may be applied

#### Scenario: Public mode is restored
- **WHEN** an account changes from restricted to public
- **THEN** all stored grants for that account MUST be removed
- **THEN** the future-user default MUST be disabled

### Requirement: Future-user defaults must create explicit grants
The system SHALL grant every restricted account marked as a future-user default to each newly inserted ordinary user in the same database transaction. Defaults SHALL NOT implicitly alter existing users.

#### Scenario: A user is created through any registration path
- **WHEN** an ordinary user row is inserted while one or more restricted accounts are marked as future-user defaults
- **THEN** explicit grants for those accounts MUST exist when user creation commits

#### Scenario: A default is enabled or disabled
- **WHEN** an administrator changes the future-user default setting
- **THEN** existing users and existing grants MUST remain unchanged
- **THEN** only later ordinary-user inserts MUST use the new default set

### Requirement: Existing installations must remain public after migration
The migration SHALL leave accounts without a policy row in public mode and SHALL NOT create restrictions or grants for existing users.

#### Scenario: The migration is applied without administrator changes
- **WHEN** an existing installation starts the new version
- **THEN** OpenAI OAuth routing MUST remain behaviorally equivalent to the pre-migration routing
