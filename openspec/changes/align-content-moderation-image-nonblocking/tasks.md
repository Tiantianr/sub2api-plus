## 1. Policy propagation

- [x] 1.1 Propagate the frozen Prompt Audit blocking exemption through the
  coordinator and legacy moderation adapter.
- [x] 1.2 Keep HTTP and WebSocket request/turn policy snapshots aligned.

## 2. Content Moderation behavior

- [x] 2.1 Route blocking-exempt images to asynchronous non-enforcing review.
- [x] 2.2 Preserve synchronous findings and independent text policy for
  non-exempt requests.
- [x] 2.3 Allow Moderation API dependency failures while retaining safe error
  telemetry and no enforcement side effects.

## 3. Verification and documentation

- [x] 3.1 Cover ordinary and exempt real image payloads across protocol shapes.
- [x] 3.2 Cover API 4xx/5xx, timeout, invalid/empty response, no-key, queue, hash,
  and extraction boundaries.
- [x] 3.3 Cover HTTP and WebSocket coordinator propagation and side-effect order.
- [x] 3.4 Update security content coverage and aligned frontend locale guidance.
- [x] 3.5 Run focused Go tests, integration/race checks, frontend locale tests,
  strict OpenSpec validation, lint/typecheck, and diff checks.
