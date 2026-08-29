# Change: Incrementally review Prompt Audit turns

## Why

Agent clients resend the same conversation after each model tool call. Prompt
Audit currently treats the latest historical user turn as new and repeatedly
sends all user, assistant, reasoning, and tool history to Guard. Long tool loops
therefore multiply Guard load even though only one canonical segment changed.

## What changes

- Synchronously review direct user content marked current by the canonical
  extractor. Historical user turns require valid receipts so client-controlled
  role ordering cannot hide unreviewed text.
- Issue per-segment Allow receipts and combine only receipt misses into one
  Guard input.
- Keep a new current user turn mandatory in its first active lane. In blocking
  mode, asynchronous deep review may reuse the exact complete synchronous
  Allow from the same request.
- Incrementally review new assistant, reasoning, prompt-variable, tool
  definition, tool-call, and tool-output segments while skipping unchanged
  history.
- Add an administrator-configurable TTL with a one-hour default.
- Preserve dual-lane scheduling, late-Block recovery, and fail-closed forced
  recovery.
- Treat receipt-store errors as misses and expose hit, miss, write, and error
  counters.

## Impact

- Redis gains bounded receipt TTL keys; no SQL migration is required.
- Prompt Audit configuration and administration APIs gain one TTL field.
- Guard input evidence records the exact segment hit/miss selection.
