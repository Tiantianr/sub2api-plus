# Design: Prompt Audit Block recovery

## Context

Synchronous Prompt Audit runs before account selection, billing, and upstream
writes. A Block currently returns HTTP 403 but records no durable recovery
state. Existing `deep_required` state is keyed only by user and is written by
an asynchronous deep Block.

## Decisions

### 1. Synchronous Block recovery is user-scoped and single-use

The recovery identity SHALL use the authenticated user. It SHALL NOT depend on
API key, group, `session_id`, `conversation_id`, or `prompt_cache_key`. The
user's next request must complete recovery; exact Allow clears the state
immediately rather than creating a permanent user freeze.

This scope prevents an agent client from avoiding recovery by rebuilding the
request or rotating client-controlled identity values. It may make one
unrelated conversation perform the recovery review, but that cost is bounded
to the first exact Allow.

### 2. Recovery uses the active deep-review selection

Recovery SHALL include every current and historical user turn plus the
optional sources enabled in the active `deep_review_modules`. It SHALL bypass
trusted same-request handoff and stored Allow receipts, so every selected
segment is sent to Guard again.

Recovery uses the active configuration at request time rather than a snapshot
from the Block. This preserves the existing asynchronous late-Block recovery
model and avoids another persisted policy representation. Canonical extraction
and existing wrapper exclusions remain unchanged.

### 3. Any complete exact Allow clears recovery

Recovery does not require a new direct user turn, a cooldown, or administrator
approval. A user, assistant, or tool continuation may clear the state when the
currently submitted canonical context completes uncached deep review with an
exact Allow. Warn, Block, invalid response, timeout, extraction failure, empty
selection, or recovery-state failure keeps upstream access blocked and leaves
recovery required.

The system does not reattach content that the client removed after the Block.
Removing the offending segment and passing full review is accepted remediation.
This intentionally does not prove that the client's higher-level objective has
changed; Prompt Audit judges the canonical content actually submitted.

### 4. Synchronous and asynchronous Blocks share one versioned state

Synchronous Prompt Audit Block and asynchronous deep Block SHALL both write the
existing user-level `deep_required` state with a new source-identifying token.
There is no independent sync and async requirement to clear.

Recovery retains compare-and-delete semantics. If a newer synchronous or
asynchronous Block refreshes the token while recovery is in progress, the
older recovery Allow cannot clear the newer finding and must fail closed. The
token source may be logged as bounded metadata, but raw prompts and client
identifiers remain excluded.

Before a combined ordinary Allow commits receipts, enqueues deep review, or
returns to the upstream path, Coordinator performs a final recovery-state
fence. A state created while that request was being synchronously reviewed
therefore converts the result to required recovery. As with asynchronous late
Block, a request that already completed the final gateway audit gate cannot be
retroactively cancelled.

### 5. Existing client error contracts remain stable

An initial synchronous Block continues to return HTTP 403 with
`prompt_guard_blocked`. A recovery result other than exact Allow returns HTTP
403 with `prompt_guard_deep_review_required`. Audit dependency, extraction, or
recovery-state failures continue to return HTTP 503 with their existing stable
error codes. Existing WebSocket mappings remain 4403 for policy/recovery Block
and 1013 for unavailable dependencies. No HTTP 409, 423, or new retry contract
is introduced.

### 6. Any selected synchronous source may trigger recovery

A synchronous aggregate Block writes recovery regardless of whether the
selected input came from current or historical user text, system/instructions,
assistant, reasoning, prompt variables, tool definitions, tool calls, or tool
outputs. The configured blocking module selection already expresses the
administrator's intent to enforce those sources, and Guard chunks may combine
multiple canonical segments.

This recovery state remains review state only. A non-user finding does not
create a Content Moderation hash, violation count, automatic ban, or
notification that attributes misconduct to the user.

### 7. Recovery state does not expire

The shared user-level `deep_required` key SHALL have no TTL, matching existing
asynchronous late-Block behavior. Time, process restart, API-key rotation, and
session rotation do not clear it. Only compare-and-delete after complete exact
Allow removes the active version through the ordinary request path.

The state is small bounded metadata in Redis. Redis persistence and recovery
remain deployment responsibilities; failure to read or write state fails
closed rather than being interpreted as no requirement.

### 8. Observability uses logs and existing runtime metrics

Prompt Audit SHALL emit bounded structured events when synchronous or
asynchronous Block writes recovery, exact Allow clears recovery, a non-Allow
result retains recovery, or recovery-state access fails. Fields may include
source, user ID, request/job ID, config version, decision, and stable error
code. They must not include raw state tokens, prompts, tool values, media, or
client session identifiers.

The existing Prompt Audit runtime snapshot SHALL expose cumulative counters for
these transitions. This change does not add a database table, recovery listing,
new management endpoint, or administrator clear action.

### 9. Disabling blocking pauses enforcement without clearing state

Recovery enforcement requires effective blocking Prompt Audit. An explicit
administrator change that disables risk control, Prompt Audit, or synchronous
blocking pauses the recovery gate but does not delete `deep_required`. When
blocking is enabled again, the next request resumes required recovery. This is
an administrative policy override, not an automatic clearing path.

## Consequences

- The existing user-keyed `deep_required` storage can represent this state.
- Switching API keys, groups, or session identifiers cannot escape recovery.
- Recovery must still run before account selection, billing, concurrency
  acquisition, or upstream writes.

## Clarifications

Recovery lookup and recovery coverage are separate decisions. The user-level
state is checked first; only then does the request use the configured recovery
selection with receipts bypassed. Client-controlled session identifiers do not
participate in this lookup.

## Resolved Questions

All product and security-boundary decisions required for implementation are
resolved.
