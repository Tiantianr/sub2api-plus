## 1. Conversation state

- [x] 1.1 Add atomic Redis CLEAN/FULL_REQUIRED state, turn leases, TTLs, and hashed parent mappings.
- [x] 1.2 Add canonical context/input/output fingerprints and full-replay continuity checks.
- [x] 1.3 Select full versus prior-output-plus-current-input snapshots in blocking Prompt Audit.

## 2. Output lifecycle

- [x] 2.1 Capture bounded final HTTP/JSON/SSE output after downstream transforms.
- [x] 2.2 Add final-frame observation to Responses WebSocket ingress modes.
- [x] 2.3 Add Live call aliasing and Sideband server-frame observation.
- [x] 2.4 Preserve FULL_REQUIRED on failure, cancellation, overflow, parse ambiguity, and non-success terminal states.

## 3. Administration and documentation

- [x] 3.1 Remove the static latest-turn toggle while retaining rolling-upgrade JSON compatibility.
- [x] 3.2 Update the normative security-audit coverage matrix and bilingual UI descriptions.

## 4. Verification

- [x] 4.1 Add state-store, state-machine, replay-bypass, parent-chain, context-change, and output-normalization tests.
- [x] 4.2 Add HTTP/SSE, all Responses WS ingress modes, and Live final-payload hook tests.
- [x] 4.3 Run focused and full backend/frontend validation, strict OpenSpec validation, formatting, and diff checks.
