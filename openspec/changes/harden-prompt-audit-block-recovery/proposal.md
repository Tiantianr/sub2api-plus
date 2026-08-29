# Harden Prompt Audit Block recovery

## Problem

A synchronous Prompt Audit Block rejects only the current request. An agent
client can submit a new, rebuilt request and continue after that new request
receives ordinary blocking Allow. Session identifiers are client-controlled,
so a session-scoped recovery key would be avoidable by changing the identifier.

## Proposal

- Persist recovery state after a synchronous Prompt Audit Block.
- Bind that state to the authenticated user so API-key, group, and session-ID
  changes cannot skip the next recovery review.
- Require the user's next request to satisfy one explicitly defined recovery
  review before upstream work resumes, then clear the state immediately.
- Use the active asynchronous deep-review module selection for recovery, with
  every user turn mandatory and all Allow receipts bypassed.
- Clear recovery after any complete exact Allow, including an automatic
  assistant or tool continuation whose currently submitted context is safe.
- Use one versioned user-level state for synchronous and asynchronous deep
  Blocks so a newer finding cannot be cleared by an older recovery request.
- Preserve existing HTTP and WebSocket error contracts for initial Block,
  required recovery, and unavailable dependencies.
- Trigger recovery for a Block from any source deliberately selected by the
  active synchronous Prompt Audit policy without attributing it as user abuse.
- Keep recovery state non-expiring so waiting cannot bypass the required
  review; exact Allow remains the ordinary clearing path.
- Add bounded structured logs and runtime counters for state creation,
  successful recovery, retained recovery, and state errors.
- Preserve pre-upstream fail-closed ordering and avoid automatic account
  punishment.

## Non-goals

- Permanently freezing the user or requiring administrator release.
- Treating a Prompt Audit finding as a Content Moderation violation.
- Relying on a client-selected API key or session ID as the recovery boundary.
- Adding a third module map dedicated only to recovery.
- Requiring the blocked content to remain present after the client removes it.
- Introducing HTTP 409 or 423 recovery responses.
- Adding a recovery-state database table, administrative listing, or manual
  clear action.

## Impact

- Prompt Audit recovery storage, synchronous evaluation, tests, protocol
  documentation, and administration text will change.
- Recovery persistence, concurrency, and observability change while existing
  asynchronous late-Block protection remains enabled.
