Sub2API Plus v0.2.0+custom.905

## Highlights

- Add OpenAI OAuth history admission policy and conversation bindings to prevent cross-account history bleed.
- Introduce database migration `266_openai_conversation_bindings.sql` for long-term session-to-account ownership tracking.
- Optimize CI and release workflows with fast local preflight and single-stage rapid arm64 image publishing channel.

## Changed

- OpenAI OAuth accounts now default to rejecting external history requests unless explicitly disabled.
- Frontend account creation, editing, and bulk-editing modals support the history admission policy toggle.
- Accelerated main push CI by skipping redundant test shards and lint already verified in pull requests.
- Added `--fast` preflight mode to `push-cli` and provided `fast-release.yml` direct publish workflow.

## Compatibility and migration

- Added forward-only migration `266_openai_conversation_bindings.sql`.
- Account settings default to `openai_oauth_reject_external_history=true`.
- Roll back application code to `v0.2.0+custom.904` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.003
Plus commit: 6ba64e382edf9fa7c33fb3e37afe6ba219ce2003
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
