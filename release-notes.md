Sub2API Plus v0.1.178+custom.901

## Highlights

- Establish the first `Tiantianr/sub2api-plus` release while preserving the
  complete `v0.1.178+custom.001` Plus application behavior.
- Publish checksummed release binaries and multi-architecture GHCR images from
  the personal GitHub release channel.
- Document the fork, upstream synchronization, custom development, release,
  ID3 deployment, and rollback workflow.

## Changed

- Point update checks, rollback links, installers, pricing manifests, user-facing
  repository links, and deployment examples to `Tiantianr/sub2api-plus`.
- Allow the protected push and release CLIs to target an explicitly trusted fork
  through `SUB2API_EXPECTED_REPOSITORY`, while retaining the original fail-closed
  default.
- Reserve custom iterations `901` through `999` for personal releases so they
  remain distinct from the inherited Plus release history.

## Compatibility and migration

- Database schema and application behavior are unchanged from
  `v0.1.178+custom.001`; this release adds no migration.
- Existing deployments remain compatible. Back up PostgreSQL and configuration
  before replacing an older application version because inherited migrations
  remain forward-only.

## Known issues

- Docker in-place binary updates live in the container writable layer. Keep the
  Compose image tag aligned with the running release before recreating a
  production container.

## Upstream baseline

Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
