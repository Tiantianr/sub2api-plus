Sub2API Plus v0.2.0+custom.908

## Highlights

- Support Codex in-input `additional_tools` declarations during fresh-turn OpenAI OAuth history admission.
- Tool declarations declared within input items are treated as tool definitions and no longer falsely rejected as external history.
- Immutable security audit and moderation extraction contracts remain preserved and fail-closed.

## Changed

- Canonical extractor recognizes `additional_tools` items on fresh turns alongside messages and agent messages.
- Audits tool schemas inside `additional_tools` under `SourceToolDefinition` while ensuring missing schemas or history siblings fail closed.
- Real-payload Codex test suite extended to cover in-input `additional_tools`, sampled models, and WebSocket turns.

## Compatibility and migration

- Forward-only database migration schema unchanged.
- Roll back application code to `v0.2.0+custom.907` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.003
Plus commit: 6ba64e382edf9fa7c33fb3e37afe6ba219ce2003
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
