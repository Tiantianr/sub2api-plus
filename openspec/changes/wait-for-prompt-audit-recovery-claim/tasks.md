# Tasks

## 1. Contract

- [x] 1.1 Define request-context waiting, owner-clear, retained-finding, and
  dependency-failure behavior.

## 2. Backend

- [x] 2.1 Wait for occupied claims without replacing claim or finding tokens.
- [x] 2.2 Re-read finding state after claim acquisition and resume the correct
  audit mode.
- [x] 2.3 Add bounded wait lifecycle logs.

## 3. Documentation

- [x] 3.1 Update protocol and security-audit failure semantics.

## 4. Verification

- [x] 4.1 Cover owner Allow, retained finding, cancellation, and concurrent
  claim ownership.
- [x] 4.2 Run focused race, full backend, lint, docs, strict OpenSpec, and diff
  checks.
