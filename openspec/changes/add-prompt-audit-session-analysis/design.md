## Session identity

The gateway resolves supported client session headers plus explicit
`conversation_id` and `thread_id` fields before Prompt Audit. Cache-routing
fields such as `prompt_cache_key` are excluded. The application stores only a
SHA-256 key scoped by user, protocol, and identifier source. If no stable identifier exists, the
request ID is hashed as a request-scoped fallback. Prompt text is never used to
infer a session.

## Content ownership

`prompt_audit_sessions` provides the user-to-session layer. Each session owns
`prompt_audit_chat_records`, keyed by a hash of the complete Guard prompt and
complete-context hash. Events store only `session_key`, `session_source`, and
the application-managed content record ID. The old in-row prompt and legacy
context table are cleared by migration after their contents are copied.

The event reference is intentionally application-managed rather than a foreign
key: logical backups omit chat rows, so restoring event metadata without
payload rows must remain valid. Event deletion performs bounded orphan cleanup.

## Analysis

The administrator calls `POST /admin/prompt-audit/events/:id/analyze`.
The service resolves the selected event's session, reads at most 200 records and
120,000 Unicode characters, and sends the bounded transcript through the
configured OpenAI-compatible audit endpoint pool. The analyzer uses a system
instruction that treats the transcript as untrusted data and applies existing
outbound credential/PII redaction before sending. Reports are returned only to
the authenticated administrator and are not persisted.

## Retention and backup

The Prompt Audit service runs a bounded hourly cleanup. Unselected Pass content
gets a seven-day `retention_until`; selected Pass evidence and risk findings use
the existing indefinite evidence policy. Cleanup removes content
rows but leaves lightweight events searchable. `pg_dump` excludes both the new chat table and the legacy
context table; event references may be dangling after restore by design.
