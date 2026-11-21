# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Expert view regression: never disable Windows audio endpoints."""
import json
import os
import tempfile
import tkinter as tk
import unittest
from unittest.mock import patch

from device_visibility import (cable_inventory, config_path, is_vb_cable_endpoint,
                               load_expert_mode, save_expert_mode)
from mic import SolarisMic


class ExpertVisibilityTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, {"LOCALAPPDATA": self.tmp.name})
        self.env.start()

    def tearDown(self):
        self.env.stop()
        self.tmp.cleanup()

    def test_config_default_off_and_boolean_persistence(self):
        self.assertFalse(load_expert_mode())
        save_expert_mode(True)
        self.assertTrue(load_expert_mode())
        self.assertEqual(json.loads(config_path().read_text())["expert_mode"], True)
        save_expert_mode(False)
        self.assertFalse(load_expert_mode())
        config_path().write_text('{"expert_mode": "yes"}')
        self.assertFalse(load_expert_mode())

    def test_exact_vb_cable_detection_only(self):
        names = [
            "CABLE Input (VB-Audio Virtual Cable)",
            "CABLE In 16ch (VB-Audio Virtual Cable)",
            "CABLE Output 16ch (VB-Audio Virtual Cable)",
            "CABLE Out 16ch (VB-Audio Virtual Cable)",
            "CABLE Output (VB-Audio Point)",
            "SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)",
        ]
        self.assertTrue(all(is_vb_cable_endpoint(n) for n in names))
        self.assertFalse(is_vb_cable_endpoint("USB Audio Device"))
        self.assertFalse(is_vb_cable_endpoint("CABLE Input (Some Other Brand)"))
        self.assertFalse(is_vb_cable_endpoint("SteelSeries Sonar - Microphone"))

    def test_inventory_prefers_wasapi_and_handles_absent_optional_cables(self):
        apis = [{"name": "MME"}, {"name": "Windows WASAPI"}]
        devices = [
            {"name": "CABLE Input (VB-Audio Virtual C", "hostapi": 0, "max_input_channels": 0, "max_output_channels": 2},
            {"name": "CABLE Input (VB-Audio Virtual Cable)", "hostapi": 1, "max_input_channels": 0, "max_output_channels": 2},
            {"name": "CABLE In 16ch (VB-Audio Virtual Cable)", "hostapi": 1, "max_input_channels": 0, "max_output_channels": 16},
            {"name": "SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)", "hostapi": 1, "max_input_channels": 2, "max_output_channels": 0},
            {"name": "Mic (USB Audio)", "hostapi": 1, "max_input_channels": 2, "max_output_channels": 0},
        ]
        rows = cable_inventory(devices, apis)
        self.assertEqual(len(rows), 3)
        self.assertIn(("playback", "CABLE Input (VB-Audio Virtual Cable)"), rows)
        self.assertIn(("capture", "SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)"), rows)
        self.assertNotIn("Mic (USB Audio)", repr(rows))

    def test_settings_toggle_and_no_virtual_reference(self):
        root = tk.Tk()
        root.withdraw()
        try:
            app = SolarisMic(root)
            self.assertFalse(app.expert_mode.get())
            self.assertEqual(app.expert_section.winfo_manager(), "")
            self.assertEqual(app.settings_driver_title.winfo_manager(), "")
            self.assertNotIn("VB-CABLE", app.dependency_status.cget("text"))
            self.assertNotIn("VB-CABLE", app.cable_explanation.cget("text"))
            self.assertFalse(any(is_vb_cable_endpoint(label) for _, label, _ in app.devices))
            app.expert_mode.set(True)
            app.set_expert_mode()
            self.assertTrue(load_expert_mode())
            self.assertEqual(app.expert_section.winfo_manager(), "pack")
            self.assertIn("VB-CABLE", app.T("expert_mode"))
            self.assertEqual(app.settings_driver_title.winfo_manager(), "pack")
            self.assertIsNotNone(app.expert_device_picker["values"])
            self.assertFalse(any(is_vb_cable_endpoint(label) for _, label, _ in app.devices))
            app.expert_mode.set(False)
            app.set_expert_mode()
            self.assertFalse(load_expert_mode())
            self.assertEqual(app.expert_section.winfo_manager(), "")
            self.assertEqual(app.settings_driver_title.winfo_manager(), "")
            self.assertNotIn("VB-CABLE", app.dependency_status.cget("text"))
        finally:
            root.destroy()


if __name__ == "__main__":
    unittest.main()
