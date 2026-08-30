Sub2API Plus v0.1.183+custom.915

## Highlights

- Retain normal Prompt Audit Pass evidence only for explicitly selected users;
  Flag and Critical findings remain mandatory for every audited user.
- Add a previewed Pass-only cleanup workflow with event, context, and estimated
  retained-byte counts before irreversible deletion.
- Let administrators explicitly allow ordinary requests when every usable
  Guard node fails because of a remote API, timeout, or capacity problem.

## Changed

- Add an independently versioned, multi-instance-safe Pass-retention user list;
  users without an explicit selection default to no normal-event storage.
- Add a default-off `allow_on_guard_unavailable` switch and a dedicated runtime
  counter for observable failure-allowed requests.
- Keep Guard failure allowance separate from invalid responses, incomplete
  extraction, undecryptable credentials, known findings, Content Moderation,
  and required user recovery; those paths continue to fail closed.

## Fixed

- Prevent normal Pass conversations from dominating future logical backups
  when no user has been selected for retention.
- Ensure a failure-allowed request cannot create an Allow receipt, including if
  its later asynchronous deep review succeeds.

## Compatibility and migration

- Migration `241_prompt_audit_remove_global_pass_retention.sql` removes the
  obsolete global `store_pass_events` JSON field; it does not change the SQL
  schema or delete existing audit events.
- The initial Pass-retention allowlist is empty. Existing Pass evidence remains
  until an administrator separately previews and confirms cleanup.
- `allow_on_guard_unavailable` is a backward-compatible configuration field
  and defaults to false, preserving fail-closed behavior until explicitly
  enabled.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Logical deletion reduces future backups but does not guarantee immediate
  PostgreSQL filesystem reclamation.
- Historical large backups and existing Pass records are not deleted by this
  release.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
