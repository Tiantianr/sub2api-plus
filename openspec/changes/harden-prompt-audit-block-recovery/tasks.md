## 1. Contract

- [x] 1.1 Define the recovery scope for synchronous Prompt Audit Blocks.
- [x] 1.2 Define identity changes and missing session identity behavior.
- [x] 1.3 Define recovery module coverage and receipt behavior.
- [x] 1.4 Define recovery clearing and error semantics.
- [x] 1.5 Define interaction with asynchronous late Blocks.
- [x] 1.6 Preserve client-facing HTTP and WebSocket error contracts.
- [x] 1.7 Define which synchronous selected sources trigger recovery.
- [x] 1.8 Define recovery lifetime and clearing authority.
- [x] 1.9 Define bounded operational observability.

## 2. Runtime

- [x] 2.1 Persist and enforce user-scoped single-use recovery state.
- [x] 2.2 Add bounded recovery logs and runtime counters.

## 3. Verification

- [x] 3.1 Cover identity changes, API-key changes, recovery, Redis failures, and
  pre-upstream side-effect ordering.
- [x] 3.2 Update protocol and security coverage documentation.
- [x] 3.3 Pass focused backend, race, lint, docs, and strict OpenSpec checks.
