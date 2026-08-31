Sub2API Plus v0.1.183+custom.923

## Highlights

- Estimate the full OpenAI OAuth seven-day platform limit when the displayed
  utilization advances to a new integer percentage.
- Show the frozen account-cost estimate directly on the OAuth account list's
  7d row, with the sampled cost and proportional formula available on hover.

## Changed

- The first eligible usage observation records only the current percentage
  baseline. A later percentage increase estimates the limit as the sampled
  platform account cost divided by the previous observed percentage, times
  100; user-billed cost is never used.
- The estimate remains frozen while the displayed percentage is unchanged.
  Skipped percentages use the last observed percentage as the basis.
- Seven-day reset anchors tolerate up to 15 minutes of drift. A new window or
  a utilization decrease clears the old estimate and establishes a new
  baseline.
- Per-account serialization prevents concurrent row and batch refreshes from
  replacing an already frozen estimate with a later cost sample.

## Fixed

- Invalid, incomplete, or non-finite stored estimates are hidden instead of
  being rendered in the account list.
- A failure to persist the observational estimate remains non-blocking for the
  underlying OpenAI usage query.

## Compatibility and migration

- No database migration, dependency, configuration, port, Compose,
  certificate, proxy, or persistent-volume change is required.
- Existing accounts establish their first baseline on the first eligible 7d
  usage query after upgrade. No estimate is shown until a later percentage
  increase is observed.
- The snapshot is stored under the account Extra key
  `codex_7d_limit_estimate`; it does not alter user billing, account cost, or
  upstream quota state.
- Roll back to `v0.1.183+custom.922` if required. The rollback removes the
  estimate marker and stops updating its observational Extra snapshot; any
  existing snapshot remains inert account metadata.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- The value is a proportional estimate based on locally recorded platform
  account cost and OpenAI's integer utilization display; it is not an upstream
  declaration of the account's exact monetary limit.
- No estimate is produced when the previous observed percentage is zero or a
  positive platform account cost is unavailable.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
