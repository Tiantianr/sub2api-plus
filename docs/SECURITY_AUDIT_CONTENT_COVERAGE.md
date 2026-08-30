# Security Audit Content Coverage

This document is the normative content-extraction matrix for Content
Moderation and Prompt Audit. The shared implementation is
`backend/internal/auditcontent`; protocol handlers and account paths must not
maintain alternate text extractors.

## Boundary And Ordering

Every accepted HTTP request, WebSocket turn, and Live Sideband client frame
must cross the same security-audit boundary after authentication and basic
request validation, but before:

1. account selection or API-key/OAuth credential normalization;
2. billing, quota reservation, or concurrency acquisition;
3. routing, retry, probe, fingerprint, or protocol transformations; and
4. any upstream request or frame write.

Session affinity, account type, inbound role labels, envelope `type` values,
and protocol adapters cannot bypass the audit hook. In blocking mode,
malformed, unsupported, partially extracted, or otherwise incomplete content
fails closed; successfully extracted siblings do not turn a partial result
into a complete audit. Async mode retains best-effort enqueue semantics.

## Canonical Result

The shared extractor returns a protocol-independent document containing text
segments and image inputs. Both carry `Role`, `Source`, `Current`, and
`ClientControlled`; the document also carries `ContentBearing` and `Incomplete`
classifications.

- `Current` identifies the new content in this request or turn. Consecutive
  trailing tool results are all current.
- `ClientControlled` is independent of the claimed role. A current
  `assistant` or `model` message remains client-controlled inbound content.
- Structured tool arguments and results are encoded as deterministic JSON.
- Recognized media blocks are explicit no-text content. Images remain in the
  canonical result with the same role, source, and current-turn attribution as
  their containing item. Prompt Audit does not scan URLs or encoded media as
  prompt text. Structured results are sanitized before text
  serialization so image/file URLs, data URLs, long base64 payloads, encrypted
  compaction data, screenshots, and image-generation results are not persisted;
  ordinary text beside those fields remains auditable.
- Any non-empty recognized content item that cannot be completely normalized
  sets `Incomplete` for metrics and structured logs. Blocking Prompt Audit
  rejects that request as an extraction failure before upstream side effects;
  async Prompt Audit may still retain successfully extracted siblings.

## Protocol Matrix

| Protocol family | Canonical text sources | Current-content rule | Explicit no-text or control cases |
| --- | --- | --- | --- |
| OpenAI Chat Completions | `instructions`; `tools` and `functions`; `messages[].content`; `tool_calls[].function.arguments`; `function_call.arguments`; tool/function-role results, including structured content | Last message is current; if the tail contains tool/function results, every consecutive trailing result is current; system/developer context is current audit context | Recognized image/video content blocks |
| Anthropic Messages | `system`; `tools`; message text and thinking text; client/server tool-use input; tool-result content, including structured content | Last message is current; system and tool definitions remain current audit context | Recognized image blocks and encrypted `redacted_thinking` blocks |
| OpenAI Responses HTTP and WebSocket | Top-level, `response`-nested, or session-update `instructions`, `tools`, native `input`, legacy `messages`/string `prompt`, and reusable `prompt.variables`; message/reasoning/refusal text; function/custom/tool-search outputs; local/hosted shell, apply-patch, computer, MCP, code-interpreter, program/program-output, additional-tools, and accepted search call payloads. Native non-null `input` has ingress-equivalent precedence over legacy aliases. `POST /v1/responses/input_tokens` uses the same document before account selection. | Last input item is current; every consecutive trailing recognized output is current; dynamic definitions are current context; a claimed system/developer role remains context and all other current roles remain client-controlled | Recognized media and shape-validated encrypted compaction fields produce no text. Unknown or partially understood content is incomplete and fails closed under blocking audit. |
| OpenAI Live | Initial session instructions, tools, input, legacy `input_audio_transcription.prompt`/`keywords`, current `audio.input.transcription.prompt`/`keywords`; `session.update`; `transcription_session.update`; `conversation.item.create`; Live-shaped `response.create` | Every initial HTTP session and accepted Sideband client frame enters the audit hook before its downstream side effect or upstream write | Known control-only events produce no text. Unknown content-bearing frames are incomplete and close a blocking Sideband connection before forwarding. |
| Alpha Search | Complete sanitized deterministic JSON for `commands`, `settings`, and `input` | All three values are current because PAT fallback can place them in one upstream user prompt | Empty fields and recognized media/opaque values |
| OpenAI Embeddings | String or string-array `input` | Every string input is current | Empty input is no text; unsupported token-ID arrays are incomplete and fail closed under blocking audit |
| Gemini | `systemInstruction`/`system_instruction`; tools; `contents`/`content`; batched `requests`; `instances[].prompt`; part text; `functionCall` arguments; `functionResponse.response` | Last content item is current; system and tools remain current audit context | `inlineData`/`fileData` media-only parts |
| Images and media | Deterministic prompt-like keys such as prompt, description, query, lyrics, negative prompt, and input | Every extracted prompt is current; duplicate text is emitted once | HTTP(S) URLs, `data:image`/`data:video` values, and large base64-like media payloads |

