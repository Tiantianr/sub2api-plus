# Release Process

This is the canonical pull-request-first process for custom Sub2API Plus
releases. Ordinary branch pushes are fast. The final PR submission performs the
complete local validation matrix once, and release-cli then relies on that exact
commit proof plus protected GitHub Actions before merging and tagging.

## Time Budget and Request Handling

This process has no five-minute end-to-end guarantee. The five-minute budget
refers only to the tag-triggered `Publish Linux image` job after all of these
prerequisites are complete:

- the final candidate was submitted and merged through the protected PR;
- `CI` and `Security Scan` succeeded for the exact main merge commit; and
- the matching reusable Linux arm64 image artifact is available.

Candidate preparation, local validation, PR checks, merge waiting, tag review,
asset publication, and final verification are outside that runner budget. Call
the bounded step the **five-minute image publication stage**, never a
five-minute release.

When a request gives the whole release a deadline that cannot include its
remaining prerequisites, report the unmet stages and stop before committing,
pushing, submitting or merging a PR, creating or pushing a tag, or publishing
an image. Continue the formal flow only after the requester explicitly accepts
the disclosed scope and timing. This repository has no local-worktree or
`publish-local` release shortcut.

## Version Format

| Surface | Format |
| --- | --- |
| Git tag and GitHub Release | `vX.Y.Z+custom.NNN` |
| Embedded application version | `X.Y.Z+custom.NNN` |
| OCI image tag | `vX.Y.Z-custom.NNN` |

Increment `NNN` on the same official `X.Y.Z` baseline and reset it to `001`
after merging a newer official baseline. `NNN` is always three digits.

The `Tiantianr/sub2api-plus` distribution reserves iterations `901` through
`999` and resets to `901` only after importing a Plus release based on a newer
official `X.Y.Z` baseline. Its release commands must set
`SUB2API_CUSTOM_ITERATION_MIN=901`; inherited Plus tags below that range remain
historical or mirrored rollback inputs, not new personal release versions.

## Repository Prerequisites

Before automatic PR promotion is enabled, repository administrators must:

1. Enable GitHub repository Auto-merge and merge-commit mode.
2. Protect `main` with a ruleset that requires pull requests.
3. Require the branch to be current with `main` and require these exact status
   contexts: `sub2api/local-validation`, `deployment-config`, `test`,
   `frontend`, `golangci-lint`, `goreleaser-config`, `linux-image-artifact`,
   `repository-policy`, `backend-security`, and `frontend-security`.
4. Keep those contexts synchronized with the CI and Security Scan job IDs so a
   renamed or later unvalidated job cannot silently leave the merge gate.
5. Block force pushes and branch deletion; do not give release-cli an admin
   bypass.
6. Configure the Actions environment named `release` with administrator bypass
   disabled, no required reviewer, wait timer, or custom gate, and exactly one
   deployment policy for tags matching `v*+custom.*`.
7. Add an active repository Tag ruleset for `refs/tags/v*+custom.*`. Do not add
   bypass actors or a creation restriction; block tag updates and deletion.

The source tree cannot create these external governance settings safely.
`release-cli promote-pr` verifies Auto-merge, merge-commit mode, the required
pull-request rule, strict current-branch policy, and the complete context list.
Before tag transfer, `release-cli publish` verifies the complete Environment
and Tag ruleset policy. Both actions fail closed when an external prerequisite
is absent.

## Prepare the Release PR

Start a working branch from the latest `origin/main`, then:

1. Confirm the official upstream tag and commit.
2. If the previous personal release remains `planned`, mark it `published` in
   this PR; `promote-pr` revalidates its tag, workflow, assets, and image. If it
   failed publication, mark it `invalid` or `withdrawn` instead.
3. Update `backend/cmd/server/VERSION`.
4. Update every `ARG VERSION=` in `Dockerfile` and `backend/Dockerfile`.
5. Add the custom version to `UPSTREAM.md` with status `planned`.
6. Synchronize install, rollback, and image examples:
   `python3 tools/update_release_docs.py`.
