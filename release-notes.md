Sub2API Plus v0.1.183+custom.921

## Highlights

- Prevent every final `prompt_guard_unavailable` outcome from blocking the
  current request, including deterministic Guard 4xx responses, missing nodes,
  recovery review outages, and asynchronous admission dependency failures.
- Redact credentials and direct identifiers immediately before every external
  Guard API request without changing canonical hashes, encrypted evidence,
  Allow receipts, or recovery state.

## Changed

- Prompt Audit availability is now a fixed gateway invariant rather than an
  administrator option. The obsolete `allow_on_guard_unavailable` field and UI
  switch are removed; older stored or submitted values are ignored.
- Failure-allowed requests keep safe failure events and metrics, never receive
  a Safe result or Allow receipt, retain existing recovery findings, and attempt
  best-effort asynchronous review with receipt writes suppressed.
- External Guard requests replace recognizable Bearer/JWT/API credentials,
  email addresses, telephone numbers, checksum-valid Chinese identity numbers
  and bank cards, and valid IPv4/IPv6 addresses with typed placeholders.
- A value-free local PII signal affects a non-Safe Guard result only when `pii`
  is enabled. Guard Safe and disabled PII remain unchanged.

## Fixed

- Prevent a safe earlier chunk followed by a deterministic Guard 4xx from
  returning a user-visible Prompt Audit 503.
- Prevent missing Guard capacity, scanner construction failures, and required
  recovery Guard outages from blocking the current request.
- Prevent database, payload, queue-capacity, or queue-publication unavailability
  during blocking-exempt admission from returning `prompt_guard_unavailable`.
- Prevent recognized credentials and direct identifiers from being copied into
  external Guard request content.

## Compatibility and migration

- No database migration, dependency, port, Compose, certificate, proxy, or
  persistent-volume change is required.
- Existing Prompt Audit scanner IDs, group scope, blocking exemptions, evidence
  retention, endpoint priority, and Content Moderation configuration remain
  unchanged.
- Older clients may still submit `allow_on_guard_unavailable`; the server
  ignores the unknown field. The public configuration no longer returns it.
- Roll back to `v0.1.183+custom.920` if required. The rollback restores the
  configurable narrow failure-allow policy and sends original selected prompt
  chunks to Guard endpoints without the new outbound identifier redaction.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Strictly invalid Guard responses, incomplete canonical extraction, encryption
  failures, configuration-version races, known Guard findings, and Content
  Moderation decisions retain their independent fail-closed behavior.
- Free-form names and postal addresses are not redacted locally because
  regex-only detection would create unacceptable false positives.
- Canonical prompt content remains available only through the existing internal
  encrypted evidence and retention controls; this release changes the external
  Guard copy, not administrator evidence policy.
- A Prompt Audit dependency outage can let the current request continue without
  completing asynchronous review. Failure events, metrics, and the existing
  five-consecutive-pool-failure email alert remain observable.
- Publishing this release does not change production Prompt Audit groups,
  enabled scanners, blocking exemptions, Content Moderation settings, email
  settings, or runtime deployment.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
