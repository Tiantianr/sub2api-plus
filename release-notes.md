Sub2API Plus v0.2.0+custom.904

## Highlights

- Synchronize upstream `v0.2.0+custom.003` for canonical GPT-6 Astra support across OpenAI discovery, Codex catalogs, client configurations, request metadata, and billing.
- Add database migration `265_channel_monitor_gpt6_astra.sql` to register default channel monitor configuration for GPT-6 Astra.
- Preserve local Pi outbound identity emulation, WebSocket connection pool isolation, and custom repository distribution settings.
- Place GPT-6 Astra first in model whitelists and preset mappings while retaining GPT-5.6 Sol as default account test model.

## Changed

- Listed GPT-6 Astra first while retaining GPT-5.6 Sol as the account-test model.
- Imported official Codex Astra instructions, default low reasoning level, multi-agent Ultra preset, and client capability metadata.
- Enabled official Standard, Flex, Fast, prompt-cache, and whole-request long-context pricing for OpenAI Platform API-key traffic.
- Preserved local `gpt-6` alias and Astra suffix compatibility normalization.

## Compatibility and migration

- Applied forward-only migration `265_channel_monitor_gpt6_astra.sql` to register channel monitor defaults for GPT-6 Astra.
- Existing model pricing and configured GPT-5.6 Sol rates remain unchanged.
- At 272,001 total input tokens and above, Astra input and cache tokens are billed at 2x and output tokens at 1.5x.
- Roll back application code to `v0.2.0+custom.903` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.
- Fast/priority processing is unavailable for GPT-6 Astra with EU data residency.

## Upstream baseline

Plus release: v0.2.0+custom.003
Plus commit: 6ba64e382edf9fa7c33fb3e37afe6ba219ce2003
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
