## 1. Contract

- [x] 1.1 Define exact blocking-current receipt reuse and async-only behavior.
- [x] 1.2 Preserve combined authorization, Redis miss, recovery, and media
  boundaries.

## 2. Implementation

- [x] 2.1 Allow stored current-user receipt lookup only in blocking mode.
- [x] 2.2 Update administration text and security coverage documentation.

## 3. Verification

- [x] 3.1 Cover repeated, changed, cross-user, config-changed, Redis-error,
  async-only, and forced-recovery behavior.
- [x] 3.2 Run backend unit/race/lint, frontend checks, docs, and strict OpenSpec.
