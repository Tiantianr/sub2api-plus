## Why

The current push tool runs the complete local validation matrix before every
branch push, which makes iterative development unnecessarily expensive. At the
same time it does not reject the repository default branch. The release tool
then repeats the complete local matrix, mixes tag publication with workflow
monitoring, and can finalize release metadata on whichever branch is checked
out.

The repository needs one explicit promotion boundary: ordinary branch pushes
are fast, while the final push that submits a pull request carries an exact,
server-visible proof of local validation. Release publication can then trust
that proof together with GitHub Actions, merge the reviewed pull request, and
tag only the tested commit that reached `main`.

## What Changes

- Make `push-cli push` a fast, non-default-branch push with no local test
  matrix and no implicit Actions wait.
- Add `push-cli submit-pr` to validate the latest default-branch base and exact
  head commit inside the platform container, push that commit, publish a
  commit-status proof, and create or reuse its pull request.
- Reject pull-request promotion when the head or base moved after local
  validation, when required Actions are not successful, or when repository
  auto-merge and default-branch protection are not enabled.
- Split release tag publication from release workflow monitoring and asset
  verification.
- Create release tags from the successfully merged pull-request commit only
  after the exact `main` push Actions succeed.
- End normal personal releases at verification and finalize their published
  metadata in the next release PR. Retain a fresh finalization branch through
  the same pull-request gate only when immediate metadata is required.

## Capabilities

### New Capabilities

- `push-release-pr-promotion`: Defines fast branch pushes, locally validated
  pull-request submission, protected automatic merge, immutable tag creation,
  deferred post-publication metadata, and optional immediate finalization.

### Modified Capabilities

None.

## Impact

- **Developer workflow**: Iterative pushes skip local checks. The final
  `submit-pr` action runs a bounded host preflight before protected Linux
  GitHub Actions execute the complete PR gates.
- **Repository governance**: Safe automatic merging requires a protected
  default branch, required Actions checks, and GitHub auto-merge. The CLI fails
  closed when those external settings are absent.
- **Release workflow**: Release validation reuses the submitted PR proof and
  remote Actions instead of rerunning the full local application matrix.
- **Compatibility**: Existing release tags and Releases remain immutable. No
  application API, database schema, or runtime configuration changes.
