## Why

Administrators can configure blocking-exempt users in Prompt Audit so their
requests do not wait for synchronous Guard checks and are not blocked.
However, local keyword detection in Content Moderation still synchronously
enforced hard blocking on those users. To align operational policy, keyword
matching for blocking-exempt users should record the hit with a clear exemption
marker for administrator review without blocking the user's upstream request or
triggering automatic account penalties.

## What Changes

- When a request carrying `blocking_exempt_at_request` hits a configured blocked
  keyword in Content Moderation pre-block mode:
  - The keyword finding does not block the request or produce a 403. Independent
    known-hash and configured blocking API findings retain their existing behavior.
  - A shadow audit record is persisted with `action: shadow`, `flagged: true`,
    `matched_keyword`, encrypted full input ciphertext, and
    `blocking_exempt_at_request: true`.
  - No automatic ban, penalty counter increment, or violation email is
    triggered.
- Add `blocking_exempt_at_request` column to `content_moderation_logs` table.
- Allow administrators to decrypt and view complete keyword-participating text
  for both blocked and exempt keyword hits.
- Display a "Non-blocking" / "免阻塞" badge in the Risk Control administration log
  view.
- Update security audit coverage documentation and automated test suites.

## Non-goals

- Disabling keyword matching entirely for exempt users (the hit is still
  detected, recorded, and audited).
- Exempting known risky hashes or successful blocking text API findings.
- Exempting non-exempt users from keyword blocking.
- Changing Prompt Guard scanner policies, receipts, or recovery mechanics.

## Impact

- Affected code: `backend/internal/service/content_moderation.go`,
  `backend/internal/repository/content_moderation_repo.go`, database migrations,
  frontend Risk Control views, and security coverage documentation.
- Compatibility: Forward-only database migration adding a nullable-safe default
  column. No breaking API changes.
