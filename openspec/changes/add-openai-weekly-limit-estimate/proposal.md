## Why

OpenAI OAuth accounts expose an upstream 7-day utilization percentage while
the account list already shows the local account-cost total accumulated in the
same window. Operators currently have to estimate the implied weekly platform
limit manually, and a live formula would drift while the upstream percentage
remains unchanged.

## What Changes

- Persist one account-scoped weekly-limit estimate snapshot in account Extra.
- Establish the first observed integer percentage as a baseline without
  estimating.
- When the displayed 7-day percentage advances, estimate the platform limit as
  the current 7-day account cost divided by the previous observed percentage
  and multiplied by 100.
- Freeze the estimate until the next percentage advance, and reset the baseline
  when the upstream 7-day window changes or utilization moves backwards.
- Show the latest estimate beside the OpenAI OAuth 7-day progress row with the
  sampled formula available as a tooltip.

## Non-goals

- Estimating user-billed cost, subscription revenue, token limits, or request
  limits.
- Changing upstream quota polling, account scheduling, billing, or reset-credit
  behavior.
- Backfilling an estimate when the feature first observes an existing window.
- Adding a database migration or historical estimate table.

## Impact

- Affected capability: `openai-oauth-weekly-limit-estimate`.
- Affected code: account usage snapshot enrichment, account Extra persistence,
  account usage API types, and the account-list 7-day progress display.
- Compatibility: existing account Extra and API clients remain valid; the new
  response field is optional and absent until a percentage transition occurs.
