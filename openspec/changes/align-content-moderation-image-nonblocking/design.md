## Context

In blocking Prompt Audit mode, the coordinator resolves whether the request is
inside Prompt Audit scope and whether its user is blocking-exempt before it
starts the Prompt Guard and legacy Content Moderation branches. Today only the
text-authority bit is passed to Content Moderation. Image inputs always force a
synchronous Moderation API call in Content Moderation pre-block mode, and an
API dependency error is represented as a blocking unavailable decision.

## Decisions

### Freeze and propagate blocking exemption before concurrent checks

Add an internal request field for the already-resolved
`blocking_exempt_at_request` policy. The coordinator sets it from the same
Prompt Audit policy snapshot used by Guard admission, then passes it through
the legacy adapter. Content Moderation does not reload or independently infer
the exempt-user list.

This preserves one request-level decision across HTTP and every independently
audited WebSocket turn.

### Make exempt images asynchronous and non-enforcing

For a pre-block request that contains images and carries the frozen exemption:

- local text keyword checks remain synchronous and authoritative;
- image-derived pre-hash blocking is skipped, while an available text-only hash
  may still be checked;
- image Moderation API work is enqueued to the existing bounded worker queue;
- the current request does not wait for that API call;
- a later image finding is stored as a shadow observation and does not create a
  flagged hash, increment violation counters, send enforcement email, or ban an
  account.

Queue admission remains best-effort for an exempt request. Queue saturation is
observable but cannot turn the exempt request back into a blocking path.

### Keep ordinary successful findings authoritative

For non-exempt image requests in Content Moderation pre-block mode, the image
API call remains synchronous. A valid successful result above configured
thresholds blocks exactly as before. Text-only blocking configuration, local
keywords, known hashes, and explicit security failures are unchanged.

### Treat Moderation API availability failures as non-findings

The following failures allow the current request:

- no configured or currently usable Moderation API key;
- proxy resolution or HTTP transport failure;
- timeout, cancellation, or upstream non-2xx response;
- malformed or empty successful response.

They create no safe result, no flagged hash, and no enforcement side effect.
The service emits stable structured logs and stores a minimal `error` audit row
when persistence is available. It never stores the raw upstream error body.

This allowance does not apply to canonical content extraction failure,
configuration failure, hash-cache failure, known findings, or local policy
matches. Those boundaries retain their current behavior.

## Risks and mitigations

- **Exempt image findings arrive after admission:** mark them as non-enforcing
  shadow observations, matching the explicit operator exemption.
- **Policy drift during a request:** propagate the coordinator's frozen policy
  instead of re-reading configuration in the worker.
- **Silent provider outage:** always emit structured failure metrics/logs and a
  stable error record independently from non-hit retention.
- **Exemption weakens text controls:** split exempt images from text so local
  keyword and text-only hash checks remain authoritative.
- **Queue overload:** retain bounded non-blocking admission and explicit drop
  telemetry; never fall back to a synchronous request after exemption.