For unknown protocol labels, the fallback recognizes Chat-shaped `messages`,
Responses-shaped `input` or `instructions`, Gemini-shaped content, Alpha
Search commands, and finally the media prompt allowlist. Unrecognized values
set an incomplete result; blocking Prompt Audit rejects them until an explicit
adapter makes the new content auditable by both engines.

An envelope `type` value or empty top-level field never overrides content
present elsewhere in the payload. In particular, a non-`response.create` type
carrying `input`, `instructions`, or nested `response.input` is still
content-bearing, and top-level plus nested Responses fields are both inspected.
An unsupported envelope type is counted and safely logged as an extraction
failure even when sibling fields were extracted. A media type label does not
suppress recognized text in the same content block. Unsupported item types,
unknown Responses/Live content frames, valid-JSON unrecognized structures,
and other incomplete content prevent blocking-mode forwarding. Responses
passthrough and Live Sideband invoke the audit hook before every upstream frame
write.

A non-empty Responses or Live root, nested `response`, or session object
containing no recognized request, control, content, or metadata field is an
observable extraction failure rather than an ordinary empty request. Blocking
Prompt Audit rejects it; async mode records it without affecting forwarding.
Unknown sibling keys remain ignored only when their containing object is fully
classified by recognized fields.

## Engine Selection

Both engines consume the same canonical document:

| Engine/mode | Segment selection |
| --- | --- |
| Content Moderation | Scans only current direct-user text and images. Chat and Anthropic require an explicit `user` role; Responses, Live, and Gemini also accept their protocol-defined roleless user forms. Direct Alpha Search queries, embedding strings, and media prompts remain eligible. Instructions, system/developer context, reusable prompt variables, assistant/model messages, reasoning, tool definitions/calls/results, approval responses, and tool-produced images are excluded so platform or external content is not attributed to the user. |
| Prompt Audit async / async-deep | Selects user turns plus configured instructions/system/developer context, assistant/model messages, reasoning, reusable prompt variables, tool definitions, tool-call arguments, and tool outputs. An async-only current user turn is mandatory. Async-deep under blocking may omit current, historical, and automatic segments with valid per-segment Allow receipts, including the same request's trusted handoff. |
| Prompt Audit blocking | Scans direct user text marked `Current` unless an unexpired blocking Allow receipt certifies the same user, policy, source, and receipt-normalized canonical text. Every historical user turn has the same receipt requirement; misses are synchronously reviewed so client-controlled role ordering cannot hide unreviewed text. Configuration independently adds the same source modules. `blocking_latest_turn_only` remains a compatibility field and does not override module selection. Any aggregate Block writes user-level deep-review state before returning 403. A user carrying that requirement is synchronously reviewed with the active async-deep module selection and all receipts bypassed, regardless of API key, group, or client session identity. |

Sharing a canonical document does not mean that the engines select identical
segments. Content Moderation preserves the `v0.1.177+custom.003` attribution
rule: only a direct user submission may produce a user content-policy
violation. Prompt Audit may inspect non-user sources when the corresponding
module is enabled, but such a finding is review state rather than proof of user
misconduct. The `system` module also retains `<system-reminder>` content so
user-installed skills can be reviewed. `<environment_context>`,
`<permission_profile>`, and `<filesystem>` blocks are still removed from user
text. A turn containing only unselected modules is a valid empty Prompt Audit
selection. Incomplete canonical extraction is observable. Blocking and forced
deep review fail closed; async modes evaluate selected siblings and record the
extraction defect.

Responses `agent_message` items retain visible supported content. Recognized
authors preserve user/system/developer/assistant/model attribution; other named
agents are assistant sources so they cannot create direct-user punishment.
Opaque `encrypted_content`, recipient routing, and bounded turn metadata are
not persisted or sent to Guard. Other unknown content blocks remain extraction
failures.

Configurations created before these module maps existed default synchronous
review to system/instructions, prompt variables, and tool definitions. Deep
review defaults all optional modules on. Administrators may independently
disable any optional module. New or changed current-user text remains mandatory
in blocking mode; receipt-equivalent repeats may reuse a valid blocking receipt.

