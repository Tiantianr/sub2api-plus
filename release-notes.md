Sub2API Plus v0.1.183+custom.904

## Highlights

- Make Prompt Audit events operationally diagnosable with client IP filtering,
  queue and audit latency, execution mode, input size, and matched-chunk data.
- Retain and download the complete audited prompt for new events while clearly
  identifying historical content that was already truncated.
- Raise the bounded per-node audit input limit from 100,000 to 500,000 Unicode
  characters and document the effective split behavior in the admin UI.
- Keep the personal Linux arm64-only immutable release and update path.

## Changed

- Add exact client-IP filtering backed by the trusted ingress IP resolution
  chain and a concurrent PostgreSQL index.
- Separate runtime last-success and last-error timestamps and surface worker,
  persistence, retry, and lease failures consistently.
- Clarify the user-facing Prompt Guard rejection response and improve keyboard
  navigation and ARIA relationships in the event detail workspace.
- Split the large backend service test package into bounded GitHub Actions
  shards and render unavailable update checks as a neutral version state.

## Fixed

- Stop truncating newly persisted Prompt Audit evidence to 65,536 characters.
- Preserve the real asynchronous queue delay across retries and record zero for
  synchronous blocking evaluations without inventing values for old events.
- Keep full prompt content out of list queries, logs, and non-admin responses;
  sensitive detail reads continue through the existing administrator audit.
- Make the configured 500,000-character limit process a 395,959-character
  prompt as one Guard request instead of four context-losing chunks.

## Compatibility and migration

- Database migrations 234 and 235 add Prompt Audit observability fields,
  historical truncation markers, and a non-blocking concurrent client-IP index.
- Existing endpoint limits remain unchanged after upgrade; administrators may
  explicitly raise them up to 500,000 characters after validating Guard model
  context and timeout capacity.
- New audit events retain complete prompt content for administrator review, so
  database retention and access policy should match local privacy requirements.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Prompt content truncated before this release cannot be reconstructed.
- Inputs above the configured node limit are still split, and all synchronous
  chunks retain the existing shared evaluation timeout behavior.
- OAuth transport plugins are experimental and default off.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
