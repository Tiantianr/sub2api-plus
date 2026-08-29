Sub2API Plus v0.1.183+custom.913

## Highlights

- Accept Responses `agent_message` histories without rejecting extractable
  requests, while preserving visible text and direct-user attribution.
- Reuse exact blocking Prompt Audit Allows for repeated fixed current-user
  prompts, including strict opaque media-hash marker changes.
- Give each synchronous Guard node and chunk its own timeout so a timed-out
  priority node can fail over to the next configured node.

## Changed

- Classify recognized `agent_message` authors as user, system, developer,
  assistant, or model; classify other named agents as non-user assistant
  sources and ignore only known opaque encrypted content and turn metadata.
- Advance the Allow receipt schema and permit unexpired stored current-user
  receipts in blocking mode while keeping async-only current input and forced
  recovery uncached.
- Clarify the audit-pool timeout as a per-node, per-chunk limit.
- Show user IDs in the Prompt Audit event identity column and make them
  clickable user filters.

## Fixed

- Stop supported `agent_message` items with visible `input_text` from producing
  duplicate Prompt Audit and Content Moderation extraction failures and HTTP
  503 responses.
- Let a second Guard node receive its own timeout after the first node consumes
  its configured timeout.
- Avoid repeated Guard calls for fixed text when only a strict
  `[images:<hex>]` reference changes.

## Compatibility and migration

- No database migration or new configuration field is required.
- Existing Allow receipts are invalidated by receipt schema version 2 and are
  rebuilt after one successful review.
- Synchronous worst-case audit latency is now bounded by required chunks times
  the sum of attempted node timeouts instead of the first node's single shared
  timeout.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Opaque `agent_message.encrypted_content` is not decrypted or audited; every
  visible supported sibling remains audited.
- Unknown future `agent_message` content blocks still fail closed while
  blocking audit applies.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
