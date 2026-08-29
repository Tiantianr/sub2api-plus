## 1. Runtime

- [x] 1.1 Add independent synchronous and asynchronous source-module config.
- [x] 1.2 Enqueue async-deep review after combined synchronous Allow.
- [x] 1.3 Fence per-user late-Block recovery with Redis compare-and-delete.

## 2. Administration

- [x] 2.1 Add module controls and async-deep event filtering.
- [x] 2.2 Preserve encrypted complete-context evidence for deep events.

## 3. Verification

- [x] 3.1 Cover selection, side-effect order, enqueue failure, late Block,
  recovery, concurrency fencing, event filtering, and migration behavior.
- [x] 3.2 Pass backend, frontend, migration, race, lint, build, and strict
  OpenSpec checks.
- [x] 3.3 Add shared value-free extraction-failure diagnostics and verify both
  engines never log raw failed-node values.
