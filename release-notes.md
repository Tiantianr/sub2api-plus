Sub2API Plus v0.2.0+custom.907

## Highlights

- Fix OpenAI OAuth history admission to permit multi-user context messages on Codex first turn.
- Codex first turns with project instructions, environment context, and prompts no longer falsely trigger external history rejection.
- Immutable security audit and moderation extraction contracts remain preserved and fail-closed.

## Changed

- Canonical extraction `HistoryBearing` classification decoupled from audit `Current` segment marker.
- History classification strictly flags assistant/model messages, tool calls/outputs, reasoning, opaque state, and continuation references.
- Multiple user messages alone do not establish prior conversation turns or require historical session affinity.
- Added real-payload Codex first-turn test suite for canonical extraction, HTTP routing, and WebSocket turns.

## Compatibility and migration

- Forward-only database migration schema unchanged.
- Roll back application code to `v0.2.0+custom.906` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.003
Plus commit: 6ba64e382edf9fa7c33fb3e37afe6ba219ce2003
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
