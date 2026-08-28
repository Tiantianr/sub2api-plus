## 1. Persist authorization state

- [x] 1.1 Add forward-only policy/grant migration, indexes, constraints, and future-user default trigger.
- [x] 1.2 Add Ent schemas and regenerate Ent/Wire output.
- [x] 1.3 Add repository/service transactions for listing, previewing, and revision-checked batch updates.

## 2. Enforce user grants

- [x] 2.1 Hydrate access policy into scheduler metadata and refresh it through existing account-change events.
- [x] 2.2 Enforce grants in advanced and legacy OpenAI selection, sticky sessions, and response continuation.
- [x] 2.3 Revalidate grants for WebSocket turns, Live sideband control, and Spark shadows.

## 3. Add administrator management

- [x] 3.1 Add administrator-only account/user matrix, preview, and atomic update endpoints.
- [x] 3.2 Add the paginated matrix page with filters, batch drafts, impact preview, conflict handling, routing, navigation, and synchronized locales.

## 4. Verify behavior

- [x] 4.1 Add focused migration, repository, service, scheduler, handler, and long-connection tests.
- [x] 4.2 Add focused frontend API, view-model, and interaction tests.
- [x] 4.3 Run generation, formatting, backend tests, frontend lint/typecheck/Vitest, builds, and strict OpenSpec validation.
