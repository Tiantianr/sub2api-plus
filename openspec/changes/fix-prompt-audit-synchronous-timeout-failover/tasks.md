## 1. Contract

- [x] 1.1 Replace the shared synchronous deadline with per-node, per-chunk
  timeout semantics.
- [x] 1.2 Preserve parent cancellation, ordered retryability, and complete
  fail-closed chunk coverage.

## 2. Implementation

- [x] 2.1 Scope each synchronous scanner call to the active endpoint timeout.
- [x] 2.2 Clarify the administration timeout label in English and Chinese.

## 3. Verification

- [x] 3.1 Cover first-node full timeout followed by successful second-node
  review and parent cancellation without failover.
- [x] 3.2 Run focused backend, race, lint, frontend, and strict OpenSpec checks.
