## Context

`prepareAllowReceipts` computes receipt keys for every selected canonical
segment, including current user text. Exact synchronous Allow already writes
that key after the combined gateway decision, but stored lookup has an
explicit `segment.CurrentUser` exclusion.

The key and Redis namespace already bind:

- authenticated user ID;
- receipt schema and Prompt Audit configuration versions;
- enabled endpoint ID, protocol, model, timeout, and input limit;
- scanner policy, source class, and exact canonical text;
- the administrator-configured bounded TTL.

## Decisions

### 1. Reuse current receipts only while blocking is active

Stored current-user lookup is enabled when the active configuration has
blocking enabled. Async-only mode keeps its current user segment mandatory.
The existing trusted same-request handoff remains valid in either path.

This uses the existing configuration-version transition to prevent an
async-only receipt from becoming blocking authority after an administrator
changes mode.

### 2. Preserve fail-to-review behavior

A missing, expired, malformed, or unavailable Redis receipt remains a miss and
therefore runs Guard. No receipt value creates an Allow unless the key matches
the current user and active policy exactly.

### 3. Preserve recovery and combined authorization

Forced recovery continues to pass `bypass=true`, so every selected segment is
reviewed regardless of receipts. Synchronous receipt writes remain pending
until Content Moderation and Prompt Audit both permit the request and the final
recovery fence passes.

### 4. Do not add a media cache contract

Prompt Audit receipts certify canonical text only. Content Moderation remains
independent and continues classifying current direct-user text and images for
every request. Changing video frames therefore do not invalidate fixed Prompt
Audit text, but they are not exempt from Content Moderation.

Some clients append an opaque `[images:<hex>]` reference to otherwise fixed
user text. Receipt schema version 2 replaces only 32-, 40-, or 64-character
hexadecimal markers in user segments with `[images]` before hashing. The
original text remains Guard input on a miss; marker count remains significant;
non-user sources, malformed markers, and all surrounding text remain exact.

## Risks

- Repetition inside the TTL reduces opportunities for a nondeterministic Guard
  to change its answer. Receipt keys are policy-bound and only exact Allow can
  create them; changing policy increments the configuration version and
  invalidates reuse.
