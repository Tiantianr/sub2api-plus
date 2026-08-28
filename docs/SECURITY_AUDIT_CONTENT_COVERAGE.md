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
| Content Moderation | Selects only current direct-user text and images. Local keyword/hash rules run first. External text uses `auto`/`blocking`/`observe`/`off`; `auto` shadows text only when blocking Prompt Guard covers that exact group, otherwise it preserves blocking text moderation. Shadow findings record no hash/notification/ban effects. Images continue to follow the global mode. Instructions, assistant/model content, tool content, and tool-produced images are excluded from direct-user attribution. |
| Prompt Audit full/async | Scans all canonical text that can affect model behavior: messages, instructions/system context, reusable prompt variables, reasoning, tool definitions/calls/results, and search/embedding/media prompts. |
| Prompt Audit blocking FULL_REQUIRED | Scans the complete canonical text present in the request. This applies to new, expired, changed, unknown-parent, branched, or replay-mismatched conversations. Any blocked chunk blocks the whole request. |
| Prompt Audit blocking CLEAN | Scans the bounded sanitized AI output captured after the previous downstream transforms plus every current client-controlled non-static segment. Static context and old history are reused only after config, context, parent, and replay-continuity checks pass. |

Sharing a canonical document does not mean that the engines attribute content
identically. Content Moderation attributes only direct current user text and
images. Prompt Audit evaluates complete model-visible text on a full scan,
including client tool schemas and tool results. A recognized media-only turn
is a valid empty text selection and remains eligible for image moderation; an
unknown content-bearing turn is not empty and fails closed in blocking mode.

## Conversation Checkpoints

Blocking Prompt Audit stores temporary Redis `CLEAN` / `FULL_REQUIRED` state.
`Begin` atomically consumes the prior checkpoint, acquires one turn lease, and
sets `FULL_REQUIRED` before Guard or upstream work. Only Guard `Allow` followed
by a complete successful downstream response can atomically restore `CLEAN`.
Flag, Block, timeout, cancellation, invalid Guard output, Redis failure,
extraction failure, downstream error, missing terminal, parse ambiguity, or
capture overflow cannot advance the checkpoint.

For full-replay clients, incremental eligibility additionally requires the
non-current canonical history to equal the prior request input fingerprint
followed by the captured AI-output fingerprint. A safe latest message cannot
hide inserted or rewritten history. For server-side continuation,
`previous_response_id` is only a hashed index to a checkpoint; it is not a
transcript and cannot prove or recover content absent from both the request and
temporary output state.

Known parent identity does not waive replay validation: when a parent request
also includes non-current canonical history, that history must still match the
prior input and output fingerprints. Active turn leases are token-bound and
fixed at 2 hours, above the current 1-hour Live and 15-minute WS limits.

HTTP/JSON/SSE output is observed through the final gateway response writer.
Responses WebSocket and Live Sideband observe frames only after final model,
tool-name, image-status, and error transformations succeed. Output is bounded
at 500000 Unicode characters; media, base64, and encrypted opaque values are
sanitized, then the retained output is application-encrypted before Redis
storage. Empty or non-canonical output, decryption failure, or overflow
invalidates the checkpoint instead of treating partial content as complete.

## Failure Semantics

All enabled engine paths expose `extraction_attempted`,
`extraction_succeeded`, `extraction_empty`, and `extraction_failed` counters.
Every extraction, evaluation, or audit-dependency exception emits a structured
log containing request ID, endpoint, protocol, stage, a stable error
code/reason, available byte counts, and bounded incomplete reasons. Logs must
not contain raw content, credentials, or unsanitized user fields. Extraction
failure is an audit-dependency outcome rather than a fabricated policy match;
blocking Prompt Audit reports it and prevents upstream side effects.

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
availability remains separate from Prompt Guard checkpoint semantics.

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
- Live lifecycle tests when Sideband classification or forwarding changes.

Route-call presence or static source-order assertions alone do not prove
content coverage.
