# Tasks

## 1. Contract

- [x] 1.1 Define the default-on failure-allow policy and exact unavailable
  boundary.
- [x] 1.2 Preserve extraction, invalid-response, finding, recovery, receipt,
  and Content Moderation security behavior.

## 2. Backend

- [x] 2.1 Persist and publish `allow_on_guard_unavailable` through the existing
  Prompt Audit configuration lifecycle.
- [x] 2.2 Convert ordinary final Guard-unavailable failures to observable Allow
  decisions without creating receipts.
- [x] 2.3 Keep required recovery and every non-unavailable failure fail closed.

## 3. Frontend

- [x] 3.1 Add the failure-allow switch with clear risk and boundary text.
- [x] 3.2 Keep English and Chinese configuration types and locale keys aligned.

## 4. Verification

- [x] 4.1 Cover default-on, explicit-off, unavailable allow, invalid response, extraction,
  known Block, recovery, metrics, and receipt behavior.
- [x] 4.2 Run focused backend race/integration tests, frontend tests, lint,
  typecheck, documentation checks, and strict OpenSpec validation.
