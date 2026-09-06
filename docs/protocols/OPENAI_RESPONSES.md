# OpenAI Responses and WebSocket Ingress

Sub2API Plus accepts OpenAI-compatible Responses requests over HTTP and
client-facing WebSocket ingress. Account routing can use an upstream WebSocket
or bridge the client WebSocket to an HTTP/SSE upstream.

## GPT-6 Astra

The gateway exposes the canonical `gpt-6-astra` model ID and retains the
existing `gpt-6` and recognized reasoning/date suffix compatibility forms.
These compatibility forms share Astra's canonical upstream behavior; unknown
model suffixes are not promoted into Astra aliases. Astra accepts `low`,
`medium`, `high`, `xhigh`, and `max` reasoning effort. Configured Codex
catalogs default it to `low` and preserve `max` as distinct from `xhigh`; the
additional `ultra` catalog level is a client-side multi-agent preset that
delegates with `xhigh` and is never sent upstream as an API reasoning effort.

Generated API client configuration advertises the 1,050,000-token API context
window and 128,000-token maximum output. Configured Codex catalogs follow the
Codex client contract separately: the active context window is 272,000 and the
maximum configuration override is 872,000. Astra advertises text and image
input, original image detail, text-and-image web search, freeform apply-patch,
code-mode-only tools, multi-agent v2, and WebSocket preference metadata.

Default OpenAI model catalogs and model-mapping presets list Astra first. The
admin account test UI still prefers `gpt-5.6-sol` when it is available, so
catalog presentation order does not silently change the connectivity probe.

Default API cost accounting uses $10 input, $1 cached input, $12.50 cache
write, and $50 output per million tokens. Flex is half of Standard and Fast is
twice Standard. When total input is 272,001 tokens or more, the whole-request
long-context policy multiplies ordinary input, cache-read input, and
cache-write input by 2 and output by 1.5. Platform API-key Astra requests
always apply this official policy; ChatGPT OAuth account policy remains
independently configurable. Group and channel custom selling-price overrides
retain their existing precedence.

`prompt_cache_options` is supported for Astra on OpenAI Platform API-key
Responses and Compact traffic. Callers should use `prompt_cache_options.ttl`
with `30m`; `mode: explicit` disables the implicit cache breakpoint.

## OAuth History Admission

Credential-owning OpenAI OAuth accounts default
`extra.openai_oauth_reject_external_history` to `true`, including accounts that
omit the field. An explicit boolean `false` disables only this candidate gate.
Spark shadows inherit the parent's policy; API-key, setup-token and other
provider accounts are unaffected. Create, edit and opt-in bulk editing expose
the setting. Bulk policy changes must target OAuth credential owners, not
shadows. Invalid boolean values are rejected by management writes.

A fresh conversation is eligible even when its newly generated session ID has
no binding. A `previous_response_id`, recognized historical message/tool/state
input, or an existing conversation binding requires history admission. Shared
canonical extraction supplies the history classification; system/developer
instructions and tool declarations alone are not history. Validated call-less
Codex automation/delegation bootstrap input retains its existing new-user-input
exception. This is ID-based routing, not proof of where every text segment was
generated; pasted ordinary user text cannot be reliably identified as history.

For historical requests, a strict OAuth candidate must match the recorded
credential owner and identity. Unknown ownership or another owner excludes
that candidate without acquiring its slot, contacting it, or marking it
unhealthy. Other candidates remain eligible under their own policies. A durable
owner is preferred over weighted sticky routing, subject to all existing
permissions, capability, health, quota and concurrency gates. Retries use the
original ownership snapshot; newly written sticky routes do not authorize the
original history. A missing response binding cannot be hidden by a matching
session binding. Disabling this policy does not bypass HTTP response-user
ownership checks, OAuth sharing scopes, moderation or existing replay safety.

