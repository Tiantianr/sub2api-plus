# Prompt Audit Dual-Lane Specification

## ADDED Requirements

### Requirement: Guard source modules are independently configurable per lane

Synchronous Prompt Audit SHALL always review the latest user turn and SHALL add
the configured synchronous source modules. Asynchronous Prompt Audit SHALL
always review all user turns and SHALL add the configured asynchronous source
modules.

#### Scenario: User-installed extension is synchronously reviewed

- **WHEN** synchronous system or tool-definition review is enabled
- **AND** a request carries a skill instruction or plugin definition
- **THEN** Guard receives that content together with the latest user turn
- **AND** assistant, reasoning, tool-call, and tool-output content remains
  excluded unless its own module is enabled

### Requirement: Blocking Allow starts deep review

The system SHALL enqueue an `async_deep` job after an applicable blocking
request passes the combined synchronous security decision and before the
handler advances. A failure to enqueue SHALL be observable but SHALL NOT change
the synchronous Allow.

#### Scenario: Content Moderation blocks the request

- **WHEN** Prompt Guard allows but Content Moderation blocks
- **THEN** no async-deep job is enqueued
- **AND** no account, billing, concurrency, or upstream side effect occurs

### Requirement: Deep Block requires a fenced synchronous recovery review

An `async_deep` Block SHALL establish a per-user deep-review requirement before
the job completes. The user's next request SHALL synchronously use the configured
deep modules. Only Allow SHALL atomically clear the same requirement version.

#### Scenario: A newer finding races an older recovery Allow

- **WHEN** a newer deep Block replaces the user's requirement while an older
  synchronous recovery is running
- **THEN** the older Allow cannot clear the newer requirement
- **AND** the request fails closed without accessing upstream

#### Scenario: Recovery returns Warn

- **WHEN** the forced synchronous deep review returns Warn
- **THEN** the request is rejected
- **AND** the deep-review requirement remains

### Requirement: Deep events are independently reviewable

Async-deep jobs and events SHALL use execution mode `async_deep`, and event
list/delete filters SHALL support that mode. Stored events SHALL retain the
same encrypted complete-context evidence contract as other Prompt Audit modes.

#### Scenario: Administrator filters deep findings

- **WHEN** an administrator filters events by `execution_mode=async_deep`
- **THEN** only asynchronous deep-review events are returned

### Requirement: Extraction failures retain safe structural diagnostics

Content Moderation and Prompt Audit SHALL use the same canonical failure paths
to log a bounded structural description of each failed node. Diagnostics SHALL
include enough protocol metadata to distinguish unsupported item types and
shapes, while excluding raw message text, tool payload values, credentials,
media, and other scalar content.

#### Scenario: A new Responses item type fails extraction

- **WHEN** a Responses input item has an unsupported type and contains a secret
  payload value
- **THEN** both engines log the item path, constrained type, value kind, object
  keys, value-free shape, byte count, and stable fingerprints
- **AND** neither log contains the secret payload value

## MODIFIED Requirements

### Requirement: Guard scans user-authored prompt text

Synchronous Prompt Audit SHALL scan the latest user-authored input and the
independently enabled synchronous source modules. Asynchronous Prompt Audit
SHALL scan all user-authored turns and the independently enabled asynchronous
source modules. Environment, permission, and filesystem wrapper blocks SHALL
remain excluded from selected user text; `system-reminder` SHALL follow the
system-module setting.

#### Scenario: Assistant review differs by lane

- **WHEN** assistant review is disabled synchronously and enabled asynchronously
- **THEN** assistant history is absent from the blocking Guard input
- **AND** the same assistant history is present in the async-deep Guard input

### Requirement: Administration controls must reflect effective blocking behavior

The administration console SHALL state that the latest user input is mandatory
for synchronous review and SHALL expose independent module controls for the
synchronous and asynchronous lanes. The compatibility field MAY remain in
storage and API schemas but SHALL NOT override module selection.

#### Scenario: Administrator changes one source module

- **WHEN** an administrator enables synchronous tool definitions without
  enabling synchronous tool outputs
- **THEN** only the tool-definition module is added to blocking Guard input
