Sub2API Plus v0.1.183+custom.916

## Highlights

- Fix the Prompt Audit administration page blank screen introduced in `.915`
  when the initial Pass-retention user list is empty.
- Enable eligible Guard-unavailable request allowance by default so a remote
  audit outage does not make every in-scope user request unavailable.
- Preserve explicit administrator opt-out and all strict content, finding,
  recovery, configuration, and Content Moderation boundaries.

## Changed

- Treat older Prompt Audit configurations without
  `allow_on_guard_unavailable` as enabled; an explicit false remains disabled.
- Preserve the availability policy while Prompt Audit or synchronous blocking
  is temporarily disabled, instead of silently clearing it.
- Normalize legacy or malformed null Pass-retention user lists at both the API
  client and page-state boundaries.

## Fixed

- Serialize an empty Pass-retention user list as `[]` rather than `null`.
- Prevent `PromptAuditView` from dereferencing `user_ids.length` on a null API
  response, with a real-child full-page regression test.
- Keep failure-allowed requests receipt-free while continuing to fail closed
  for invalid Guard responses, extraction failure, known findings,
  undecryptable credentials, missing usable nodes, and required recovery.

## Compatibility and migration

- `v0.1.183+custom.915` is invalid and must not be deployed; use `.916` for the
  retention and Guard-availability feature set.
- Deployments upgrading directly from `.914` apply migration
  `241_prompt_audit_remove_global_pass_retention.sql`; it removes only the old
  global JSON field and does not delete existing audit events.
- Existing configurations that omit `allow_on_guard_unavailable` default to
  true. Administrators may explicitly turn it off from Prompt Audit settings.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Existing Pass events and historical backups are not deleted automatically.
- Logical event cleanup reduces future backups but does not guarantee immediate
  PostgreSQL filesystem reclamation.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
