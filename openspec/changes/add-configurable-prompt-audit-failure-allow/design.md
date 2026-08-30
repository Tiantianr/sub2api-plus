# Design

## Configuration

Persist `allow_on_guard_unavailable` in the existing versioned Prompt Audit
configuration. Its default is false. Saving it uses the existing CAS,
configuration invalidation, audit metadata, and active-snapshot flow.

The administration console exposes the value as a switch that is meaningful
only while synchronous blocking is enabled. Disabling synchronous blocking or
Prompt Audit clears the draft value so a later activation cannot silently
inherit a failure-allow policy.

## Decision boundary

The Guard evaluator keeps its strict node ordering, per-node timeout, complete
chunk coverage, and error classification. The Prompt service converts the
final `prompt_guard_unavailable` error into an Allow only when the active
configuration enables failure allow, the error is classified as an eligible
remote API/timeout/capacity failure, and the request is not a required recovery
review. Missing usable nodes, undecryptable credentials, local client
construction failures, and scanner wiring failures are ineligible.

The conversion occurs after canonical extraction and context encryption. It
therefore cannot bypass extraction failures, invalid Guard responses, missing
encryption, untrusted configuration, or deep-review state failures. Known Flag
and Block results continue through their existing decision path. Content
Moderation remains independent and may still reject or fail closed.

## Evidence and observability

A failure-allowed decision carries no normalized Safe result and no pending
Allow-receipt commit, so it cannot certify content for a later request. Any
best-effort asynchronous deep job inherits explicit receipt-write suppression;
a later successful deep review still cannot certify the failure-allowed
request. The existing unavailable/timeout metrics continue to count the Guard
failure, and a separate `failure_allowed` counter plus structured event
identifies requests that continued because of the policy. Asynchronous deep
review may still be queued through the existing best-effort path.
