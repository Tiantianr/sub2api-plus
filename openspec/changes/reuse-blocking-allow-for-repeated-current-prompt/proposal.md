# Reuse blocking Allow for repeated current prompts

## Problem

Prompt Audit writes a user-scoped, policy-bound receipt after an exact complete
synchronous Allow, but deliberately ignores stored receipts whenever the same
canonical segment is marked as the current user turn. Workloads that submit a
fixed instruction with changing media therefore pay for and wait on the same
text review on every request.

## Proposal

- In blocking mode, allow a current user segment to reuse an unexpired stored
  Allow receipt for the same user, configuration, Guard policy, source, and
  receipt-equivalent canonical text.
- Normalize only strict user-text `[images:<hex>]` media markers to a stable
  receipt placeholder while sending the original text to Guard on every miss.
- Keep asynchronous-only current user segments mandatory unless they receive
  the existing trusted same-request handoff from blocking review.
- Preserve receipt misses and Redis errors as ordinary Guard review.
- Preserve forced recovery as a complete uncached deep review.
- Continue running Content Moderation independently on every applicable
  request, including changing media.

## Non-goals

- Adding administrator-managed prompt allowlists.
- Sharing receipts between users or across Prompt Audit configuration changes.
- Caching Warn, Block, invalid, timeout, extraction failure, or partial results.
- Treating media identity as part of Prompt Audit's text receipt key.
- Changing receipt TTL, storage schema, or recovery behavior.

## Impact

Repeated fixed text can skip both blocking Prompt Audit and its redundant deep
job while the configured receipt remains valid, even when only a recognized
opaque media identifier changes. Other new or changed canonical text still
runs normal synchronous Guard review and remains fail closed.
