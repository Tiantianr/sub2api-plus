# Design

The canonical extractor remains the only source classifier. A logical review
segment contains a source class, exact text, and whether it belongs to the
current direct user turn. Consecutive user content blocks form one user-turn
segment so synchronous and asynchronous lanes derive the same receipt key.

Normal blocking selection includes current direct user content plus the
configured synchronous modules. Every historical user turn remains a receipt
candidate: valid receipts remove it, while misses are synchronously reviewed.
A genuine tool continuation therefore avoids repeated review, but a client
cannot hide unreviewed user text before a benign current turn or by appending
an assistant/tool item.

Deep and standalone asynchronous selection includes user turns and configured
optional segments, then removes valid receipt hits. A current user segment is
always a miss unless the blocking lane passes the exact receipt key to the deep
lane after a complete Allow from the same request. This in-process handoff
cannot be supplied by a client. It avoids a duplicate Guard call without
letting an old identical user string bypass the first review of a new turn.

Each receipt key binds a receipt-schema revision, user ID, configuration
version, enabled Guard endpoint and scanner policy, source class, and exact
segment SHA-256. One Redis MGET
resolves all eligible segments; misses are concatenated into one prioritized
Guard input and one pipeline writes their Allow receipts. The TTL defaults to
3600 seconds and is bounded by configuration validation.

Lookup occurs before complete-context encryption. On full or partial hits, the
actual Guard input and event prompt metadata contain only missed segments,
while encrypted context retains every selected source and hit/miss counts. If
every segment hits, no empty job or Guard call is created. Only a complete
aggregate Allow from a request permitted by Content Moderation writes all
scanned keys. Warn, Block, invalid, timeout, partial extraction, or a combined
gateway rejection never writes a receipt.

Dual-lane ordering remains unchanged: Content Moderation and synchronous Prompt
Audit run concurrently; combined Allow starts asynchronous deep review; normal
upstream work and the deep job then proceed concurrently. A late deep Block
still creates recovery state. Forced recovery bypasses all receipts and must
complete a full synchronous deep review before clearing that state.

Redis lookup or write failure is observable and falls back to ordinary review.
It never creates Allow, Block, or user-penalty state. Config version changes
invalidate keys immediately; TTL bounds an external Guard policy update that
occurs without a local config change.

An asynchronous worker processes a job only when the active configuration
version still equals the job's version. A stale job fails without scanning or
writing receipts; current policy must never be used to certify an old-policy
key.
