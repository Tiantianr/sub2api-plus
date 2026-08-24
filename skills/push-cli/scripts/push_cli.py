#!/usr/bin/env python3
"""Validate and safely push a Sub2API Plus branch."""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Sequence


ROOT = Path(__file__).resolve().parents[3]
DEFAULT_REMOTE = "origin"
EXPECTED_REPOSITORY = os.environ.get(
    "SUB2API_EXPECTED_REPOSITORY",
    "LuckyKuang/sub2api-plus",
)
LOCAL_VALIDATION_CONTEXT = "sub2api/local-validation"
VALIDATION_MARKER_RE = re.compile(
    r"<!--\s*sub2api-submit-pr:\s*(\{.*?\})\s*-->", re.DOTALL
)
GO_VERSION_RE = re.compile(r"^go\s+(\d+\.\d+(?:\.\d+)?)\s*$", re.MULTILINE)
class PushCliError(RuntimeError):
    """A hard failure that must stop validation or pushing."""


@dataclass(frozen=True)
class DeclaredToolchains:
    go: str
    pnpm: str
    node_major_minimum: int


@dataclass(frozen=True)
class ValidationProof:
    base: str
    head: str


def display(command: Sequence[str]) -> str:
    return shlex.join(str(item) for item in command)


def run_command(
    command: Sequence[str],
    *,
    cwd: Path | None = None,
    capture: bool = False,
    merge_stderr: bool = True,
) -> subprocess.CompletedProcess[str]:
    actual_cwd = ROOT if cwd is None else cwd
    try:
        return subprocess.run(
            [str(item) for item in command],
            cwd=actual_cwd,
            check=False,
            text=True,
            encoding="utf-8",
            errors="replace",
            stdout=subprocess.PIPE if capture else None,
            stderr=(subprocess.STDOUT if merge_stderr else subprocess.PIPE)
            if capture
            else None,
        )
    except FileNotFoundError as error:
        raise PushCliError(f"required command is unavailable: {error.filename}") from error


def capture(command: Sequence[str], *, cwd: Path | None = None) -> str:
    result = run_command(command, cwd=cwd, capture=True)
    if result.returncode != 0:
        detail = (result.stdout or "").strip()
        raise PushCliError(
            f"{display(command)} failed with exit code {result.returncode}"
            + (f": {detail[-2000:]}" if detail else "")
        )
    return (result.stdout or "").strip()


def require_command(command: str) -> None:
    if shutil.which(command) is None:
        raise PushCliError(f"required command is unavailable: {command}")


def repo_from_remote(url: str) -> str:
    match = re.search(r"github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?$", url.strip())
    if not match:
        raise PushCliError(f"origin is not a GitHub repository URL: {url}")
    return f"{match.group(1)}/{match.group(2)}"


def github_gate(remote: str) -> str:
    require_command("gh")
    version = capture(["gh", "--version"]).splitlines()[0]
    print(f"GitHub CLI: {version}")

    auth = run_command(
        ["gh", "auth", "status", "--hostname", "github.com"],
        capture=True,
    )
    if auth.returncode != 0:
        detail = (auth.stdout or "").strip()
        raise PushCliError(
            "GitHub CLI is missing a valid github.com login. Run "
            "gh auth login and retry."
            + (f"\n{detail[-2000:]}" if detail else "")
        )

    remote_url = capture(["git", "remote", "get-url", remote])
    repository = repo_from_remote(remote_url)
    if repository != EXPECTED_REPOSITORY:
        raise PushCliError(
            f"origin resolves to {repository}; expected {EXPECTED_REPOSITORY}"
        )

    capture(["gh", "repo", "view", repository, "--json", "nameWithOwner"])
    push_permission = capture(
        ["gh", "api", f"repos/{repository}", "--jq", ".permissions.push"]
    ).lower()
    if push_permission != "true":
        raise PushCliError(
            f"authenticated GitHub account has no push permission for {repository}"
        )
    print(f"GitHub repository: {repository} (push permission confirmed)")
    return repository


def repository_default_branch(repository: str) -> str:
    branch = capture(
        ["gh", "api", f"repos/{repository}", "--jq", ".default_branch"]
    )
    if not branch or any(char.isspace() for char in branch):
        raise PushCliError("GitHub returned an invalid repository default branch")
    print(f"Default branch: {branch}")
    return branch


