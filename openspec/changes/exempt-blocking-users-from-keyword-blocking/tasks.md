# Tasks

- [x] Add forward-only database migration `246_content_moderation_blocking_exempt.sql` and migration test.
- [x] Update `ContentModerationLog` and repository select/insert logic for `blocking_exempt_at_request`.
- [x] Update `ContentModerationService.Check` to record blocking-exempt keyword matches without bypassing independent hash or API checks.
- [x] Update `GetLogInput` to decrypt purpose-bound keyword input for shadow keyword hits.
- [x] Update `docs/SECURITY_AUDIT_CONTENT_COVERAGE.md`.
- [x] Update frontend models, locales, and RiskControlView for `blocking_exempt_at_request`.
- [x] Add backend and frontend unit tests and verify complete test suite.
