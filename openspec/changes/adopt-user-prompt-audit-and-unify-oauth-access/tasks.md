## 1. Prompt Audit

- [x] 1.1 Adopt Plus user-authored synchronous/async selection and remove checkpoint runtime behavior.
- [x] 1.2 Serialize, compress, encrypt, persist, retrieve, and cascade-delete complete canonical event context.
- [x] 1.3 Add authenticated admin download API and UI action without exposing context in ordinary JSON or logs.
- [x] 1.4 Cover HTTP, SSE, Responses WS, Live, extraction failures, harness stripping, async jobs, and migration behavior.

## 2. OAuth access

- [x] 2.1 Centralize effective group/session/user eligibility.
- [x] 2.2 Make group duplication and account-copy paths fail closed without expanding OAuth allowlists.
- [x] 2.3 Use effective eligibility in admin matrices, impact previews, Responses WS, and Live revalidation.
- [x] 2.4 Cover dormant public defaults, ghost bindings, revocation, sticky sessions, and long-lived connections.

## 3. Release

- [x] 3.1 Update protocol/security documentation, locales, release metadata, and migration checks.
- [ ] 3.2 Pass backend, frontend, race, lint, build, OpenSpec, and protected release validation.
