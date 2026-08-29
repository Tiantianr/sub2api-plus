# Accept Responses agent messages

## Problem

Production Responses clients submit historical `agent_message` items with
visible `input_text`, optional `encrypted_content`, author/recipient routing,
and bounded turn metadata. The shared canonical extractor treats the whole
item as unsupported, so both blocking audit engines reject otherwise
extractable requests with HTTP 503.

## Proposal

- Recognize `agent_message` in the shared Responses canonical extractor.
- Extract visible text and classify recognized authors by role.
- Treat unknown named agents as non-user assistant sources.
- Ignore only the observed opaque encrypted-content and routing/turn metadata.
- Keep unknown content-bearing blocks observable and fail closed under the
  existing blocking policy.

## Non-goals

- Decrypting or auditing opaque `encrypted_content`.
- Treating arbitrary unknown Responses types as supported.
- Changing Content Moderation or Prompt Audit policy decisions.
- Deploying or changing production configuration in this release action.

## Impact

Observed `agent_message` histories no longer create extraction failures.
Visible user-authored text remains available to both engines, while assistant
and named-agent output is not attributed to the direct user.
