Sub2API Plus v0.1.178+custom.903

## Highlights

- Synchronize the published `LuckyKuang/sub2api-plus`
  `v0.1.178+custom.005` release into the personal distribution while preserving
  its independent update, release, and GHCR channels.
- Harden the shared security-audit content-extraction contract so Content
  Moderation and Prompt Audit consume the same canonical protocol document.
- Restore Content Moderation to current direct-user text and images so a
  policy violation is attributed only to a user submission, not to platform
  or tool content.
- Preserve independently extracted sibling content while incomplete or
  unrecognized audit input is logged and passed through without an
  audit-derived block.
- Restore Prompt Audit conversation-text selection so ordinary Codex requests
  are not blocked by static client tool schemas.
- Prebuild the exact Linux arm64 image during protected main validation so an
  explicitly published release tag can make the immutable GHCR image available
  within a five-minute runner execution budget.
- Surface extraction metrics in risk-control and prompt-audit runtimes, and
  document the protocol matrix in `docs/SECURITY_AUDIT_CONTENT_COVERAGE.md`.
- Personalize the complete web interface with the `#3c80e6` brand-blue theme,
  including balance highlights while preserving semantic and payment-brand
  colors.
- Make balance recharge the first and default purchase tab, while preserving
  explicit subscription deep links and balance-disabled behavior.
- Hide the asynchronous image-generation entry from user and administrator
  support sidebars without removing its protected direct route or backend API.

## Changed

- Route every content-bearing HTTP request, WebSocket turn, and Live Sideband
  client frame through `backend/internal/auditcontent` after authentication
  and before account selection, billing, concurrency, routing, or upstream
  writes.
- Select only protocol-defined direct-user content for Content Moderation while
  keeping instructions, tool traffic, and assistant or model content available
  to Prompt Audit.
- Exclude static `tools` and `functions` schemas, structured tool-call
  arguments, and tool or function outputs from Prompt Audit while retaining
  messages, instructions, reusable prompt variables, reasoning, and media
  prompts.
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
- Apply the personalized primary color consistently across public, user, and
  administrator views.

## Fixed

- Restore the `v0.1.177+custom.003` user-attribution rule that a later
  shared-extractor expansion had broadened beyond direct-user content.
- Prevent extraction failures from becoming policy blocks, HTTP 503 responses,
  or WebSocket closes while preserving recognized sibling content for
  independent policy evaluation.
- Keep a simple Codex `hi` request plus a large tool schema from producing a
  Prompt Audit block.

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

Plus release: v0.1.178+custom.005
Plus commit: 594d5fb2526ce4981d1ad06cd83893f075f494bb
Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
