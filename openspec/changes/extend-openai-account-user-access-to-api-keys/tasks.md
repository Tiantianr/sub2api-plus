## 1. Extend persistence and repository scope

- [x] 1.1 Add the forward-only trigger-function migration for OpenAI API-key roots.
- [x] 1.2 Include account type in the access-account response and accept OAuth/API-key roots during list and apply validation.

## 2. Extend runtime enforcement

- [x] 2.1 Apply the existing access gate to OpenAI API-key accounts.
- [x] 2.2 Verify scheduler, sticky/failover, WebSocket, Live, and terminal recheck paths.

## 3. Update administration UI

- [x] 3.1 Distinguish OAuth and API-key accounts in the matrix.
- [x] 3.2 Align Chinese and English page copy and preserve legacy API error mappings.

## 4. Verify

- [x] 4.1 Add repository, migration, service, handler, and runtime regression coverage.
- [x] 4.2 Add frontend API and matrix rendering coverage.
- [x] 4.3 Run focused backend/frontend checks and strict OpenSpec validation.
