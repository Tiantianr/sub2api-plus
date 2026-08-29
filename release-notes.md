Sub2API Plus v0.1.183+custom.910

## Highlights

- Add configurable dual-lane Prompt Audit: the latest user turn is reviewed
  synchronously, and every user turn receives a best-effort asynchronous deep
  review after the combined synchronous security decision allows the request.
- Fence late asynchronous Blocks with a per-user synchronous deep-review
  requirement on the next request; only a matching Allow restores normal mode.
- Record value-free structural diagnostics for failed content extraction so
  unsupported protocol item types can be identified without logging prompt,
  tool, credential, or media values.

## Changed

- Configure synchronous and asynchronous review independently for system
  instructions, assistant history, reasoning, prompt variables, tool
  definitions, tool calls, and tool outputs. Mandatory user coverage cannot be
  disabled.
- Store asynchronous deep jobs and events under `execution_mode=async_deep`,
  with independent administration filters and the existing encrypted complete
  context evidence.
- Preserve rolling-upgrade compatibility for omitted module fields while
  accepting explicitly supplied all-false optional module maps.

## Fixed

- Require a user with a late deep Block to pass a fenced synchronous deep
  review even after leaving the ordinary Prompt Audit group scope.
- Keep non-user deep findings out of Content Moderation hashes, violation
  counts, permanent bans, and automatic user penalties.

## Compatibility and migration

- Migrations `239_prompt_audit_deep_review.sql` and
  `240_prompt_audit_events_mode_index_notx.sql` run automatically. They admit
  `async_deep` rows and create the event-mode index concurrently.
- Existing Prompt Audit configurations inherit compatibility defaults when the
  new module maps are absent. No manual configuration change is required.
- No Compose, port, certificate, proxy, or persistent-volume change is required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Asynchronous deep review begins only after synchronous Allow and cannot
  retroactively cancel content already sent upstream or delivered to a client.
- Blocking and forced recovery review remain fail closed on extraction,
  encryption, Guard, and deep-review state-store failures.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
