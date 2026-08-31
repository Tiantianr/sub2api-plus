# Tasks

## 1. Contract

- [x] 1.1 Define the external-only redaction boundary and supported deterministic identifier classes.
- [x] 1.2 Preserve scanner allowlist, Safe behavior, hashes, evidence, receipts, and recovery semantics.

## 2. Backend

- [x] 2.1 Implement validated typed-placeholder redaction with overlap priority and a no-match fast path.
- [x] 2.2 Apply redaction at the shared OpenAI-compatible Guard request boundary.
- [x] 2.3 Merge value-free local PII signals into non-Safe Guard results only when PII is enabled.

## 3. Verification

- [x] 3.1 Cover all supported types, invalid numeric candidates, overlap, and absence of plaintext in real Guard payloads.
- [x] 3.2 Cover Safe, enabled PII, disabled PII, credential-only, sync, async, retry, and failover semantics.
- [x] 3.3 Benchmark large matching and non-matching chunks and run backend, OpenSpec, lint, and diff checks.
