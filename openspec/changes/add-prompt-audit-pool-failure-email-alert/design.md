# Design: Prompt Audit pool failure email alert

## Outcome boundary

The existing Prompt Audit metrics observer is the shared completion point for
synchronous Guard evaluation and asynchronous deep review. A final
`unavailable` or `invalid` outcome increments the streak. Allow, Flag, and Block
all prove that the pool returned a complete decision and reset the streak.
Endpoint attempts recovered by failover do not increment it. Failures before a
Guard evaluation, including recovery claim contention, are outside this metric.
Parent request cancellation and service shutdown are not pool outcomes and do
not increment or reset the streak.

## Notification lifecycle

The fifth consecutive failure claims the notification for the current streak.
Later failures do not send duplicates. A complete Guard result resets the
streak, allowing a later group of five failures to send another email. Email
work runs asynchronously and cannot alter the audit decision, upstream dispatch,
billing, or response.

The notification uses the existing Ops alert sender, recipients, template,
critical-severity filtering, silencing, and hourly rate limiter. It includes
only the failure count, bounded outcome kind, and trigger time. It excludes
prompts, request bodies, users, credentials, endpoint URLs, and raw Guard
responses.

## State scope

The current deployment has one application process, so a mutex-protected
in-memory streak is sufficient and resets on process restart. A future
multi-instance deployment that requires one global alert must replace it with a
Redis atomic counter and generation key.
