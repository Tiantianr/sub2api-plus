## ADDED Requirements

### Requirement: Security audit input must cover effective upstream text semantics

Content Moderation and Prompt Audit SHALL receive every client-controlled text value that an accepted request can cause the gateway to send as upstream prompt or conversation text. Post-audit compatibility conversion, account selection, and fallback routing MUST NOT introduce unaudited client text.

#### Scenario: Responses receives legacy messages without native input

- **WHEN** an accepted Responses request contains legacy `messages` and no non-null native `input`
- **THEN** the canonical extractor MUST expose the same message text that ingress compatibility converts into upstream `input`
- **THEN** both audit engines MUST apply their existing selection policy before routing or account selection

#### Scenario: Responses receives a legacy string prompt

- **WHEN** an accepted Responses request contains a string `prompt` and no non-null native `input`
- **THEN** the prompt MUST be treated as current direct-user input by both audit engines

#### Scenario: Native Responses input is authoritative

- **WHEN** a Responses request contains non-null native `input` together with legacy `messages` or a string `prompt`
- **THEN** audit extraction MUST select native `input` and MUST NOT scan aliases that ingress compatibility deletes

#### Scenario: Alpha Search can use PAT fallback

- **WHEN** Alpha Search contains non-empty `commands`, `settings`, or `input`
- **THEN** all three values MUST be available to both audit engines before account selection
- **THEN** selecting a PAT account MUST NOT add client-controlled text to the fallback user prompt that was absent from audit input

### Requirement: Supported visible and opaque history fields must be classified correctly

The canonical extractor SHALL retain visible conversation reasoning and visible/unknown compaction fields while treating only supported encrypted compaction fields as known opaque data.

#### Scenario: Chat carries reasoning content

- **WHEN** a Chat message contains non-empty `reasoning_content`
- **THEN** full Prompt Audit MUST include that text as reasoning content
- **THEN** Content Moderation MUST attribute user-role reasoning to the direct user and MUST NOT attribute assistant/model reasoning to the direct user

#### Scenario: Responses carries compaction state or trigger

- **WHEN** Responses input contains `compaction`, `compaction_summary`, or `compaction_trigger`
- **THEN** extraction MUST recognize the item without persisting encrypted or opaque state
- **THEN** pure known opaque/control fields MUST NOT produce an incomplete extraction warning
- **THEN** summary text and unknown non-empty visible fields MUST remain auditable
- **THEN** visible sibling user content MUST remain auditable

### Requirement: Untrusted reminder markup must not suppress direct-user moderation

Content Moderation SHALL treat reminder-like markup in client-provided direct-user text as ordinary untrusted text. A marker MUST NOT suppress text inside or outside the marked block.

#### Scenario: User text contains a system-reminder block and a question

- **WHEN** current direct-user text contains `<system-reminder>...</system-reminder>` followed by or surrounding ordinary user text
- **THEN** Content Moderation MUST receive the complete normalized direct-user text
- **THEN** Prompt Audit MUST continue to receive the same canonical message content

### Requirement: Administration controls must reflect effective blocking behavior

The administration console MUST NOT present a mutable setting that the active Prompt Guard runtime ignores. Compatibility fields MAY remain in storage and API schemas when removing them would break existing clients.

#### Scenario: Administrator edits blocking configuration

- **WHEN** the active runtime uses conversation checkpoints instead of a static latest/full policy
- **THEN** the console MUST NOT render a latest-turn toggle
- **THEN** saving unrelated Prompt Audit settings MUST remain possible without changing the runtime policy

### Requirement: Semantic parity must have production-shaped regression coverage

The security-audit suite SHALL exercise canonical extraction, Content Moderation selection, and Prompt Audit selection using payloads shaped like accepted HTTP and WebSocket traffic.

#### Scenario: Security-audit regressions run

- **WHEN** repository validation executes
- **THEN** tests MUST cover Responses legacy aliases, system-reminder text, Alpha Search fallback fields, Chat reasoning content, and supported compaction items
- **THEN** route-order tests MUST continue proving that audit completes before account, billing, concurrency, and upstream side effects
