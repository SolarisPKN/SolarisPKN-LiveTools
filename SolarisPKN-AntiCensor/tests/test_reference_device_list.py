# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Regression: each physical Windows microphone appears once in the selector."""
import os
import tempfile
import tkinter as tk
import unittest
from unittest.mock import patch

import mic
from device_visibility import reference_microphones

APIS = [
    {"name": "MME"},
    {"name": "Windows DirectSound"},
    {"name": "Windows WASAPI"},
    {"name": "Windows WDM-KS"},
]


def device(name, api, input_channels=1, output_channels=0):
    return {"name": name, "hostapi": api,
            "max_input_channels": input_channels,
            "max_output_channels": output_channels}


class ReferenceDeviceListTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.env = patch.dict(os.environ, {"LOCALAPPDATA": self.tmp.name})
        self.env.start()

    def tearDown(self):
        self.env.stop()
        self.tmp.cleanup()

    def test_same_mic_repeated_across_host_apis_keeps_only_wasapi(self):
        devices = [
            device("USB Mic", 0), device("USB Mic", 1),
            device("USB Mic", 2), device("USB Mic", 3),
            device("Realtek Microphone", 1), device("Realtek Microphone", 2),
            device("SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)", 2),
            device("CABLE Out 16ch (VB-Audio Virtual Cable)", 2),
            device("CABLE Input (VB-Audio Virtual Cable)", 2, 0, 2),
        ]
        actual = reference_microphones(devices, APIS)
        self.assertEqual([x[1] for x in actual], ["Realtek Microphone", "USB Mic"])
        self.assertTrue(all("WASAPI" in item[2] for item in actual))
        self.assertEqual([x[0] for x in actual], [5, 2])

    def test_fallback_without_wasapi_deduplicates_backends_only(self):
        devices = [
            device("USB Mic", 0), device("USB Mic", 1),
            device("USB Mic", 1),  # two same-name devices on the same backend
            device("Laptop Mic", 0),
        ]
        actual = reference_microphones(devices, APIS)
        self.assertEqual([x[1] for x in actual], ["Laptop Mic", "USB Mic", "USB Mic"])
        self.assertEqual([x[0] for x in actual], [3, 1, 2])

    def test_gui_has_one_row_for_usb_mic_and_preserves_choice(self):
        devices = [
            device("USB Mic", 0), device("USB Mic", 1),
            device("USB Mic", 2), device("USB Mic", 3),
            device("Realtek Microphone", 2),
            device("SolarisPKN-Mic Lelele (VB-Audio Virtual Cable)", 2),
        ]
        with patch.object(mic.sd, "query_hostapis", return_value=APIS), \
             patch.object(mic.sd, "query_devices", return_value=devices), \
             patch.object(mic.sd, "default") as defaults, \
             patch.object(mic, "find_cable_pair", return_value=(77, "SolarisPKN-Mic Lelele")):
            defaults.device = [2, 0]
            root = tk.Tk()
            root.withdraw()
            try:
                app = mic.SolarisMic(root)
                self.assertEqual(len(app.devices), 2)
                self.assertEqual([i for i, label, key in app.devices if "USB Mic" in label], [2])
                # First run never silently picks an arbitrary device.
                self.assertEqual(app.in_selector.current(), -1)
                app.in_selector.current(1)
                app.source_changed()
                self.assertEqual(app.selected_device()[0], 2)
                app.refresh()
                self.assertEqual(app.selected_device()[0], 2)
                # When a saved microphone is missing, do not fall back to
                # Realtek/USB/virtual captures and overwrite the preference.
                app.settings["last_source"] = "windows wasapi|missing gadnic|1"
                app.in_selector.set("")
                app.refresh()
                self.assertEqual(app.in_selector.current(), -1)
                self.assertEqual(app.settings["last_source"],
                                 "windows wasapi|missing gadnic|1")
            finally:
                root.destroy()


if __name__ == "__main__":
    unittest.main()
