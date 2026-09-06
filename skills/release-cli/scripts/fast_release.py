#!/usr/bin/env python3
"""Execute a fast release for Sub2API Plus fork by directly publishing to GHCR."""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Sequence

ROOT = Path(__file__).resolve().parents[3]
DEFAULT_REMOTE = "origin"
DEFAULT_REPOSITORY = os.environ.get(
    "SUB2API_EXPECTED_REPOSITORY",
    "Tiantianr/sub2api-plus",
)
TAG_RE = re.compile(r"^v(\d+\.\d+\.\d+)\+custom\.(\d{3})$")
VERSION_FILE = ROOT / "backend/cmd/server/VERSION"
UPSTREAM_FILE = ROOT / "UPSTREAM.md"
DOCKERFILE = ROOT / "Dockerfile"
BACKEND_DOCKERFILE = ROOT / "backend/Dockerfile"


class FastReleaseError(RuntimeError):
    """A fatal release error."""


def run_command(
    command: Sequence[str],
    *,
    cwd: Path | None = None,
    capture: bool = False,
) -> subprocess.CompletedProcess[str]:
    actual_cwd = ROOT if cwd is None else cwd
    return subprocess.run(
        [str(item) for item in command],
        cwd=actual_cwd,
        check=False,
        text=True,
        encoding="utf-8",
        errors="replace",
        stdout=subprocess.PIPE if capture else None,
        stderr=subprocess.STDOUT if capture else None,
    )


def capture(command: Sequence[str], *, cwd: Path | None = None) -> str:
    result = run_command(command, cwd=cwd, capture=True)
    if result.returncode != 0:
        detail = (result.stdout or "").strip()
        raise FastReleaseError(
            f"{' '.join(command)} failed with exit code {result.returncode}"
            + (f": {detail[-1000:]}" if detail else "")
        )
    return (result.stdout or "").strip()


def resolve_current_version() -> tuple[str, int]:
    content = VERSION_FILE.read_text(encoding="utf-8").strip()
    match = re.fullmatch(r"^(\d+\.\d+\.\d+)\+custom\.(\d{3})$", content)
    if not match:
        raise FastReleaseError(f"invalid version format in {VERSION_FILE}: {content}")
    return match.group(1), int(match.group(2))


def next_custom_tag(base_version: str, current_iteration: int) -> str:
    next_iteration = current_iteration + 1
    if next_iteration > 999:
        raise FastReleaseError("custom iteration exceeded 999")
    return f"v{base_version}+custom.{next_iteration:03d}"


def update_version_files(next_tag: str, current_tag: str) -> None:
    version = next_tag.lstrip("v")
    base_version, iteration_str = version.split("+custom.")

    # 1. Update VERSION file
    VERSION_FILE.write_text(f"{version}\n", encoding="utf-8")

    # 2. Update Dockerfiles
    for dockerfile in (DOCKERFILE, BACKEND_DOCKERFILE):
        content = dockerfile.read_text(encoding="utf-8")
        updated = re.sub(
            r"ARG VERSION=\d+\.\d+\.\d+\+custom\.\d{3}",
            f"ARG VERSION={version}",
            content,
        )
        dockerfile.write_text(updated, encoding="utf-8")

    # 3. Update UPSTREAM.md
    upstream_content = UPSTREAM_FILE.read_text(encoding="utf-8")
    # Mark previous planned tag as published
    prev_planned_pattern = rf"\|\s*`{re.escape(current_tag)}`\s*\|\s*`([^`]+)`\s*\|\s*`([^`]+)`\s*\|\s*planned\s*\|"
    match = re.search(prev_planned_pattern, upstream_content)
    if match:
        official_ver, official_sha = match.group(1), match.group(2)
        upstream_content = re.sub(
            prev_planned_pattern,
            f"| `{current_tag}` | `{official_ver}` | `{official_sha}` | published |",
            upstream_content,
        )
        new_entry = f"| `{next_tag}` | `{official_ver}` | `{official_sha}` | planned |"
        # Insert after the updated entry
        needle = f"| `{current_tag}` | `{official_ver}` | `{official_sha}` | published |"
        upstream_content = upstream_content.replace(needle, f"{needle}\n{new_entry}")
    else:
        # Fallback: update release section directly
        pass

    # Update Current Version block in UPSTREAM.md
    upstream_content = re.sub(
        r"Git/GitHub:\s*v\d+\.\d+\.\d+\+custom\.\d{3}",
        f"Git/GitHub: {next_tag}",
        upstream_content,
    )
    upstream_content = re.sub(
        r"Application:\s*\d+\.\d+\.\d+\+custom\.\d{3}",
        f"Application: {version}",
        upstream_content,
    )
    upstream_content = re.sub(
        r"GHCR:\s*ghcr\.io/[^:]+:v\d+\.\d+\.\d+-custom\.\d{3}",
        f"GHCR: ghcr.io/{DEFAULT_REPOSITORY.split('/')[0].lower()}/sub2api-plus:{next_tag.replace('+', '-')}",
        upstream_content,
    )
    UPSTREAM_FILE.write_text(upstream_content, encoding="utf-8")

    # 4. Synchronize release docs
    updater = ROOT / "tools/update_release_docs.py"
    if updater.exists():
        run_command([sys.executable, str(updater)], capture=True)


