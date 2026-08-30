# Design: Prompt Audit blocking exemptions

## Configuration

`blocking_exempt_user_ids` is part of the existing CAS-protected Prompt Audit
configuration. Values are positive, deduplicated, sorted user IDs with a limit
of 100. An update that omits the additive field preserves the current list so
an older administration page cannot silently remove enforcement exceptions.
The public administration response returns the list, while change summaries
retain only its count and hash.

## Request behavior

Group scope is evaluated before Prompt Audit recovery state. An out-of-scope
request does not enter synchronous evaluation, asynchronous enqueue, recovery
claiming, or the final recovery fence. Any existing user recovery finding stays
stored and becomes enforceable again on a later in-scope request.

If an administrator removes a queued job's group before it starts, the worker
terminates that job without calling Guard or creating an event. If scope or
exemption changes while recovery Guard evaluation is already running, an Allow
does not clear the finding after enforcement has become paused.

An in-scope blocking-exempt user still enters synchronous review and the
post-allow asynchronous deep review. Guard findings remain Critical or Flag in
stored events, but a synchronous Block is returned to the coordinator as an
allowing Flag and neither synchronous nor asynchronous review writes recovery
state. Existing recovery state is retained but ignored while the exemption is
active.

Extraction, encryption, invalid-response, recovery-store, and Guard outage
semantics are unchanged. The exemption applies only to valid Prompt Audit
content findings. Content Moderation remains an independent engine with its own
scope and enforcement configuration.
