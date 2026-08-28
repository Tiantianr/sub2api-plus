Sub2API Plus v0.1.183+custom.909

## Highlights

- Synchronize the published Plus `v0.1.183+custom.003` release and retire its
  upstream billing probe and public billing endpoint.
- Use Plus user-authored Prompt Audit selection while preserving the complete
  canonical request context for authorized audit-event download.
- Unify OpenAI OAuth group, session-sharing, and per-user access scope across
  scheduling, administration, group duplication, Responses WS, and Live.

## Changed

- Synchronous Prompt Audit always scans only the latest user input; asynchronous
  Prompt Audit scans all user-authored turns.
- Guard excludes system/developer instructions, assistant/model output,
  reasoning, prompt variables, tool definitions/calls/results, and Live session
  configuration.
- Supported client harness XML is stripped from Guard input but retained in the
  separately encrypted complete-context artifact.

## Fixed

- Prevent duplicated groups from creating effective OAuth session-sharing
  access outside the account allowlist.
- Revalidate current group eligibility as well as per-user grants on long-lived
  Responses WebSocket and Live paths.
- Keep Codex auto-review billing accurate when an unmapped request observes a
  Luna response.

## Compatibility and migration

- Removes obsolete upstream billing-probe database state and adds encrypted
  complete-context storage tied to Prompt Audit event retention.
- The legacy latest-turn configuration field remains accepted but no longer
  changes synchronous Prompt Audit selection.
- No Compose, port, certificate, proxy, or persistent-volume change is required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- PostgreSQL Prompt Audit integration tests still require an external
  `PROMPT_AUDIT_TEST_POSTGRES_DSN`; repository migration coverage remains in the
  protected Linux test matrix.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
