# Change: Adopt user-authored Prompt Audit and unify OAuth access

## Why

Full canonical Prompt Guard scans create unacceptable latency and false positives from client harness, assistant, and tool content. Administrators still need the complete canonical request context for reviewing hits and improving policy. OpenAI OAuth group bindings, session-sharing allowlists, and per-user grants also need one effective-access rule across management and runtime paths.

## What Changes

- Synchronous Prompt Audit scans only the latest user-authored input; asynchronous Prompt Audit scans all user-authored turns.
- Supported client harness XML is removed from Guard input.
- Every stored Prompt Audit event can retain an encrypted, downloadable complete canonical context artifact containing selected and excluded content.
- OpenAI OAuth eligibility uses the intersection of group binding and session-sharing allowlist before applying per-user policy.
- Group copy paths fail closed for OAuth bindings, and HTTP, Responses WebSocket, Live, and admin previews use the same effective scope.

## Impact

- A new forward migration stores encrypted event context separately from list rows.
- Prompt Audit no longer uses conversation checkpoints or downstream output capture.
- No OAuth policy or grant is created automatically; existing public defaults remain unchanged.
