## 1. Policy and HTTP paths

- [x] 1.1 Define local, upstream, and hidden quota modes with immutable
  precedence.
- [x] 1.2 Finalize default quota after generic filtering without mutating
  source headers.
- [x] 1.3 Propagate the selected account through native, converted, raw,
  media, Alpha Search, Embeddings, compact, and error response writers.
- [x] 1.4 Preserve first-output per-attempt header staging and failover.

## 2. WebSocket paths

- [x] 2.1 Replace, preserve, or suppress the default `codex.rate_limits`
  family according to the same policy.
- [x] 2.2 Preserve named model-specific limit families and binary frames.
- [x] 2.3 Apply the transform in direct ingress, WS v2, and shared passthrough
  relay paths.

## 3. Verification and documentation

- [x] 3.1 Cover policy precedence, source immutability, native and converted
  HTTP/SSE paths, and WebSocket transform/drop behavior.
- [x] 3.2 Update the OpenAI Responses protocol documentation.
- [x] 3.3 Run focused and repository-required backend checks plus strict
  OpenSpec validation.
