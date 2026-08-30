# Design: Optimize Prompt Audit hot path

## Immutable synchronous body

Ingress supplies a frozen request body. Coordinator passes the same byte slice
to its two synchronous read-only audit branches. Neither branch may mutate it.
The asynchronous enqueue boundary retains the existing deep copy before the
goroutine outlives the request.

## Zero-copy rune chunking

The chunker iterates UTF-8 rune start offsets and slices the original immutable
string at exact rune-count boundaries. Priority segments remain independent and
empty segments remain omitted. Chunk logging uses `utf8.RuneCountInString`
instead of allocating `[]rune` values.

## Safety

Canonical extraction, priority order, configured input limits, sequential
complete coverage, node failover, and early Block termination are unchanged.
No mutable object is shared with asynchronous processing.
