Sub2API Plus v0.1.178+custom.902

## Highlights

- Synchronize the published `LuckyKuang/sub2api-plus`
  `v0.1.178+custom.002` release into the personal distribution while preserving
  its independent update, release, and GHCR channels.
- Harden the shared security-audit content-extraction contract so Content
  Moderation and Prompt Audit consume the same canonical protocol document.
- Fail closed on incomplete or partial extraction whenever a blocking audit
  mode is active, including sibling content that would otherwise hide a failed
  field.
- Surface extraction metrics in risk-control and prompt-audit runtimes, and
  document the protocol matrix in `docs/SECURITY_AUDIT_CONTENT_COVERAGE.md`.

## Changed

- Route every content-bearing HTTP request, WebSocket turn, and Live Sideband
  client frame through `backend/internal/auditcontent` after authentication
  and before account selection, billing, concurrency, routing, or upstream
  writes.
- Keep update checks, installers, repository links, release tooling, and OCI
  metadata bound to `Tiantianr/sub2api-plus`.
- Preserve the official v0.1.178 baseline and embedded Codex identity
  precedence.

## Fixed

- Treat a partial extraction hidden by successful sibling content as an
  extraction failure, not a successful or empty request.
- Satisfy staticcheck tagged-switch lint in audit extraction role handling.

## Compatibility and migration

- Existing data remains compatible; this iteration adds no database migration.
- Back up PostgreSQL and configuration before replacing an older application
  version because inherited migrations remain forward-only.

## Known issues

- This sync intentionally excludes commits after the published Plus
  `v0.1.178+custom.002` tag, including its later direct-user moderation scope
  fix. Do not publish or deploy this candidate until that fix appears in a
  formal Plus release or is separately approved for adoption.
- Docker in-place binary updates live in the container writable layer. Keep the
  Compose image tag aligned with the running release before recreating a
  production container.

## Upstream baseline

Plus release: v0.1.178+custom.002
Plus commit: 0d3a8a8c4ad9e591960df5aae193347435d852d9
Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
