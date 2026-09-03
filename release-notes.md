Sub2API Plus v0.1.183+custom.925

## Highlights

- Retain the complete normalized input behind new Content Moderation keyword
  findings as purpose-bound AES-256-GCM ciphertext, with an audited
  administrator-only detail view and excerpt-only list responses.
- Organize Prompt Audit payloads by user and session, apply bounded Pass
  retention, and add administrator-triggered analysis limited to the selected
  event's session through the existing Guard endpoint pool.
- Show account group-coverage warnings and make OpenAI OAuth restricted-access
  preview and save robust to empty or legacy nullable API collections.

## Changed

- New keyword evidence is redacted before encryption and can be retrieved only
  through `GET /admin/risk-control/logs/:id/input`. Sensitive reads use private
  no-store response headers and administrator audit logging.
- Prompt Audit stores deduplicated chat records under a user-scoped session.
  Unselected Pass payloads expire after seven days; selected Pass evidence and
  risk findings remain until manual cleanup, while event metadata remains.
- Session analysis is capped at 200 records and 120,000 Unicode characters,
  prioritizes the selected event and recent context, compacts common cumulative
  prefixes, and does not persist or log the generated report.
- Scheduled cleanup drains bounded batches within its time budget and removes
  orphaned chat records. Database backups retain audit metadata while excluding
  Prompt Audit chat and legacy context payload rows.
- Content Moderation keyword enforcement becomes shadow-only for the existing
  blocking-exempt users. Hash checks and independent moderation checks remain
  active, and the exemption is frozen once per request.
- Account administration distinguishes groups with no linked accounts from
  groups with no currently available accounts and refreshes coverage after
  account mutations.

## Fixed

- Prevent complete keyword evidence from falling back to plaintext when
  encryption fails; the keyword decision is preserved and the missing evidence
  remains observable.
- Prevent Prompt Audit session deduplication from merging unrelated users,
  protocols, or identifier sources, and stop using `prompt_cache_key` as a
  conversation identity.
- Prevent deletion previews from counting shared chat records that remain
  referenced by other events.
- Prevent the OpenAI OAuth restricted-account flow from failing with
  `Cannot read properties of null (reading 'length')` when no users lose all
  access. Backend collection fields now serialize as arrays and the frontend
  normalizes nullable responses from older instances.

## Compatibility and migration

- Forward-only migrations `245`, `246`, and `247` add encrypted moderation
  evidence, Content Moderation blocking exemptions, and Prompt Audit session
  and chat-record storage. Existing audit rows remain valid and are not
  rewritten with reconstructed content.
- Configure a stable `TOTP_ENCRYPTION_KEY` before expecting recoverable complete
  keyword evidence. Missing or invalid encryption configuration does not weaken
  keyword enforcement and never stores plaintext evidence.
- Rolling deployments remain compatible: `.925` can read legacy Prompt Audit
  payloads written by `.924` instances after migration. New `.925` chat payloads
  are not readable by `.924` code because new events intentionally leave the
  legacy `full_prompt` field empty.
- Roll back application code to `v0.1.183+custom.924` if required. Leave the
  forward migrations in place; rolled-back code ignores the added schema but
  cannot display Prompt Audit payloads captured only by `.925`.
- No dependency, port, certificate, proxy, or persistent-volume change is
  required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Historical Content Moderation rows have no recoverable complete input and
  return `complete: false`.
- Prompt Audit chat payloads are intentionally omitted from database backups;
  restoring a backup preserves event metadata but not retained conversations.
- Session analysis is bounded context, not a complete account history, and is
  available only while the selected event still references retained content.
- Unselected Pass content can expire after seven days. Selecting evidence or
  receiving a risk finding changes retention to indefinite until manual
  cleanup.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
