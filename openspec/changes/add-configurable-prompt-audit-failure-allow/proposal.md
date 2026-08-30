# Configurable Prompt Audit failure allow

## Problem

Blocking Prompt Audit currently rejects every request when all Guard nodes time
out or the Guard API is unavailable. A network or provider outage can therefore
make the gateway unavailable to every user in the configured scope.

## Proposal

- Add an explicit, default-off `allow_on_guard_unavailable` policy to the main
  Prompt Audit configuration.
- When enabled, allow an ordinary blocking request after all Guard nodes end in
  `prompt_guard_unavailable`.
- Keep known findings, invalid Guard responses, incomplete content extraction,
  configuration failures, Content Moderation, and required recovery fail closed.
- Never create an Allow receipt from a failure-allowed request.
- Expose failure-allowed events through structured logs and a runtime counter.

## Non-goals

- Treating malformed or partially understood content as safe.
- Clearing or bypassing a user's required deep review.
- Changing Content Moderation failure behavior.
- Disabling node failover or reducing configured node timeouts.

## Impact

The existing Prompt Audit settings JSON gains one backward-compatible boolean.
No SQL migration or new dependency is required. Existing installations preserve
fail-closed behavior until an administrator explicitly enables the policy.
