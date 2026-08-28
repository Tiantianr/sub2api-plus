#!/usr/bin/env python3
"""Focused tests for release-cli state and mutation boundaries."""

from __future__ import annotations

import argparse
import importlib.util
import json
import os
import re
import subprocess
import sys
import unittest
from pathlib import Path
from unittest import mock


os.environ["SUB2API_EXPECTED_REPOSITORY"] = "LuckyKuang/sub2api-plus"
os.environ["SUB2API_CUSTOM_ITERATION_MIN"] = "1"

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "release_cli.py"
SPEC = importlib.util.spec_from_file_location("release_cli_under_test", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
release_cli = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = release_cli
SPEC.loader.exec_module(release_cli)

TAG = "v1.2.3+custom.009"
BASE = "a" * 40
HEAD = "b" * 40
MERGE = "c" * 40
REPOSITORY = "LuckyKuang/sub2api-plus"
ROOT = Path(__file__).resolve().parents[3]
CURRENT_MAPPING = f"| `{TAG}` | `v1.2.3` | `{'a' * 40}` | planned |\n"


def marker(base: str = BASE, head: str = HEAD) -> str:
    payload = json.dumps({"base": base, "head": head}, separators=(",", ":"))
    return f"<!-- sub2api-submit-pr: {payload} -->"


class ReleaseArtifactValidationTests(unittest.TestCase):
    def test_personal_iteration_floor_is_enforced(self) -> None:
        with mock.patch.object(release_cli, "CUSTOM_ITERATION_MIN", 901):
            release_cli.validate_tag("v1.2.3+custom.901")
            with self.assertRaisesRegex(
                release_cli.ReleaseCliError,
                "between 901 and 999",
            ):
                release_cli.validate_tag("v1.2.3+custom.900")

    def test_verify_release_requires_all_binary_assets_and_ghcr(self) -> None:
        self.assertEqual(
            release_cli.required_release_assets(TAG),
            {
                "checksums.txt",
                "model-pricing.json",
                "model-pricing-manifest.json",
                "sub2api_1.2.3+custom.009_linux_arm64.tar.gz",
            },
        )
        assets = [
            {"name": name}
            for name in sorted(release_cli.required_release_assets(TAG))
        ]
        release = {
            "tagName": TAG,
            "isDraft": False,
            "isPrerelease": False,
            "url": "https://github.com/example/release",
            "assets": assets,
        }
        with (
            mock.patch.object(release_cli, "release_details", return_value=release),
            mock.patch.object(release_cli, "verify_public_ghcr") as verify_ghcr,
        ):
            release_cli.verify_release(REPOSITORY, TAG)
        verify_ghcr.assert_called_once_with(REPOSITORY, TAG)

    def test_verify_public_ghcr_accepts_linux_arm64_index(self) -> None:
        responses = [
            {"token": "anonymous-token"},
            {
                "manifests": [
                    {"platform": {"os": "linux", "architecture": "arm64"}},
                    {"platform": {"os": "unknown", "architecture": "unknown"}},
                ]
            },
        ]
        with mock.patch.object(
            release_cli,
            "fetch_public_registry_json",
            side_effect=responses,
        ):
            release_cli.verify_public_ghcr(REPOSITORY, TAG)

    def test_verify_public_ghcr_accepts_single_linux_arm64_manifest(self) -> None:
        responses = [
            {"token": "anonymous-token"},
            {"config": {"digest": "sha256:config"}},
            {"os": "linux", "architecture": "arm64"},
        ]
        with mock.patch.object(
            release_cli,
            "fetch_public_registry_json",
            side_effect=responses,
        ) as fetch:
            release_cli.verify_public_ghcr(REPOSITORY, TAG)
        accept = fetch.call_args_list[1].kwargs["headers"]["Accept"]
        self.assertIn("application/vnd.oci.image.manifest.v1+json", accept)
        self.assertIn("application/vnd.docker.distribution.manifest.v2+json", accept)

    def test_verify_public_ghcr_requires_linux_arm64(self) -> None:
        responses = [
            {"token": "anonymous-token"},
            {"config": {"digest": "sha256:config"}},
            {"os": "linux", "architecture": "amd64"},
        ]
        with (
            mock.patch.object(
                release_cli,
                "fetch_public_registry_json",
                side_effect=responses,
            ),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "linux/arm64"),
        ):
            release_cli.verify_public_ghcr(REPOSITORY, TAG)

    def test_expired_linux_image_artifact_stops_publication(self) -> None:
        runs = [
            {
                "databaseId": 123,
                "workflowName": "CI",
                "status": "completed",
                "conclusion": "success",
            }
        ]
        with (
            mock.patch.object(release_cli, "find_branch_runs", return_value=runs),
            mock.patch.object(
                release_cli,
                "json_capture",
                return_value={
                    "artifacts": [
                        {"name": f"linux-image-{MERGE}", "expired": True}
                    ]
                },
            ),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "missing, expired"),
        ):
            release_cli.require_linux_image_artifact(REPOSITORY, "main", MERGE)


