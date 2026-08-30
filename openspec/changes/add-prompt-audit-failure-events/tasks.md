# Tasks

## 1. Contract and migration

- [x] 1.1 Define the lightweight failed-event and confidentiality contract.
- [x] 1.2 Add migration 243 and backfill terminal failed jobs.

## 2. Backend

- [x] 2.1 Persist synchronous and terminal asynchronous failed events.
- [x] 2.2 Return stable error code/message through the existing event API.

## 3. Frontend

- [x] 3.1 Show failed results and their safe reason in the existing event list.
- [x] 3.2 Keep English and Chinese locale keys aligned.

## 4. Verification

- [x] 4.1 Cover migration, persistence, failure allowance, terminal retry, and
  confidentiality behavior.
- [x] 4.2 Cover failed list rendering and filters.
- [x] 4.3 Update audit documentation and run backend, frontend, OpenSpec, lint,
  and diff checks.
