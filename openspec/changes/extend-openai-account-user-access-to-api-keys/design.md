## Scope

The managed set becomes non-deleted OpenAI root accounts whose type is either
`oauth` or `apikey`. The existing policy and grant tables, revision protocol,
administrator routes, and frontend matrix remain in use; the legacy
`openai-oauth-access` endpoint path is retained for compatibility.

The account response adds a `type` field so the matrix distinguishes OAuth
and API-key credentials. The page title and messages use OpenAI account
access terminology while preserving existing error reason codes.

## Enforcement

The same snapshot is hydrated into scheduler metadata for both account types.
Restricted accounts require the trusted local user ID after normal API-key
group intersection. Public accounts and accounts without a policy keep current
behavior. Existing sticky, failover, WebSocket, Live, and terminal rechecks
reuse the generalized account gate.

The future-user default trigger grants restricted OpenAI OAuth and API-key
root accounts to newly inserted ordinary users. It does not alter existing
users or grant administrators automatically.

## Migration and compatibility

No data backfill is required. Migration 248 replaces the two existing trigger
functions so new policy rows for API-key roots survive account writes and
future-user defaults include them. The original migration remains immutable.
OpenAI OAuth policies and grants are unchanged.
