Sub2API Plus v0.1.183+custom.924

## Highlights

- Make image Content Moderation follow the frozen Prompt Audit blocking-exempt
  user policy: exempt images are reviewed asynchronously without delaying or
  enforcing against the current conversation.
- Allow the current conversation when the external Moderation API is
  unavailable or returns an error, while preserving observable safe failure
  records and all explicit local security boundaries.
- Hide the 1000, 2000, and 5000 balance-recharge quick amount buttons while
  retaining custom amount entry.

## Changed

- The coordinator now resolves and forwards the request-time Prompt Audit
  blocking exemption before concurrent Guard and Content Moderation checks for
  HTTP, first WebSocket turns, and subsequent WebSocket turns.
- Exempt image findings are stored as non-enforcing shadow observations. They
  do not create flagged hashes, violation counts, enforcement email, recovery
  state, or automatic account bans.
- Ordinary pre-block images remain synchronous, and a valid successful risk
  finding retains its configured blocking authority.
- Moderation API credential, proxy, transport, timeout, non-2xx, malformed
  response, empty response, and no-usable-key failures record stable error
  telemetry but permit the current request without fabricating a Safe result.
- Recharge quick amounts now stop at 500. Server-side amount limits and the
  custom amount field remain unchanged.

## Fixed

- Prevent 413, timeout, upstream 4xx/5xx, proxy-resolution, invalid-response,
  and temporary key-health failures from returning a user-facing Content
  Moderation 503 for an otherwise admissible conversation.
- Prevent users explicitly configured as Prompt Audit blocking-exempt from
  waiting for image moderation or receiving image-derived hashes and
  enforcement side effects.
- Preserve synchronous local text keyword and text-only hash enforcement for
  mixed text/image requests even when image review is exempt.

## Compatibility and migration

- No database migration, dependency, configuration, port, Compose,
  certificate, proxy, or persistent-volume change is required.
- Existing Prompt Audit `blocking_exempt_user_ids` automatically become the
  request-time authority for image Content Moderation exemption when blocking
  Prompt Audit applies.
- Canonical extraction failure, active-configuration failure, required hash
  state failure, known keyword/hash findings, and successful non-exempt risk
  findings retain their previous security behavior.
- Roll back to `v0.1.183+custom.923` if required. The rollback restores
  synchronous image moderation for blocking-exempt users, restores
  Moderation-API availability failures as blocking unavailable decisions, and
  restores the three high-value recharge quick buttons.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- A blocking-exempt image finding is intentionally non-enforcing and cannot
  retroactively cancel a request that was already admitted.
- Moderation API availability failures intentionally permit the current
  request. Operators must monitor error records and key-health metrics because
  these failures are not Safe moderation results.
- Prompt Audit exemption can only be propagated while blocking Prompt Audit is
  active and its request scope applies; otherwise Content Moderation uses its
  ordinary independent policy.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
