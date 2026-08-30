# User-scoped Prompt Audit Pass retention

## Problem

Prompt Audit currently uses one global `store_pass_events` switch. When it is
enabled, every stored Pass event retains both the exact Guard input and an
encrypted complete canonical context. Long conversations repeat prior context
on every request, so normal traffic can dominate PostgreSQL and every logical
backup even though risk evidence is small.

Changing a user's evidence-retention preference through the main Prompt Audit
configuration would also increment the Guard policy version, invalidating
Allow receipts and making already queued jobs stale for a storage-only change.

## Proposal

- Replace global Pass-event storage with an independently versioned list of
  users whose Pass events should be retained.
- Default to an empty list, so users without an explicit selection do not
  retain Pass events or complete contexts.
- Continue storing every Flag and Critical event for every audited user.
- Keep review, blocking, metrics, deep recovery, and Allow receipts independent
  from the evidence-retention choice.
- Add a Pass-only cleanup flow that requires an explicit time range, a server
  preview, estimated retained-content bytes, and the existing bound
  confirmation token before deletion.

## Non-goals

- Deduplicating prompt text or encrypted context blobs.
- Adding automatic retention or scheduled deletion.
- Deleting risk events through the new cleanup shortcut.
- Changing Allow-receipt matching, TTL, or recovery behavior.
- Reclaiming PostgreSQL files with `VACUUM FULL`.

## Impact

The change adds one settings-backed administration resource and a data-only
forward migration that removes the obsolete global field without changing the
SQL schema. Existing Pass events remain until an administrator explicitly
cleans them. The old global value does not opt any user into the new retention
list and cannot reactivate global persistence after rollback.
