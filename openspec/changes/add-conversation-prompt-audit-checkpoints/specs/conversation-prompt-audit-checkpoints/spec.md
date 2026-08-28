## ADDED Requirements

### Requirement: Blocking Prompt Audit must maintain fail-closed conversation checkpoints

The gateway SHALL maintain an atomic temporary checkpoint for each identifiable in-scope conversation. Only a completely allowed input audit followed by a complete successful downstream response MAY establish `CLEAN`; every other state SHALL be `FULL_REQUIRED`.

#### Scenario: A new conversation starts

- **WHEN** no valid CLEAN checkpoint exists
- **THEN** Prompt Audit MUST scan all canonical text present in the current request
- **THEN** account selection, billing, concurrency acquisition, and upstream writes MUST wait for the full allow decision

#### Scenario: An audited turn is interrupted

- **WHEN** Guard, Redis, extraction, downstream capture, parsing, or transport completion fails
- **THEN** the request or checkpoint transition MUST fail closed
- **THEN** the conversation MUST NOT retain or acquire CLEAN eligibility from that turn

#### Scenario: Two turns overlap

- **WHEN** a second content-bearing turn starts while the first turn lease is active
- **THEN** the second turn MUST be rejected before upstream side effects
- **THEN** it MUST NOT overwrite the first turn's lease or output
- **THEN** the token-bound lease duration MUST exceed every supported transport's maximum turn duration

### Requirement: Incremental audit must use trusted prior output and current input

A CLEAN continuous conversation SHALL audit the exact sanitized AI output captured after the previous downstream transformations together with all canonical current client-controlled text. Client-supplied historical assistant text MUST NOT replace the stored prior output.

#### Scenario: A stable conversation continues

- **WHEN** config version, static context, parent continuity, and replay fingerprints match the CLEAN checkpoint
- **THEN** blocking Prompt Audit MUST scan the stored prior AI output and current client-controlled input
- **THEN** it MUST NOT rescan already fingerprinted older history

#### Scenario: Any incremental chunk is blocked

- **WHEN** Guard returns Block for any prior-output or current-input chunk
- **THEN** the entire turn MUST be blocked before upstream dispatch
- **THEN** the conversation MUST remain FULL_REQUIRED

### Requirement: Full replay must not bypass historical blocking

For a request without an authoritative parent continuation, the gateway SHALL verify that its non-current canonical history is exactly the prior audited input sequence followed by the captured AI output sequence.

#### Scenario: A client inserts safe-latest historical text

- **WHEN** a client inserts, deletes, rewrites, or reorders historical content and appends a safe latest user message
- **THEN** replay continuity MUST fail
- **THEN** the complete current request body MUST be scanned
- **THEN** a blocked historical chunk MUST block the entire request

#### Scenario: The client performs legitimate full replay

- **WHEN** historical input and captured output fingerprints match exactly
- **THEN** the turn MAY use incremental scanning

### Requirement: Context and parent changes must invalidate incremental eligibility

Configuration and continuation references SHALL be treated as checkpoint inputs, not as trusted transcripts.

#### Scenario: System or tool context changes

- **WHEN** present system/developer instructions or tool definitions differ from the CLEAN context hash
- **THEN** the request MUST receive a full scan

#### Scenario: A known latest parent continues

- **WHEN** `previous_response_id` resolves to the same checkpoint and equals its latest successful response id
- **THEN** omitted unchanged static context MAY inherit the checkpoint hash
- **THEN** the turn MAY use incremental scanning

#### Scenario: A parent is unknown or stale

- **WHEN** a parent mapping is missing, expired, belongs to another API key, or identifies an older branch
- **THEN** the request MUST NOT use the stored latest output as that parent's output
- **THEN** it MUST receive a full scan of canonical text actually present in the request
- **THEN** the gateway MUST NOT claim that absent parent history was re-audited

#### Scenario: A known parent also carries replayed history

- **WHEN** a known latest parent request contains non-current canonical history
- **THEN** parent identity alone MUST NOT authorize incremental scanning
- **THEN** the replayed history MUST also match the prior input and output fingerprints

### Requirement: Downstream output capture must be complete, bounded, and protocol-aware

The checkpoint output SHALL represent data successfully written to the client after gateway transformations. Media and opaque encrypted data SHALL be sanitized, the retained output SHALL be application-encrypted in Redis, and an over-limit output SHALL invalidate rather than truncate a checkpoint.

#### Scenario: HTTP or SSE succeeds

- **WHEN** a 2xx JSON response is valid, or an SSE stream reaches its protocol success terminal
- **THEN** the bounded sanitized final output and response id MUST be eligible for atomic CLEAN commit

#### Scenario: Responses WebSocket succeeds

- **WHEN** any supported WS ingress mode writes a transformed success-terminal frame to the client
- **THEN** the same final frame sequence MUST establish the output checkpoint

#### Scenario: Live Sideband succeeds

- **WHEN** a Live call's audited control turn writes a success-terminal server frame
- **THEN** the call alias MUST resolve to the same checkpoint and the final output MUST be committed

#### Scenario: Output is incomplete or ambiguous

- **WHEN** output overflows, cannot be parsed, lacks a success terminal, reports failure/incomplete/cancelled, or fails during downstream write
- **THEN** CLEAN MUST NOT be committed
- **THEN** the next content-bearing turn MUST require full audit

### Requirement: Legacy latest-turn configuration must not misrepresent runtime behavior

The administration console SHALL describe conversation checkpoint behavior. A legacy latest-turn JSON field MAY remain during rolling upgrades but MUST NOT control the new blocking state machine.

#### Scenario: An administrator saves Prompt Audit configuration

- **WHEN** the new runtime receives either legacy latest-turn value
- **THEN** blocking scope MUST still be selected from checkpoint state and continuity
- **THEN** the console MUST NOT offer the legacy value as an active policy control
