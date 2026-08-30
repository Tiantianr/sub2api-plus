# Design

## Independent retention snapshot

Store the canonical sorted user ID list under
`prompt_audit_pass_retention`. The value has its own revision, update metadata,
compare-and-swap save, Redis invalidation, and bounded periodic reload. It does
not increment Prompt Audit's Guard `config_version` and is not included in an
Allow-receipt key.

Missing or invalid retention data activates an empty list and reports a
bounded administration load error. This fail-minimized behavior affects only
optional Pass evidence; it must not make request auditing unavailable.

The active in-memory snapshot exposes a user lookup used when synchronous or
asynchronous evaluation completes. A Pass event is stored only when its user ID
is selected at completion time. The repository's existing rule continues to
store every non-Pass event. Anonymous or invalid user IDs are never selected.

## Administration contract

Expose a separate GET/PUT resource with `revision`, canonical `user_ids`, and
update metadata. PUT requires `expected_revision`. The audit log records only
the revision, selected/add/remove counts, and a digest of the canonical ID
list, not the full list.

The Prompt Audit page presents a searchable selected-user list with its own
dirty and save state. User search reuses existing administrator user APIs.
Deleted or otherwise unresolved configured IDs stay visible as stable `#ID`
entries so administrators can remove them deliberately.

The old global `store_pass_events` UI and update field are removed. Current
runtime behavior is determined only by the independent selected-user list.

## Pass-only cleanup

Reuse the existing event-filter delete repository and confirmation flow. A
dedicated UI always submits `decision=pass`; administrators can choose all
users or one user and an explicit cutoff/custom range. Confirmation remains
disabled until a preview has been displayed.

The preview transaction additionally returns the number of matched context
artifacts and an estimated retained-content size. The estimate sums stored
encrypted context and event full-prompt datum sizes without reading,
decrypting, or returning content. It describes expected reduction in future
logical backups, not immediate filesystem reclamation.

The existing snapshot maximum event ID, filter digest, administrator-bound
five-minute token, bounded delete batches, and `ON DELETE CASCADE` context
cleanup remain authoritative. Events created after preview survive.

## Compatibility and rollout

An old persisted global `store_pass_events=true` value is ignored by the new
runtime. A forward data migration removes the field from the stored main
configuration, so an older process defaults it to false after its bounded
reload and a rollback cannot reactivate global Pass persistence. Operators may
still disable the old switch before a mixed-version rollout to eliminate that
short reload window.

Existing Pass evidence is not deleted by upgrade. Initial production cleanup
is a separate authorized operation: preview Pass events, confirm deletion,
create and verify a new backup, then separately decide whether old backups may
be deleted.
