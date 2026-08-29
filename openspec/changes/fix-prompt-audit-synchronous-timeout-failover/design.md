## Context

`GuardEvaluator.Evaluate` currently derives one context deadline from the
first enabled endpoint before splitting and scanning chunks. `scanChunk`
correctly recognizes timeout as retryable, but every later endpoint inherits
the expired context after the first endpoint reaches that deadline.

## Decisions

### 1. Scope timeout to one node attempt for one chunk

`Evaluate` will retain the caller context without adding a first-node
deadline. `scanChunk` will derive a child context from the current endpoint's
`timeout_ms` for each scanner call and cancel it immediately when the call
returns.

This matches the existing asynchronous scanner contract and gives ordered
failover real capacity to recover. Total evaluation duration remains bounded
by the number of required chunks and the sum of enabled endpoint timeouts.

### 2. Parent cancellation remains authoritative

If the ingress request context is cancelled, scanning stops without trying
another node. An endpoint-local deadline is normalized to a retryable timeout;
it may advance to the next node. A scanner result returned after its local
deadline is not accepted.

### 3. Preserve terminal and fail-closed behavior

Only existing retryable failures advance to the next endpoint. Authentication
failure, other non-retryable HTTP responses, invalid Guard output, and parser
errors remain terminal. Every required chunk must still produce a valid result
before Allow, and Block may still stop early.

### 4. Make configuration text describe the actual boundary

The node editor will label `timeout_ms` as a per-node, per-chunk timeout. No
configuration field, storage format, API, default, or valid range changes.

## Risks

- A long multi-chunk request can wait longer than the former first-node total
  budget. The administrator controls this bound through per-node timeout and
  the minimum enabled input limit; introducing another total-budget field is
  deferred until runtime data demonstrates that it is needed.
