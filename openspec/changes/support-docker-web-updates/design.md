## Persistent runtime binary

The image retains `/app/sub2api` as its immutable bootstrap binary and embeds a
build identity derived from the exact bootstrap binary SHA-256. The
entrypoint runs the application from `/app/data/.sub2api-runtime/sub2api`, which
is on the deployment's existing persistent data mount and is writable by the
non-root application user.

On first startup, the root entrypoint performs best-effort data-mount ownership
preparation. A non-root child copies the image binary through an exclusive
temporary directory and sets executable permissions. Only after that succeeds
does the root entrypoint flush the runtime filesystem, atomically commit and
flush the image identity on the trusted state mount, and launch the application
as UID 1000. The existing update service therefore
resolves its own executable inside the persistent runtime directory and can reuse
its current same-filesystem download, checksum, backup, and atomic rename flow.

## Trusted image precedence and recovery

Compose mounts a separate root-only state directory outside `/app/data`. The
entrypoint records the SHA-256 identity of the image binary there only after a
non-root seed succeeds. The application process cannot read or replace that
trusted marker. If the selected image has the same identity, the entrypoint
preserves a website-installed or website-rolled-back binary. If the identity
changes, the operator has explicitly changed or rolled back the image, so the
image binary replaces the persisted runtime binary and any stale local backup is
removed.

An interrupted image seed leaves either the previous executable or a complete
new executable. If a website update stops between moving the old executable to
`.backup` and installing the replacement, the next startup restores that
complete backup. A missing executable without a valid backup, or a mismatched
identity, causes the next startup to seed from the image again. The immutable
image binary remains the recovery source and is never writable by the
application process.

## Security boundary

The design deliberately avoids mounting `/var/run/docker.sock`, invoking Docker
from the API process, or adding a privileged updater. The website continues to
use the existing administrator-only endpoints, system-operation lock, trusted
GitHub download hosts, archive size limits, checksum validation, and explicit
restart action. This change grants write access only to the executable copy in
the already writable application data mount.

The container entrypoint must retain its default root user so it can maintain
the root-only identity marker. The application binary itself always runs as UID
1000.

## Rollout

The first image containing this behavior must be installed through the existing
Compose deployment procedure because older images do not run from the persistent
runtime path. After that one bootstrap, website updates survive normal Docker
restarts and recreations. Updating the Compose image remains a supported explicit
override and recovery path.