def current_branch() -> str:
    branch = capture(["git", "branch", "--show-current"])
    if not branch or branch == "HEAD":
        raise PushCliError("detached HEAD is not pushable; check out a branch first")
    if any(char.isspace() for char in branch):
        raise PushCliError(f"invalid current branch name: {branch}")
    print(f"Current branch: {branch}")
    return branch


def require_working_branch(branch: str, default_branch: str) -> None:
    if branch == default_branch:
        raise PushCliError(
            f"refusing to push repository default branch {default_branch!r}; "
            "create or switch to a working branch first"
        )


def require_no_git_operation() -> None:
    git_dir = Path(capture(["git", "rev-parse", "--git-dir"]))
    if not git_dir.is_absolute():
        git_dir = ROOT / git_dir
    active = [
        name
        for name in (
            "MERGE_HEAD",
            "CHERRY_PICK_HEAD",
            "REVERT_HEAD",
            "rebase-merge",
            "rebase-apply",
        )
        if (git_dir / name).exists()
    ]
    if active:
        raise PushCliError(
            "an unfinished Git operation is present: " + ", ".join(active)
        )


def require_clean_worktree() -> None:
    status = capture(
        ["git", "status", "--porcelain=v1", "--untracked-files=all"]
    )
    if status:
        raise PushCliError(
            "worktree is not clean; commit or remove these paths before pushing:\n"
            + status
        )
    diff_check = run_command(["git", "diff", "--check"], capture=True)
    if diff_check.returncode != 0:
        raise PushCliError(
            "git diff --check failed:\n" + (diff_check.stdout or "").strip()
        )


def declared_toolchains() -> DeclaredToolchains:
    package = json.loads((ROOT / "frontend/package.json").read_text(encoding="utf-8"))
    package_manager = package.get("packageManager", "")
    pnpm_match = re.fullmatch(r"pnpm@(.+)", package_manager)
    if not pnpm_match:
        raise PushCliError("frontend/package.json must declare packageManager as pnpm@VERSION")

    go_mod = (ROOT / "backend/go.mod").read_text(encoding="utf-8")
    go_match = GO_VERSION_RE.search(go_mod)
    if not go_match:
        raise PushCliError("backend/go.mod does not declare a Go version")

    node_minimum = re.search(
        r">=\s*(\d+)(?:\.(\d+))?", package.get("engines", {}).get("node", "")
    )
    if not node_minimum:
        raise PushCliError("unable to determine the Node.js minimum version")

    return DeclaredToolchains(
        go=go_match.group(1),
        pnpm=pnpm_match.group(1),
        node_major_minimum=int(node_minimum.group(1)),
    )


def current_go_version() -> str:
    return capture(["go", "env", "GOVERSION"], cwd=ROOT / "backend")


def current_pnpm_version() -> str:
    return capture(["pnpm", "--version"], cwd=ROOT / "frontend").splitlines()[-1]


def current_node_version() -> str:
    return capture(["node", "--version"])


def node_major(version: str) -> int:
    match = re.search(r"(\d+)", version)
    if not match:
        raise PushCliError(f"unable to parse Node.js version from {version!r}")
    return int(match.group(1))


def check_toolchains() -> None:
    declared = declared_toolchains()
    go_actual = current_go_version()
    if go_actual != f"go{declared.go}":
        raise PushCliError(f"Go {declared.go} is required; found {go_actual}")

    pnpm_actual = current_pnpm_version()
    if pnpm_actual != declared.pnpm:
        raise PushCliError(f"pnpm {declared.pnpm} is required; found {pnpm_actual}")

    node_actual = current_node_version()
    if node_major(node_actual) < declared.node_major_minimum:
        raise PushCliError(
            f"Node.js {declared.node_major_minimum}+ is required; found {node_actual}"
        )

    print(
        "Toolchains: "
        f"Go {go_actual}; pnpm {pnpm_actual}; Node.js {node_actual}"
    )


def run_step(
    name: str,
    command: Sequence[str],
    cwd: Path | None = None,
) -> None:
    print(f"\n[{name}]")
    print(f"$ {display(command)}")
    result = run_command(command, cwd=cwd)
    if result.returncode != 0:
        raise PushCliError(f"{name} failed with exit code {result.returncode}")


