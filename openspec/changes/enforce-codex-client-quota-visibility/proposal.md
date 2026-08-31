## Why

Codex quota response fields currently pass through generic response-header
filtering before the gateway applies an optional local subscription overlay.
When the local view is disabled, an additional header allowance can expose the
selected upstream account's default quota even when that account is not an
automatic-passthrough account. Several converted, media, error, and WebSocket
paths also apply the local view inconsistently.

## What Changes

- Define one account-aware client quota policy with local, upstream, and hidden
  outcomes.
- Make an enabled local subscription quota authoritative over upstream values,
  including for automatic-passthrough accounts.
- Preserve real upstream default quota only for an automatic-passthrough
  account when the local view is disabled.
- Remove upstream default quota for all other accounts after generic header
  filtering, including converted, media, and error responses.
- Apply the same policy to default `codex.rate_limits` WebSocket events while
  preserving named model-specific limit families.

## Non-goals

- Changing upstream quota ingestion, account scheduling, local subscription
  accounting, or `/backend-api/wham/usage` authentication.
- Adding settings, database migrations, frontend controls, or support for
  Codex App API-key `account/rateLimits/read`.
- Changing model-specific metered limit visibility.

## Impact

- Affected capability: `codex-rate-limit-headers`.
- Affected code: OpenAI HTTP/SSE response finalization, protocol conversion,
  media and error response writing, and Responses WebSocket relays.
- Compatibility: clients may stop receiving the selected account's default
  quota unless that account explicitly enables automatic passthrough. Local
  quota clients continue receiving the existing fields and reset formats.
