from __future__ import annotations

import argparse
import importlib.util
import json
import subprocess
import sys
import unittest
from pathlib import Path
from unittest import mock


SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "push_cli.py"
SPEC = importlib.util.spec_from_file_location("push_cli_under_test", SCRIPT)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f"unable to load {SCRIPT}")
push_cli = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = push_cli
SPEC.loader.exec_module(push_cli)


class DeclaredToolchainsTest(unittest.TestCase):
    def test_reads_repository_pins(self) -> None:
        declared = push_cli.declared_toolchains()
        self.assertRegex(declared.go, r"^\d+\.\d+\.\d+$")
        self.assertRegex(declared.pnpm, r"^\d+\.\d+\.\d+$")
        self.assertGreaterEqual(declared.node_major_minimum, 20)

    def test_toolchain_probes_use_module_directories(self) -> None:
        with (
            mock.patch.object(push_cli, "ROOT", Path("/repo")),
            mock.patch.object(push_cli, "capture", return_value="version") as capture,
        ):
            push_cli.current_go_version()
            push_cli.current_pnpm_version()

        self.assertEqual(
            mock.call(["go", "env", "GOVERSION"], cwd=Path("/repo/backend")),
            capture.call_args_list[0],
        )
        self.assertEqual(
            mock.call(["pnpm", "--version"], cwd=Path("/repo/frontend")),
            capture.call_args_list[1],
        )

    def test_pnpm_probe_ignores_leading_warning(self) -> None:
        with mock.patch.object(
            push_cli,
            "capture",
            return_value="warning from pnpm\n9.15.9",
        ):
            self.assertEqual("9.15.9", push_cli.current_pnpm_version())


class LocalChecksTest(unittest.TestCase):
    def test_preflight_runs_expected_checks(self) -> None:
        git_miss = subprocess.CompletedProcess(["git"], 1, "")
        with (
            mock.patch.object(push_cli, "ROOT", Path("/repo")),
            mock.patch.object(push_cli, "run_command", return_value=git_miss),
            mock.patch.object(push_cli, "run_step") as run_step,
        ):
            push_cli.run_local_checks("origin", "feature")

        names = [call.args[0] for call in run_step.call_args_list]
        self.assertEqual("Go module tidiness", names[0])
        for expected in (
            "Compress CLI self-tests",
            "Push CLI self-tests",
            "Release CLI self-tests",
            "Backend unit tests",
            "Frontend frozen install",
            "Frontend lint",
            "Frontend typecheck",
            "Frontend tests",
            "Release policy tests",
            "Codex outbound identity",
            "README synchronization",
            "Release metadata sources",
            "Linux installer syntax",
            "Docker Compose security",
            "Docker runtime resources",
            "Caddy cache policy",
        ):
            self.assertIn(expected, names)
        self.assertNotIn("Backend integration tests", names)
        self.assertNotIn("Backend lint", names)
        self.assertNotIn("Frontend production build", names)

        frontend_test = next(
            call for call in run_step.call_args_list if call.args[0] == "Frontend tests"
        )
        self.assertEqual(
            [
                "pnpm",
                "--dir",
                "frontend",
                "run",
                "test:run",
                "--maxWorkers=4",
            ],
            frontend_test.args[1],
        )

    def test_preflight_checks_migrations_against_exact_base(self) -> None:
        git_hit = subprocess.CompletedProcess(["git"], 0, "")
        with (
            mock.patch.object(push_cli, "ROOT", Path("/repo")),
            mock.patch.object(push_cli, "run_command", return_value=git_hit),
            mock.patch.object(push_cli, "run_step") as run_step,
        ):
            push_cli.run_local_checks(
                "origin",
                "feature",
                base_ref="origin/main",
            )

        migration = next(
            call
            for call in run_step.call_args_list
            if call.args[0] == "Migration policy"
        )
        self.assertEqual(
            [
                sys.executable,
                "tools/check_new_migrations.py",
                "--base",
                "origin/main",
            ],
            migration.args[1],
        )


