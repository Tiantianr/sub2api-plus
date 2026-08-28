## Why

OpenAI OAuth accounts are currently selected only from an API key's scheduling group. Administrators cannot limit individual users to specific OAuth credentials without exposing account details, splitting public groups, or changing subscription behavior.

## What Changes

- Add explicit public/restricted access policies for OpenAI OAuth root accounts and per-user grants for restricted accounts.
- Apply user access after ordinary group eligibility across every OpenAI scheduling path, including sticky sessions, response continuation, WebSocket turns, Live, and Spark shadows.
- Add an administrator-only user/account matrix with search, batch grant/revoke, impact preview, optimistic revision checks, and default grants for future users.
- Keep existing accounts public by default and leave user-facing API keys, groups, subscriptions, models, and account identities unchanged.

## Impact

- Persistent data: additive policy and grant tables plus a trigger that grants configured defaults to newly inserted ordinary users.
- API: new administrator-only list, preview, and atomic policy-update endpoints.
- Security: restricted accounts fail closed when trusted user identity or a matching grant is absent; unstarted requests and new long-connection turns cannot reuse revoked credentials.
- Compatibility: migration alone preserves existing routing. Before rolling back to code that does not understand the policy, every account must be returned to public mode.