7. Write `release-notes.md` from the template below.
8. Commit all repository changes. Never commit on or push directly to `main`.

Intermediate pushes are intentionally fast and skip local validation:

```bash
python3 skills/push-cli/scripts/push_cli.py push
```

When the branch is the final merge candidate, submit it once through the local
promotion gate:

```bash
python3 skills/push-cli/scripts/push_cli.py submit-pr
```

`submit-pr` fetches the current `origin/main`, requires it in the branch,
records exact base/head SHAs, runs the host repository preflight, refetches and
rechecks both SHAs, pushes the exact head, publishes
`sub2api/local-validation`, and creates or reuses the PR. Protected Linux
GitHub Actions run integration, lint, production build, security, and Docker
gates. Any later head or base change requires another `submit-pr`.

The documentation updater reads the current version from
`backend/cmd/server/VERSION`. Its rollback example uses the nearest lower
`published` entry in `UPSTREAM.md`; it skips `planned`, `historical`,
`withdrawn`, and `invalid`. Check without writing files with:

```bash
python3 tools/update_release_docs.py --check
```

## Release Notes

The first non-empty line is the annotated-tag subject. `Changed` and `Fixed`
are optional; every other section below is required and non-empty.

```markdown
Sub2API Plus vX.Y.Z+custom.NNN

## Highlights

Describe the primary user-visible changes.

## Changed

Optional details.

## Compatibility and migration

None.

## Known issues

None.

## Upstream baseline

Official release: vX.Y.Z
Official commit: <40-character commit>
```

## Promote the Release PR

Promotion requires the explicit PR number, intended tag, and reviewed notes:

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md
```

The tool verifies the PR is open, non-draft, same-repository, and targets
`main`. Its machine marker and successful local-validation status must match
the current head and current `main` base exactly. It waits for GitHub required
checks, rechecks both SHAs, and enables native `--auto --merge` without admin
bypass.

After GitHub merges the PR, release-cli resolves the actual merge commit,
fetches `origin/main`, and waits for push-triggered `CI` and `Security Scan`
runs at that exact SHA. The main CI run also builds a Linux arm64 Docker image
artifact for that commit without publishing it. A successful PR check alone is
insufficient because the merge commit may differ from the PR head.

If protected review or another required condition is still pending, promotion
returns status 2. Complete the GitHub requirement and rerun the same command.

## Validate and Create the Tag

After PR promotion, run the focused release metadata gate against the merged
PR commit, then repeat it while creating the local annotated tag:

```bash
python3 skills/release-cli/scripts/release_cli.py validate \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md

python3 skills/release-cli/scripts/release_cli.py tag \
  --tag vX.Y.Z+custom.NNN \
  --pr <number> \
  --notes-file release-notes.md
```

This gate does not repeat Go, frontend, lint, integration, or deployment
matrices. Those ran in `submit-pr`, PR Actions, and merged-main Actions. It
validates release metadata, notes, synchronized examples, exact tree identity,
and local/remote tag absence. `tag` targets the PR's actual merge commit and
preserves the notes verbatim. It never pushes.

Review the local tag, then explicitly publish only that tag:

```bash
git show --no-patch vX.Y.Z+custom.NNN
python3 skills/release-cli/scripts/release_cli.py publish \
  --tag vX.Y.Z+custom.NNN
```

`publish` first verifies the automatic `release` Environment and immutable
custom-tag ruleset, then returns after exact tag transfer. This explicit command
is the irreversible publication authorization point. Never use `git push
--tags`, reuse a version, retag, force push, or create the GitHub Release
manually.

## Monitor and Verify

Monitor publication separately. The remote annotated tag is the source of
truth, so this action can resume without relying on a local tag:

```bash
python3 skills/release-cli/scripts/release_cli.py monitor \
  --tag vX.Y.Z+custom.NNN