class BranchAndProofTest(unittest.TestCase):
    def test_default_branch_is_never_pushable(self) -> None:
        with self.assertRaisesRegex(push_cli.PushCliError, "default branch"):
            push_cli.require_working_branch("main", "main")

    def test_validation_marker_replaces_stale_marker(self) -> None:
        old = push_cli.ValidationProof("a" * 40, "b" * 40)
        new = push_cli.ValidationProof("c" * 40, "d" * 40)
        body = push_cli.with_validation_marker("Summary", old)
        updated = push_cli.with_validation_marker(body, new)
        self.assertEqual(1, len(push_cli.VALIDATION_MARKER_RE.findall(updated)))
        self.assertIn('"base":"' + "c" * 40 + '"', updated)
        self.assertNotIn('"base":"' + "a" * 40 + '"', updated)

    def test_latest_base_rejects_stale_branch(self) -> None:
        stale = subprocess.CompletedProcess([], 1, "")
        with (
            mock.patch.object(
                push_cli,
                "fetch_default_branch",
                return_value="a" * 40,
            ),
            mock.patch.object(push_cli, "pushed_sha", return_value="b" * 40),
            mock.patch.object(push_cli, "run_command", return_value=stale),
        ):
            with self.assertRaisesRegex(push_cli.PushCliError, "does not contain"):
                push_cli.require_latest_base("origin", "main")

    def test_status_describes_repository_preflight(self) -> None:
        proof = push_cli.ValidationProof("a" * 40, "b" * 40)
        with mock.patch.object(push_cli, "run_step") as run_step:
            push_cli.publish_validation_status("owner/repo", proof)

        command = run_step.call_args.args[1]
        self.assertIn("context=sub2api/local-validation", command)
        self.assertIn("description=Repository preflight passed", command)


class PullRequestQueryTest(unittest.TestCase):
    def test_hydrates_base_oid_without_pr_list_base_ref_oid(self) -> None:
        base_oid = "a" * 40
        listing = json.dumps(
            [
                {
                    "number": 22,
                    "url": "https://github.com/LuckyKuang/sub2api-plus/pull/22",
                    "isDraft": False,
                    "headRefOid": "b" * 40,
                    "body": "Summary",
                }
            ]
        )
        with mock.patch.object(
            push_cli,
            "capture",
            side_effect=[listing, base_oid],
        ) as capture:
            prs = push_cli.open_pull_requests(
                "LuckyKuang/sub2api-plus",
                "feature",
                "main",
            )

        self.assertEqual(base_oid, prs[0]["baseRefOid"])
        list_command = capture.call_args_list[0].args[0]
        self.assertIn("number,url,isDraft,headRefOid,body", list_command)
        self.assertNotIn("baseRefOid", list_command)


class PullRequestUpdateTest(unittest.TestCase):
    def test_updates_validation_marker_through_rest_api(self) -> None:
        repository = "LuckyKuang/sub2api-plus"
        proof = push_cli.ValidationProof("a" * 40, "b" * 40)
        stale_proof = push_cli.ValidationProof("c" * 40, "d" * 40)
        existing_body = push_cli.with_validation_marker("Summary", stale_proof)
        pull_request = {
            "number": 22,
            "url": "https://github.com/LuckyKuang/sub2api-plus/pull/22",
            "headRefOid": proof.head,
            "baseRefOid": proof.base,
            "body": existing_body,
        }
        with (
            mock.patch.object(
                push_cli,
                "open_pull_requests",
                return_value=[pull_request],
            ),
            mock.patch.object(push_cli, "run_step") as run_step,
        ):
            url = push_cli.create_or_update_pull_request(
                repository,
                "feature",
                "main",
                proof,
                title=None,
                body_file=None,
            )

        self.assertEqual(pull_request["url"], url)
        command = run_step.call_args.args[1]
        expected_body = push_cli.with_validation_marker(existing_body, proof)
        self.assertEqual(
            [
                "gh",
                "api",
                "--method",
                "PATCH",
                "repos/LuckyKuang/sub2api-plus/pulls/22",
                "-f",
                f"body={expected_body}",
            ],
            command,
        )


