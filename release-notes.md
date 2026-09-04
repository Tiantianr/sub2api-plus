Sub2API Plus v0.1.183+custom.927

## Highlights

- Extend OpenAI account user access management to include OpenAI API key accounts
  alongside OAuth accounts in the administrative access matrix.
- Redesign and optimize the user Billing & Subscription checkout page (`/purchase`)
  with a centered sliding-pill switcher, dedicated recharge balance layout,
  feature-rich plan cards with stats metrics, and unified primary action buttons.

## Changed

- Allow administrators to manage user access grants for both OpenAI OAuth and OpenAI API Key accounts.
- Reorganize checkout tabs to place recharge balance first and subscription plans second with smooth animated indicators.
- Upgrade subscription plan cards with structured rate, limit, and quota metrics boxes and feature check lists.
- Standardize checkout confirmation buttons to the system royal-blue color scheme.

## Fixed

- Prevent confirmation buttons from inheriting inconsistent payment-channel background colors during checkout.
- Ensure all subscription plan cards remain interactive and visible when selecting plans.

## Compatibility and migration

- No database migration, dependency, configuration, port, certificate, proxy,
  or persistent-volume change is required.
- Existing user subscriptions, payment orders, and provider configurations remain fully compatible.
- Roll back application code to `v0.1.183+custom.926` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations
  and are not part of release publication.

## Upstream baseline

Plus release: v0.1.183+custom.003
Plus commit: e94f300b586d8ceb91ba526b13313407b99ffbff
Official release: v0.1.183
Official commit: e8cb019fabf8b55199436229044cbf9aa7a82564
