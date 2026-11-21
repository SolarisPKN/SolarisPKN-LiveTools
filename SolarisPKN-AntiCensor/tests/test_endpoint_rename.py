# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Regression tests for automatic Windows microphone naming.

Audio hardware is not modified in these tests. A Windows runtime probe is run
separately with Renombrar-Mic.ps1 -Probe; real renaming requires VB-CABLE.
"""
import os
import sys
import unittest

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)

from tray_app import TrayAntiCensor
from vbcable_backend import find_cable_pair, is_vbcable_capture_device
from version import APP_VERSION

SCRIPT = os.path.join(ROOT, "packaging", "Renombrar-Mic.ps1")


class RenameTests(unittest.TestCase):
    def test_program_version_and_installer_hook(self):
        self.assertEqual(APP_VERSION, "0.8.1")
        with open(os.path.join(ROOT, "packaging", "AntiCensor.iss"), encoding="utf-8") as f:
            installer = f.read()
        self.assertIn('Source: "{#ProjectRoot}\\packaging\\Renombrar-Mic.ps1"', installer)
        self.assertIn('Renombrar-Mic.ps1', installer)
        self.assertIn('Source: "{#ProjectRoot}\\packaging\\Instalar-VBCable.ps1"', installer)

    def test_target_name_and_capture_only(self):
        with open(SCRIPT, encoding="utf-8") as f:
            contents = f.read()
        self.assertIn("SolarisPKN-Mic Lelele", contents)
        self.assertIn("EnumAudioEndpoints(1, 0xF", contents)
        self.assertIn('description.Equals("CABLE Output"', contents)
        self.assertIn("ContainsVendor(adapter)", contents)
        self.assertIn("matches.Count > 1", contents)
        self.assertIn("OpenPropertyStore(2", contents)
        self.assertNotIn("DeviceIoControl", contents)
        self.assertNotIn("Restart-Service", contents)

    def test_tray_will_retry_after_reboot(self):
        import inspect
        src = inspect.getsource(TrayAntiCensor)
        self.assertIn("def periodic_rename(", src)
        self.assertIn("self.window.after(60_000, self.periodic_rename)", src)
        self.assertIn('self.window.after(3500, self.periodic_rename)', src)
        self.assertIn('creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0)', src)

    def test_renamed_audio_does_not_become_reference(self):
        self.assertTrue(is_vbcable_capture_device("SolarisPKN-Mic Lelele"))
        devices = [
            {"name": "Realtek Speakers", "hostapi": 0,
             "max_input_channels": 0, "max_output_channels": 2},
            {"name": "CABLE Input (VB-Audio Virtual Cable)", "hostapi": 0,
             "max_input_channels": 0, "max_output_channels": 2},
            {"name": "SolarisPKN-Mic Lelele", "hostapi": 0,
             "max_input_channels": 2, "max_output_channels": 0},
        ]
        self.assertEqual(find_cable_pair(devices, [{"name": "Windows WASAPI"}]),
                         (1, "SolarisPKN-Mic Lelele"))


if __name__ == "__main__":
    unittest.main()
