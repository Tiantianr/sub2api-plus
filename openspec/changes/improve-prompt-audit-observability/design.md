## Data flow

The gateway resolves the client address once through the existing security client-IP contract and copies it into the Prompt Audit request and snapshot. Async jobs persist the IP with redacted metadata; workers reconstruct the complete review text from the Redis payload before deleting it. Blocking events use the same snapshot directly.

Async queue delay is measured from job creation to the processing claim that produces the stored event. This intentionally includes retry/backoff waiting. Blocking events record zero queue delay. Historical events use a null queue delay because it cannot be reconstructed accurately after lease refreshes.

## Complete content

New events retain the complete extracted audit text after PostgreSQL-incompatible NUL removal. The list query continues to exclude `full_prompt`; only the single-event admin detail loads it. The UI creates the text download from that authorized detail response, avoiding a duplicate content endpoint and a second unredacted read path.

The existing administration audit middleware treats the event-detail GET as a sensitive read. It records the administrator, route, event ID, result, and request metadata without recording the prompt content.

Migration 234 backfills prompt length, message count, and execution mode from the associated job. It marks an existing event as truncated when its retained character count is lower than the recorded prompt length. Truncated historical content is never presented as complete.

## Chunk diagnostics and limits

The configured input limit remains a bounded per-chunk Unicode-character limit. The maximum increases from 100,000 to 500,000, which allows the observed 395,959-character compaction request to remain one chunk while retaining an operational ceiling. When several endpoints are enabled, the minimum enabled endpoint limit remains authoritative.

Each new event stores the effective input limit and the first highest-severity chunk index. These fields explain the recorded aggregate without persisting another copy of the matched content.

## Runtime errors

Worker runtime state keeps the last error code/message as before and adds its own timestamp. Successful processing does not rewrite that timestamp, so the UI can display last success and last error as separate facts instead of combining a fresh success time with a stale error.
