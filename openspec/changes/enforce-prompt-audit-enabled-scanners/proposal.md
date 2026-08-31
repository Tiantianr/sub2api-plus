# Enforce enabled Prompt Audit scanners

## Problem

Qwen3Guard may report a known category that an administrator did not enable.
The parser excludes that category from `matched_scanners`, but still stores it
in `categories` and lets the global Safety value produce a warning. Event views
therefore present a disabled category as an effective finding and can obscure
which enabled category actually caused a block.

## Proposal

- Treat configured `scanners` as a server-enforced allowlist for known risk
  categories.
- Remove disabled known categories before decision, aggregation, persistence,
  issue-summary derivation, and administration display.
- Preserve the existing fail-closed behavior for unknown categories and an
  Unsafe response that contains no recognized category.
- Normalize historical event reads to their persisted `matched_scanners`
  without rewriting stored rows.

## Non-goals

- Changing the nine stable category IDs or configuration API.
- Allowing administrators to disable unknown-category fail-closed behavior.
- Rewriting historical event rows or storing raw Guard output.

## Impact

No migration or configuration change is required. Future events persist only
effective enabled known categories. Historical API reads and UI views expose
their already persisted `matched_scanners` as effective categories.
