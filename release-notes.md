Sub2API Plus v0.1.178+custom.902

## Highlights

- Synchronize the published `LuckyKuang/sub2api-plus`
  `v0.1.178+custom.003` release into the personal distribution while preserving
  its independent update, release, and GHCR channels.
- Harden the shared security-audit content-extraction contract so Content
  Moderation and Prompt Audit consume the same canonical protocol document.
- Restore Content Moderation to current direct-user text and images so a
  policy violation is attributed only to a user submission, not to platform
  or tool content.
- Fail closed on incomplete or partial extraction whenever a blocking audit
  mode is active, including sibling content that would otherwise hide a failed
  field.
- Prebuild the exact Linux arm64 image during protected main validation so an
  explicitly published release tag can make the immutable GHCR image available
  within a five-minute runner execution budget.
- Surface extraction metrics in risk-control and prompt-audit runtimes, and
  document the protocol matrix in `docs/SECURITY_AUDIT_CONTENT_COVERAGE.md`.

## Changed

- Route every content-bearing HTTP request, WebSocket turn, and Live Sideband
  client frame through `backend/internal/auditcontent` after authentication
  and before account selection, billing, concurrency, routing, or upstream
  writes.
- Select only protocol-defined direct-user content for Content Moderation while
  keeping instructions, tool traffic, and assistant or model content available
  to Prompt Audit.
- Keep update checks, installers, repository links, release tooling, and OCI
  metadata bound to `Tiantianr/sub2api-plus`.
- Reuse successful CI and Security Scan provenance at release time instead of
  repeating the complete matrix, and avoid duplicate branch-push and PR runs.
- Remove Apple Containers deployment scripts, documentation, environment
  settings, and validation; supported deployments remain Linux Docker Compose
  and the Linux binary installer.
- Publish only Linux arm64 images and archives; Linux amd64, Darwin, and Windows
  artifacts are no longer produced by this personal distribution.
- Preserve the official v0.1.178 baseline and embedded Codex identity
  precedence.

## Fixed

- Restore the `v0.1.177+custom.003` user-attribution rule that a later
  shared-extractor expansion had broadened beyond direct-user content.
- Treat a partial extraction hidden by successful sibling content as an
  extraction failure, not a successful or empty request.
- Satisfy audit-content lint after the extraction-scope change.

## Compatibility and migration

- Existing data remains compatible; this iteration adds no database migration.
- Hosts must run Linux arm64 to use new personal images or binary archives.
- Back up PostgreSQL and configuration before replacing an older application
  version because inherited migrations remain forward-only.

## Known issues

- Docker in-place binary updates live in the container writable layer. Keep the
  Compose image tag aligned with the running release before recreating a
  production container.
- GitHub-hosted runner queue time is external to the five-minute image-job
  execution budget; a missing validated main artifact stops publication.

## Upstream baseline

Plus release: v0.1.178+custom.003
Plus commit: bfa1220152a309ec94a5fed52f02fbceccc27055
Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
