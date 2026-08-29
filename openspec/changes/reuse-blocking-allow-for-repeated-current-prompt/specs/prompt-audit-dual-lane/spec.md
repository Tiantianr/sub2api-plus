## MODIFIED Requirements

### Requirement: Guard source modules are independently configurable per lane

Synchronous Prompt Audit SHALL select the latest user turn unless an exact
blocking Allow receipt certifies it, and SHALL add the configured synchronous
source-module receipt misses. Asynchronous Prompt Audit SHALL select all user
turn and configured source-module receipt misses while keeping an async-only
current user turn mandatory.

#### Scenario: User-installed extension is synchronously reviewed

- **WHEN** synchronous system or tool-definition review is enabled
- **AND** a request carries a previously uncertified skill instruction or
  plugin definition
- **THEN** Guard receives that content together with any current-user receipt
  miss
- **AND** assistant, reasoning, tool-call, and tool-output content remains
  excluded unless its own module is enabled

### Requirement: Guard scans user-authored prompt text

Synchronous Prompt Audit SHALL scan every selected user-authored receipt miss,
including current text without a valid blocking receipt, plus the independently
enabled synchronous source-module misses. Asynchronous Prompt Audit SHALL scan
all user-authored and configured asynchronous source-module misses while
keeping async-only current text mandatory. Environment, permission, and
filesystem wrapper blocks SHALL remain excluded from selected user text;
`system-reminder` SHALL follow the system-module setting.

#### Scenario: Repeated fixed current text is certified

- **WHEN** blocking Prompt Audit receives current user text with a valid exact
  blocking Allow receipt, including an equivalent strict media-marker form
- **THEN** that text is absent from Guard input
- **AND** any other selected receipt miss remains present

### Requirement: Administration controls must reflect effective blocking behavior

The administration console SHALL state that new or changed current user input
requires synchronous review while exact repeated current text may reuse a
valid blocking Allow receipt. It SHALL expose independent module controls for
the synchronous and asynchronous lanes. The compatibility field MAY remain in
storage and API schemas but SHALL NOT override module selection.

#### Scenario: Administrator changes one source module

- **WHEN** an administrator enables synchronous tool definitions without
  enabling synchronous tool outputs
- **THEN** only uncertified tool-definition segments are added to blocking
  Guard input
