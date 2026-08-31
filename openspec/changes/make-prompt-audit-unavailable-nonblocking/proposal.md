# Make Prompt Audit unavailability non-blocking

## Problem

The current failure-allow policy classifies only selected Guard outages as
eligible. Deterministic Guard 4xx responses, missing Guard capacity, recovery
review outages, and required asynchronous admission dependencies can still
surface `prompt_guard_unavailable` as a gateway 503. This violates the product
requirement that Prompt Audit availability must never determine whether a user
request can proceed.

## Proposal

- Treat every final `prompt_guard_unavailable` outcome as non-blocking at the
  gateway boundary, regardless of Guard HTTP status, retryability, user
  exemption, recovery state, or infrastructure source.
- Remove the configurable `allow_on_guard_unavailable` switch and its narrow
  eligibility classification.
- Preserve failure evidence and pending recovery state without creating Allow
  receipts or certifying partially reviewed content as Safe.
- Keep known findings, invalid Guard output, incomplete extraction,
  configuration-version races, encryption failures, and Content Moderation
  independently fail closed.

## Non-goals

- Treating a partial Guard result as a complete Safe result.
- Clearing an existing recovery finding because Guard was unavailable.
- Allowing malformed Guard output or incomplete canonical extraction.
- Changing Content Moderation admission behavior.

## Impact

The Prompt Audit configuration API no longer publishes or accepts
`allow_on_guard_unavailable`; older persisted values are ignored and removed on
the next configuration save. No SQL migration or dependency change is needed.
Prompt Audit outage events and metrics remain visible, but they no longer
produce a user-facing 503.
