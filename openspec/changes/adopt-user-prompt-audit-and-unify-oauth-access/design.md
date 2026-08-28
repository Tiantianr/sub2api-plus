# Design

## Prompt selection and evidence

The canonical extractor remains the single protocol contract. Guard selection follows the published Plus behavior: blocking uses the latest user turn, async uses all user turns, and known harness XML is stripped. Blocking extraction failures remain fail closed.

Before selection is discarded, the service serializes the complete canonical document with segment provenance, images, extraction status, and the exact Guard input. The JSON is gzip-compressed, application-encrypted, and stored in `prompt_audit_event_contexts`. Async jobs carry only the encrypted artifact in their transient Redis payload. Event list/detail responses never contain ciphertext or complete context. An authenticated admin download decrypts and streams JSON with `Cache-Control: no-store`; deleting the event cascades to the artifact.

## OAuth effective access

An account is eligible for a group only when an `account_groups` binding exists and, for a session-sharing OAuth account, the group is also in `allowed_group_ids`. Per-user public/restricted policy is evaluated only after that group eligibility succeeds.

Group duplication and account-copy operations skip OAuth bindings that are not explicitly authorized for the destination group. They never expand `allowed_group_ids`. Admin matrices and impact previews expose only effective group eligibility. Long-lived Responses WebSocket and Live revalidation repeat the same group and user checks.

## Compatibility

Missing OAuth access policies remain public. Existing allowlists are not expanded. Existing Prompt Audit events have no downloadable artifact and return not found for the new endpoint. Existing queued async payloads remain readable as legacy plain scan text but cannot gain context retroactively.
