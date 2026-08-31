# Tasks

## 1. Contract

- [x] 1.1 Define configured scanners as a server-enforced known-category allowlist.
- [x] 1.2 Preserve unknown and unclassified Unsafe fail-closed behavior.

## 2. Backend

- [x] 2.1 Filter disabled known categories before normalized decision construction.
- [x] 2.2 Normalize historical event API categories from matched scanners.
- [x] 2.3 Derive issue summaries only from effective matched scanners.

## 3. Frontend

- [x] 3.1 Use effective categories in list, detail, and structured Guard result views.

## 4. Verification

- [x] 4.1 Cover disabled-only Safe, Controversial, and Unsafe results.
- [x] 4.2 Cover mixed enabled/disabled and unknown-category results.
- [x] 4.3 Run backend, frontend, OpenSpec, lint, and diff checks.
