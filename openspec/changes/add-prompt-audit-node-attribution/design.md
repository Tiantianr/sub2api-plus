# Design

`OpenAICompatibleScanner.Scan` already receives the exact `ActiveEndpoint` used
after failover. It attaches that endpoint's ID, name, and model to the normalized
result. Aggregation copies all three fields together whenever it selects the
result that determines the event decision; all-Pass aggregation keeps the first
successful result, matching the existing endpoint-ID behavior.

The repository stores `guard_endpoint_name` and `guard_model` beside the
existing `guard_endpoint_id`. They are event-time snapshots, so later config
edits do not rewrite history. Historical rows remain untouched: the UI uses
`scanner_version` as the model fallback because that field previously held the
endpoint model, and it uses the stable endpoint ID when the name is empty.

The list adds one compact audit-node column containing name, model, and ID. The
detail view exposes separate node, model, and ID rows. No filter or index is
added because attribution is for inspection, not a requested query dimension.
