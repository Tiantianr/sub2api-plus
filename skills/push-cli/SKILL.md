---
name: push-cli
description: >-
  Safely push Sub2API Plus working branches and submit the final locally
  validated pull request. Use when the user asks to push code, publish the
  current branch, run the repository validation matrix, create or update a
  pull request, or verify branch CI. Ordinary push is fast and never runs the
  local matrix. submit-pr is the only final promotion boundary: it requires
  the latest default-branch base, runs the host repository preflight, binds the
  result to exact base/head SHAs, pushes, publishes the local-validation commit
  status, and creates or reuses the pull request. Never push the repository
  default branch.
---

# Push CLI

Run commands from the repository root:

    python3 skills/push-cli/scripts/push_cli.py push
    python3 skills/push-cli/scripts/push_cli.py submit-pr
    python3 skills/push-cli/scripts/push_cli.py check
    python3 skills/push-cli/scripts/push_cli.py ensure
    python3 skills/push-cli/scripts/push_cli.py watch

`push` performs an authenticated exact-ref push of the clean current working
branch. It does not probe a container runtime, run local tests, create a pull
request, or wait for Actions. It rejects the GitHub default branch, detached
HEAD, unfinished Git operations, an unexpected remote, and every force/tag/all
push form.

`submit-pr` is the final promotion action. It fetches the current GitHub default
branch, requires that commit to be contained by the candidate branch, records
the exact base and head, and runs the repository preflight with pinned host
toolchains. After the preflight it refetches the default branch and requires the
worktree, base, and head to be unchanged. It then pushes the exact head,
publishes `sub2api/local-validation=success`, and creates or reuses one pull
request to the default branch. Protected Linux GitHub Actions own the complete
integration, lint, build, security, and Docker gates.

`check` runs the same local preflight without pushing or creating a PR.
`ensure` only checks the required host toolchains. `watch` observes
push-triggered Actions for the current branch and SHA.

## Mandatory GitHub CLI Gate

Every action requires an installed and authenticated GitHub CLI. By default,
resolve the origin repository exactly as `LuckyKuang/sub2api-plus`. An
explicitly trusted fork may set `SUB2API_EXPECTED_REPOSITORY=owner/repository`;
the same exact-repository, access, and push-permission checks still apply.
Resolve the default branch from GitHub. Never run `gh auth login` automatically
or fall back to another credential or HTTP client.

Before an authorized push, configure Git transport with `gh auth setup-git`.
Git transfers only `HEAD:<current-branch>`. Never use `--force`, `--all`,
`--tags`, or another local ref.

## Local Preflight

`check` and `submit-pr` run Go unit tests, frontend lint/typecheck/Vitest, CLI
self-tests, release and documentation policy, migration policy, installer
syntax, and deployment static checks with the pinned host toolchains. `ensure`
checks those toolchains without running tests. Linux GitHub Actions run backend
integration tests, golangci-lint, frontend production builds, security scans,
and Docker Compose validation.

## Pull-Request Proof

The PR body contains one machine-readable base/head marker and its head commit
carries the `sub2api/local-validation` status. A later branch commit has no
matching status. A later default-branch update no longer matches the marker.
Either condition requires another `submit-pr`; never edit the marker or status
manually.

`submit-pr` creates a ready PR by default. `--title` and `--body-file` may
provide reviewed PR text. If exactly one open PR already exists for the same
head/base, update only its validation marker. Refuse ambiguous PRs or mismatched
head/base SHAs.

## Safety

- Never push the repository default branch, a dirty worktree, detached HEAD,
  or an unfinished merge/rebase/cherry-pick/revert.
- Never imply that ordinary `push` performed local validation.
- Never publish a success status until the post-matrix base/head/worktree
  recheck and exact branch push both succeed.
- Never create a PR for a stale default-branch base.
- Never silently downgrade required tool versions.
- Treat a local validation failure as a hard stop before push or PR mutation.

Read `references/push-cli.md` for the exact matrix, proof format, and recovery
rules. Use `scripts/push_cli.py` for every push or submission action.
