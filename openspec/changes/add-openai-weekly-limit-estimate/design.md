## Context

The account usage API combines two independently observed values for an OpenAI
OAuth account: upstream 7-day utilization and local account cost accumulated
since the upstream reset anchor. The estimate must update only at utilization
transitions, survive page reloads and different administrator browsers, and
avoid treating user-billed cost as platform cost.

## Decisions

### Persist an account Extra snapshot

Store the estimator state under `codex_7d_limit_estimate`. Account Extra is the
existing ownership boundary for mutable Codex usage observations and requires
no schema migration. The snapshot contains the reset anchor, last observed
rounded percentage, and, after a transition, the basis percentage, sampled
account cost, estimated account cost, and sample time.

### Use the displayed percentage transition

Round utilization with the same positive-number semantics as the frontend
display. The first observation records only a baseline. When a later
observation advances from `P` to a larger displayed percentage, and `P > 0`,
compute:

`estimated account cost = sampled 7-day account cost / P * 100`

A skipped observation still uses the previous observed percentage. This makes
the estimate describe the actual observation transition rather than assuming
that every intermediate percentage was seen.

### Freeze until another transition

Local account cost continues to increase while utilization remains at one
displayed percentage. Return the persisted estimate unchanged during that
period. Persist a replacement only after a larger percentage is observed.

### Re-baseline on a new or corrected window

Treat reset anchors within 15 minutes as the same upstream window because a
relative reset countdown may introduce small observation drift. A larger reset
anchor change, an expired prior anchor followed by a new future anchor, or a
decrease in displayed utilization clears the estimate and records the current
percentage as the new baseline.

### Expose only account cost

The estimate uses `WindowStats.Cost`, labeled `A` in the account list. It never
uses standard cost or user-billed cost. The response includes sampled cost and
basis percentage only so the administrator tooltip can explain the formula.

## Risks and mitigations

- **Concurrent observations:** per-account striped serialization and the latest
  successfully persisted in-process snapshot prevent overlapping row and batch
  refreshes with stale account copies from replacing the first frozen estimate.
  The snapshot is merged into account Extra instead of replacing unrelated
  observations.
- **Reset-anchor drift:** tolerate small reset-time movement while treating a
  weekly-scale jump as a new window.
- **Premature estimate:** the first observation is baseline-only, matching the
  requested no-backfill behavior.
- **Layout growth:** render a compact, non-wrapping marker only on the OpenAI
  OAuth 7-day row and keep the detailed arithmetic in a tooltip.
