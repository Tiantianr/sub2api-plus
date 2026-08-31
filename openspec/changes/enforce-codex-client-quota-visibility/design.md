## Context

The gateway already computes local 7-day and rolling 5-hour subscription
windows and can overlay them on selected responses. The existing helper does
not know which account produced a response. As a result, it cannot distinguish
an account whose automatic-passthrough contract permits real upstream metadata
from an account whose upstream quota must remain private. The official Codex
WebSocket client also consumes in-band `codex.rate_limits` events rather than
the already committed client upgrade headers.

## Decisions

### One three-state policy

Every client-facing default Codex quota value resolves in this order:

1. `local`: an enabled local quota view for the request's OpenAI subscription
   group is authoritative.
2. `upstream`: with local quota disabled, the selected account may expose its
   real values only when OpenAI automatic passthrough is enabled.
3. `hidden`: all other requests remove or suppress default quota values.

The policy uses the selected account only as a visibility decision. It does not
select, mutate, cool down, or persist that account.

### Finalize after generic filtering

HTTP response fields are finalized after generic response-header filtering and
before a body, SSE event, JSON conversion, or error payload is committed. This
keeps an administrator's additional response-header allowance from bypassing
the product-level visibility rule. First-output failover paths finalize a
per-attempt header map before that map is committed, preserving existing
failover isolation and leaving the upstream response header map unchanged.

### Transform only the default WebSocket family

Text WebSocket frames whose type is `codex.rate_limits` and whose limit name is
empty or `codex` use the same three-state policy. Local mode replaces the whole
event with local Primary and Secondary windows, upstream mode preserves it,
and hidden mode drops it. Named model-specific families such as
`codex_bengalfox` remain byte-for-byte unchanged. Binary frames are never
parsed or transformed.

### Keep the client upgrade boundary

A client-facing WebSocket `101` is committed before account selection and the
upstream handshake. It can therefore contain an enabled local view, but never
real upstream account quota. In-band events apply the account-aware policy
after an account is selected.

### Suppress incomplete local views instead of falling back

When local mode is selected but no active local window can be built, the
gateway removes upstream default fields and suppresses the default WebSocket
event. It never falls through to upstream account quota because doing so would
mix visibility sources under a request whose configured authority is local.

## Risks and mitigations

- **Missed response path:** keep the policy in shared finalizers and cover
  native Responses, Chat, Messages, raw compatibility, media, errors, and all
  WebSocket relays with compile-time signature propagation and focused tests.
- **WebSocket relay lifecycle regression:** transform frames before client
  observation and writes, count suppressed frames as dropped, and preserve the
  existing terminal/error hooks for emitted frames.
- **Header source mutation:** finalize cloned or destination header maps only;
  tests retain and inspect the original upstream map.
- **Unexpected model-limit suppression:** classify only the empty/default
  `codex` family and preserve any named family.