def protected_rules(
    *,
    strict: bool = True,
    contexts: frozenset[str] = release_cli.REQUIRED_PR_STATUS_CONTEXTS,
) -> list[dict[str, object]]:
    return [
        {
            "type": "pull_request",
            "parameters": {"allowed_merge_methods": ["merge"]},
        },
        {
            "type": "required_status_checks",
            "parameters": {
                "strict_required_status_checks_policy": strict,
                "required_status_checks": [
                    {"context": context} for context in sorted(contexts)
                ],
            },
        },
    ]


def automated_environment(
    *,
    can_admins_bypass: bool = False,
    rule_types: tuple[str, ...] = ("branch_policy",),
) -> dict[str, object]:
    return {
        "name": release_cli.RELEASE_ENVIRONMENT,
        "can_admins_bypass": can_admins_bypass,
        "protection_rules": [{"type": rule_type} for rule_type in rule_types],
        "deployment_branch_policy": {
            "protected_branches": False,
            "custom_branch_policies": True,
        },
    }


def release_deployment_policies(
    *, name: str = release_cli.RELEASE_TAG_POLICY, kind: str = "tag"
) -> dict[str, object]:
    return {
        "total_count": 1,
        "branch_policies": [{"name": name, "type": kind}],
    }


def tag_ruleset_summary() -> dict[str, object]:
    return {
        "id": 42,
        "target": "tag",
        "enforcement": "active",
        "source_type": "Repository",
    }


def tag_ruleset(
    *,
    rules: tuple[str, ...] = ("deletion", "update"),
    bypass_actors: list[dict[str, object]] | None = None,
) -> dict[str, object]:
    return {
        "id": 42,
        "name": "Protect immutable custom release tags",
        "target": "tag",
        "enforcement": "active",
        "conditions": {
            "ref_name": {
                "include": [release_cli.RELEASE_TAG_RULESET_REF],
                "exclude": [],
            }
        },
        "rules": [{"type": rule_type} for rule_type in rules],
        "bypass_actors": [] if bypass_actors is None else bypass_actors,
    }


def pull_request(
    *,
    state: str = "OPEN",
    base: str = BASE,
    head: str = HEAD,
    merge: str | None = None,
    auto_merge: bool = False,
) -> object:
    return release_cli.PullRequest(
        number=17,
        state=state,
        is_draft=False,
        base_branch="main",
        base_oid=base,
        head_branch="release/candidate",
        head_oid=head,
        head_owner="LuckyKuang",
        merge_state="CLEAN",
        merge_commit=merge,
        auto_merge_enabled=auto_merge,
        body=marker(base, head),
        url="https://github.com/LuckyKuang/sub2api-plus/pull/17",
    )


class ValidationProofTest(unittest.TestCase):
    def test_marker_requires_exact_base_and_head(self) -> None:
        proof = release_cli.parse_validation_proof(marker())
        self.assertEqual(release_cli.ValidationProof(BASE, HEAD), proof)

    def test_duplicate_marker_is_rejected(self) -> None:
        with self.assertRaisesRegex(release_cli.ReleaseCliError, "exactly one"):
            release_cli.parse_validation_proof(marker() + "\n" + marker())

    def test_changed_pr_head_is_rejected(self) -> None:
        pr = pull_request(head="d" * 40)
        pr = release_cli.PullRequest(**{**pr.__dict__, "body": marker(BASE, HEAD)})
        with self.assertRaisesRegex(release_cli.ReleaseCliError, "head changed"):
            release_cli.require_promotable_pr(REPOSITORY, pr, "main")


