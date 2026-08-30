# Optimize Prompt Audit hot path

## Problem

The synchronous audit coordinator deep-copies the same immutable request body
for both audit engines and again before asynchronous handoff. Prompt chunking
also materializes and copies a full `[]rune` representation. Large Responses
payloads therefore create avoidable memory traffic before Guard evaluation.

## Proposal

- Share the frozen request body read-only between synchronous Content
  Moderation and Prompt Audit branches.
- Keep one deep copy only when asynchronous work crosses the request lifetime.
- Split UTF-8 Guard input at rune boundaries by byte offsets without allocating
  a full rune slice or copying each chunk.
- Count chunk runes without temporary rune allocations.

## Non-goals

- Sharing mutable request state with background work.
- Changing canonical extraction, selected content, chunk ordering, or limits.
- Parallelizing required chunks or reducing complete Guard coverage.
- Changing fail-closed, receipt, finding, or evidence behavior.

## Impact

No API, configuration, migration, dependency, or frontend change is required.
The change reduces peak allocation and copy volume for large request bodies
while retaining identical audit input and side-effect order.