def trigger_fast_release_workflow(
    repository: str,
    tag: str,
    notes: str | None = None,
) -> None:
    if shutil.which("gh") is None:
        raise FastReleaseError("GitHub CLI (gh) is required to trigger release workflow")

    print(f"Triggering Fast Release workflow for {tag} on {repository}...")
    cmd = [
        "gh",
        "workflow",
        "run",
        "fast-release.yml",
        "--repo",
        repository,
        "-f",
        f"tag={tag}",
    ]
    if notes:
        cmd.extend(["-f", f"notes={notes}"])

    capture(cmd)
    print("Workflow dispatched. Waiting for run to start...")
    time.sleep(5)

    # Find the run
    runs_output = capture(
        [
            "gh",
            "run",
            "list",
            "--repo",
            repository,
            "--workflow",
            "fast-release.yml",
            "--limit",
            "5",
            "--json",
            "databaseId,status,url",
        ]
    )
    runs = json.loads(runs_output)
    if not runs:
        print("Workflow triggered successfully. Check status on GitHub Actions.")
        return

    latest_run = runs[0]
    run_id = str(latest_run["databaseId"])
    run_url = latest_run.get("url", "")
    print(f"Running Fast Release [{run_id}]: {run_url}")
    print("Watching workflow progress...")
    result = run_command(
        ["gh", "run", "watch", run_id, "--repo", repository, "--exit-status"]
    )
    if result.returncode != 0:
        raise FastReleaseError(f"Fast release workflow failed. Log: {run_url}")
    print(f"\nFast release {tag} successfully completed and published to GHCR!")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Fast release generator and publisher for Sub2API Plus fork."
    )
    parser.add_argument("--tag", help="target release tag (auto-incremented if omitted)")
    parser.add_argument("--notes", help="release notes content")
    parser.add_argument("--notes-file", type=Path, help="path to release notes file")
    parser.add_argument(
        "--repo",
        default=DEFAULT_REPOSITORY,
        help=f"target GitHub repository (default: {DEFAULT_REPOSITORY})",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="only update files and calculate versions without triggering workflow",
    )
    parser.add_argument(
        "--skip-bump",
        action="store_true",
        help="do not bump files; only trigger workflow for current or specified tag",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        base_ver, current_iter = resolve_current_version()
        current_tag = f"v{base_ver}+custom.{current_iter:03d}"

        if args.tag:
            target_tag = args.tag
            if not TAG_RE.match(target_tag):
                raise FastReleaseError(f"invalid tag format: {target_tag}")
        else:
            target_tag = next_custom_tag(base_ver, current_iter)

        print(f"Current version: {current_tag}")
        print(f"Target release:  {target_tag}")

        notes = args.notes
        if not notes and args.notes_file and args.notes_file.exists():
            notes = args.notes_file.read_text(encoding="utf-8")

        if not args.skip_bump:
            print("Updating release metadata files...")
            update_version_files(target_tag, current_tag)
            print(f"Files bumped to {target_tag}.")

        if args.dry_run:
            print("Dry-run complete. No remote actions taken.")
            return 0

        trigger_fast_release_workflow(args.repo, target_tag, notes=notes)
        return 0
    except FastReleaseError as error:
        print(f"fast-release error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
