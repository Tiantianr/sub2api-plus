Sub2API Plus v0.1.178+custom.904

## Highlights

- Complete the `#3c80e6` brand-blue rollout across public, user, and
  administrator interfaces.
- Replace residual green brand accents in dashboards, monetary values,
  OpenAI platform badges, model chips, action buttons, and hover states.
- Preserve green for semantic success, health, enabled, and verified states,
  together with payment-provider brand colors and independent chart series.

## Changed

- Make OpenAI platform styling follow the configured site primary palette in
  shared badges, group selectors, channel tables, subscription views, and
  account controls.
- Use the primary palette for positive balance activity, billed-cost emphasis,
  affiliate amounts, recharge actions, and non-semantic table interactions.
- Keep danger, warning, health, quota, payment status, and provider-brand
  colors distinct from the site theme.

## Compatibility and migration

- Existing data remains compatible; this iteration adds no database migration.
- Hosts must run Linux arm64 to use personal images or binary archives.

## Known issues

- Custom home-page HTML remains administrator-managed database content and is
  not overwritten by the application image; update its embedded colors
  separately when a site uses a custom home page.

## Upstream baseline

Plus release: v0.1.178+custom.005
Plus commit: 594d5fb2526ce4981d1ad06cd83893f075f494bb
Official release: v0.1.178
Official commit: e0c48a19ed794a565e3858662520afe0a1f9f0ba