class RepositoryPolicyTest(unittest.TestCase):
    def test_required_contexts_are_emitted_by_pull_request_workflows(self) -> None:
        workflow_job_re = re.compile(r"^  ([a-zA-Z0-9_-]+):\n", re.MULTILINE)
        workflow_jobs: set[str] = set()
        for name in ("backend-ci.yml", "security-scan.yml"):
            workflow = ROOT.joinpath(".github", "workflows", name).read_text(
                encoding="utf-8"
            )
            workflow_jobs.update(workflow_job_re.findall(workflow))

        expected = release_cli.REQUIRED_PR_STATUS_CONTEXTS - {
            "sub2api/local-validation"
        }
        self.assertEqual(expected - workflow_jobs, set())

    def test_auto_merge_must_be_enabled(self) -> None:
        with mock.patch.object(
            release_cli,
            "repository_settings",
            return_value={"allow_auto_merge": False},
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "disabled"):
                release_cli.require_protected_auto_merge(REPOSITORY, "main")

    def test_merge_commit_mode_must_be_enabled(self) -> None:
        with mock.patch.object(
            release_cli,
            "repository_settings",
            return_value={"allow_auto_merge": True, "allow_merge_commit": False},
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "merge-commit"):
                release_cli.require_protected_auto_merge(REPOSITORY, "main")

    def test_required_rules_must_be_present(self) -> None:
        with (
            mock.patch.object(
                release_cli,
                "repository_settings",
                return_value={"allow_auto_merge": True, "allow_merge_commit": True},
            ),
            mock.patch.object(
                release_cli,
                "json_capture",
                return_value=[{"type": "pull_request"}],
            ),
        ):
            with self.assertRaisesRegex(
                release_cli.ReleaseCliError, "required_status_checks"
            ):
                release_cli.require_protected_auto_merge(REPOSITORY, "main")

    def test_branch_must_require_current_head_and_complete_matrix(self) -> None:
        incomplete = release_cli.REQUIRED_PR_STATUS_CONTEXTS - {"backend-security"}
        with (
            mock.patch.object(
                release_cli,
                "repository_settings",
                return_value={"allow_auto_merge": True, "allow_merge_commit": True},
            ),
            mock.patch.object(
                release_cli,
                "json_capture",
                return_value=protected_rules(strict=False, contexts=incomplete),
            ),
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "current"):
                release_cli.require_protected_auto_merge(REPOSITORY, "main")

        with (
            mock.patch.object(
                release_cli,
                "repository_settings",
                return_value={"allow_auto_merge": True, "allow_merge_commit": True},
            ),
            mock.patch.object(
                release_cli,
                "json_capture",
                return_value=protected_rules(contexts=incomplete),
            ),
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "backend-security"):
                release_cli.require_protected_auto_merge(REPOSITORY, "main")

    def test_complete_protected_policy_is_accepted(self) -> None:
        with (
            mock.patch.object(
                release_cli,
                "repository_settings",
                return_value={"allow_auto_merge": True, "allow_merge_commit": True},
            ),
            mock.patch.object(
                release_cli,
                "json_capture",
                return_value=protected_rules(),
            ),
        ):
            release_cli.require_protected_auto_merge(REPOSITORY, "main")


class ReleasePublicationPolicyTest(unittest.TestCase):
    def test_automatic_environment_and_immutable_tag_ruleset_are_accepted(
        self,
    ) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            side_effect=[
                automated_environment(),
                release_deployment_policies(),
                [tag_ruleset_summary()],
                tag_ruleset(),
            ],
        ):
            release_cli.require_automated_release_policy(REPOSITORY)

    def test_environment_reviewers_are_rejected(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            return_value=automated_environment(
                rule_types=("required_reviewers", "branch_policy")
            ),
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "not automatic"):
                release_cli.require_automated_release_environment(REPOSITORY)

    def test_environment_admin_bypass_is_rejected(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            return_value=automated_environment(can_admins_bypass=True),
        ):
            with self.assertRaisesRegex(
                release_cli.ReleaseCliError, "administrator bypass"
            ):
                release_cli.require_automated_release_environment(REPOSITORY)

    def test_environment_requires_exact_custom_tag_policy(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            side_effect=[
                automated_environment(),
                release_deployment_policies(name="v*"),
            ],
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "exactly"):
                release_cli.require_automated_release_environment(REPOSITORY)

    def test_tag_ruleset_must_block_update_and_deletion_without_bypass(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            side_effect=[
                [tag_ruleset_summary()],
                tag_ruleset(
                    rules=("deletion",),
                    bypass_actors=[{"actor_type": "RepositoryRole"}],
                ),
            ],
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "no-bypass"):
                release_cli.require_immutable_release_tag_ruleset(REPOSITORY)

    def test_tag_ruleset_must_allow_initial_creation(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            side_effect=[
                [tag_ruleset_summary()],
                tag_ruleset(rules=("creation", "deletion", "update")),
            ],
        ):
            with self.assertRaisesRegex(
                release_cli.ReleaseCliError, "initial release-tag creation"
            ):
                release_cli.require_immutable_release_tag_ruleset(REPOSITORY)


