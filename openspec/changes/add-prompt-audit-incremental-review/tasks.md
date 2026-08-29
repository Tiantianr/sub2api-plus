## 1. Contract

- [x] 1.1 Define current-user selection, per-segment receipts, same-request
  handoff, dual-lane ordering, forced recovery, TTL, and fail-safe behavior.

## 2. Runtime

- [x] 2.1 Preserve canonical current-user and per-segment boundaries in
  snapshots.
- [x] 2.2 Add receipt lookup/write, same-request handoff, and incremental
  synchronous/asynchronous integration.
- [x] 2.3 Add TTL configuration, administration controls, and cache metrics.

## 3. Verification

- [x] 3.1 Cover new user turns, tool continuations, same-request reuse, repeated
  text, policy invalidation, Redis errors, late Block, and forced recovery.
- [x] 3.2 Pass backend, frontend, race, lint, build, and strict OpenSpec checks.
