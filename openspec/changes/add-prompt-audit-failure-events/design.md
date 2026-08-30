# Design: Prompt Audit failure events

## Storage

Failure outcomes reuse `prompt_audit_events`. A failed event uses decision
`failed`, risk `unknown`, action `Error`, a stable error code, and a generic
safe message. It contains the existing redacted request snapshot but never a
full prompt or encrypted context.

Synchronous failure persistence creates a terminal failed blocking job and its
event in one transaction. Terminal asynchronous failure persistence links one
event to the existing failed job. Retry attempts remain visible through runtime
metrics and logs but do not create list rows.

Migration 243 backfills terminal failed jobs that have no event. Historical
synchronous failures cannot be reconstructed.

## Ordering and confidentiality

The service determines the original audit outcome before best-effort failure
persistence. A persistence error increments the existing record-failure metric
and emits a bounded log, but does not alter blocking, failure allowance, retry,
billing, or upstream dispatch.

The persisted reason is generated from a stable error-code mapping. Database and
API paths never consume the underlying error string. Failure rows exclude raw
Guard responses, endpoint URLs, credentials, complete prompts, media, tool
payloads, and encrypted context.
