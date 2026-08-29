Sub2API Plus v0.1.183+custom.914

## Highlights

- Show user IDs directly in the order-management user column.
- Let administrators reorder Prompt Audit nodes; the first enabled node is
  attempted first and retryable failures continue through the saved order.
- List OAuth account-authorization users by descending user ID so recent users
  appear first across filtered pages.

## Changed

- Add accessible priority controls and visible sequence numbers to the Prompt
  Audit node pool without changing the persisted configuration format.
- Clarify that the five-minute release budget applies only to the tag-triggered
  Linux image publication job after protected PR and exact-main checks pass.
- Require release requests with incompatible end-to-end deadlines to stop
  before mutating Git or publication state until the remaining stages are
  explicitly accepted.

## Fixed

- Replace the OAuth authorization user's email-based ordering with stable
  descending user-ID ordering before access filtering and pagination.
- Keep order user IDs visible even when an email or username is available.

## Compatibility and migration

- No database migration, new configuration field, or API compatibility change
  is required.
- Existing Prompt Audit endpoint arrays retain their stored order; saving a
  reordered draft changes runtime priority immediately after configuration
  activation.
- No Compose, port, certificate, proxy, or persistent-volume change is
  required. Personal images and binary archives remain Linux arm64 only.

## Known issues

- The five-minute budget is not an end-to-end release guarantee; candidate
  validation, protected merge, asset publication, and verification remain
  separate stages.
- Production deployment remains a separate operation and is not part of this
  release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
