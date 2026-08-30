Sub2API Plus v0.1.183+custom.918

## Highlights

- Wait for an in-progress Prompt Audit recovery instead of returning an
  immediate recovery-state 503 to concurrent requests.
- Send a rate-limited P0 Ops email after five consecutive Prompt Audit pool
  failures, with successful pool outcomes resetting the streak.
- Reduce Prompt Audit hot-path memory and CPU cost for large request bodies and
  UTF-8 Guard chunks without reducing audited content.

## Changed

- Recovery claim acquisition now uses one atomic Redis operation with
  exponential waiting backoff. The exact current claim owner must match the
  exact finding version before recovery can clear.
- Only network/read failures, timeout/capacity, 401/403, 429, and 5xx remain
  eligible for configured Guard failure allowance. Deterministic 400/404-class
  errors stay fail closed.
- Prompt Audit pool alerts reuse the existing Ops recipients, templates,
  severity filtering, silencing, and hourly email limiter. Client cancellation
  and service shutdown do not count as pool failures.
- Synchronous Content Moderation and Prompt Audit share the frozen request body
  read-only; the asynchronous boundary retains the only deep copy.
- UTF-8 chunking now slices at rune boundaries without building a full rune
  array or copying every chunk.

## Fixed

- Prevent claim contention from being reported as unavailable recovery state.
- Prevent an expired or replaced recovery owner from clearing a finding owned
  by another request.
- Prevent deterministic Guard client/configuration errors from silently
  bypassing Prompt Audit through failure allowance.
- Prevent caller cancellation from producing false consecutive-pool-failure
  email alerts.
- Clarify Prompt Audit Block responses without exposing raw Guard evidence,
  prompt content, tool values, or recovery tokens.

## Compatibility and migration

- **Before upgrading production, manually disable Prompt Audit synchronous
  blocking.** New and missing configurations already default to
  `blocking_enabled=false`, but this release intentionally does not overwrite
  an existing persisted `true` value.
- Asynchronous Prompt Audit may remain enabled while synchronous blocking is
  off. Pending recovery findings are retained and resume if blocking is enabled
  again.
- No SQL migration is added. Existing finding and claim Redis keys remain
  compatible; claim operations become atomic after the binary upgrade.
- Roll back to `v0.1.183+custom.917` if required. Its published assets and image
  remain immutable.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Publishing this release does not change production Prompt Audit settings.
  Upgrading while the existing synchronous-blocking toggle remains on preserves
  that enabled intent and its fail-closed dependency behavior.
- Existing full Pass evidence and historical backups are not deleted
  automatically; cleanup still requires explicit preview and confirmation.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