class MainFlowTest(unittest.TestCase):
    @staticmethod
    def args(action: str) -> argparse.Namespace:
        return argparse.Namespace(
            action=action,
            remote="origin",
            base_ref=None,
            title=None,
            body_file=None,
        )

    def test_check_rejects_dirty_worktree_before_preflight(self) -> None:
        args = self.args("check")
        with (
            mock.patch.object(push_cli, "parse_args", return_value=args),
            mock.patch.object(
                push_cli,
                "github_gate",
                return_value="LuckyKuang/sub2api-plus",
            ),
            mock.patch.object(push_cli, "current_branch", return_value="feature"),
            mock.patch.object(
                push_cli,
                "repository_default_branch",
                return_value="main",
            ),
            mock.patch.object(
                push_cli,
                "require_clean_worktree",
                side_effect=push_cli.PushCliError("worktree is not clean"),
            ),
            mock.patch.object(push_cli, "run_local_checks") as preflight,
        ):
            self.assertEqual(1, push_cli.main())

        preflight.assert_not_called()

    def test_ensure_checks_toolchains_only(self) -> None:
        args = self.args("ensure")
        with (
            mock.patch.object(push_cli, "parse_args", return_value=args),
            mock.patch.object(
                push_cli,
                "github_gate",
                return_value="LuckyKuang/sub2api-plus",
            ),
            mock.patch.object(push_cli, "check_toolchains") as check,
            mock.patch.object(push_cli, "current_branch") as current_branch,
            mock.patch.object(push_cli, "run_local_checks") as preflight,
        ):
            self.assertEqual(0, push_cli.main())

        check.assert_called_once_with()
        current_branch.assert_not_called()
        preflight.assert_not_called()

    def test_check_runs_host_preflight(self) -> None:
        args = self.args("check")
        with (
            mock.patch.object(push_cli, "parse_args", return_value=args),
            mock.patch.object(
                push_cli,
                "github_gate",
                return_value="LuckyKuang/sub2api-plus",
            ),
            mock.patch.object(push_cli, "current_branch", return_value="feature"),
            mock.patch.object(
                push_cli,
                "repository_default_branch",
                return_value="main",
            ),
            mock.patch.object(push_cli, "require_clean_worktree"),
            mock.patch.object(push_cli, "check_toolchains") as check,
            mock.patch.object(push_cli, "run_local_checks") as preflight,
            mock.patch.object(push_cli, "ensure_clean_after_checks"),
        ):
            self.assertEqual(0, push_cli.main())

        check.assert_called_once_with()
        preflight.assert_called_once_with("origin", "feature", base_ref=None)

    def test_fast_push_never_runs_preflight(self) -> None:
        args = self.args("push")
        with (
            mock.patch.object(push_cli, "parse_args", return_value=args),
            mock.patch.object(
                push_cli,
                "github_gate",
                return_value="LuckyKuang/sub2api-plus",
            ),
            mock.patch.object(push_cli, "current_branch", return_value="feature"),
            mock.patch.object(
                push_cli,
                "repository_default_branch",
                return_value="main",
            ),
            mock.patch.object(push_cli, "require_working_branch"),
            mock.patch.object(push_cli, "require_no_git_operation"),
            mock.patch.object(push_cli, "require_clean_worktree"),
            mock.patch.object(push_cli, "push_branch") as push,
            mock.patch.object(push_cli, "check_toolchains") as check,
            mock.patch.object(push_cli, "run_local_checks") as preflight,
        ):
            self.assertEqual(0, push_cli.main())

        push.assert_called_once_with("origin", "feature")
        check.assert_not_called()
        preflight.assert_not_called()

    def test_submit_pr_runs_preflight_before_mutation(self) -> None:
        args = self.args("submit-pr")
        proof = push_cli.ValidationProof("a" * 40, "b" * 40)
        order: list[str] = []

        def record(name: str):
            def _inner(*_args: object, **_kwargs: object) -> None:
                order.append(name)

            return _inner

        with (
            mock.patch.object(push_cli, "parse_args", return_value=args),
            mock.patch.object(
                push_cli,
                "github_gate",
                return_value="LuckyKuang/sub2api-plus",
            ),
            mock.patch.object(push_cli, "current_branch", return_value="feature"),
            mock.patch.object(
                push_cli,
                "repository_default_branch",
                return_value="main",
            ),
            mock.patch.object(push_cli, "require_working_branch"),
            mock.patch.object(push_cli, "require_no_git_operation"),
            mock.patch.object(push_cli, "require_clean_worktree"),
            mock.patch.object(
                push_cli,
                "require_latest_base",
                return_value=proof,
            ),
            mock.patch.object(
                push_cli,
                "check_toolchains",
                side_effect=record("toolchains"),
            ),
            mock.patch.object(
                push_cli,
                "run_local_checks",
                side_effect=record("preflight"),
            ),
            mock.patch.object(push_cli, "ensure_clean_after_checks"),
            mock.patch.object(
                push_cli,
                "require_unchanged_proof",
                side_effect=record("recheck"),
            ),
            mock.patch.object(
                push_cli,
                "push_branch",
                side_effect=record("push"),
            ),
            mock.patch.object(
                push_cli,
                "publish_validation_status",
                side_effect=record("status"),
            ),
            mock.patch.object(
                push_cli,
                "create_or_update_pull_request",
                side_effect=record("pr"),
            ),
        ):
            self.assertEqual(0, push_cli.main())

        self.assertEqual(
            ["toolchains", "preflight", "recheck", "push", "status", "pr"],
            order,
        )


if __name__ == "__main__":
    unittest.main()