PostgreSQL `openai_conversation_bindings` stores user-scoped session/response
key hashes and account/credential metadata, never prompts or tool content.
The mappings have no idle TTL. One request reuses its durable ownership lookup
across candidate checks. Redis and process-cache expiry, process restarts and
multi-day inactivity therefore do not lose recorded ownership. HTTP response
authorization uses the same durable records. Account cooldown and quota changes
do not delete them. Normal OAuth refresh preserves ownership; a different
upstream OAuth identity or sharing scope invalidates the old identity snapshot.
User/account deletion removes the associated records. Response ownership cannot
be reassigned; mutable session routing uses conditional writes to avoid silent
concurrent overwrites. Real upstream response IDs are persisted before being
exposed to clients, including streaming and WebSocket delivery.

Coverage includes Responses HTTP and Compact, all Responses WebSocket modes,
Chat Completions/Messages compatibility and upstream history-bearing token
counting. Token counting checks admission but never creates conversation
ownership. Each later WS content request is audited and checked before upstream
writes. A connection that can no longer use its selected account requests
reconnection for normal candidate selection, not immediate conversation reset.

If this gate excluded otherwise serviceable candidates and no eligible account
can continue, HTTP returns `400`, type `invalid_request_error`, code
`external_history_not_allowed`, with message
`当前没有可接续此历史对话的账号，请新建对话后重试。` Messages uses its existing
Anthropic-compatible error envelope. WebSocket uses an error event. Ownership
storage failures return a service error, and concurrent routing conflicts are
retryable conflicts; neither is reported as an external-history denial.

An upgrade can backfill still-authorized cached response ownership. A legacy
group-only session cache is not user ownership evidence and is not promoted.
Already expired legacy bindings cannot be reconstructed. Preserving gateway
ownership also does not extend the provider's response or connection lifetime.
See the [deployment notes](../../deploy/OPENAI_HISTORY_ADMISSION_CN.md).

## Prompt Cache Identity and Usage

Current Codex clients can supply the canonical `session-id`, `thread-id`, and
`x-client-request-id` headers. The gateway also accepts the supported legacy
session aliases, including `session_id`, for sticky routing compatibility.
When direct session headers are absent, the stable string `session_id` inside
`X-Codex-Turn-Metadata` is also accepted; request-scoped `turn_id` is never a
routing key. This lets `/v1/responses` WebSocket ingress and
`/v1/alpha/search` share one account route even when Alpha Search would
otherwise fall back to its search ID.
The same sanitized value is recorded as `usage_logs.session_id` and participates
in cyber-session blocking. Endpoint fallbacks and `turn_id` remain routing
details and are not persisted as client session IDs.
Thread and client-request identifiers remain paired request context and never
replace the session-scoped cache identity.

For OpenAI Responses and Compact requests, the gateway resolves one opaque,
tenant-isolated cache identity from an explicit `prompt_cache_key`, a supported
session header, or a stable content prefix with a meaningful user/input anchor.
It writes the finalized UUID to both the upstream `prompt_cache_key` and the
canonical `session-id` header; the legacy `session_id` alias carries the same
value. A model-only request does not receive a content-derived key. API-key
Chat Completions requests converted to Responses use the same behavior, while
raw Chat Completions forwarding does not receive Responses-only cache fields.

Under the default hard-affinity mode, account priority changes do not replace a
valid active session route. The optional sticky-weighted scheduler mode remains
score-based except for a durable OAuth history owner as described above. If the configured health/concurrency sticky escape
temporarily bypasses a degraded account, a movable Responses continuation does
not fall back to that account through its older response ID; the temporary
candidate order is derived from the shared session identity, while the
canonical sticky binding remains unchanged. Removing an account from the
requesting group, disabling it, or making it incompatible is always a hard
invalidation: both ordinary sticky routing and `previous_response_id` routing
recheck the current account before reuse. Long-lived Responses WebSocket
connections also run current billing and reload the selected account before
every turn in `ctx_pool`, `http_bridge`, and `passthrough` modes. The refreshed
account must still satisfy group, status, schedulability, quota, client-model,
endpoint-capability, transport, parent-health, runtime-block, scheduling
threshold, and proxy-quarantine gates. Follow-up image intent also repeats the
group permission and Responses-capability checks. If any gate fails, the
gateway closes before sending that turn upstream; a reconnect then performs
normal account selection. The first turn is billed-eligible once before
selection, while every later turn is checked once at its pre-turn boundary. A
movable WebSocket continuation follows a session route that has already failed
over to another account and removes the old account's response ID before
forwarding. Tool-output continuations without complete call context remain on
the response owner and keep the existing fail-closed OAuth ownership checks.