class RemoteTagTest(unittest.TestCase):
    def test_annotated_remote_tag_resolves_exact_target_and_subject(self) -> None:
        tag_object = "d" * 40
        with mock.patch.object(
            release_cli,
            "json_capture",
            side_effect=[
                [
                    {
                        "ref": f"refs/tags/{TAG}",
                        "object": {"type": "tag", "sha": tag_object},
                    }
                ],
                {
                    "tag": TAG,
                    "message": f"Sub2API Plus {TAG}\n\nRelease notes",
                    "object": {"type": "commit", "sha": MERGE},
                },
            ],
        ):
            details = release_cli.require_published_remote_tag(REPOSITORY, TAG)

        self.assertEqual(
            release_cli.RemoteTag(True, tag_object, MERGE, f"Sub2API Plus {TAG}"),
            details,
        )

    def test_lightweight_remote_tag_is_rejected(self) -> None:
        with mock.patch.object(
            release_cli,
            "json_capture",
            return_value=[
                {
                    "ref": f"refs/tags/{TAG}",
                    "object": {"type": "commit", "sha": MERGE},
                }
            ],
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "not annotated"):
                release_cli.require_published_remote_tag(REPOSITORY, TAG)


class PromotionTest(unittest.TestCase):
    def test_promote_uses_native_auto_merge_and_waits_for_merge_sha(self) -> None:
        candidate = pull_request()
        merged = pull_request(state="MERGED", merge=MERGE, auto_merge=True)
        proof = release_cli.ValidationProof(BASE, HEAD)
        completed = subprocess.CompletedProcess([], 0, "")
        with (
            mock.patch.object(release_cli, "require_clean_worktree"),
            mock.patch.object(release_cli, "repository_default_branch", return_value="main"),
            mock.patch.object(release_cli, "require_protected_auto_merge"),
            mock.patch.object(
                release_cli,
                "pull_request_details",
                side_effect=[candidate, candidate, merged],
            ),
            mock.patch.object(release_cli, "require_promotable_pr", return_value=proof),
            mock.patch.object(release_cli, "current_head", return_value=HEAD),
            mock.patch.object(release_cli, "fetch_default_branch", return_value=BASE),
            mock.patch.object(release_cli, "setup_git_transport"),
            mock.patch.object(release_cli, "remote_tag_exists", return_value=False),
            mock.patch.object(
                release_cli, "verify_deferred_finalizations"
            ) as verify_deferred,
            mock.patch.object(release_cli, "run_metadata_preflight"),
            mock.patch.object(release_cli, "require_required_pr_checks"),
            mock.patch.object(release_cli, "run_step") as run_step,
            mock.patch.object(release_cli, "run_command", return_value=completed),
            mock.patch.object(release_cli, "watch_branch_runs") as watch,
        ):
            result = release_cli.promote_pull_request(
                REPOSITORY, 17, TAG, Path("release-notes.md"), "origin"
            )

        self.assertEqual(MERGE, result)
        merge_commands = [call.args[1] for call in run_step.call_args_list]
        self.assertIn(
            [
                "gh",
                "pr",
                "merge",
                "17",
                "--repo",
                REPOSITORY,
                "--auto",
                "--merge",
                "--match-head-commit",
                HEAD,
            ],
            merge_commands,
        )
        self.assertFalse(any("--admin" in command for command in merge_commands))
        verify_deferred.assert_called_once_with(REPOSITORY, proof, TAG)
        watch.assert_called_once_with(REPOSITORY, "main", MERGE)

    def test_finalize_promotion_requires_published_tag_and_no_notes(self) -> None:
        final_branch = release_cli.finalization_branch(TAG)
        candidate = release_cli.PullRequest(
            **{**pull_request().__dict__, "head_branch": final_branch}
        )
        proof = release_cli.ValidationProof(BASE, HEAD)
        with (
            mock.patch.object(release_cli, "require_clean_worktree"),
            mock.patch.object(release_cli, "repository_default_branch", return_value="main"),
            mock.patch.object(release_cli, "require_protected_auto_merge"),
            mock.patch.object(release_cli, "pull_request_details", return_value=candidate),
            mock.patch.object(release_cli, "require_promotable_pr", return_value=proof),
            mock.patch.object(release_cli, "current_head", return_value=HEAD),
            mock.patch.object(release_cli, "fetch_default_branch", return_value=BASE),
            mock.patch.object(release_cli, "setup_git_transport"),
            mock.patch.object(
                release_cli,
                "require_published_remote_tag",
                side_effect=release_cli.ReleaseCliError(
                    "release-finalization promotion requires the published remote tag"
                ),
            ),
        ):
            with self.assertRaisesRegex(release_cli.ReleaseCliError, "published remote tag"):
                release_cli.promote_pull_request(REPOSITORY, 17, TAG, None, "origin")


