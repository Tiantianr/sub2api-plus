Sub2API Plus v0.2.0+custom.901

## Highlights

- Synchronize the published Plus `v0.2.0+custom.002` release while retaining the
  personal fork's security-audit, Codex identity, billing, deployment, and
  release behavior.
- Add client-disconnect lifecycle tracking, ordered streak enforcement,
  automatic user disabling, and administrator event review.
- Add durable Content Moderation session blocks, endpoint failover, and bounded
  redacted input retention for administrator review.
- Add Codex fingerprint convergence (`codex_fingerprint_mode`) for OpenAI API Key
  accounts, preventing upstream sticky-session routing lock on degraded accounts.
- Support Pi Agent wire identity simulation (`pi/0.85.0`) and one-click channel
  switching in the Admin UI.

## Changed

- Allow administrators to manage user access grants for both OpenAI OAuth and OpenAI API Key accounts.
- Extend Codex fingerprint convergence (`off`, `device`, `session`, `full`) to OpenAI
  API Key accounts; unconfigured API Key accounts default to `off` for 100% backward compatibility.
- Support Pi Agent client profile and wire identity across OAuth refresh and gateway paths.
- Preserve the redesigned Billing & Subscription checkout and multi-currency
  recharge presentation introduced in the previous personal release.
- Prompt Audit scans the canonical client-controlled transcript; latest-turn
  blocking prioritizes current input while retaining the personal asynchronous
  deep-review, exemption, fail-closed, evidence, and session-analysis policies.
- Content Moderation stores bounded redacted `input_content` and no longer keeps
  the superseded encrypted `input_ciphertext` evidence chain.

## Fixed

- Settle admitted OpenAI HTTP and WebSocket usage after client disconnects
  without silently dropping lifecycle or billing outcomes.
- Trigger failover for terminal failures returned through successful HTTP
  responses and keep PostgreSQL authoritative for session blocks.
- Avoid rendering empty IP last-seen values as permanent bans.

## Compatibility and migration

- Forward-only database migrations add moderation observability, native
  compaction metadata, OpenAI Fast controls, disconnect lifecycle state,
  durable moderation session blocks, redacted moderation input, and OpenAI API
  Key Codex fingerprint support (migration 264). A final migration removes
  the superseded `input_ciphertext` column.
- Existing published migrations remain byte-for-byte unchanged; imported
  migrations are renumbered above the personal fork's existing `248` boundary.
- Existing user subscriptions, payment orders, provider configurations, and
  Prompt Audit evidence remain compatible.
- Roll back application code to `v0.1.183+custom.927` if required; database
  migrations remain forward-only.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.002
Plus commit: cd1d8438cbe19358936605af7e6b20954283bf15
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
