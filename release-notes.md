Sub2API Plus v0.1.183+custom.917

## Highlights

- Restore lightweight Pass rows in the Prompt Audit event list for every
  completed review without restoring large full-context storage by default.
- Snapshot and display the actual Guard pool node name, node ID, and model that
  produced each event decision, including after endpoint failover.
- Fix concurrent Block-recovery requests overwriting one another and returning
  503 after every Guard chunk had already completed with Allow.

## Changed

- Treat the selected-user list as full Pass-evidence retention only. Every Pass
  keeps a lightweight redacted event; selected users additionally retain the
  full Guard prompt and encrypted canonical context.
- Add an Audit node column and explicit node/model fields to event details.
  Historical events fall back to their stored endpoint ID and scanner model.
- Separate the non-expiring recovery finding from a bounded per-user Redis
  claim lease. Claim expiry or process failure cannot clear the finding.
- Return a recovery-state-specific message for
  `prompt_guard_deep_review_state_unavailable` instead of reporting it as an
  ordinary Guard outage.
- Include redacted localized risk-category labels in Prompt Audit Block
  responses without exposing raw Guard evidence or prompt content.

## Fixed

- Prevent unselected Pass results from disappearing entirely from the event
  list; their `full_prompt` stays empty and no context artifact is created.
- Prevent a later recovery request from replacing an in-progress request's
  claim and stranding an immortal `review:` token in the finding key.
- Preserve newer synchronous or asynchronous Block findings when an older
  recovery completes with Allow.
- Allow `.916` historical `review:` state tokens to complete one full recovery
  review and clear safely after upgrade.
- Keep ordinary eligible Guard network/API/timeout/capacity failures subject to
  the default-on failure allowance while recovery, extraction, invalid response,
  configuration, finding, and Content Moderation boundaries remain fail closed.

## Compatibility and migration

- `v0.1.183+custom.915` is invalid and must not be deployed; use `.916` for the
  previous rollback target or `.917` for the corrected event and recovery flow.
- Migration `242_prompt_audit_guard_node_snapshot.sql` adds only non-secret
  `guard_endpoint_name` and `guard_model` event snapshot columns. It does not
  rewrite historical rows or touch encrypted context data.
- The recovery claim uses a new Redis key namespace. Existing finding keys,
  including historical `review:` values, remain enforceable and recoverable.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Pass decisions omitted entirely while `.915`/`.916` retention behavior was
  active cannot be reconstructed retroactively.
- Historical events cannot recover an old configured node name and therefore
  display their stable endpoint ID instead.
- Existing full Pass evidence and historical backups are not deleted
  automatically. Cleanup still requires an explicit preview and confirmation.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
