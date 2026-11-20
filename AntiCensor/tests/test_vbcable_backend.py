# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Pruebas de enrutamiento VB-CABLE sin instalación ni acceso a audio del host."""
import os
import sys
import unittest
import numpy as np

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)

from vbcable_backend import (find_cable_pair, pcm_to_float32,
                             is_vbcable_capture_device)


class VBCableTests(unittest.TestCase):
    def setUp(self):
        self.apis = [{"name": "MME"}, {"name": "Windows WASAPI"}]
        self.devices = [
            {"name": "Speakers (Realtek)", "hostapi": 1, "max_input_channels": 0, "max_output_channels": 2},
            {"name": "Microphone (USB)", "hostapi": 1, "max_input_channels": 1, "max_output_channels": 0},
            {"name": "CABLE Input (VB-Audio Virtual Cable)", "hostapi": 0, "max_input_channels": 0, "max_output_channels": 2},
            {"name": "CABLE Output (VB-Audio Virtual Cable)", "hostapi": 0, "max_input_channels": 2, "max_output_channels": 0},
            {"name": "CABLE Input (VB-Audio Virtual Cable)", "hostapi": 1, "max_input_channels": 0, "max_output_channels": 2},
            {"name": "CABLE Output (VB-Audio Virtual Cable)", "hostapi": 1, "max_input_channels": 2, "max_output_channels": 0},
        ]

    def test_select_wasapi_playback_not_real_speakers(self):
        self.assertEqual(find_cable_pair(self.devices, self.apis),
                         (4, "CABLE Output (VB-Audio Virtual Cable)"))

    def test_no_cable_means_no_output(self):
        self.assertIsNone(find_cable_pair(self.devices[:2], self.apis))

    def test_only_playback_not_enough(self):
        data = [d for d in self.devices if "CABLE Output" not in d["name"]]
        self.assertIsNone(find_cable_pair(data, self.apis))

    def test_only_capture_not_enough(self):
        data = [d for d in self.devices if "CABLE Input" not in d["name"]]
        self.assertIsNone(find_cable_pair(data, self.apis))

    def test_never_select_cable_as_reference_mic(self):
        self.assertTrue(is_vbcable_capture_device("CABLE Output (VB-Audio Virtual Cable)"))
        self.assertFalse(is_vbcable_capture_device("Microphone (USB)"))

    def test_pcm_to_wasapi_float32(self):
        pcm = np.zeros((480, 2), dtype="<i4")
        pcm[0] = (2147483647, -2147483647)
        result = pcm_to_float32(pcm.tobytes(), 480)
        self.assertEqual(result.shape, (480, 2))
        self.assertEqual(result.dtype, np.float32)
        self.assertAlmostEqual(float(result[0, 0]), 1.0, places=6)
        self.assertAlmostEqual(float(result[0, 1]), -1.0, places=6)
        self.assertEqual(float(result[200, 0]), 0.0)

    def test_invalid_pcm_packet(self):
        with self.assertRaises(ValueError):
            pcm_to_float32(b"\x00" * 10, 480)

    def test_audio_callback_handles_variable_frames_and_mono(self):
        from mic import SolarisMic
        obj = SolarisMic.__new__(SolarisMic)
        obj.output_pending = np.empty((0, 2), dtype=np.float32)
        obj.error = ""
        raw = np.full((480, 2), 1073741823, dtype="<i4").tobytes()
        obj.render = lambda: raw
        stereo = np.zeros((600, 2), dtype=np.float32)
        obj.cable_callback(stereo, 600, None, None)
        self.assertTrue(np.all(stereo > 0.49))
        self.assertEqual(obj.output_pending.shape[0], 360)
        mono = np.zeros((100, 1), dtype=np.float32)
        obj.cable_callback(mono, 100, None, None)
        self.assertTrue(np.all(mono > 0.49))
        self.assertEqual(obj.output_pending.shape[0], 260)
        self.assertEqual(obj.error, "")


if __name__ == "__main__":
    unittest.main()
