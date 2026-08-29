Sub2API Plus v0.1.183+custom.911

## Highlights

- Reduce Prompt Audit load during agent tool loops by reviewing each new user
  turn synchronously and only newly introduced canonical segments afterward.
- Preserve the existing dual-lane flow: combined synchronous Allow starts
  asynchronous deep review while normal upstream processing continues.

## Changed

- Add user-scoped, policy-bound Allow receipts for historical user, system,
  assistant, reasoning, prompt-variable, tool-definition, tool-call, and
  tool-output segments.
- Reuse an exact synchronous Allow in the same request's asynchronous deep
  review while preserving late-Block recovery.
- Add a configurable receipt TTL with a one-hour default, Redis batch lookup
  and pipeline writes, encrypted hit/miss evidence, and runtime counters.
- Keep unverified historical user turns in synchronous receipt selection so
  client-controlled role ordering cannot hide new user content.

## Fixed

- Stop repeatedly submitting complete user and tool history on every automatic
  assistant or tool continuation.
- Write receipts only after Content Moderation and Prompt Guard jointly permit
  the original request.
- Reject queued asynchronous jobs whose configuration version no longer
  matches the active Guard policy.

## Compatibility and migration

- No database migration is required. Configurations that omit
  `allow_receipt_ttl_seconds` use 3600 seconds; the accepted range is
  60-86400 seconds.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- Asynchronous deep findings affect the next request and cannot retroactively
  cancel content already sent upstream or delivered to a client.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
