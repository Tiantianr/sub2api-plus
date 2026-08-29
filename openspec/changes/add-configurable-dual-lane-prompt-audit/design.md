# Design

## Module selection

The canonical extractor remains the only content parser. Synchronous review
always starts with the latest user turn; asynchronous review always starts with
all user turns. Each lane independently selects system/instructions, assistant
messages, reasoning, prompt variables, tool definitions, tool-call arguments,
and tool outputs. The system module also controls whether `system-reminder`
content inside user text is retained. Environment, permission, and filesystem
wrapper blocks remain excluded from user text.

## Dual lane

Blocking Prompt Audit and Content Moderation finish before account selection,
billing, concurrency, or upstream writes. Only an allowed combined decision
enqueues `async_deep`. Queue admission remains best effort and does not delay
the allowed request. Deep jobs use a distinct execution mode so their events
can be listed independently.

## Late findings and recovery

An `async_deep` Block writes a version token to the user's Redis state before
the job is completed. The next request for that user synchronously uses the
configured deep modules, even when its group is outside the ordinary Prompt
Audit group scope. Only Guard Allow removes the exact token with compare-and-
delete. Flag and Block replace the token and reject the request; unavailable,
invalid, extraction, encryption, and state-store failures fail closed while the
requirement remains. This fencing prevents an older concurrent Allow from
clearing a newer deep finding.

The state is a review requirement, not a policy strike. Assistant, system,
reasoning, or tool-originated findings do not enter Content Moderation hashes,
violation counts, or automatic user penalties.

## Evidence

Every stored synchronous, async-only, and async-deep event retains the exact
Guard input and encrypted complete canonical context. Event APIs expose
`execution_mode=async_deep`; list and delete filters may select that mode.
