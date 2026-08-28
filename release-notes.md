Sub2API Plus v0.1.183+custom.908

## Highlights

- Close security-audit extraction gaps across Responses, Chat, Alpha Search,
  tools, reasoning, compaction, WebSocket, SSE, and Live traffic.
- Replace static latest-turn Prompt Guard blocking with fail-closed
  conversation checkpoints that audit complete context first and trusted
  incremental context afterward.
- Separate Prompt Guard text authority from Content Moderation while retaining
  independent local keyword/hash and image controls.

## Changed

- Audit full canonical context for new, expired, changed, branched, or
  continuity-invalid conversations. Stable continuations audit the captured
  previous AI output together with current client-controlled content.
- Commit a clean conversation checkpoint only after a complete successful
  HTTP, SSE, Responses WebSocket, or Live downstream write.
- Add Content Moderation text API policies for automatic authority selection,
  explicit blocking, shadow observation, and off. Shadow findings have no hash,
  notification, ban, or violation-count effects.
- Keep blocking image moderation independent from text policy and sampling.

## Fixed

- Recognize Responses legacy `messages` and string `prompt` aliases, visible
  `reasoning_content`/`reasoning_text`, compaction summaries and triggers, and
  Alpha Search fallback fields without persisting opaque or media payloads.
- Prevent safe latest turns, known parent IDs, replay insertion, stale branches,
  extraction failures, or incomplete output from establishing incremental
  eligibility.
- Prevent blocking moderation dependency failures from silently allowing
  requests, and distinguish external API blocks, local keyword/hash blocks,
  shadow findings, and business-upstream `cyber_policy` records.

## Compatibility and migration

- No database migration is required.
- Blocking Prompt Audit requires Redis and the existing application secret
  encryptor for temporary conversation checkpoints.
- The legacy latest-turn JSON field remains accepted for rolling-upgrade
  compatibility but no longer controls blocking behavior.
- Existing Content Moderation configuration defaults missing text API policy to
  `auto`; images and local keyword/hash behavior remain independently scoped.
- No Compose, port, certificate, proxy, or persistent-volume change is required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- PostgreSQL Prompt Audit integration tests still require an external
  `PROMPT_AUDIT_TEST_POSTGRES_DSN`; repository migration coverage remains in the
  protected Linux test matrix.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.002
Plus commit: 2b5bd31478415617831d49eea9988be90111d3b7
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
