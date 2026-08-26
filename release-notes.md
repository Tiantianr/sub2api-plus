Sub2API Plus v0.1.183+custom.901

## Highlights

- Import the published Plus `v0.1.183+custom.001` baseline and its official
  Sub2API v0.1.183 integration.
- Preserve the personal billing navigation, hidden asynchronous-image menu,
  `#3c80e6` brand palette, and semantic status colors from v0.1.178+custom.904.
- Keep the personal Linux arm64-only immutable release and update path.

## Changed

- Add service-tier pricing, channel pricing multipliers, Composite CN-provider
  routes, OpenAI quota reset controls, and the default-disabled OAuth transport
  plugin framework from Plus.
- Upgrade the backend toolchain to Go 1.27 and retain the personal GitHub,
  installer, release, and GHCR distribution identity.

## Fixed

- Codex `session-id` affinity now takes priority over `session_id` on sticky routing and WebSocket session resolution.
- OpenAI sticky sessions keep a one-request capacity spillover temporary, so a full wait queue does not rewrite the durable account binding.
- OpenAI OAuth 429 quota-exhausted responses can pause the account instead of treating every 429 as a same-account retry.
- Responses custom tool-call item IDs stay typed after restore.
- Email rebind adds alias and concurrency guards.
- Kimi concurrency 403 stays recoverable; Antigravity compatible token limits are clamped; channel-monitor v2 composite aggregation uses NULLIF.

## Compatibility and migration

- Published Plus migrations 224/225/226/228 are unchanged; prefix 227 remains unused.
- Upgrades from v0.1.178+custom.904 apply Plus migrations 229-233 for usage-log
  indexes, Composite CN providers, channel pricing multipliers, plugins, and
  plugin artifacts. Back up PostgreSQL before deployment.
- Experimental OAuth transport plugins remain disabled by default.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- OAuth transport plugins are experimental and default off.
- Custom home-page HTML remains administrator-managed database content and is
  not overwritten by the application image.

## Upstream baseline

Plus release: v0.1.183+custom.001
Plus commit: 86eda1ccb152f8f2d0c3c96bd5cf741c51c80aed
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
