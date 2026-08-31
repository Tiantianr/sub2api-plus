Sub2API Plus v0.1.183+custom.922

## Highlights

- Make local Codex subscription quota authoritative across HTTP, SSE, and
  WebSocket responses, including accounts that enable automatic passthrough.
- Expose real upstream default Codex quota only for automatic-passthrough
  accounts when local quota is disabled; hide it for every other account.

## Changed

- Client-facing default quota now follows one account-aware `local`, `upstream`,
  or `hidden` policy after generic response-header filtering.
- Native Responses, converted Chat and Messages, raw compatibility, Compact,
  Embeddings, Alpha Search, Images, protocol-error, and upstream-error paths
  apply the same final quota policy before committing a response.
- Default `codex.rate_limits` WebSocket events are replaced with local windows,
  preserved for automatic passthrough, or suppressed. Named model-specific
  limit families and binary frames remain unchanged.

## Fixed

- Prevent a generic response-header allowance from exposing a shared OAuth or
  API-key account's real default Codex quota.
- Prevent an upstream WebSocket quota event from replacing an enabled local
  subscription quota after the client upgrade response.
- Prevent converted, media, error, and first-output failover paths from using a
  quota source inconsistent with the final selected account and local policy.

## Compatibility and migration

- No database migration, dependency, port, Compose, certificate, proxy, or
  persistent-volume change is required.
- Existing local Codex quota settings and subscription accounting remain
  unchanged. The dedicated `/backend-api/wham/usage` route remains local-only.
- Clients using non-passthrough accounts may stop receiving upstream default
  Primary and Secondary quota fields; this is the intended privacy boundary.
- Roll back to `v0.1.183+custom.921` if required. The rollback restores generic
  eligible upstream quota-header passthrough and unmodified in-band WebSocket
  quota events when the local view does not replace them.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Codex App API-key calls to `account/rateLimits/read` remain outside the
  gateway compatibility path.
- Named model-specific metered-limit events remain visible independently from
  the default quota policy.
- Real upstream default quota requires the selected account to enable OpenAI
  automatic passthrough; response-header configuration alone cannot expose it.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
