Sub2API Plus v0.1.183+custom.926

## Highlights

- Allow administrators to include their own local identity in explicit
  restricted OpenAI OAuth account grants.
- Keep administrator grants subject to the same API-key group eligibility,
  atomic policy updates, revision checks, and scheduler enforcement as ordinary
  user grants.

## Changed

- The OpenAI OAuth user/account matrix now lists non-deleted `user` and `admin`
  local identities.
- Transactional grant validation accepts either supported role while continuing
  to reject missing, deleted, or unsupported identities.
- Future-user defaults remain limited to newly inserted ordinary users and do
  not automatically grant administrator identities.

## Fixed

- Prevent administrator identities from being absent from the OpenAI OAuth
  access matrix.
- Prevent an explicit restricted-account grant for an administrator from being
  rejected as an invalid user.

## Compatibility and migration

- No database migration, dependency, configuration, port, certificate, proxy,
  or persistent-volume change is required.
- Existing public/restricted policies and grants remain unchanged. `.925` can
  enforce administrator grant rows created by `.926`, although its matrix does
  not list administrators for further editing.
- Administrators still require an active API key whose group intersects the
  selected OpenAI OAuth account's effective groups.
- Roll back application code to `v0.1.183+custom.925` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Future-user defaults intentionally do not grant administrators; administrator
  access must be selected explicitly in the matrix.
- A disabled administrator remains visible when no status filter is selected,
  but ordinary authentication status checks still prevent API use.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
