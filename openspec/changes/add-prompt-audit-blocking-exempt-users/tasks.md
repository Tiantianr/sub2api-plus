# Tasks

## 1. Contract

- [x] 1.1 Define strict group-scope and blocking-exemption semantics.
- [x] 1.2 Document recovery-state retention and unchanged failure behavior.

## 2. Backend

- [x] 2.1 Add versioned blocking-exempt user configuration.
- [x] 2.2 Apply group scope before recovery and suppress finding enforcement for exempt users.
- [x] 2.3 Prevent asynchronous findings from creating exempt-user recovery state.

## 3. Frontend

- [x] 3.1 Add the existing remote user selector to the Prompt Audit policy panel.
- [x] 3.2 Keep English and Chinese labels aligned and describe audit-versus-block behavior.

## 4. Verification

- [x] 4.1 Cover configuration, selected-user review, finding allowance, recovery retention, and group exclusion.
- [x] 4.2 Run backend, frontend, OpenSpec, lint, and diff checks.
