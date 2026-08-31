## ADDED Requirements

### Requirement: Client default quota follows account-aware precedence

The gateway SHALL resolve client-facing default Codex quota visibility in the
following immutable order: enabled local subscription quota, selected-account
upstream quota when OpenAI automatic passthrough is enabled, otherwise hidden.

#### Scenario: Local quota overrides automatic passthrough

- **WHEN** local Codex subscription quota is enabled and the selected account
  also enables OpenAI automatic passthrough
- **THEN** the client SHALL receive only local Primary and Secondary values
- **THEN** no upstream default percentage, reset, window, or ratio value SHALL
  remain

#### Scenario: Automatic passthrough exposes real upstream quota

- **WHEN** local Codex subscription quota is disabled and the selected account
  enables OpenAI automatic passthrough
- **THEN** eligible upstream default quota fields SHALL remain unchanged after
  generic response-header filtering

#### Scenario: Non-passthrough account hides real upstream quota

- **WHEN** local Codex subscription quota is disabled and the selected account
  does not enable OpenAI automatic passthrough
- **THEN** the gateway SHALL remove every default Primary and Secondary quota
  field before committing the response

#### Scenario: Local quota has no active window

- **WHEN** local mode is selected but no active local quota window can be built
- **THEN** the gateway SHALL NOT fall through to real upstream default quota

### Requirement: Header policy covers every OpenAI client response path

The gateway SHALL apply the account-aware default quota policy after generic
header filtering and before committing native Responses, converted Chat or
Messages, raw compatibility, Embeddings, Alpha Search, image, compact,
non-streaming protocol-error, and client-visible upstream-error responses.

#### Scenario: Additional header allowance attempts to expose quota

- **WHEN** generic response-header configuration allows default Codex quota
  fields for a non-passthrough selected account
- **THEN** the final account-aware policy SHALL remove those fields

#### Scenario: First output is staged for failover

- **WHEN** an OpenAI attempt stages response headers until meaningful output
- **THEN** quota finalization SHALL apply to the per-attempt staged header map
- **THEN** a discarded attempt SHALL NOT commit its quota source to the client

#### Scenario: Upstream source headers are reused internally

- **WHEN** the gateway finalizes a cloned or destination header map
- **THEN** it SHALL NOT mutate the original upstream response header map

### Requirement: WebSocket default quota events follow the same policy

Every Responses WebSocket relay SHALL apply the account-aware policy to text
frames in the default `codex.rate_limits` family before client observation or
write.

#### Scenario: Local WebSocket quota replaces upstream event

- **WHEN** local quota is enabled and upstream emits a default
  `codex.rate_limits` event
- **THEN** the client SHALL receive one locally constructed default event with
  local Primary and Secondary windows

#### Scenario: Hidden WebSocket quota is suppressed

- **WHEN** local quota is disabled, the selected account is not automatic
  passthrough, and upstream emits a default `codex.rate_limits` event
- **THEN** that event SHALL NOT be written to the client
- **THEN** the relay SHALL continue processing subsequent frames

#### Scenario: Automatic-passthrough WebSocket quota is preserved

- **WHEN** local quota is disabled and the selected account enables automatic
  passthrough
- **THEN** the default upstream event SHALL remain unchanged

#### Scenario: Named model-specific quota is independent

- **WHEN** a `codex.rate_limits` event names a model-specific metered limit
- **THEN** the gateway SHALL preserve the event regardless of the default
  quota visibility mode

#### Scenario: Binary WebSocket frame

- **WHEN** a relay receives a binary upstream frame
- **THEN** the quota policy SHALL NOT parse or transform that frame

### Requirement: Client WebSocket upgrade remains pre-account

The client-facing WebSocket upgrade SHALL use only request-scoped local quota
known before account selection and SHALL NOT expose a later selected account's
upstream quota.

#### Scenario: Upgrade with local quota disabled

- **WHEN** local quota is disabled and the gateway commits the client `101`
  before selecting or connecting an upstream account
- **THEN** the upgrade SHALL contain no injected default Codex quota fields
- **THEN** account-aware upstream visibility SHALL begin only with later
  in-band WebSocket events