Prompt Audit issues per-segment Allow receipts for canonical user turns and
individually classified optional segments. A receipt is reusable only for the
same receipt-schema revision, user, config version, enabled Guard
endpoint/scanner policy, source class, and receipt-normalized canonical text.
For user segments only, exact `[images:<hex>]` markers with a 32-, 40-, or
64-character hexadecimal identifier are replaced by `[images]` for the receipt
key; Guard still reviews the original marker on a miss, marker count remains
significant, and all other text remains exact. Adding
one tool result therefore misses only that result instead of invalidating the
entire tool-output history. Misses are
concatenated into one Guard input, so a cold request does not make one call per
segment. In blocking mode, current user text may reuse a stored exact receipt,
and an exact complete synchronous Allow is handed to the same request's
`async_deep` enqueue in-process. An async-only current user always ignores
stored receipts. Only a complete
aggregate `Allow` from a request permitted by Content Moderation writes every
submitted receipt. The TTL is
administrator-configurable and defaults to one hour. A Redis error falls back
to ordinary Guard review. Warn, Block, timeout, invalid, extraction failure,
and partial results never create receipts. Redis keys contain hashes, not
prompt or tool values.

Synchronous receipts remain pending until Content Moderation and Prompt Guard
both permit the request. Async jobs receive internal receipt-write permission
only after the original request is permitted; blocked or unavailable requests
may still be observed but cannot create reusable receipts.

Queued jobs are bound to their configuration version. A worker fails a stale
job without calling Guard or writing receipts rather than using a newer policy
to certify an older key.

## Prompt Audit Event Evidence

Guard selection and review evidence are separate. The event `full_prompt`
contains the exact unredacted Guard input. For each newly stored event, the
complete canonical document is serialized with segment source, role, current,
and client-controlled attributes, plus the exact Guard input and extraction
diagnostics. This complete context includes instructions, assistant/model text,
reasoning, prompt variables, tool definitions/calls/results, and harness blocks
that were excluded from Guard selection.

Complete context is gzip-compressed, application-encrypted, and stored in
`prompt_audit_event_contexts`, separate from event list/detail rows. Only the
authenticated admin download endpoint decrypts it; responses use `no-store`
and downloads never enter application logs. Event deletion cascades to the
context artifact. Events created before this migration have no recoverable
complete-context artifact.

Complete context records `allow_receipt_status` and hit/miss counts. On a full
or partial hit, `guard_input`, event prompt metadata, hashes, and chunk counts
describe only the receipt misses; every selected canonical source segment
remains in encrypted context for review.

Pass evidence retention is independently selected by authenticated user ID and
defaults to an empty selection. An unselected user's Pass result still
completes review, metrics, deep-review scheduling, and applicable Allow receipt
work, but it creates no event or complete-context artifact. Flag and Critical
events always retain the same encrypted evidence regardless of this optional
Pass-retention setting. Changing the retention list has its own revision and
does not change Guard policy identity, invalidate receipts, or make queued jobs
stale.

The administration cleanup shortcut is fixed to Pass events and requires a
displayed server preview with an explicit time range, snapshot high water,
administrator-bound confirmation token, context count, and estimated stored
content bytes. Cleanup reuses event cascade deletion and never deletes Allow
receipts. The estimate describes future logical-backup reduction; it does not
promise immediate filesystem reclamation.

Blocking Allow starts a best-effort `async_deep` job only after Content
Moderation and Prompt Guard both permit the request. A synchronous aggregate
Block writes a versioned per-user Redis requirement before returning 403; an
asynchronous deep Block writes the same state before its job completes. The
next request uses the active configured deep modules synchronously, bypasses
all receipts, and may clear only the exact version it claimed after complete
Allow. Flag, Block, empty selection, dependency failure, and a newer concurrent
finding keep the non-expiring requirement and prevent upstream access. The
state is independent of API key, group, and client session identity. It does
not create a Content Moderation hash, violation count, or automatic penalty.
Coordinator performs a final state fence before an ordinary combined Allow can
persist receipts, enqueue deep review, or return to the upstream path. Explicit
administrator disabling of risk control, Prompt Audit, or blocking mode pauses
enforcement without clearing the Redis state; re-enabling blocking resumes it.
Requests that already completed the final audit gate cannot be retroactively
cancelled.

## Failure Semantics

