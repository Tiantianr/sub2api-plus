# Design: Prompt Audit outbound sensitive-data redaction

## Boundary

Canonical extraction must remain complete because hashes, Allow receipts,
encrypted evidence, recovery review, and administrator evidence retention all
depend on the original content. Redaction therefore runs inside
`OpenAICompatibleScanner.Scan` immediately before the chunk is assigned to the
external request payload. This single boundary is shared by synchronous
evaluation, asynchronous workers, retries, and failover.

## Detector

The detector uses package-level RE2 patterns and byte-indexed replacement
spans. Credential spans have priority over identifiers; validated identity,
bank-card, phone, and IP spans cannot overlap. Chinese identity candidates must
pass the standard checksum, bank cards must pass Luhn, and IP candidates must
pass `net.ParseIP`. Telephone candidates are bounded by digit count to avoid
turning arbitrary long numbers into phone matches.

Recognized values are replaced by fixed typed placeholders:
`<CREDENTIAL>`, `<EMAIL>`, `<PHONE>`, `<CN_ID>`, `<BANK_CARD>`, and
`<IP_ADDRESS>`. No plaintext-derived hash, suffix, domain, area code, or global
mapping is sent. If there is no match, the original string is returned without
building a second content string. If there is a match, one `strings.Builder`
constructs the outbound copy. Sensitive buffers are not pooled or cached.

## PII decision parity

Credentials are always redacted but do not create a PII signal. Email, phone,
identity, bank-card, and IP replacements set a local boolean PII signal. After
strict Guard parsing:

- Guard `Safe` remains unchanged.
- If `pii` is disabled, the signal cannot affect categories or decisions.
- If `pii` is enabled and Guard returns `Controversial` or `Unsafe`, the server
  adds effective `pii` metadata using the existing scanner catalog, score, and
  elevated-category decision rules.

This preserves the configured scanner boundary without sending the sensitive
value. Unknown categories and invalid Guard responses keep their existing
behavior.

## Data lifecycle

Prompt hashes, canonical `ScanText`, encrypted complete context, transient job
payloads, and retained administrator evidence are unchanged. Logs and events
receive no matched values or new raw diagnostics. The outbound placeholder copy
is scoped to one scanner call and becomes unreachable after request
construction and completion.
