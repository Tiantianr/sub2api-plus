# Prompt Audit node attribution

## Problem

Prompt Audit events retain only the Guard endpoint ID. The administration list
does not show it, and it does not retain the configured node name or an explicit
model snapshot. Looking up the current configuration would make historical
events misleading after a node is renamed or removed.

## Proposal

- Snapshot the actual Guard node ID, configured name, and model on each
  completed scan result.
- Carry the node that determines the aggregate decision into the audit event.
- Show the node name and model in the event list and detail view, with endpoint
  ID and legacy scanner version as compatibility fallbacks.
- Display the already stored scanner version as the model fallback for existing
  Qwen3Guard events; do not rewrite rows or invent historical node names.

## Impact

This adds two non-secret columns to `prompt_audit_events` and two additive event
API fields. It does not store node URLs, credentials, request content, or raw
Guard responses and does not change routing, failover, blocking, or billing.
