# Change: Retain full audited text for Content Moderation keyword hits

## Why

Content Moderation currently labels its administrator input dialog as full
content, but every row persists only a 240-rune excerpt. Operators therefore
cannot inspect the complete normalized text that caused a configured keyword
block.

## What changes

- Persist encrypted, secret-redacted normalized text only for new
  `keyword_block` records.
- Keep the paginated log API limited to the existing excerpt.
- Add an administrator-only, record-scoped endpoint that decrypts keyword-hit
  content on demand.
- Make the UI load full content only after the administrator opens a record.
- Identify historical rows without retained ciphertext as incomplete excerpts.

## Impact

- Adds one forward-only nullable-by-content `TEXT` column with an empty default.
- Reuses the existing AES-256-GCM secret encryptor and fixed deployment key.
- Does not change keyword matching, blocking, hash state, violation counting,
  notifications, retention, or non-keyword evidence.