def run_local_checks(
    remote: str,
    branch: str,
    *,
    base_ref: str | None = None,
) -> None:
    python = sys.executable
    backend = ROOT / "backend"
    steps: list[tuple[str, Sequence[str], Path]] = [
        ("Go module tidiness", ["go", "mod", "tidy", "-diff"], backend),
        (
            "Compress CLI self-tests",
            [python, "skills/compress-cli/tests/test_compress_cli.py"],
            ROOT,
        ),
        (
            "Push CLI self-tests",
            [python, "skills/push-cli/tests/test_push_cli.py"],
            ROOT,
        ),
        (
            "Release CLI self-tests",
            [python, "skills/release-cli/tests/test_release_cli.py"],
            ROOT,
        ),
        ("Backend unit tests", ["go", "test", "-tags=unit", "./..."], backend),
        (
            "Frontend frozen install",
            ["pnpm", "--dir", "frontend", "install", "--frozen-lockfile"],
            ROOT,
        ),
        (
            "Frontend lint",
            ["pnpm", "--dir", "frontend", "run", "lint:check"],
            ROOT,
        ),
        (
            "Frontend typecheck",
            ["pnpm", "--dir", "frontend", "run", "typecheck"],
            ROOT,
        ),
        (
            "Frontend tests",
            [
                "pnpm",
                "--dir",
                "frontend",
                "run",
                "test:run",
                "--maxWorkers=4",
            ],
            ROOT,
        ),
        (
            "Release policy tests",
            [python, "tools/test_release_policy.py"],
            ROOT,
        ),
        (
            "Codex outbound identity",
            [python, "tools/check_openai_codex_identity.py"],
            ROOT,
        ),
        ("README synchronization", [python, "tools/check_readme_sync.py"], ROOT),
        ("Release metadata sources", [python, "tools/check_release.py"], ROOT),
        ("Linux installer syntax", ["bash", "-n", "deploy/install.sh"], ROOT),
        (
            "Docker Compose security",
            ["sh", "deploy/tests/docker-compose-security-test.sh"],
            ROOT,
        ),
        (
            "Docker runtime resources",
            ["sh", "deploy/tests/docker-runtime-resources-test.sh"],
            ROOT,
        ),
        (
            "Caddy cache policy",
            ["bash", "deploy/test-caddyfile-cache.sh"],
            ROOT,
        ),
    ]

    migration_base = base_ref or f"{remote}/{branch}"
    base_check = run_command(
        ["git", "rev-parse", "--verify", migration_base], capture=True
    )
    if base_check.returncode == 0:
        steps.append(
            (
                "Migration policy",
                [python, "tools/check_new_migrations.py", "--base", migration_base],
                ROOT,
            )
        )

    for name, command, cwd in steps:
        run_step(name, command, cwd)


def ensure_clean_after_checks() -> None:
    status = capture(
        ["git", "status", "--porcelain=v1", "--untracked-files=all"]
    )
    if status:
        raise PushCliError(
            "checks created or exposed worktree changes; refusing to push:\n" + status
        )


def pushed_sha() -> str:
    return capture(["git", "rev-parse", "HEAD"])


def fetch_default_branch(remote: str, default_branch: str) -> str:
    run_step(
        f"Fetch {remote}/{default_branch}",
        [
            "git",
            "fetch",
            "--no-tags",
            remote,
            f"+refs/heads/{default_branch}:refs/remotes/{remote}/{default_branch}",
        ],
    )
    return capture(["git", "rev-parse", f"{remote}/{default_branch}"])


def require_latest_base(remote: str, default_branch: str) -> ValidationProof:
    base = fetch_default_branch(remote, default_branch)
    head = pushed_sha()
    ancestor = run_command(
        ["git", "merge-base", "--is-ancestor", base, head], capture=True
    )
    if ancestor.returncode == 1:
        raise PushCliError(
            f"current branch does not contain latest {remote}/{default_branch} "
            f"({base}); update the branch before submit-pr"
        )
    if ancestor.returncode != 0:
        raise PushCliError("unable to compare the pull-request base and head")
    return ValidationProof(base=base, head=head)


