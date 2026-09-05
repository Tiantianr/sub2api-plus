Sub2API Plus v0.2.0+custom.903

## Highlights

- Add official outbound Pi client identity emulation across ChatGPT privacy, PAT verification, model manifest query, and gateway forwarding paths without leaking Codex version headers.
- Isolate WebSocket connection pool by client identity (User-Agent, originator, version), dial with processed headers, and discard stale prewarmed connections.
- Enable API Key account Codex fingerprint convergence on Chat Completions and Messages gateway bridge endpoints.
- Unify frontend OpenAI client identity presets (`OpenAIIdentityPresetSelector`) with authentic Pi User-Agent format and token boundary matching.

## Changed

- Outbound client identity (`ApplyOutboundClientIdentity`) enforces stripping `Version` header for Pi or empty versions across all ChatGPT endpoints, PAT validation, and OAuth code exchange.
- Upstream model manifest query URL omits `client_version` query parameter when version is empty.
- Inbound request classifier decouples Pi from built-in Codex client profiles to avoid inappropriate credential requirements.
- WebSocket connection pool ties handshake compatibility and routing affinity to actual dialed headers, updating acquire history and discarding stale prewarm connections.
- Refactored frontend identity preset selection across Settings, Create Account, and Edit Account modals using the shared component and strict token boundaries.

## Fixed

- Prevent cross-identity WebSocket connection reuse between Codex and Pi clients.
- Stop sending empty `Version: ` header during OAuth authorization code exchange for Pi identity.
- Fix missing fingerprint convergence for API Key accounts on Chat Completions and Messages bridge routes.
- Prevent overly permissive prefix matching (`startsWith('pi')`, `startsWith('codex-tui')`) in frontend identity recognition.

## Compatibility and migration

- Fully backward-compatible; no database schema migrations required.
- Inbound Codex endpoints reject unauthorized non-official profiles when strict profile whitelisting is enabled.
- Roll back application code to `v0.2.0+custom.902` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.002
Plus commit: cd1d8438cbe19358936605af7e6b20954283bf15
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