`prompt_cache_options` is forwarded only for GPT-6 Astra and GPT-5.6-family
OpenAI Platform API-key Responses/Compact traffic. ChatGPT OAuth and older or
unknown model families have that field removed. Deprecated
`prompt_cache_retention` is removed on every path. Current callers should use
`prompt_cache_options.ttl` with the only supported value, `30m`; setting
`mode` to `explicit` disables the implicit cache breakpoint.

Usage ingestion treats ordinary input, cache-read input, and cache-write input
as mutually exclusive stored buckets. Usage pages may report each bucket's
share of total tokens, but do not derive a prompt-cache hit rate from these
counters.
Canonical nested usage details take priority by field presence, including an
explicit zero, before known top-level compatibility aliases are considered.

## Codex Rate-Limit Response Headers

For Codex Responses requests, successful HTTP and SSE responses can include
the following rate-limit fields for both `Primary` and `Secondary` windows.
WebSocket upgrade responses include the same fields when the local Codex
subscription quota view is enabled:

- `X-Codex-*-Used-Percent` is the consumed percentage.
- `X-Codex-*-Window-Minutes` is the window length in minutes.
- `X-Codex-*-Reset-At` is the next reset time as Unix seconds.
- `X-Codex-*-Reset-After-Seconds` remains alongside `Reset-At` for older
  clients that only understand a relative countdown.

When local Codex subscription quota is enabled, `Primary` represents the local
7-day window and `Secondary` represents the local rolling 5-hour window. This
local view is authoritative even when the selected OpenAI account enables
automatic passthrough. The gateway clears all upstream default rate-limit
fields before writing the local values, so one response never mixes upstream
reset times with local percentages or window sizes.

When the local view is disabled, the selected account's real default quota is
visible only when that account enables OpenAI automatic passthrough. Otherwise
the gateway removes the default `Primary` and `Secondary` quota fields after
generic response-header filtering, so an additional response-header allowance
cannot bypass this policy.

A client WebSocket `101` response is committed before the gateway connects to
the selected upstream. The gateway therefore writes only the local quota view
known before the upgrade. The official WebSocket client updates quota from
in-band `codex.rate_limits` events, so the gateway applies the same policy to
the default `codex` event family: replace it with local subscription windows,
pass it through for an automatic-passthrough account, or suppress it. Named
model-specific limit families remain independent. HTTP and SSE headers are
finalized before their response bodies are written.

The dedicated `/backend-api/wham/usage` route remains a local-only view. It
returns the API key subscription quota when the local setting is enabled and
returns `404` otherwise; it does not select an account or proxy upstream quota.

This response-header compatibility does not make Codex App API-key calls to
`account/rateLimits/read` available; that App Server authentication behavior is
outside this gateway's request path.

## Codex Fingerprint Convergence

OpenAI OAuth accounts may rewrite outbound Codex installation, session, and
thread carriers. Unset accounts use `device` mode. Ordinary Responses,
Chat-Completions-to-Responses, Messages-to-Responses, HTTP-to-WebSocket, and
direct Responses WebSocket turns use the configured account mode. Native
remote Compact v2 is an ordinary Responses session for fingerprint purposes
and therefore also uses the full configured mode. The ChatGPT Codex OAuth
legacy compact compatibility path uses installation-only convergence for every
non-`off` mode and preserves its own compact session, cache, and thread
namespace.