class MainFlowTest(unittest.TestCase):
    def args(self, action: str) -> argparse.Namespace:
        return argparse.Namespace(
            action=action,
            tag=TAG,
            pr=None,
            notes_file=None,
            remote="origin",
            repo_root=Path("/repo"),
        )

    def test_publish_only_pushes_tag(self) -> None:
        args = self.args("publish")
        completed = subprocess.CompletedProcess([], 0, "")
        with (
            mock.patch.object(release_cli, "parse_args", return_value=args),
            mock.patch.object(release_cli, "github_gate", return_value=REPOSITORY),
            mock.patch.object(release_cli, "require_clean_worktree"),
            mock.patch.object(release_cli, "require_publishable_local_tag", return_value=MERGE),
            mock.patch.object(release_cli, "repository_default_branch", return_value="main"),
            mock.patch.object(release_cli, "fetch_default_branch", return_value=MERGE),
            mock.patch.object(release_cli, "run_command", return_value=completed),
            mock.patch.object(release_cli, "watch_branch_runs") as watch_main,
            mock.patch.object(
                release_cli, "require_linux_image_artifact"
            ) as require_image,
            mock.patch.object(release_cli, "remote_tag_exists", return_value=False),
            mock.patch.object(release_cli, "release_details", return_value=None),
            mock.patch.object(
                release_cli, "require_automated_release_policy"
            ) as release_policy,
            mock.patch.object(release_cli, "setup_git_transport"),
            mock.patch.object(release_cli, "run_step") as run_step,
            mock.patch.object(release_cli, "watch_release") as watch,
            mock.patch.object(release_cli, "verify_release") as verify,
        ):
            self.assertEqual(0, release_cli.main())

        run_step.assert_called_once_with(
            "Push exact release tag", ["git", "push", "origin", TAG]
        )
        release_policy.assert_called_once_with(REPOSITORY)
        watch_main.assert_called_once_with(REPOSITORY, "main", MERGE)
        require_image.assert_called_once_with(REPOSITORY, "main", MERGE)
        watch.assert_not_called()
        verify.assert_not_called()

    def test_verify_does_not_watch_workflow(self) -> None:
        args = self.args("verify")
        details = release_cli.RemoteTag(
            True, "d" * 40, MERGE, f"Sub2API Plus {TAG}"
        )
        with (
            mock.patch.object(release_cli, "parse_args", return_value=args),
            mock.patch.object(release_cli, "github_gate", return_value=REPOSITORY),
            mock.patch.object(
                release_cli, "require_published_remote_tag", return_value=details
            ),
            mock.patch.object(release_cli, "require_release_workflow_success") as success,
            mock.patch.object(release_cli, "verify_release") as verify,
            mock.patch.object(release_cli, "watch_release") as watch,
        ):
            self.assertEqual(0, release_cli.main())

        success.assert_called_once_with(REPOSITORY, TAG, MERGE)
        verify.assert_called_once_with(REPOSITORY, TAG)
        watch.assert_not_called()


