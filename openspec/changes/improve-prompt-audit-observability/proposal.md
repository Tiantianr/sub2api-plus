## Why

Prompt Audit events currently omit client IP and queue delay, truncate stored review content at 65,536 Unicode characters, and expose node limits only through HTML input validation. This prevents administrators from reproducing false positives, distinguishing queue pressure from Guard latency, and understanding why a long request was split.

## What Changes

- Persist the complete extracted text actually submitted for audit on new events and mark historical or otherwise incomplete stored content explicitly.
- Capture the trusted client IP and support exact IP filtering from the event list.
- Persist queue delay, execution mode, prompt size, configured chunk limit, and matched chunk index with each event.
- Allow a bounded 500,000-character chunk size and show the accepted timeout/input ranges and splitting behavior beside the controls.
- Let administrators download the already-authorized event content as a local text file.
- Separate the latest successful processing time from the latest worker error time.

## Impact

- Persistent data: migration 234 adds additive Prompt Audit job/event metadata; migration 235 builds the IP index concurrently.
- API: event and runtime responses gain additive fields; event filtering gains `client_ip`.
- Security: complete event content remains admin-only, is excluded from list queries and logs, and downloads are generated from the authorized detail response.
- Compatibility: historical events retain their existing content and are marked when it is incomplete; existing configurations remain valid.
