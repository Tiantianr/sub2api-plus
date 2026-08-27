## 1. Persist observability metadata

- [x] 1.1 Add migration 234 for client IP, prompt metadata, queue delay, execution mode, chunk diagnostics, and historical truncation state, plus concurrent index migration 235.
- [x] 1.2 Capture the trusted client IP and persist complete new-event audit content.
- [x] 1.3 Record queue delay, effective chunk limit, matched chunk index, and a distinct last-error timestamp.
- [x] 1.4 Add exact client-IP event filtering and increase the bounded input limit to 500,000 characters.

## 2. Improve the administration UI

- [x] 2.1 Add client IP and queue/audit duration columns, including click-to-filter behavior.
- [x] 2.2 Show complete content status, chunk diagnostics, prompt metadata, and a local text download action in event details.
- [x] 2.3 Show accepted timeout/input ranges and splitting behavior beside endpoint controls.
- [x] 2.4 Separate latest successful processing from latest error time in runtime status.

## 3. Verify behavior

- [x] 3.1 Add focused backend migration, repository, handler, snapshot, Guard, and runtime tests.
- [x] 3.2 Add focused frontend view-model and component tests.
- [x] 3.3 Run Go formatting/tests, frontend lint/typecheck/Vitest, strict OpenSpec validation, and diff checks.
