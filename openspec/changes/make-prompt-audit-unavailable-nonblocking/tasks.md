# Tasks

## 1. Contract

- [x] 1.1 Define `prompt_guard_unavailable` as an unconditional non-blocking outcome.
- [x] 1.2 Preserve explicit security failures, Content Moderation, evidence, receipts, and recovery findings.

## 2. Backend

- [x] 2.1 Normalize every final Prompt Audit unavailable result at the service and coordinator boundaries.
- [x] 2.2 Remove failure-allow eligibility and the obsolete configuration switch.
- [x] 2.3 Preserve failure events, receipt suppression, recovery findings, and cancellation semantics.

## 3. Frontend

- [x] 3.1 Remove the obsolete availability switch and synchronize frontend types and locales.

## 4. Verification

- [x] 4.1 Cover deterministic 4xx, missing nodes, required recovery, exempt admission dependencies, and Content Moderation precedence.
- [x] 4.2 Run backend, frontend, OpenSpec, lint, and diff checks.
