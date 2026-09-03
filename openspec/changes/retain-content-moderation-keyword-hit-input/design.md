# Design

## Storage boundary

`content_moderation_logs.input_excerpt` remains the list-safe 240-rune value.
The new `input_ciphertext` column contains AES-256-GCM ciphertext only for
keyword finding rows. Plaintext is never written to the database column, list
response, application logs, notifications, or error responses.

Before encryption, the service applies the existing Content Moderation secret
redactor to the normalized text. The retained value is the exact text used by
Content Moderation after its existing 12,000-rune input bound; image data URLs
are not included.

The service prepends `sub2api:content-moderation-keyword-input:v1:` inside the
authenticated ciphertext and strips it only after successful decryption and
exact purpose validation. This prevents ciphertext produced for another field
with the shared application encryptor from being disclosed through this route.

Encryption failure must not weaken or reverse an already-known keyword
finding. The request remains blocked and the normal excerpt remains available,
while the failure is logged with a stable operation name and error kind only.

## Read boundary

The existing paginated list query does not select `input_ciphertext`. A new
administrator route loads one row by numeric ID. The service decrypts content
only when a matched keyword and ciphertext exist and the decrypted envelope has
the expected purpose/version prefix. This supports ordinary `keyword_block`
rows and blocking-exempt keyword `shadow` rows without accepting unrelated
ciphertext.

The response returns:

- `id`
- `content`
- `complete`

Historical rows and non-keyword rows return the existing excerpt with
`complete=false`. Missing rows return a generic 404. Decryption failures return
a generic internal error without ciphertext or cryptographic details.

## UI behavior

The table continues rendering and tooltiping only `input_excerpt`. Opening the
detail dialog starts a record-scoped fetch. The dialog distinguishes loading,
complete retained content, historical/incomplete excerpt, and load failure.
Closing or switching records prevents a late response from replacing the
current dialog content.

## Retention and rollback

Existing Content Moderation cleanup deletes the complete row, so ciphertext
inherits hit-retention behavior without a second cleanup path. Rollback leaves
the additive column inert. Historical rows cannot be backfilled because their
discarded content is unavailable.
