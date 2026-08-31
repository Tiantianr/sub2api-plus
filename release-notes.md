Sub2API Plus v0.1.183+custom.920

## Highlights

- Let blocking-exempt users continue after a complete Prompt Audit job is
  reliably queued, without waiting for synchronous Guard evaluation.
- Show an immutable request-time blocking-exempt marker in the existing event
  list for successful and failed asynchronous reviews.
- Enforce configured Qwen3Guard risk categories as a server-side allowlist so
  disabled known categories cannot affect decisions or appear as findings.

## Changed

- Blocking-exempt requests complete canonical extraction, context encryption,
  database admission, transient payload storage, and queue publication before
  continuing. Content Moderation remains an independent synchronous authority.
- Asynchronous exempt jobs bypass Allow receipts and never create recovery
  state, while stored Guard results retain their original Critical, Flag, risk,
  action, and evidence policy.
- Event APIs and views use persisted `matched_scanners` as the effective
  category set. Existing database rows are not rewritten.

## Fixed

- Prevent blocking-exempt requests from waiting for synchronous Guard or a
  recovery claim before upstream processing.
- Prevent a Prompt Audit configuration race from letting Content Moderation
  use stale shadow authority while Prompt Audit has already left blocking mode.
- Prevent disabled known categories such as PII from producing warnings,
  blocks, scores, evidence, issue summaries, or administration labels.
- Persist safe failed events when payload storage or queue publication fails
  after a staging job was created.

## Compatibility and migration

- Migration 244 adds `blocking_exempt_at_request` to Prompt Audit jobs and
  events with a non-null `false` default. Historical rows are not inferred or
  backfilled from the current exemption list.
- The Prompt Audit configuration API and its nine stable scanner IDs are
  unchanged. Existing recovery findings and Allow receipts are not cleared.
- Roll back to `v0.1.183+custom.919` if required. The additive schema remains;
  `.919` waits for synchronous Guard for exempt users and may display disabled
  known categories returned by Guard.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Blocking exemption applies only to Prompt Audit. Content Moderation remains
  independently configured and may still block the same user.
- Exempt requests do not wait for Guard, but reliable admission still adds
  database and Redis latency. Extraction, encryption, queue-capacity, payload,
  or publication failure remains fail closed before upstream side effects.
- Unknown categories and Unsafe output without a recognized category retain
  the existing fail-closed behavior even when known categories are disabled.
- Publishing this release does not change production Prompt Audit groups,
  enabled scanners, blocking exemptions, Content Moderation settings, or
  runtime deployment.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
