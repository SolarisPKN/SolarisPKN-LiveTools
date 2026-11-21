# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""No-network tests of fixed-repo updater and install-after-exit staging."""
import hashlib
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
from updater import (UpdateChecker, module_release_version, auto_enabled,
                     set_auto_enabled, load_state, save_state, now_utc, due,
                     pending_path, pending_update, UpdateResult, REPO, API,
                     release_asset_name)


class FakeResponse:
    status = 200
    def __init__(self, data, address=API):
        self.payload = json.dumps(data).encode()
        self.address = address
        self.position = 0
    def read(self, amount):
        chunk = self.payload[self.position:self.position+amount]
        self.position += len(chunk)
        return chunk
    def geturl(self):
        return self.address
    def __enter__(self):
        return self
    def __exit__(self, *args):
        pass


class BinaryResponse(FakeResponse):
    def __init__(self, content, address):
        self.payload = content
        self.position = 0
        self.address = address


class FakeOpener:
    def __init__(self, content, url):
        self.content = content
        self.url = url
    def open(self, req, timeout=None):
        return BinaryResponse(self.content, self.url)


class UpdaterTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.patcher = patch.dict(os.environ, {"LOCALAPPDATA": self.temp.name})
        self.patcher.start()
        self.updater = UpdateChecker()
        self.repo = "SolarisPKN/SolarisPKN-LiveTools"

    def tearDown(self):
        self.patcher.stop()
        self.temp.cleanup()

    def test_fixed_repository_no_edit_methods(self):
        self.assertEqual(self.updater.repository(), self.repo)
        self.assertFalse(hasattr(self.updater, "set_repository"))
        self.assertIn(self.repo, API)

    def test_module_tags(self):
        self.assertEqual(module_release_version("solarispkn-anticensor-v0.9.0"), (0, 9, 0))
        self.assertEqual(module_release_version("anticensor-v0.8.0"), (0, 8, 0))
        self.assertIsNone(module_release_version("publisher-v99.0.0"))
        self.assertIsNone(module_release_version("latest"))

    def test_auto_default_off_toggle(self):
        self.assertFalse(auto_enabled())
        set_auto_enabled(True)
        self.assertTrue(auto_enabled())
        set_auto_enabled(False)
        self.assertFalse(auto_enabled())

    def test_cadence(self):
        self.assertTrue(due())
        save_state({"checked_at": now_utc().isoformat(), "check_success": True})
        self.assertFalse(due())
        save_state({"checked_at": (now_utc()-timedelta(days=8)).isoformat(),
                    "check_success": True})
        self.assertTrue(due())

    def release(self, tag, content):
        version = module_release_version(tag)
        name = release_asset_name(version) if version else "Publisher-Setup.exe"
        return {"tag_name": tag,
                "html_url": f"https://github.com/{REPO}/releases/tag/{tag}",
                "prerelease": False, "draft": False,
                "assets": [{"name": name,
                    "browser_download_url": f"https://github.com/{REPO}/releases/download/{tag}/{name}",
                    "digest": "sha256:" + hashlib.sha256(content).hexdigest(),
                    "size": len(content)}]}

    def test_check_skips_unrelated_releases(self):
        content = b"Z" * 1_000_100
        releases = [self.release("publisher-v99.0.0", content),
                    self.release("solarispkn-anticensor-v0.9.0", content),
                    self.release("anticensor-v0.8.1", content)]
        with patch("updater.urlopen", return_value=FakeResponse(releases)):
            result = self.updater.check(force=True)
        self.assertEqual(result.status, "available")
        self.assertEqual(result.tag, "solarispkn-anticensor-v0.9.0")
        self.assertEqual(result.version, (0, 9, 0))

    def test_missing_digest_refuses_update(self):
        content = b"Z" * 1_000_100
        item = self.release("solarispkn-anticensor-v0.9.0", content)
        item["assets"][0]["digest"] = ""
        with patch("updater.urlopen", return_value=FakeResponse([item])):
            self.assertEqual(self.updater.check(force=True).status, "missing")

    def test_download_staged_no_execution(self):
        content = b"SolarisTest" * 100_000
        item = self.release("solarispkn-anticensor-v0.9.0", content)
        with patch("updater.urlopen", return_value=FakeResponse([item])):
            release = self.updater.check(force=True)
        self.assertEqual(release.status, "available")
        set_auto_enabled(True)
        with patch("updater.build_opener", return_value=FakeOpener(content, release.asset_url)):
            result = self.updater.stage(release, automatic=True)
        self.assertEqual(result.status, "ready")
        pending = pending_update()
        self.assertEqual(pending["version"], "0.9.0")
        self.assertEqual(pending["source"], "automatic")
        self.assertTrue(os.path.exists(pending["file"]))
        set_auto_enabled(False)
        self.assertEqual(pending_update(), {})
        self.assertFalse(os.path.exists(pending["file"]))

    def test_no_auto_download_if_unchecked(self):
        release = UpdateResult("available", "ok", version=(0, 9, 0),
                               asset_url="https://github.com/example",sha256="b"*64,size=1000000)
        self.assertEqual(self.updater.stage(release, automatic=True).status, "disabled")


if __name__ == "__main__":
    unittest.main()
