# Design: non-blocking Prompt Audit availability failures

## Final availability boundary

`prompt_guard_unavailable` becomes an availability signal rather than an
admission decision. PromptService converts evaluator failures with that exact
stable code into an Allow decision. Coordinator applies the same normalization
as a final guard for unavailable engines, missing results, and required
asynchronous admission dependencies so no HTTP or WebSocket path can bypass
the invariant.

The conversion is unconditional: Guard 4xx, 401/403, 429, 5xx, timeouts,
connection failures, bulkhead saturation, missing usable nodes, scanner
construction failures, and Prompt Audit queue or payload dependencies all
continue the current request when their final stable code is
`prompt_guard_unavailable`. Retryability still controls ordered node failover;
it no longer controls user admission.

Errors with another stable code retain their existing behavior. In particular,
`prompt_guard_invalid_response`, content extraction failures, encryption
failures, configuration-version races, and recovery-state storage failures are
not normalized to Allow. Content Moderation remains an independent synchronous
authority.

## Evidence and recovery

A failure-allowed request has no normalized Safe result and never creates an
Allow receipt. Synchronous failure persistence remains best effort and stores
only the stable failure code and generic safe reason. A best-effort deep job may
still be queued with receipt writes suppressed.

If the user already has a recovery finding, Prompt Audit unavailability leaves
that finding intact, releases any temporary claim, skips the final recovery
fence for the current failure-allowed request, and attempts asynchronous review.
A later successful exact-owner review may still clear the finding under the
existing recovery contract.

## Configuration removal

The availability behavior is invariant, so an administrator switch would be
misleading. The backend storage, active/public/update DTOs, audit summary, and
frontend draft remove `allow_on_guard_unavailable`. JSON decoders continue to
ignore an older persisted or submitted field through their normal unknown-field
behavior; no compatibility branch reads or applies its value.

## Cancellation

Caller cancellation and service shutdown do not create failure events or
increment failure-allowed metrics. The request context may still terminate the
transport, but Prompt Audit does not replace cancellation with a user-facing
503.
