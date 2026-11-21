# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Interactive Windows Tk smoke; no physical microphone capture or driver changes."""
import os
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)

from mic import SolarisMic, STAGES
from theme import BG
import tkinter as tk


class NoAudioStream:
    def __init__(self):
        self.started = False
    def start(self):
        self.started = True
    def stop(self):
        pass
    def close(self):
        pass


class InterfaceTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, {"LOCALAPPDATA": self.temp.name})
        self.env.start()
        self.root = tk.Tk()
        self.root.withdraw()
        self.app = SolarisMic(self.root)

    def tearDown(self):
        self.root.destroy()
        self.env.stop()
        self.temp.cleanup()

    def test_dark_and_only_lelele(self):
        self.assertEqual(self.root.cget("bg"), BG)
        self.assertEqual(self.app.tabs.index("end"), 4)  # Audio, Calibración, Ajustes, Acerca de
        self.assertFalse(hasattr(self.app, "mode_combo"))
        self.assertFalse(hasattr(self.app, "mode_value"))

    def test_fixed_repo_checkbox_language(self):
        self.assertEqual(self.app.auto_updates.get(), False)
        self.app.language_selector.set("English")
        self.app.change_language()
        self.assertEqual(self.app.tabs.tab(self.app.settings_tab, "text"), "Settings")
        self.assertIn("SolarisPKN", self.app.tabs.tab(self.app.settings_tab, "text") + self.app.T("repo_fixed"))
        self.app.language_selector.set("Español")
        self.app.change_language()
        self.assertEqual(self.app.tabs.tab(self.app.cal_tab, "text"), "Calibración")

    def test_five_second_calibration_wizard_waits_for_next(self):
        if not self.app.devices:
            self.skipTest("No microphone devices present for fake capture")
        self.app.in_selector.current(0)
        self.app.source_changed()
        fake = NoAudioStream()
        self.app.open_reference_stream = lambda _idx, _cb: fake
        self.app.start_calibration()
        self.assertTrue(fake.started)
        self.assertTrue(self.app.calibrating)
        self.assertTrue(self.app.cal_running)
        self.assertEqual(self.app.cal_step_name, "noise")
        self.assertEqual(STAGES["noise"][2], 5.0)
        with self.app.cal_lock:
            self.app.cal_samples = [-70.0] * 40
        self.app._finish_step()
        self.assertTrue(self.app.calibrating)
        self.assertFalse(self.app.cal_running)
        self.assertEqual(self.app.cal_step_name, "noise")
        self.app.next_stage()
        self.assertEqual(self.app.cal_step_name, "whisper")
        self.assertFalse(self.app.cal_running)
        self.app.repeat_stage()
        self.assertTrue(self.app.cal_running)
        self.app.cancel_calibration()
        self.assertFalse(self.app.calibrating)
        self.assertFalse(self.app.cal_running)


if __name__ == "__main__":
    unittest.main()
