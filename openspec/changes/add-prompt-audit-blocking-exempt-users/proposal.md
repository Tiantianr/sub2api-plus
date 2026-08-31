# Prompt Audit blocking-exempt users and strict group scope

## Problem

Prompt Audit has no per-user way to keep review evidence while suppressing
content-finding blocks. In addition, pending recovery is checked before the
configured group scope, so a user with recovery state can still be reviewed
from an unselected group and create an event.

## Proposal

- Add a versioned `blocking_exempt_user_ids` list to Prompt Audit configuration.
- Reliably enqueue full asynchronous Prompt Audit review for selected users
  without waiting for synchronous Guard, while preserving event evidence.
- Persist whether the user was exempt when the request was admitted so event
  history remains immutable and later removal cannot enable recovery for that
  already admitted job.
- Apply configured group scope before ordinary review, recovery review, and the
  final recovery fence.
- Retain existing recovery state while its user is exempt or outside scope so
  removing the exemption or returning in scope resumes enforcement.

## Non-goals

- Best-effort enqueue or hiding events for blocking-exempt users.
- Exempting Content Moderation decisions, content extraction, encryption, or
  queue-admission failures.
- Clearing existing recovery state when configuration changes.
- Adding a new database table or endpoint.

## Impact

The existing Prompt Audit configuration API gains one additive user-ID list.
The administration policy panel reuses the existing user selector. Prompt
Audit jobs and events gain a request-time exemption marker used by workers and
the existing event list.
