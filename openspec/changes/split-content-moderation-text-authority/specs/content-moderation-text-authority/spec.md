## ADDED Requirements

### Requirement: Prompt Guard must be the sole blocking text authority where it applies

The gateway SHALL treat blocking Prompt Guard as the authoritative external text decision only for requests included by its active group scope. Content Moderation SHALL retain local keyword/hash checks and image decisions independently.

#### Scenario: Prompt Guard covers the request in auto mode

- **WHEN** Prompt Guard blocking is active for the request's group and Content Moderation text API mode is `auto`
- **THEN** Content Moderation text API results MUST NOT block the request
- **THEN** the Prompt Guard text decision MUST remain authoritative
- **THEN** current images MUST still follow the Content Moderation global mode

#### Scenario: Prompt Guard does not cover the request

- **WHEN** the group is outside Prompt Audit scope or Prompt blocking is not active
- **THEN** `auto` MUST preserve Content Moderation's blocking text behavior

### Requirement: Text API transition modes must have explicit semantics

Content Moderation SHALL support `auto`, `blocking`, `observe`, and `off` text API policies. Missing or unknown persisted values SHALL normalize to `auto`.

#### Scenario: Administrator selects blocking

- **WHEN** `text_api_mode=blocking`
- **THEN** text MUST follow the global Content Moderation mode even when Prompt Guard also runs

#### Scenario: Administrator selects observe

- **WHEN** `text_api_mode=observe` under global pre-block mode
- **THEN** text MUST be submitted asynchronously for comparison
- **THEN** a text finding MUST NOT block, create a risk hash, send notification, count toward a ban, or disable an identity

#### Scenario: Administrator selects off

- **WHEN** `text_api_mode=off`
- **THEN** direct-user text MUST NOT be sent to the external Moderation API
- **THEN** image API, local keywords, and enabled pre-hash behavior MUST remain available

#### Scenario: Keyword-only text includes an image

- **WHEN** keyword mode is `keyword_only` and a request also contains a current user image
- **THEN** text MUST skip the external API after a keyword miss
- **THEN** the image MUST still follow the global Content Moderation mode

### Requirement: Multimodal pre-block requests must separate text and image authority

When text is non-blocking but images remain blocking, the gateway SHALL create independent sanitized API inputs so a text score cannot affect the image decision.

#### Scenario: Shadow text is flagged and image is safe

- **WHEN** one request contains text and an image, shadow text exceeds a threshold, and image moderation passes
- **THEN** Content Moderation MUST allow its stage
- **THEN** the text comparison MAY be recorded as flagged with `shadow` action
- **THEN** shadow records MUST be excluded from future violation counts

#### Scenario: Text is off and image is flagged

- **WHEN** one request contains text and an image, text API mode is off, and image moderation exceeds a threshold
- **THEN** only the image MUST be sent to the Moderation API
- **THEN** the request MUST be blocked by the image decision

### Requirement: Blocking Content Moderation dependencies must fail closed without fabricating a policy hit

An incomplete blocking extraction, hash-store failure, missing moderation credential, or synchronous image/text API failure SHALL prevent upstream dispatch with `content_moderation_unavailable` and SHALL NOT count as a violation.

#### Scenario: Image Moderation API is unavailable

- **WHEN** a pre-block image call times out, is cancelled, returns an error, or returns an invalid response
- **THEN** the request MUST receive an unavailable decision before business upstream dispatch
- **THEN** it MUST NOT be recorded as a flagged content category

#### Scenario: Blocking extraction is incomplete

- **WHEN** canonical extraction cannot completely classify a content-bearing pre-block request
- **THEN** Content Moderation MUST return unavailable rather than allow a partial audit

### Requirement: Risk Control records must identify local hash blocks

The administration UI SHALL distinguish local `hash_block`, external Moderation API `block`, local keyword block, and business-upstream `cyber_policy` actions.

#### Scenario: Administrator reviews a hash hit

- **WHEN** a record action is `hash_block`
- **THEN** the result label MUST identify it as a Hash Block rather than a generic hit
