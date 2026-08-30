# Design: Wait for Prompt Audit recovery claims

## Claim wait loop

One Redis Lua operation reads the current finding and conditionally creates the
bounded claim with `NX` and `PX`. Claims use independent 128-bit random owner
tokens. The operation returns missing, busy, or acquired together with the
exact finding token. A busy result starts polling at 100ms with
exponential backoff capped at one second and deterministic per-request jitter.
The loop has no independent deadline and exits when the request context ends,
Redis returns an error, the finding is cleared, or the request acquires the
claim.

## Safety

Only one request enters uncached recovery review for a user. Waiters never
replace the claim or finding. If recovery retains the finding, one waiter may
acquire the released claim and perform the next complete review. Redis errors
remain `prompt_guard_deep_review_state_unavailable`; request cancellation or
deadline ends waiting without clearing either key.

An exact Allow clears a finding only through one Lua operation that verifies
both the exact finding token and current claim token. An expired or replaced
claim owner therefore cannot clear the finding. Claim release remains
token-checked and best-effort after the decision.

## Observability

One wait-started event and one wait-finished event are emitted per waiting
request. They contain bounded request metadata, status, and elapsed time only.
They exclude prompts, raw claim/finding tokens, credentials, and Guard responses.
