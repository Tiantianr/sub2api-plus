# Fix synchronous Prompt Audit timeout failover

## Problem

Synchronous Prompt Audit uses the first enabled node's timeout as one shared
deadline for every chunk and failover attempt. When the first node consumes
that deadline, the next node receives an already-cancelled context and cannot
perform the promised ordered failover.

## Proposal

- Apply each enabled node's configured timeout to that node's attempt for one
  chunk.
- Give the next node its own timeout after a retryable timeout, network error,
  HTTP 429, HTTP 5xx response, or saturated node bulkhead.
- Preserve immediate parent-request cancellation, ordered priority, terminal
  handling for non-retryable responses, and fail-closed complete coverage.
- Clarify the administration label so `timeout_ms` no longer implies one
  shared evaluation budget.

## Non-goals

- Adding parallel or hedged Guard requests.
- Adding retries against the same node within one synchronous attempt.
- Changing asynchronous worker retry or failover behavior.
- Allowing partial chunk results or any implicit fail-open path.

## Impact

Synchronous requests may wait for the sum of attempted node timeouts for each
required chunk. This is the intended availability trade-off for effective
ordered failover and remains bounded by configured node and chunk limits.