def require_unchanged_proof(
    proof: ValidationProof,
    remote: str,
    default_branch: str,
) -> None:
    if pushed_sha() != proof.head:
        raise PushCliError("HEAD changed while local validation was running")
    latest_base = fetch_default_branch(remote, default_branch)
    if latest_base != proof.base:
        raise PushCliError(
            f"{remote}/{default_branch} changed from {proof.base} to {latest_base} "
            "while local validation was running; rerun submit-pr"
        )


def validation_marker(proof: ValidationProof) -> str:
    payload = json.dumps(
        {"base": proof.base, "head": proof.head},
        separators=(",", ":"),
        sort_keys=True,
    )
    return f"<!-- sub2api-submit-pr: {payload} -->"


def with_validation_marker(body: str, proof: ValidationProof) -> str:
    cleaned = VALIDATION_MARKER_RE.sub("", body).rstrip()
    marker = validation_marker(proof)
    return f"{cleaned}\n\n{marker}\n" if cleaned else f"{marker}\n"


def publish_validation_status(repository: str, proof: ValidationProof) -> None:
    run_step(
        "Publish local-validation commit status",
        [
            "gh",
            "api",
            "--method",
            "POST",
            f"repos/{repository}/statuses/{proof.head}",
            "-f",
            "state=success",
            "-f",
            f"context={LOCAL_VALIDATION_CONTEXT}",
            "-f",
            "description=Repository preflight passed",
        ],
    )


def open_pull_requests(
    repository: str,
    branch: str,
    default_branch: str,
) -> list[dict[str, object]]:
    output = capture(
        [
            "gh",
            "pr",
            "list",
            "--repo",
            repository,
            "--state",
            "open",
            "--head",
            branch,
            "--base",
            default_branch,
            "--json",
            "number,url,isDraft,headRefOid,body",
        ]
    )
    try:
        prs = json.loads(output)
    except json.JSONDecodeError as error:
        raise PushCliError("gh pr list returned invalid JSON") from error
    if not isinstance(prs, list) or any(not isinstance(pr, dict) for pr in prs):
        raise PushCliError("gh pr list returned an unexpected JSON value")
    for pr in prs:
        number = pr.get("number")
        if not isinstance(number, int):
            raise PushCliError("gh pr list returned a pull request without a number")
        base_oid = capture(
            [
                "gh",
                "api",
                f"repos/{repository}/pulls/{number}",
                "--jq",
                ".base.sha",
            ]
        )
        if not re.fullmatch(r"[0-9a-f]{40}", base_oid):
            raise PushCliError(
                f"GitHub returned an invalid base commit for pull request #{number}"
            )
        pr["baseRefOid"] = base_oid
    return prs


def default_pr_body(branch: str) -> str:
    return (
        "## Summary\n\n"
        f"Submit `{branch}` after the repository local-validation gate.\n\n"
        "## Validation\n\n"
        "- Host repository preflight passed.\n"
        "- Protected Linux GitHub Actions run the complete PR checks.\n"
    )


def create_or_update_pull_request(
    repository: str,
    branch: str,
    default_branch: str,
    proof: ValidationProof,
    *,
    title: str | None,
    body_file: Path | None,
) -> str:
    prs = open_pull_requests(repository, branch, default_branch)
    if len(prs) > 1:
        raise PushCliError(
            f"multiple open pull requests exist for {branch} -> {default_branch}"
        )
    supplied_body = (
        default_pr_body(branch)
    )
    if body_file is not None:
        try:
            supplied_body = body_file.read_text(encoding="utf-8")
        except OSError as error:
            raise PushCliError(f"cannot read pull-request body {body_file}: {error}") from error
    if prs:
        pr = prs[0]
        if pr.get("headRefOid") != proof.head:
            raise PushCliError("existing pull request head does not match validated HEAD")
        if pr.get("baseRefOid") != proof.base:
            raise PushCliError(
                "existing pull request base does not match the validated default branch"
            )
        body = with_validation_marker(str(pr.get("body") or supplied_body), proof)
        number = str(pr.get("number"))
        run_step(
            "Update pull-request validation marker",
            [
                "gh",
                "api",
                "--method",
                "PATCH",
                f"repos/{repository}/pulls/{number}",
                "-f",
                f"body={body}",
            ],
        )
        url = str(pr.get("url") or number)
        print(f"Pull request ready: {url}")
        return url

    pr_title = title or capture(["git", "log", "-1", "--format=%s"])
    body = with_validation_marker(supplied_body, proof)
    url = capture(
        [
            "gh",
            "pr",
            "create",
            "--repo",
            repository,
            "--base",
            default_branch,
            "--head",
            branch,
            "--title",
            pr_title,
            "--body",
            body,
        ]
    )
    if not url:
        raise PushCliError("gh pr create did not return a pull-request URL")
    print(f"Pull request created: {url}")
    return url