```

The Release workflow does not rerun the application matrix. Its
`Publish Linux image` job verifies the annotated tag, main ancestry, successful
`CI` and `Security Scan` push runs for the exact SHA, release metadata, and the
matching unexpired image artifact. It then enters the checked `release`
Environment and runs the five-minute image publication stage. That runner
budget starts here; it excludes every prerequisite listed in
[Time Budget and Request Handling](#time-budget-and-request-handling). `Build
release assets` starts only after the image is available. If either job
unexpectedly waits, `monitor` reports policy drift and the Actions URL; restore
the Environment policy and rerun `monitor`. The CLI never approves or bypasses
a deployment.

After the workflow succeeds, verify the published state:

```bash
python3 skills/release-cli/scripts/release_cli.py verify \
  --tag vX.Y.Z+custom.NNN
```

Verification requires a successful workflow, a non-draft/non-prerelease GitHub
Release, `checksums.txt`, the Linux arm64 archive, both immutable pricing
assets, and a publicly pullable `linux/arm64` GHCR image:

- `checksums.txt`
- `sub2api_<version>_linux_arm64.tar.gz`
- `model-pricing.json`
- `model-pricing-manifest.json`
- `ghcr.io/<owner>/sub2api-plus:<OCI version>`

The workflow accepts an existing pricing asset only when its bytes are
identical. Correct a bad asset with a new custom version, never by replacement
or retagging.

The release workflow publishes only the immutable OCI version tag. It does not
move `latest`, major, or minor aliases, so rerunning an older release cannot
silently roll back another deployment. If the validated main image artifact has
expired, rerun that exact main CI before `publish`; release-cli refuses to push
the immutable Git tag until the artifact is available.

## Defer Published Mapping

For this single-maintainer distribution, successful `verify` completes the
release. Do not open another PR immediately. During preparation of the next
release, change the previous verified version from `planned` to `published` in
the same release PR, then add the new version as `planned` and regenerate the
release documents. The metadata gate rejects stale planned versions, and
`release-cli promote-pr` verifies each deferred transition against its remote
annotated tag, successful Release workflow, immutable assets, and anonymous
GHCR image. It also rejects deleted mappings, changed upstream ancestry,
unrelated new mappings, and invalid status reversals.

This removes all post-publication PR and CI time. GitHub Release and the
immutable remote tag remain the publication source of truth between releases.

When immediate Git metadata is explicitly required, use the optional recovery
path:

```bash
python3 skills/release-cli/scripts/release_cli.py finalize \
  --tag vX.Y.Z+custom.NNN
```

`finalize` fetches the latest `origin/main`, creates deterministic branch
`release/finalize-X.Y.Z-custom.NNN`, and changes exactly one `UPSTREAM.md`
status from `planned` to `published`. It validates that historical mapping
independently from the current embedded version. If a newer release was already
prepared, it also synchronizes the generated rollback examples; otherwise only
`UPSTREAM.md` changes. It then calls `push-cli submit-pr` and never commits or
pushes `main` directly.

After that optional follow-up PR passes, promote it without release notes:

```bash
python3 skills/release-cli/scripts/release_cli.py promote-pr \
  --tag vX.Y.Z+custom.NNN \
  --pr <finalization-pr-number>
```

The no-notes form is accepted only for the deterministic finalization branch
and requires the release mapping to be `published`.

## Pricing Assets

The manifest binds the release tag, fixed asset URL, and data SHA-256. Runtime
loading also validates dedicated HTTPS hosts, response sizes, JSON shape, and
version rollback. Release publication authority is the pricing trust boundary,
so tag-creation authority and Release/package permissions must remain limited
to trusted maintainers.

## Failed or Invalid Releases

- Never reuse or retag a published version.
- Resolve a failed previous `planned` release as `withdrawn` or `invalid` in
  the next correction release PR.
- Treat `historical`, `invalid`, and `withdrawn` mappings as terminal; never
  restore one to `planned` or `published`.
- Publish corrections under the next custom iteration.
- If tag push succeeded but local observation was interrupted, resume with
  `monitor`; do not rerun `publish`.
- If a release publication job unexpectedly waits, restore the checked Environment
  policy and rerun `monitor`; do not approve the drifted deployment through the
  CLI.
- Deleting an unpublished tag or artifact requires an explicit audit and
  maintainer decision.
