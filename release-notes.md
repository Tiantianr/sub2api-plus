Sub2API Plus v0.1.183+custom.919

## Highlights

- Show synchronous Guard failures and terminal asynchronous Prompt Audit
  failures in the existing event list with stable, safe reasons.
- Add blocking-exempt users who remain fully audited and visible while Prompt
  Audit findings do not reject their requests.
- Enforce selected-group scope before every Prompt Audit review, queue,
  recovery, and final enforcement path.

## Changed

- Blocking-exempt users preserve the original Critical, Flag, risk, and Guard
  action in stored events. Only the Prompt Audit gateway enforcement decision
  is converted to a non-blocking Flag.
- Existing recovery findings pause while a user is exempt or outside selected
  group scope and resume if the policy later applies again.
- Prompt Audit configuration accepts up to 100 blocking-exempt users and
  preserves the current list when an older client omits the additive field.
- Failed events retain only a redacted request snapshot, stable error code, and
  bounded generic reason. They never retain raw Guard errors or full context.

## Fixed

- Prevent pending recovery state from causing Prompt Audit review and events
  for a group that is not selected.
- Prevent an in-progress recovery Allow from clearing its finding after the
  user becomes exempt or the request group leaves scope.
- Prevent stale queued work for a removed group from calling Guard or creating
  an audit event.
- Prevent active queue jobs from being deleted when an administrator removes
  related event history.

## Compatibility and migration

- Migration 243 adds bounded `error_code` and `error_message` event columns,
  permits `failed / unknown / Error`, and backfills terminal failed jobs that
  do not already have events. The migration is forward-only and idempotent.
- `blocking_exempt_user_ids` is an additive field in the existing versioned
  Prompt Audit JSON configuration and defaults to an empty list.
- Existing recovery findings and Allow receipts are not cleared by this
  upgrade.
- Roll back to `v0.1.183+custom.918` if required. The additive schema remains;
  `.918` does not display the new failure reasons or blocking-exempt controls.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Blocking exemption applies only to Prompt Audit findings. Content Moderation
  remains independently configured and may still block the same user.
- Extraction, encryption, invalid Guard output, recovery-store errors, and
  Guard availability continue to follow the existing fail-closed or configured
  failure-allow policy for blocking-exempt users.
- Publishing this release does not change production Prompt Audit groups,
  blocking exemptions, Content Moderation settings, or runtime deployment.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