class ReleaseMonitoringTest(unittest.TestCase):
    def test_monitor_watches_automatic_publication_through_success(self) -> None:
        run = release_cli.WorkflowRun(
            database_id=123,
            url="https://github.com/LuckyKuang/sub2api-plus/actions/runs/123",
            status="in_progress",
            conclusion=None,
        )
        watched = subprocess.CompletedProcess([], 0, "")
        with (
            mock.patch.object(release_cli, "find_release_run", return_value=run),
            mock.patch.object(
                release_cli,
                "workflow_state",
                side_effect=[
                    {
                        "status": "in_progress",
                        "conclusion": None,
                        "jobs": [],
                        "url": run.url,
                    },
                    {
                        "status": "completed",
                        "conclusion": "success",
                        "jobs": [
                            {
                                "name": "Build release assets",
                                "status": "completed",
                            }
                        ],
                        "url": run.url,
                    },
                ],
            ),
            mock.patch.object(
                release_cli, "run_command", return_value=watched
            ) as run_command,
        ):
            release_cli.watch_release(REPOSITORY, TAG, MERGE)

        run_command.assert_called_once_with(
            [
                "gh",
                "run",
                "watch",
                "123",
                "--repo",
                REPOSITORY,
                "--exit-status",
            ],
            timeout=release_cli.WATCH_SECONDS,
        )

    def test_waiting_environment_is_policy_drift(self) -> None:
        run = release_cli.WorkflowRun(
            database_id=123,
            url="https://github.com/LuckyKuang/sub2api-plus/actions/runs/123",
            status="waiting",
            conclusion=None,
        )
        with (
            mock.patch.object(release_cli, "find_release_run", return_value=run),
            mock.patch.object(
                release_cli,
                "workflow_state",
                return_value={
                    "status": "waiting",
                    "conclusion": None,
                    "jobs": [
                        {"name": "Publish Linux image", "status": "waiting"}
                    ],
                    "url": run.url,
                },
            ),
        ):
            with self.assertRaisesRegex(
                release_cli.ReleaseCliError, "policy drifted"
            ):
                release_cli.watch_release(REPOSITORY, TAG, MERGE)


