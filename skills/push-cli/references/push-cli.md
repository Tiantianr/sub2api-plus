# Push CLI Reference

## Action Contract

| Action | Local work | GitHub mutation | Result |
| --- | --- | --- | --- |
| `push` | None | Exact current-branch push | Fast intermediate branch publication. |
| `submit-pr` | Host repository preflight | Exact push, commit status, PR create/update | Final preflight-validated candidate. |
| `check` | Host repository preflight | None | Read-only preflight result. |
| `ensure` | Toolchain check only | None | Required host toolchains are ready. |
| `watch` | None | None | Watches branch push runs at current HEAD. |

All actions run the GitHub CLI repository gate. `push` and `submit-pr` also
resolve and reject the default branch and require a clean worktree with no
unfinished Git operation.

The repository gate defaults to `LuckyKuang/sub2api-plus`. Maintainers of an
explicitly trusted fork must set `SUB2API_EXPECTED_REPOSITORY=owner/repository`
for every command; the override changes the exact expected repository and does
not disable any safety check.

## Fast Push

The only branch transfer is:

    git push origin HEAD:<current-branch>

Ordinary push returns after Git accepts the ref. It does not run the local
matrix and does not wait for Actions. Use `watch` when remote observation is
needed. Use `submit-pr`, never ordinary push, for the final PR candidate.

## Validated Submission

`submit-pr` performs this fail-closed sequence:

1. Fetch `refs/heads/<default>` into `refs/remotes/origin/<default>` without
   fetching tags.
2. Require that exact base commit to be an ancestor of HEAD.
3. Record the 40-character base and head SHAs.
4. Verify the pinned host toolchains.
5. Run the repository preflight.
6. Require a clean worktree and unchanged HEAD.
7. Refetch the default branch and require the same base SHA.
8. Push exactly `HEAD:<current-branch>`.
9. Publish commit status context `sub2api/local-validation` for the head.
10. Create or update one PR whose base/head marker matches those SHAs.

The marker format is implementation-owned and must occur exactly once:

    <!-- sub2api-submit-pr: {"base":"<sha>","head":"<sha>"} -->

## Local Preflight

The preflight includes:

- Go module tidiness and unit tests.
- Compress CLI, push CLI, and release CLI self-tests.
- Frozen pnpm install, lint, typecheck, and Vitest.
- Release policy, release metadata, README synchronization, Codex outbound
  identity, and migration checks against the validated default-branch base.
- Installer syntax, Docker deployment security/resources, and Caddy cache
  policy.

Protected Linux GitHub Actions run backend integration tests, golangci-lint,
frontend production builds, production dependency audits, security scans, and
Docker Compose validation.

## Recovery

- If ordinary push succeeds but remote Actions fail, fix on the branch and push
  again; no local proof was issued.
- If `submit-pr` preflight fails, no push/status/PR mutation occurs.
- If the default branch changes during validation, rerun `submit-pr` after
  updating the branch.
- If push succeeds but status or PR creation fails, rerun `submit-pr`; the
  preflight runs again and the exact matching PR may be reused.
- If the PR head changes later, its old status and marker cannot authorize
  release promotion.
