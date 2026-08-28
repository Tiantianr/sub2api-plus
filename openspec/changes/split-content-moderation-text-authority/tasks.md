## 1. Request authority

- [x] 1.1 Expose request-scoped Prompt Guard blocking coverage to the coordinator.
- [x] 1.2 Pass text authority to the legacy Content Moderation adapter without changing unrelated modes.

## 2. Moderation execution

- [x] 2.1 Add and normalize auto/blocking/observe/off text API modes.
- [x] 2.2 Split shadow text and blocking image API inputs under pre-block mode.
- [x] 2.3 Remove hash, notification, and ban side effects from shadow text findings.
- [x] 2.4 Fail closed on blocking extraction, hash, key, and synchronous API dependency failures.

## 3. Administration

- [x] 3.1 Add the text API policy to config/view/update/status APIs and the Risk Control page.
- [x] 3.2 Add bilingual descriptions and a distinct Hash Block record label.
- [x] 3.3 Persist text API mode through the admin handler and distinguish shadow findings.

## 4. Verification

- [x] 4.1 Cover auto in/out of Prompt scope, multimodal split, text off, local keywords, shadow hash isolation, and image dependency failure.
- [x] 4.2 Run full backend/frontend tests, strict OpenSpec validation, formatting, and diff checks.
