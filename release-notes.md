Sub2API Plus v0.1.183+custom.905

## Highlights

- Make the administrator website update and rollback flow work for Docker
  deployments without exposing the Docker socket to the application.
- Persist verified website-installed binaries across process, container, and
  host restarts while keeping an explicit Compose image change authoritative.
- Retain the non-root application runtime and immutable Linux arm64 release
  model with a separate root-only image identity state.

## Changed

- Run Docker applications from `/app/data/.sub2api-runtime/sub2api` and bind
  the selected image identity to an independent root-only persistent mount.
- Seed, recover, and replace the persistent executable as UID 1000; root only
  prepares mounts and commits the trusted image identity after filesystem sync.
- Derive image build identity from the exact bootstrap binary SHA-256 and add
  protected arm64 smoke coverage for update, interruption, tampering, and image
  override paths.
- Synchronize the legacy deployment Dockerfile's frontend memory budget with
  the production image build.

## Fixed

- Eliminate Docker website update failures caused by the non-root process being
  unable to create `/app/.sub2api-update-*` beside the image binary.
- Prevent a container recreation from silently discarding a website-installed
  binary, and recover a complete `.backup` after an interrupted replacement.
- Reject update and versioned rollback releases that do not provide the required
  `checksums.txt` asset instead of installing an unverified executable.
- Prevent application-writable state from forging the selected image identity
  or blocking an explicit image update or rollback.

## Compatibility and migration

- Existing Docker deployments require one conventional Compose image upgrade
  to this release and the new `/app/.sub2api-update-state` persistent mount.
  Older images cannot bootstrap this capability from the website.
- After that bootstrap, website updates remain staged until the administrator
  uses the existing website restart action; Docker then starts the new binary.
- Keep the image's default container user. The entrypoint starts as container
  root to maintain trusted state and launches the application itself as UID 1000.
- Local-directory deployments must preserve `update_state/`; named-volume
  deployments must preserve `sub2api_update_state` together with application
  data when migrating hosts.
- No database migration is included. Direct binary and systemd deployments are
  unchanged except that update assets now require `checksums.txt`.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Activating this capability on an existing `.903` or `.904` Docker deployment
  still requires one normal container replacement during a maintenance window.
- The running process continues serving its current version until the website
  restart action is used after an update or rollback.
- Direct binary/systemd deployment can require manual `.backup` restoration if
  the host stops exactly between the two executable rename operations.
- Multiple application containers must not share the same runtime and trusted
  state mounts concurrently.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
