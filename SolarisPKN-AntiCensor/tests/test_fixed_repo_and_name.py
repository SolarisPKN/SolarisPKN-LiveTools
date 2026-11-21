# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Regression for fixed LiveTools repo and VB-CABLE name-management helpers."""
import os
import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from updater import UpdateChecker, REPO, API, load_state, save_state
from i18n import STRINGS
from version import APP_VERSION, DEFAULT_RELEASE_REPOSITORY, MODULE_PATH


class FixedSourceAndNameTests(unittest.TestCase):
    def test_release_source_cannot_be_changed_in_settings(self):
        self.assertEqual(DEFAULT_RELEASE_REPOSITORY, "SolarisPKN/SolarisPKN-LiveTools")
        self.assertEqual(MODULE_PATH, "SolarisPKN-AntiCensor")
        self.assertEqual(REPO, DEFAULT_RELEASE_REPOSITORY)
        self.assertIn("api.github.com/repos/SolarisPKN/SolarisPKN-LiveTools/", API)
        checker = UpdateChecker()
        self.assertEqual(checker.repository(), REPO)
        self.assertFalse(hasattr(checker, "set_repository"))

        with tempfile.TemporaryDirectory() as temp:
            with patch.dict(os.environ, {"LOCALAPPDATA": temp}):
                save_state({"repo": "other-owner/evil-repo", "auto_updates": True})
                self.assertNotIn("repo", load_state())
                self.assertEqual(UpdateChecker().repository(), REPO)

    def test_no_edit_repository_ui_strings_or_dialog(self):
        tray = (ROOT / "tray_app.py").read_text(encoding="utf-8")
        mic = (ROOT / "mic.py").read_text(encoding="utf-8")
        self.assertNotIn("configure_repository(", tray)
        self.assertNotIn("simpledialog", tray)
        self.assertNotIn("enqueue_repo", tray)
        self.assertNotIn("repo_selector", mic)
        self.assertNotIn("repo_entry", mic)
        for lang in ("es", "en"):
            for key in ("tray_repo", "repo_title", "repo_question", "repo_updated", "repo_invalid"):
                self.assertNotIn(key, STRINGS[lang])
            self.assertIn("SolarisPKN/SolarisPKN-LiveTools/SolarisPKN-AntiCensor", STRINGS[lang]["repo_fixed"])

    def test_signed_vendor_driver_is_never_modified_for_name(self):
        core = (ROOT / "packaging" / "Renombrar-Mic.ps1").read_text(encoding="utf-8")
        gui = (ROOT / "packaging" / "Renombrar-Mic-GUI.ps1").read_text(encoding="utf-8")
        iss = (ROOT / "packaging" / "AntiCensor.iss").read_text(encoding="utf-8")
        self.assertIn("Renombrar-Mic-GUI.ps1", core)
        self.assertIn('return "ALREADY";', core)
        self.assertIn("CABLE Output (VB-Audio Virtual Cable)", gui)
        self.assertIn("0x00F5", gui)  # BM_CLICK on verified Windows control
        self.assertIn("Get-PnpDevice -Class AudioEndpoint", gui)
        self.assertIn("FIELD_BEFORE", gui)
        self.assertIn("Renombrar-Mic-GUI.ps1", iss)
        self.assertNotIn("takeown.exe", gui.lower())
        self.assertNotIn("bcdedit", gui.lower())

    def test_current_version(self):
        self.assertEqual(APP_VERSION, "0.8.2")


if __name__ == "__main__":
    unittest.main()
