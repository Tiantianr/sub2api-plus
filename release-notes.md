Sub2API Plus v0.2.0+custom.906

## Highlights

- Preserve OpenAI OAuth history admission policy in slim scheduler cache projections.
- Incomplete cached OAuth snapshots without history admission boolean are treated as cache misses and rebuilt from database.
- Record history admission rejections into Ops failed-request logs as local routing business limitations.

## Changed

- Slim scheduler cache metadata includes explicit opt-outs for `openai_oauth_reject_external_history`.
- Cached snapshots missing history admission flags trigger repository rebuild instead of defaulting to rejection.
- Ops error logging captures history admission rejections for HTTP, SSE, and WebSocket turns while excluding them from upstream availability failure metrics.
- Admin Usage error requests include policy rejections under `all` and `excluded` filters.

## Compatibility and migration

- Forward-only database migration schema unchanged.
- Slim scheduler cache auto-rebuilds without manual Redis flushing.
- Roll back application code to `v0.2.0+custom.905` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.003
Plus commit: 6ba64e382edf9fa7c33fb3e37afe6ba219ce2003
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