Here `legacy` refers only to the ChatGPT Codex OAuth compatibility branch used
by this gateway. The public API-key
[`/v1/responses/compact`](https://developers.openai.com/api/reference/java/resources/responses/methods/compact)
endpoint remains a distinct supported OpenAI API surface. Response retrieve,
cancel, and other non-create subpaths are not session turns and receive no
fingerprint mutation.

Fingerprint preparation runs before final request construction. Plus
prompt-cache/session isolation is authoritative for the final `session-id` and
`session_id` headers, while fingerprint convergence remains authoritative for
installation and thread/turn metadata. `off` disables only fingerprint-owned
header and body mutation; it does not disable Plus cache isolation, security,
session sharing, or compact policy. WebSocket connection reuse compares final
stable handshake carriers even when `off` or `device` leaves those values
client-owned. Usage-log `session_id` stays the sanitized client-original value.

Only credential-owning OpenAI OAuth accounts participate. Personal access
token and Agent Identity accounts follow the same endpoint semantics because
they are OpenAI OAuth credential owners. API-key, setup-token, and non-session
endpoints such as count-tokens and alpha-search are excluded. User-Agent,
Originator, and Version use one source chain: valid credential-owner
`credentials.user_agent`, then valid global `openai_codex_user_agent`, then the
compiled default. Version synchronization changes only the version declaration
of the selected identity.

## Security Audit Content Boundary

Inbound Responses content is normalized for Content Moderation and Prompt
Audit before account selection, billing, concurrency acquisition, fingerprint
convergence, request adaptation, or upstream writes. API-key and OAuth account
paths therefore use the same audit content.

The canonical boundary covers top-level and `response`-nested `instructions`,
`tools`, `input`, reusable `prompt.variables`, message text, tool definitions,
and the arguments, input, output, result, or dynamic tools carried by function,
custom, tool-search, local/hosted shell, apply-patch, computer, MCP,
code-interpreter, and programmatic-tool-calling items. Media fields and
encoded screenshots are removed before text serialization and persistence;
ordinary text in the same structured result is retained.

Content Moderation consumes the same canonical result but selects only the
current direct-user message text and images. It excludes `instructions`, tool
definitions, reusable prompt variables, assistant/model messages, reasoning,
tool calls/results, approval responses, and tool-produced screenshots. This
prevents platform context or external tool content from being reported as a
user policy violation. Prompt Audit consumes the same canonical result.
Synchronous review always includes direct user text marked current. Every
historical user turn requires a valid receipt and misses are synchronously
reviewed, preventing client-controlled role ordering from hiding unreviewed
text. Each lane
independently configures whether
instructions, assistant/model output, reasoning, reusable prompt variables,
tool definitions, arguments, and results enter Guard input. The system module
also includes `system-reminder` skill content, while environment, permission,
and filesystem wrappers remain excluded from user text. Canonical user turns
and optional segments receive independent exact-content Allow receipts. A new
current user turn ignores old receipts; a complete synchronous Allow may be
reused only by the same request's `async_deep` handoff. Historical user,
assistant, reasoning, and tool segments are omitted when their user-scoped
receipt and active Guard policy still match. New receipt misses are combined in
one incremental Guard input. Receipts are written only after Content Moderation
also permits the original request. A blocking Allow starts an `async_deep`
job. A synchronous Block writes user-level recovery before returning 403, and
an asynchronous deep Block writes the same state before its job completes.
Either forces the user's next request through the active configured deep
selection synchronously with all receipts bypassed before WebSocket or HTTP
upstream writes can resume. API-key, group, and client session identity changes
do not avoid recovery; only complete exact Allow clears the observed finding
version. A separate bounded per-user claim lease serializes recovery without
replacing the non-expiring finding, so concurrent requests and process failure
cannot strand a claim as the recovery requirement. A concurrent request waits
within its own request context instead of returning a claim-contention error.
If the claim owner clears the finding, the waiter resumes ordinary blocking
review; otherwise one waiter acquires the released claim and performs the next
complete recovery review. Redis access failures remain fail closed.
For ordinary blocking review, failure-allow is limited to network/read errors,
timeouts or capacity, 401/403, 429, and 5xx. Other deterministic 4xx responses,
invalid Guard output, extraction failures, and required recovery remain fail
closed.
Coordinator rechecks recovery state before an ordinary combined Allow commits
receipts, enqueues deep review, or returns to HTTP/WebSocket upstream handling.
Disabling blocking Prompt Audit pauses this gate without clearing pending state.
Newly stored audit events retain a
separately encrypted complete canonical context artifact for authorized admin
download, including text excluded from Guard selection.
A supported WebSocket control frame may produce no audit input. Unknown sibling
keys, unsupported event/item types, and valid-JSON unrecognized structures are
observable extraction failures. When the canonical extractor recognizes
`input`, `instructions`, or nested `response.input`, an envelope `type` value
does not suppress those extracted segments. An unsupported envelope type is
still counted and safely logged as an extraction failure while those extracted
segments remain auditable.
Non-empty root, nested `response`, and session objects with no recognized field
are counted and safely logged as extraction failures;
unknown sibling keys on an otherwise recognized object remain ordinary success.
Direct passthrough runs the audit hook for every client text or binary frame,
including `conversation.item.create` and `session.update`, before any
non-`response.create` frame is forwarded. Unsupported binary/JSON content and
recognized items that cannot be normalized are logged. Successfully extracted
sibling content remains auditable. Blocking Prompt Audit fails closed on a
content-bearing extraction failure before upstream writes; async audit records
the defect without affecting forwarding. Compact
keepalive output and channel mapping start only after this gate. The audit uses
an immutable copy of the inbound body so compact normalization and reasoning
policy rewrites cannot remove content from the audited view.

The complete protocol/source matrix is maintained in
[`docs/SECURITY_AUDIT_CONTENT_COVERAGE.md`](../SECURITY_AUDIT_CONTENT_COVERAGE.md).

## Request Replay and Upstream Failures

When a Responses request replays a previous tool call, the gateway preserves an
item `id` only when its prefix matches the item type. In particular,
`custom_tool_call` uses `ctc...`; a mismatched item ID is removed rather than
rewritten. Its `call_id` remains unchanged so the paired
`custom_tool_call_output` continues to reference the original call.

The exact local proxy response `507` with
`exceeded request buffer limit while retrying upstream` is not an account or
model failure. The gateway stops replaying the request, keeps the selected
account eligible, and returns an OpenAI-compatible `413` with:

```text
Request payload is too large to retry safely
```

Reduce the request size or adjust the reverse-proxy retry-buffer policy before
retrying. The gateway does not retry that request through another account,
because doing so can duplicate work and billing while encountering the same
buffer limit.

Connection refusal/reset and HTTP `504` gateway failures are recorded as
provider or proxy transport failures rather than model-capacity failures. A
proxy-backed OpenAI account is isolated by the bounded proxy circuit, keyed by
the configured proxy ID; a shared proxy incident therefore does not directly
put every associated account into an account-level cooldown. Management error
records retain only structured, bounded diagnostic categories and never store
raw proxy URLs, credentials, or outbound User-Agent values.

For the native HTTP/SSE and WebSocket paths, diagnostics distinguish an edge
gateway timeout from the gateway's own response-header, first-semantic-output,
and WebSocket first-semantic-output deadlines. A first-output timeout may use
the existing single, pre-output controlled failover path; it is never enabled
after semantic output has reached the client.

## WebSocket Ingress Limits

`gateway.openai_ws` bounds the lifetime and aggregate count of client-facing
sessions independently from per-turn user and account concurrency:

```yaml
gateway:
  openai_ws:
    client_first_message_timeout_seconds: 30
    ingress_inter_turn_idle_timeout_seconds: 300
    max_ingress_connections_per_api_key: 64
```

- The first-message timeout covers receiving and decompressing the complete
  first client message.
- The inter-turn timeout closes idle sockets after a completed turn; `0`
  disables it.
- The API-key connection cap is distributed through Redis; `0` disables it.

Large contexts or slow image-heavy requests may require a higher first-message
timeout. The timeout expires before HTTP bridge routing and is not overridden
by bridge mode.

Distributed connection leases last 60 seconds and refresh every 20 seconds. If
a process cannot confirm a lease for a full lease lifetime, it closes the local
socket instead of continuing outside the global cap.

## Mode Router

Enable the v2 mode router before selecting an account WebSocket mode such as
`http_bridge`:

```yaml
gateway:
  openai_ws:
    mode_router_v2_enabled: true
```

The environment equivalent is
`GATEWAY_OPENAI_WS_MODE_ROUTER_V2_ENABLED=true`. Use `http_bridge` when the
client keeps a WebSocket while the selected upstream uses HTTP/SSE.
