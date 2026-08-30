# Prompt Audit failure events

## Problem

Prompt Audit persists only complete Guard decisions. Synchronous failures exist
only in logs, while terminal asynchronous failures remain on queue jobs and are
absent from the administration event list. Administrators can therefore see
failed requests or alerts without a matching audit event or stable reason.

## Proposal

- Store a lightweight failed event for synchronous Guard failures and terminal
  asynchronous audit failures.
- Backfill existing terminal failed jobs into the event list.
- Show the stable error code and generic safe reason in the existing event list.
- Reuse existing event pagination, detail, filtering, and deletion behavior.

## Non-goals

- Recording each asynchronous retry attempt.
- Adding a separate failure dashboard or detailed HTTP/chunk/attempt diagnostics.
- Persisting caller cancellation, raw dependency errors, Guard responses,
  endpoint URLs, credentials, or complete prompt content.
- Changing Prompt Audit blocking, retry, failover, or failure-allow behavior.

## Impact

Migration 243 adds two bounded error fields and failed result values to Prompt
Audit events. Existing event API responses gain additive fields and the current
frontend list gains a failed result option.
