# Wait for Prompt Audit recovery claims

## Problem

Prompt Audit serializes user recovery with a bounded claim, but a concurrent
request currently treats an occupied claim as unavailable recovery state and
returns HTTP 503 immediately. Claim contention is expected coordination, not a
Redis or audit dependency failure.

## Proposal

- Wait for an occupied per-user recovery claim while the request context remains
  active.
- Acquire the finding and claim atomically in one Redis operation.
- Resume ordinary blocking review when the claim owner clears the finding.
- Acquire the released claim and perform recovery when the finding remains.
- Keep finding version checks, claim ownership, and fail-closed dependency
  behavior unchanged.
- Require the exact current claim owner to clear the exact finding version.
- Record bounded wait lifecycle logs without prompt or claim values.

## Non-goals

- Allowing concurrent recovery execution for one user.
- Clearing or bypassing a pending finding because a waiter times out.
- Adding a server timeout beyond the existing request context.
- Guaranteeing strict FIFO ordering among multiple waiters.

## Impact

Concurrent recovery requests may remain open longer instead of receiving an
immediate 503. No API schema, migration, dependency, or frontend change is
required.
