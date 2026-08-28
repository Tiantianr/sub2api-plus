Sub2API Plus v0.1.183+custom.907

## Highlights

- End a verified personal release without creating a post-publication
  finalization pull request or triggering another PR/main CI cycle.
- Defer the verified `planned` to `published` mapping transition into the next
  already-required release pull request.
- Preserve immutable publication checks while removing redundant release
  finalization wait time.

## Changed

- Make `verify` the normal end of a personal release while retaining explicit
  immediate `finalize` as an optional protected-PR recovery path.
- Revalidate every deferred `planned` to `published` transition against its
  canonical annotated tag, successful Release workflow, immutable assets, and
  anonymous Linux arm64 GHCR image during the next release promotion.
- Enforce a base/head release mapping state machine: only the current `planned`
  row may be added, existing ancestry is immutable, and failed releases resolve
  to terminal `invalid` or `withdrawn` states.

## Fixed

- Remove the mandatory post-publication finalization PR that repeated local,
  pull-request, and merged-main validation for a one-line status update.
- Reject duplicate or malformed `UPSTREAM.md` mappings, deleted history,
  unrelated published rows, ancestry changes, and terminal-state resurrection.
- Keep failed release correction possible without falsely marking an incomplete
  publication as successful or selecting it as a rollback baseline.

## Compatibility and migration

- No database migration, API, scheduling, billing, authentication, or runtime
  behavior changes in this release.
- Normal release PR checks, merged-main checks, immutable annotated tags,
  publication monitoring, asset verification, and public image verification
  remain required.
- Existing deployment and rollback requirements from `.906` remain unchanged.
- No Compose, port, certificate, proxy, or persistent-volume change is required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- The first deferred transition occurs when preparing the release after `.907`;
  `.906` was already finalized under the previous process.
- Explicit immediate finalization remains available but intentionally uses a
  separate protected pull request and its required checks.
- This optimization removes only post-publication finalization work; it does not
  shorten the normal release PR, merged-main Actions, or Release workflow.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
