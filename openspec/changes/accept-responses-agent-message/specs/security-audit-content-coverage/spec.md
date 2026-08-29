## ADDED Requirements

### Requirement: Responses agent messages must preserve visible content and attribution

The shared canonical extractor SHALL recognize Responses `agent_message`
items. It SHALL extract visible supported content blocks, classify recognized
authors by their canonical role, and classify other named agents as non-user
assistant sources. It MAY ignore opaque `encrypted_content` and bounded
routing/turn metadata. Unknown content-bearing blocks MUST retain ordinary
incomplete-extraction behavior.

#### Scenario: Named agent sends visible text with opaque state

- **WHEN** an `agent_message` has an unrecognized named author, visible
  `input_text`, optional `encrypted_content`, recipient, and turn metadata
- **THEN** extraction SHALL be complete
- **THEN** visible text SHALL be an assistant message available to configured
  Prompt Audit assistant review
- **THEN** Content Moderation SHALL NOT attribute it to the direct user

#### Scenario: User-authored agent message

- **WHEN** an `agent_message` has `author=user` and visible `input_text`
- **THEN** both engines SHALL receive the text according to their ordinary
  current-user selection policies

#### Scenario: Agent message contains a future content block

- **WHEN** an `agent_message` contains a non-empty unsupported content block
  other than the explicitly opaque encrypted block
- **THEN** extraction SHALL remain incomplete and observable

#### Scenario: Opaque block carries an extra visible field

- **WHEN** an `encrypted_content` block or agent-message metadata contains an
  additional non-empty field outside its strict allowlist
- **THEN** visible supported siblings SHALL remain extracted
- **THEN** extraction SHALL remain incomplete and fail closed under blocking
  audit
