# Prompt Audit pool failure email alert

## Problem

Prompt Audit exposes runtime failure metrics, but a Guard pool outage requires an
administrator to notice the dashboard or request errors manually. Repeated pool
failures can therefore continue without an active notification.

## Proposal

- Count final Prompt Audit Guard pool evaluations that end as unavailable or
  invalid.
- Send one critical email after five consecutive failed evaluations.
- Reset the streak after any complete Guard result, including Allow, Flag, or
  Block, and permit a later failure streak to alert again.
- Reuse the existing Ops alert recipients, SMTP configuration, and `ops.alert`
  email template.
- Keep alert delivery asynchronous and independent from request enforcement.

## Non-goals

- Treating extraction, configuration, encryption, or recovery-state failures as
  Guard pool failures.
- Changing Prompt Audit fail-closed or failure-allow behavior.
- Adding new recipient configuration or a Prompt Audit alert history table.
- Coordinating the partial streak across multiple application processes.

## Impact

No API, database migration, dependency, or frontend change is required. The
counter is process-local and uses the existing operations email configuration.
