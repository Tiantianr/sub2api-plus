# Redact sensitive data before Prompt Audit outbound calls

## Problem

Prompt Audit already redacts stored previews, but its canonical `ScanText`
remains intentionally complete and each raw chunk is currently placed directly
in the external Guard API request. User-provided credentials and direct
identifiers can therefore leave the gateway even though the event preview is
redacted.

## Proposal

- Create a typed, redacted copy only at the Guard outbound boundary.
- Always replace recognizable credentials, email addresses, telephone numbers,
  checksum-valid Chinese identity numbers and bank cards, and valid IP
  addresses before JSON request construction.
- Preserve PII policy semantics by retaining a local PII signal and applying it
  only when the `pii` scanner is enabled and Guard returns a non-Safe safety
  result.
- Keep canonical extraction, prompt hashes, encrypted evidence, receipt keys,
  event retention, and recovery behavior based on the original content.

## Non-goals

- Recognizing free-form names or postal addresses with regex heuristics.
- Adding a local ML model, external DLP service, configuration switch, database
  field, or persistent identifier mapping.
- Storing or logging matched sensitive values.
- Treating a local identifier match by itself as a risk decision when Guard
  returns Safe.

## Impact

No migration, dependency, public API, or frontend change is required. Every
synchronous and asynchronous OpenAI-compatible Guard call uses the same
outbound scanner boundary. Requests without a match reuse the original string;
matched requests allocate one replacement string before the existing JSON
marshal.