def find_actions_runs(
    repository: str,
    branch: str,
    sha: str,
) -> list[dict[str, object]]:
    for _ in range(10):
        output = capture(
            [
                "gh",
                "run",
                "list",
                "--repo",
                repository,
                "--branch",
                branch,
                "--event",
                "push",
                "--limit",
                "50",
                "--json",
                "databaseId,headSha,status,conclusion,url,workflowName,headBranch",
            ]
        )
        try:
            runs = json.loads(output)
        except json.JSONDecodeError as error:
            raise PushCliError("gh run list returned invalid JSON") from error
        matches = [
            item
            for item in runs
            if item.get("headSha") == sha and item.get("headBranch") == branch
        ]
        if matches:
            return matches
        time.sleep(3)
    raise PushCliError(
        f"no GitHub Actions push run for {branch} at {sha} appeared within 30 seconds"
    )


def watch_actions(repository: str, branch: str) -> None:
    sha = pushed_sha()
    runs = find_actions_runs(repository, branch, sha)
    for run in runs:
        run_id = str(run["databaseId"])
        print(
            f"GitHub Actions run: {run_id} "
            f"({run.get('workflowName', 'unknown')}) {run.get('url', '')}"
        )
        result = run_command(
            ["gh", "run", "watch", run_id, "--repo", repository, "--exit-status"]
        )
        if result.returncode != 0:
            raise PushCliError(
                f"GitHub Actions run failed: {run.get('url', '') or run_id}"
            )
    print(f"All {len(runs)} GitHub Actions runs passed.")


def push_branch(remote: str, branch: str) -> None:
    run_step("Configure Git transport from GitHub CLI", ["gh", "auth", "setup-git"])
    run_step(
        "Push current branch",
        ["git", "push", remote, f"HEAD:{branch}"],
    )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run strict Sub2API Plus checks and push the current branch."
    )
    parser.add_argument(
        "action",
        choices=("check", "push", "submit-pr", "watch", "ensure"),
        help="check toolchains, run the local preflight, push, submit a PR, or watch branch Actions",
    )
    parser.add_argument("--remote", default=DEFAULT_REMOTE)
    parser.add_argument("--base-ref", help=argparse.SUPPRESS)
    parser.add_argument("--title", help="pull-request title for submit-pr")
    parser.add_argument("--body-file", type=Path, help="pull-request body for submit-pr")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        repository = github_gate(args.remote)
        if args.action == "ensure":
            check_toolchains()
            print("\nLocal validation toolchains are ready. No checks were run.")
            return 0

        branch = current_branch()
        if args.action == "watch":
            watch_actions(repository, branch)
            return 0

        default_branch = repository_default_branch(repository)
        if args.action in {"push", "submit-pr"}:
            require_working_branch(branch, default_branch)
            require_no_git_operation()
            require_clean_worktree()

        if args.action == "push":
            push_branch(args.remote, branch)
            print(
                f"\nPushed {branch} without local validation. "
                "Use submit-pr for the final validated pull-request submission."
            )
            return 0

        proof = None
        base_ref = args.base_ref
        if args.action == "submit-pr":
            proof = require_latest_base(args.remote, default_branch)
            base_ref = f"{args.remote}/{default_branch}"
        else:
            require_clean_worktree()
        check_toolchains()
        run_local_checks(args.remote, branch, base_ref=base_ref)
        ensure_clean_after_checks()
        print("\nLocal repository preflight passed. No branch was pushed.")

        if args.action == "submit-pr":
            assert proof is not None
            require_unchanged_proof(proof, args.remote, default_branch)
            push_branch(args.remote, branch)
            publish_validation_status(repository, proof)
            create_or_update_pull_request(
                repository,
                branch,
                default_branch,
                proof,
                title=args.title,
                body_file=args.body_file,
            )
        return 0
    except PushCliError as error:
        print(f"push-cli stopped: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
