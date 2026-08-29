## Context

Observed items use:

```text
type, id, author, recipient, content[],
internal_chat_message_metadata_passthrough{create_time,turn_id}
```

`content` contains visible `input_text` and may contain an opaque
`encrypted_content` companion. The current generic Responses path extracts the
visible sibling but still records unsupported-item and unextractable-content
reasons.

## Decisions

### 1. Add one canonical adapter

Both engines continue consuming `auditcontent.Document`; no Prompt Audit or
Content Moderation parser is added. The adapter accepts the observed outer
fields and delegates visible content blocks to the existing content parser.

### 2. Preserve attribution

Recognized `author` values `user`, `system`, `developer`, `assistant`, and
`model` retain their canonical roles. Empty or other named authors are
classified as assistant so internal Agent output cannot create direct-user
Content Moderation punishment. System and developer authors remain instruction
sources.

### 3. Ignore only known opaque blocks

`encrypted_content` blocks are skipped without persistence or logging. Other
content block types continue through the ordinary strict parser and remain
incomplete if unsupported. Routing and turn metadata are non-content and are
accepted only under the observed field names.

## Risks

- A future client may put sole semantic content in `encrypted_content`; the
  gateway cannot inspect it. This release follows the existing handling of
  opaque Responses reasoning/compaction data and still audits every visible
  sibling.
