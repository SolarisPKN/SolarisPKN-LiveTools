# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""No-network tests for AntiCensor's module-scoped LiveTools updater."""
import json
import os
import sys
import tempfile
import unittest
from datetime import timedelta
from unittest.mock import patch
from urllib.error import HTTPError

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)
from updater import (UpdateChecker, version_triplet, validate_repo,
                     module_release_version, save_state, now_utc)


class FakeResponse:
    status = 200
    def __init__(self, data):
        self.data = json.dumps(data).encode()
    def read(self, length):
        return self.data[:length]
    def __enter__(self):
        return self
    def __exit__(self, *args):
        return None


class UpdaterTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, {"LOCALAPPDATA": self.tmp.name})
        self.env.start()
        self.repo = "SolarisPKN/SolarisPKN-LiveTools"
        self.checker = UpdateChecker()

    def tearDown(self):
        self.env.stop()
        self.tmp.cleanup()

    def release(self, tag):
        return {"tag_name": tag,
                "html_url": f"https://github.com/{self.repo}/releases/tag/{tag}",
                "draft": False, "prerelease": False}

    def test_versions(self):
        self.assertEqual(version_triplet("v0.5.1"), (0, 5, 1))
        self.assertEqual(module_release_version("AntiCensor-v0.8.1"), (0, 8, 1))
        self.assertIsNone(module_release_version("v1.0.3"))
        self.assertIsNone(module_release_version("MovilCam-v99.0.0"))
        self.assertIsNone(version_triplet("latest"))

    def test_repository_validation(self):
        self.assertEqual(validate_repo(self.repo), self.repo)
        for invalid in ("github.com/a", "../evil", "bad repo/name", "repo", "https://github.com/a/b"):
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                validate_repo(invalid)

    def test_weekly_interval(self):
        self.assertTrue(self.checker.due())
        save_state({"repo": self.repo, "checked_at": now_utc().isoformat(), "check_success": True})
        self.assertFalse(self.checker.due())
        save_state({"repo": self.repo, "checked_at": (now_utc()-timedelta(days=8)).isoformat(),
                    "check_success": True})
        self.assertTrue(self.checker.due())

    def test_select_only_latest_anticensor_version(self):
        releases = [self.release("V1.0.3"), self.release("MovilCam-v9.0.0"),
                    self.release("anticensor-v0.8.0"), self.release("anticensor-v0.7.1")]
        with patch("updater.urlopen", return_value=FakeResponse(releases)):
            result = self.checker.check(force=True)
        self.assertEqual(result.status, "available")
        self.assertEqual(result.tag, "anticensor-v0.8.0")
        self.assertFalse(self.checker.due())

    def test_no_anticensor_release_is_not_an_update(self):
        with patch("updater.urlopen", return_value=FakeResponse([self.release("V1.0.3")])):
            result = self.checker.check(force=True)
        self.assertEqual(result.status, "missing")
        self.assertFalse(self.checker.due())

    def test_invalid_release_url_is_ignored(self):
        releases = [self.release("anticensor-v200.0.0")]
        releases[0]["html_url"] = "https://example.com/fake"
        with patch("updater.urlopen", return_value=FakeResponse(releases)):
            result = self.checker.check(force=True)
        self.assertEqual(result.status, "missing")

    def test_missing_repo_nonfatal(self):
        err = HTTPError("https://api.github.com/repos/...", 404, "Not Found", {}, None)
        with patch("updater.urlopen", side_effect=err):
            self.assertEqual(self.checker.check(force=True).status, "missing")
        self.assertFalse(self.checker.due())

    def test_reset_when_repo_changes(self):
        save_state({"checked_at": now_utc().isoformat(), "check_success": True})
        self.checker.set_repository("SolarisPKN/TestRelease")
        self.assertTrue(self.checker.due())


if __name__ == "__main__":
    unittest.main()
