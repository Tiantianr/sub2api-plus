## Why

Prompt Audit already supports request-scoped blocking-exempt users, but legacy
Content Moderation still synchronously waits for and enforces image moderation
for those requests. A Moderation API outage, timeout, oversized-body response,
or invalid response also currently returns a 503 and prevents the user's model
request even though no risk finding was produced.

## What Changes

- Propagate the frozen Prompt Audit blocking-exempt decision to Content
  Moderation before the synchronous engines start.
- Keep image moderation synchronous and enforcing for ordinary pre-block
  requests.
- Send images from blocking-exempt requests to the existing asynchronous
  Content Moderation worker as non-enforcing observations.
- Allow the current request when the Moderation API or its credentials, proxy,
  transport, or response are unavailable, while recording a stable failure.
- Preserve fail-closed handling for canonical extraction, configuration, hash
  state, known keyword/hash findings, and successful risk findings.

## Non-goals

- Making every Content Moderation request asynchronous.
- Exempting text keyword enforcement or unrelated Content Moderation policy.
- Treating an API failure as a safe moderation result.
- Changing Prompt Audit receipts, recovery findings, scanner policy, or Guard
  availability semantics.
- Changing request payload extraction, image bytes, hashes, evidence, billing,
  scheduling, or upstream model requests.

## Impact

- Affected capabilities: Content Moderation image enforcement and Prompt Audit
  blocking-exempt coordination.
- Affected code: security audit request policy propagation, legacy moderation
  adapter, Content Moderation synchronous/async routing, failure logging, tests,
  and security coverage documentation.
- Compatibility: no migration or configuration change is required. Existing
  blocking-exempt user IDs become authoritative for image pre-block behavior.
