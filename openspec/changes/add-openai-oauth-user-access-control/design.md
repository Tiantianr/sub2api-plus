## Authorization model

The effective candidate set is the intersection of the API key group's ordinary schedulable accounts and the authenticated local user's OpenAI OAuth grants. Public accounts remain eligible for every user allowed by their groups. Restricted accounts require an explicit grant.

Policies apply only to non-deleted OpenAI OAuth root accounts. Spark shadows inherit their root account's policy and never have independent grants. OpenAI API-key accounts, setup-token accounts, and other platforms are unchanged.

The authenticated user ID comes only from API-key middleware context. A restricted account rejects requests with a missing or invalid trusted user ID. User-facing errors remain generic and do not expose account names, IDs, or grant state.

## Persistence

`openai_oauth_account_access_policies` stores one row per configured root account with mode, future-user default, and an incrementing revision. Missing policy rows are public for upgrade compatibility. Public updates clear all grants and cannot enable the future-user default.

`openai_oauth_account_user_grants` stores unique account/user pairs. Foreign keys use the existing account and user IDs. Soft-deleted accounts and users are excluded by runtime and administration queries.

An `AFTER INSERT` PostgreSQL trigger inserts grants for every restricted policy marked as a future-user default when an ordinary user is created. The trigger runs in the user creation transaction, covers every current and future registration path, and does not backfill or revoke existing users.

## Scheduling enforcement

The scheduler metadata carries policy mode and granted user IDs so unauthorized accounts are removed before priority and Top-K selection. Policy mutations emit the existing account-changed event and refresh the affected account snapshot.

Every selected account is checked again using durable state before upstream use. This protects multi-instance deployments while snapshot updates propagate. Advanced and legacy eligibility checks share one authorization helper and normalize Spark shadows to the root policy.

HTTP and SSE requests already sent upstream may finish naturally. The next HTTP request is checked against current grants. WebSocket turns and authenticated Live sideband control periodically reload and recheck the account; a revoked caller must reconnect and cannot send another upstream turn.

## Administration workflow

The administrator page is a paginated local-identity-by-account matrix. It lists ordinary users and administrators together with only OpenAI OAuth root accounts, shows public/restricted/default state, supports user and eligibility filters, and keeps edits in a local draft until preview and confirmation. Either role may receive an explicit grant, while the future-user default remains limited to newly inserted ordinary users.

Preview reports mode changes, grant additions/removals, future-user defaults, and users whose current API-key groups would have no permitted OAuth account. Updates validate the expected policy revision and apply all submitted account changes in one transaction. Revision conflicts return HTTP 409 instead of overwriting another administrator's changes.

Audit records contain administrator identity, account IDs, old/new revisions, modes, and grant counts, but never OAuth credentials or tokens.

## Rollout and rollback

The additive migration leaves every existing account public. Administrators populate grants and review impact before atomically switching accounts to restricted mode. Application rollback is safe only after all policies have been returned to public mode because older binaries do not enforce grants. Tables may remain in place for a later rollout.