All enabled engine paths expose `extraction_attempted`,
`extraction_succeeded`, `extraction_empty`, and `extraction_failed` counters.
Prompt Audit also exposes incremental Allow receipt hit, miss, write, and error
counters. Blocking Guard failures additionally expose `failure_allowed` when
the active, default-on `allow_on_guard_unavailable` policy lets an ordinary
request continue after all Guard nodes end in `prompt_guard_unavailable`.
Unavailable and timeout counters still record the underlying failure.
Every extraction, evaluation, or audit-dependency exception emits a structured
log containing request ID, endpoint, protocol, stage, a stable error
code/reason, available byte counts, and bounded incomplete reasons. Extraction
failure logs also contain a shared `failure_nodes` description resolved from
the canonical failure path: the constrained item type and role, value kind,
sorted bounded object keys, a value-free JSON shape, node byte count, and
stable item-type/shape fingerprints. Scalar values other than protocol
discriminators are replaced with type markers such as `$string`; suspicious
or credential-like identifiers are redacted and retain only a fingerprint.
Logs must not contain raw content, credentials, media, or unsanitized user
fields. Extraction failure is an audit-dependency outcome rather than a
fabricated policy match; blocking Prompt Audit reports it and prevents
upstream side effects.

Content Moderation applies the same log contract to asynchronous persistence,
hash-cache, account-side-effect, notification, worker, cleanup, runtime, and
post-upstream cyber-policy failures. These logs use stable error categories;
they do not include raw dependency errors, panic values, or recipient email
addresses. Prompt Audit applies it to enqueue, payload-store, job-claim,
completion/retry/failure persistence, worker, reclaim, startup,
shutdown, and runtime health failures.

| Engine/mode | Content-bearing extraction failure |
| --- | --- |
| Content Moderation observe | Record failure; evaluate any selected extracted content; otherwise allow |
| Content Moderation pre-block | Return `content_moderation_unavailable` for incomplete extraction, hash-store failure, missing required API credentials, or synchronous text/image API failure; do not count it as a policy violation. |
| Prompt Audit async | Record failure; enqueue successfully extracted content or skip an empty snapshot; never affect request forwarding |
| Prompt Audit blocking | Reject malformed or incomplete content before Guard/upstream; allow only a completely recognized media/control-only empty text selection. |

A confirmed policy match continues to use `content_policy_violation` or the
Prompt Audit block decision. Extraction failure uses a distinct dependency
error code rather than a content category. Content Moderation external API
availability remains separate from Prompt Guard selection semantics.

Blocking Prompt Audit failure-allows eligible Guard unavailability by default,
including when an older persisted configuration lacks the field. An
administrator may explicitly disable `allow_on_guard_unavailable` to require
fail-closed availability. When enabled, the policy changes
only the final action for ordinary synchronous requests after node timeout,
connection/API failure, authentication failure, or capacity saturation.
Missing usable nodes, undecryptable credentials, local client construction,
and scanner wiring failures are not eligible. A failure-allowed request has no
Safe result and creates no Allow receipt, including if its best-effort
asynchronous deep review later succeeds. Strictly invalid Guard responses,
partial or failed content extraction, encryption/configuration failure, known
Flag or Block results, Content Moderation failure, and required user recovery
remain fail closed. Required recovery also retains its Redis state when Guard
is unavailable. The structured `prompt_guard.failure_allowed` event and
runtime counter make every use observable; best-effort asynchronous deep
review may still be queued after the request passes the independent final
recovery fence.

Deterministic structured serialization is part of extraction. Sanitization or
JSON serialization failure sets `Incomplete`; async audit may retain extracted
siblings, while blocking audit rejects the partial result.

Live Sideband and Responses passthrough are control connections: every client
text or binary frame enters the audit hook before `upstream.WriteFrame`.
Unsupported, binary, or otherwise unextractable content closes a blocking
control connection before upstream forwarding. WebRTC media does not
traverse the Sideband control connection and is outside this text-extraction
boundary.

## Change Evidence

Any endpoint, accepted payload field, tool form, role rule, control event, or
protocol transform that can affect inbound content must update this matrix in
the same change and provide all of the following evidence:

- production-shaped shared-extractor tests;
- the dual-engine contract in
  `backend/internal/handler/security_audit_content_contract_test.go`;
- Content Moderation and Prompt Audit payload/selection tests;
- HTTP and WebSocket ordering tests proving blocking extraction failures and
  confirmed policy blocks both produce zero account, billing,
  concurrency, or upstream side effects; and
- HTTP and WebSocket tests for API-key and OAuth identities proving an explicit
  Guard-unavailable policy allows downstream stages without creating an Allow
  receipt, while extraction and recovery failures remain before side effects;
  and
- Live lifecycle tests when Sideband classification or forwarding changes.

Route-call presence or static source-order assertions alone do not prove
content coverage.
