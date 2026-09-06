#!/usr/bin/env python3
"""Tests for the fast release utility."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path
from unittest import mock

SCRIPT = Path(__file__).resolve().parents[1] / "scripts" / "fast_release.py"
SPEC = importlib.util.spec_from_file_location("fast_release_under_test", SCRIPT)
assert SPEC is not None and SPEC.loader is not None
fast_release = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = fast_release
SPEC.loader.exec_module(fast_release)


class FastReleaseTest(unittest.TestCase):
    def test_next_custom_tag(self) -> None:
        self.assertEqual(
            "v0.2.0+custom.905",
            fast_release.next_custom_tag("0.2.0", 904),
        )
        self.assertEqual(
            "v1.0.0+custom.002",
            fast_release.next_custom_tag("1.0.0", 1),
        )

    def test_next_custom_tag_overflow(self) -> None:
        with self.assertRaises(fast_release.FastReleaseError):
            fast_release.next_custom_tag("0.2.0", 999)

    def test_resolve_current_version(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir) / "VERSION"
            temp_path.write_text("0.2.0+custom.904\n", encoding="utf-8")
            with mock.patch.object(fast_release, "VERSION_FILE", temp_path):
                base, iter_num = fast_release.resolve_current_version()
                self.assertEqual(base, "0.2.0")
                self.assertEqual(iter_num, 904)


if __name__ == "__main__":
    unittest.main()
