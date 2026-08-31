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

An in-scope blocking-exempt request does not call synchronous Guard or claim
recovery state. Before it may continue, Prompt Audit extracts the complete deep
review snapshot, encrypts retained context, and synchronously confirms that the
job and transient payload reached `queued`. The worker then performs the full
Guard review and stores the original decision, risk, action, and evidence.
Exempt jobs never create Allow receipts or user recovery state.

The job and event persist `blocking_exempt_at_request`. Workers use this
immutable snapshot so removing an exemption cannot turn an already admitted
exempt job into recovery enforcement; an active exemption may still pause
recovery for older non-exempt jobs. The administration list uses only the
snapshot and never infers historical status from the current user list.
Existing recovery state is retained and ignored for a request admitted while
exempt.

Content extraction, encryption, database admission, payload storage, and queue
publication remain pre-dispatch fail-closed boundaries. Guard invalid-response
and outage handling occurs asynchronously after reliable admission and creates
the existing terminal failure event without retroactively blocking the request.
Content Moderation remains an independent synchronous engine with its own scope
and enforcement configuration; `text_api_mode=auto` keeps blocking authority
because asynchronous Prompt Audit is not authoritative for the current text.
