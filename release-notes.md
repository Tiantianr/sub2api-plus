Sub2API Plus v0.1.183+custom.906

## Highlights

- Add first-class per-user access control for OpenAI OAuth/Codex accounts while
  preserving existing group, subscription, API key, and session-sharing rules.
- Provide an administrator authorization matrix with impact preview, optimistic
  revision checks, step-up authentication, and atomic batch updates.
- Enforce restricted account access across HTTP scheduling, sticky sessions,
  previous responses, WebSocket turns, and Live sideband traffic.

## Changed

- Add normalized account policy and user grant tables in migration 236. Missing
  policies and public mode preserve existing scheduling behavior.
- Support restricted mode, future-user default grants, root-account policy
  ownership, and Spark shadow inheritance without exposing OAuth identities.
- Propagate authorization snapshots through scheduler metadata and outbox events,
  with durable post-slot checks for stale cache and revocation safety.
- Add audited admin APIs and the `/admin/openai-oauth-access` matrix with user,
  account, status, and authorization filters.

## Fixed

- Prevent sticky bindings, `previous_response_id`, scheduler cache staleness,
  WebSocket continuation, or Live sideband traffic from bypassing a revoked
  OpenAI OAuth authorization.
- Keep policy and grant reads atomic and fail closed when durable authorization
  cannot be verified.
- Use a consistent account-before-policy database lock order so concurrent user
  registration and policy updates cannot form a deadlock cycle.

## Compatibility and migration

- Migration 236 creates the policy and grant tables plus the future-user default
  grant trigger. Existing OpenAI OAuth accounts remain public until an
  administrator explicitly restricts them.
- Enabling a future-user default does not backfill existing users; disabling it
  does not revoke grants already issued.
- Before rolling back to software that does not understand these access-control
  tables, switch every affected account policy back to public mode.
- No Compose, port, certificate, proxy, or persistent-volume change is required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- An already-running SSE request may finish after authorization is revoked;
  subsequent HTTP requests, WebSocket turns, and Live sideband frames revalidate.
- One policy update accepts at most 1000 submitted user IDs per account.
- The authorization console manages root OpenAI OAuth accounts only; Spark
  shadows intentionally inherit and are not editable as separate credentials.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
