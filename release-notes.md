Sub2API Plus v0.2.0+custom.902

## Highlights

- Align complete official OpenAI GPT-6 Astra support from upstream PR #6572 (`gpt-6-astra` and `gpt-6` alias).
- Fix GPT-6 Astra pricing resolution, fallback rates, reasoning max effort normalization, and context limits.
- Fix Group Model Pricing fallback matching so custom group pricing matches normalized request models (e.g. `gpt-6`, `openai/gpt-6-astra`).
- Update frontend model presets and whitelist to include `gpt-6` and `gpt-6-astra`.

## Changed

- Model pricing resolver introduces symmetric normalization fallback in group model pricing, ensuring custom group pricing correctly matches request variants like `gpt-6`, `openai/gpt-6-astra`, and reasoning effort suffixes (`-high`, `-max`).
- OpenAI reasoning effort normalization preserves `max` for GPT-6 Astra, Codex 5.6 Max, and supported reasoning architectures.
- Added official GPT-6 Astra fallback rates ($10/M input, $50/M output, $12.5/M cache write, $1/M cache read, 2x Fast tier, 0.5x Flex tier, 272K long context threshold).
- Updated frontend model whitelist and preset mappings with GPT-6 and GPT-6 Astra entries.

## Fixed

- Resolve pricing resolution mismatch where custom group pricing for `gpt-6-astra` failed to match requests targeting `gpt-6` or prefixed names.
- Correct context limits and capability flags for GPT-6 Astra (1,050,000 tokens context window, vision support enabled, lite responses disabled).

## Compatibility and migration

- Fully backward-compatible; no database schema migrations required.
- Existing custom pricing rules and group configurations remain compatible.
- Roll back application code to `v0.2.0+custom.901` if required.
- Personal images and binary archives remain Linux arm64 only.

## Known issues

- Invoicing remains unsupported for direct online recharges.
- Production deployment and configuration changes remain separate operations and are not part of release publication.

## Upstream baseline

Plus release: v0.2.0+custom.002
Plus commit: cd1d8438cbe19358936605af7e6b20954283bf15
Official release: v0.2.0
Official commit: aa236488351eb71e120fc2b6fb32e36b0374c918