class FinalizationTest(unittest.TestCase):
    def test_branch_name_is_deterministic_and_oci_safe(self) -> None:
        self.assertEqual(
            "release/finalize-1.2.3-custom.009",
            release_cli.finalization_branch(TAG),
        )

    def test_finalization_validation_is_mapping_only(self) -> None:
        self.assertEqual(
            [
                sys.executable,
                "tools/check_release.py",
                "--tag",
                TAG,
                "--require-status",
                "published",
                "--mapping-only",
            ],
            release_cli.finalization_metadata_command(TAG),
        )

    def test_delayed_finalization_allows_only_synchronized_release_docs(self) -> None:
        self.assertEqual(
            {
                "UPSTREAM.md",
                "README.md",
                "README_CN.md",
                "README_JA.md",
                "deploy/README.md",
            },
            release_cli.FINALIZATION_ALLOWED_PATHS,
        )

    def test_deferred_transition_is_detected_from_exact_pr_proof(self) -> None:
        before = f"| `v1.2.3+custom.008` | `v1.2.3` | `{'a' * 40}` | planned |\n"
        after = before.replace(
            "`v1.2.3+custom.008` | `v1.2.3` | "
            f"`{'a' * 40}` | planned",
            "`v1.2.3+custom.008` | `v1.2.3` | "
            f"`{'a' * 40}` | published",
        ) + CURRENT_MAPPING
        with mock.patch.object(release_cli, "capture", side_effect=[before, after]):
            self.assertEqual(
                ["v1.2.3+custom.008"],
                release_cli.deferred_published_transitions(
                    release_cli.ValidationProof(BASE, HEAD), TAG
                ),
            )

    def test_deferred_transition_rejects_unresolved_historical_status(self) -> None:
        previous = "v1.2.3+custom.008"
        before = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | planned |\n"
        after = before.replace("planned", "historical") + CURRENT_MAPPING
        with (
            mock.patch.object(release_cli, "capture", side_effect=[before, after]),
            self.assertRaisesRegex(release_cli.ReleaseCliError, re.escape(previous)),
        ):
            release_cli.deferred_published_transitions(
                release_cli.ValidationProof(BASE, HEAD), TAG
            )

    def test_failed_deferred_transition_accepts_terminal_status(self) -> None:
        previous = "v1.2.3+custom.008"
        before = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | planned |\n"
        for status in ("invalid", "withdrawn"):
            with (
                self.subTest(status=status),
                mock.patch.object(
                    release_cli,
                    "capture",
                    side_effect=[
                        before,
                        before.replace("planned", status) + CURRENT_MAPPING,
                    ],
                ),
            ):
                self.assertEqual(
                    [],
                    release_cli.deferred_published_transitions(
                        release_cli.ValidationProof(BASE, HEAD), TAG
                    ),
                )

    def test_terminal_release_cannot_be_restored_to_published(self) -> None:
        previous = "v1.2.3+custom.008"
        before = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | invalid |\n"
        after = before.replace("invalid", "published") + CURRENT_MAPPING
        with (
            mock.patch.object(release_cli, "capture", side_effect=[before, after]),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "invalid release mapping"),
        ):
            release_cli.deferred_published_transitions(
                release_cli.ValidationProof(BASE, HEAD), TAG
            )

    def test_release_pr_rejects_new_published_mapping(self) -> None:
        previous = "v1.2.3+custom.008"
        added = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | published |\n"
        with (
            mock.patch.object(
                release_cli,
                "capture",
                side_effect=[CURRENT_MAPPING, added + CURRENT_MAPPING],
            ),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "may add only"),
        ):
            release_cli.deferred_published_transitions(
                release_cli.ValidationProof(BASE, HEAD), TAG
            )

    def test_existing_published_mapping_cannot_be_removed(self) -> None:
        previous = "v1.2.3+custom.008"
        published = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | published |\n"
        with (
            mock.patch.object(
                release_cli,
                "capture",
                side_effect=[published, CURRENT_MAPPING],
            ),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "cannot be removed"),
        ):
            release_cli.deferred_published_transitions(
                release_cli.ValidationProof(BASE, HEAD), TAG
            )

    def test_deferred_transition_rejects_upstream_ancestry_change(self) -> None:
        previous = "v1.2.3+custom.008"
        before = f"| `{previous}` | `v1.2.3` | `{'a' * 40}` | planned |\n"
        after = (
            before.replace(f"`{'a' * 40}` | planned", f"`{'b' * 40}` | published")
            + CURRENT_MAPPING
        )
        with (
            mock.patch.object(release_cli, "capture", side_effect=[before, after]),
            self.assertRaisesRegex(release_cli.ReleaseCliError, "ancestry"),
        ):
            release_cli.deferred_published_transitions(
                release_cli.ValidationProof(BASE, HEAD), TAG
            )

    def test_current_planned_release_does_not_require_finalization(self) -> None:
        with mock.patch.object(
            release_cli, "capture", side_effect=[CURRENT_MAPPING, CURRENT_MAPPING]
        ):
            self.assertEqual(
                [],
                release_cli.deferred_published_transitions(
                    release_cli.ValidationProof(BASE, HEAD), TAG
                ),
            )

    def test_deferred_transition_requires_published_artifacts(self) -> None:
        proof = release_cli.ValidationProof(BASE, HEAD)
        previous = "v1.2.3+custom.008"
        remote = release_cli.RemoteTag(True, "d" * 40, MERGE, f"Sub2API Plus {previous}")
        with (
            mock.patch.object(
                release_cli,
                "deferred_published_transitions",
                return_value=[previous],
            ),
            mock.patch.object(
                release_cli,
                "require_published_remote_tag",
                return_value=remote,
            ) as require_tag,
            mock.patch.object(release_cli, "require_release_workflow_success") as workflow,
            mock.patch.object(release_cli, "verify_release") as verify,
        ):
            release_cli.verify_deferred_finalizations(REPOSITORY, proof, TAG)

        require_tag.assert_called_once_with(REPOSITORY, previous)
        workflow.assert_called_once_with(REPOSITORY, previous, MERGE)
        verify.assert_called_once_with(REPOSITORY, previous)


if __name__ == "__main__":
    unittest.main()
