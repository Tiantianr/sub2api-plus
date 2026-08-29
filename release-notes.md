Sub2API Plus v0.1.183+custom.912

## Highlights

- Require a user whose synchronous Prompt Audit request was blocked to pass one
  complete synchronous deep review before upstream processing can resume.
- Prevent API-key, group, or client session changes from skipping that recovery
  review while preserving exact-Allow automatic recovery.

## Changed

- Make synchronous and asynchronous Prompt Audit Blocks share the same
  non-expiring, versioned user recovery state.
- Use the active deep-review module selection for recovery, force all user
  turns, and bypass trusted and stored Allow receipts.
- Add bounded recovery-required, cleared, retained, and error logs and runtime
  counters without exposing state tokens or audited content.

## Fixed

- Stop an agent from resuming through ordinary incremental Allow immediately
  after a synchronous Prompt Audit Block.
- Add a final recovery-state fence before receipt persistence, deep-job enqueue,
  account selection, billing, or upstream writes.
- Preserve a newer concurrent synchronous or asynchronous Block when an older
  recovery request finishes with Allow.

## Compatibility and migration

- No database migration or new configuration field is required. Recovery uses
  the existing `deep_review_modules` policy and Redis state.
- A request after Prompt Audit Block may return
  `prompt_guard_deep_review_required` until complete deep review returns exact
  Allow.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Disabling risk control or synchronous Prompt Audit pauses recovery
  enforcement without clearing pending state; enabling it resumes recovery.
- Requests that already completed the final audit gate cannot be retroactively
  cancelled. Removed prior content is not reattached to recovery input.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
