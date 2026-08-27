## Why

The administrator version menu downloads and atomically replaces the running
binary, but Docker runs the application from `/app/sub2api` while the non-root
process cannot create the same-directory temporary file. Even granting that
directory write access would keep the update only in the container layer, so a
later container recreation could silently restore the older image binary.

Docker deployments need the existing authenticated web update flow to write to
a persistent executable location without giving the application access to the
Docker socket or weakening the non-root runtime.

## What Changes

- Keep the image-provided binary as a read-only bootstrap copy.
- Run Docker deployments from a synchronized binary under the existing
  persistent `/app/data` mount.
- Record the selected image identity in a separate root-only persistent mount
  that the application process cannot read, replace, or forge.
- Preserve web-installed binaries across process, container, and host restarts.
- Make an explicit image change or image rollback authoritative over the
  persisted runtime binary.
- Extend the protected Linux image smoke test to verify initial seeding and
  image-authoritative recovery.
- Document the one-time bootstrap requirement and the remaining restart step.

## Capabilities

### New Capabilities

- `docker-web-update`: Allows the authenticated website update and rollback
  flow to persist verified release binaries in Docker deployments.

### Modified Capabilities

None.

## Impact

- **Docker runtime**: The process executable moves from `/app/sub2api` to
  `/app/data/.sub2api-runtime/sub2api`, with a small root-only image-state mount;
  application data formats are unchanged.
- **Security**: The container remains non-root and receives no Docker socket.
  Existing GitHub host validation, checksum verification, operation locking,
  administrator authorization, and atomic replacement remain in force.
- **Compatibility**: Direct binary and systemd deployments are unchanged.
  Existing Docker deployments require one conventional image upgrade before
  later releases can be installed from the website.
- **Operations**: A web update is staged without stopping the current process;
  the new binary becomes active only after the existing restart action.
