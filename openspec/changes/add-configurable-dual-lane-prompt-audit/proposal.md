# Change: Add configurable dual-lane Prompt Audit

## Why

Latest-user-only blocking misses user-installed skills, system instructions,
prompt variables, and plugin definitions. Async-only review cannot ensure that
a later deep finding is reviewed before the same user continues.

## What Changes

- Configure synchronous and asynchronous Guard input by canonical source module.
- Keep latest user text mandatory for synchronous review and all user turns
  mandatory for asynchronous review.
- Enqueue an `async_deep` review after a blocking request is allowed.
- Require the next request from a user with a deep Block to pass synchronous
  deep review before normal dual-lane processing resumes.
- Expose deep-review events through a dedicated execution-mode filter.

## Impact

- Prompt Audit configuration gains two source-module maps.
- Redis retains a versioned per-user deep-review requirement without a TTL.
- Forward migrations admit `async_deep` job/event rows and add a concurrent mode
  index.
- Shared extraction failures record bounded value-free node structure so new
  protocol shapes can be diagnosed without logging prompt or tool values.
- Guard failures and deep-review state failures remain fail closed on the
  synchronous boundary.
