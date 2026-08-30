# Prompt Audit blocking-exempt users and strict group scope

## Problem

Prompt Audit has no per-user way to keep review evidence while suppressing
content-finding blocks. In addition, pending recovery is checked before the
configured group scope, so a user with recovery state can still be reviewed
from an unselected group and create an event.

## Proposal

- Add a versioned `blocking_exempt_user_ids` list to Prompt Audit configuration.
- Continue full Prompt Audit review and event persistence for selected users,
  but do not block them or create recovery state from Prompt Audit findings.
- Apply configured group scope before ordinary review, recovery review, and the
  final recovery fence.
- Retain existing recovery state while its user is exempt or outside scope so
  removing the exemption or returning in scope resumes enforcement.

## Non-goals

- Skipping review or hiding events for blocking-exempt users.
- Exempting Content Moderation decisions or Prompt Audit dependency failures.
- Clearing existing recovery state when configuration changes.
- Adding a new database table or endpoint.

## Impact

The existing Prompt Audit configuration API gains one additive user-ID list.
The administration policy panel reuses the existing user selector. No SQL
migration is required because Prompt Audit policy is stored as versioned JSON.
