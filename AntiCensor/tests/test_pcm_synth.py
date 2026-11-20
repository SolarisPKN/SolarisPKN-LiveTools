# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Synthesizer PCM privacy, frame size and intensity: no kernel driver needed."""
import os
import sys
import time
import unittest
import numpy as np

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)
from mic import SolarisMic, BLOCK, RATE


class SynthesizerTests(unittest.TestCase):
    @staticmethod
    def synth(mode="Lelele", volume=45):
        obj = SolarisMic.__new__(SolarisMic)
        obj.rng = np.random.default_rng(1234)
        obj.mode_value = mode
        obj.threshold_value = -44.0
        obj.response_value = 65.0
        obj.volume_value = float(volume)
        obj.target = 0.9
        obj.outlevel = 0.0
        obj.last_input = time.monotonic()
        obj.pos = 0
        obj.level_db = -25.0
        return obj

    def test_pcm_format(self):
        self.assertEqual(RATE, 48000)
        self.assertEqual(BLOCK, 480)
        self.assertEqual(BLOCK * 2 * 4, 3840)

    def test_four_synthetic_modes_never_reuse_input_audio(self):
        for mode in ("Lelele", "Robot", "Pitidos", "Ruido"):
            with self.subTest(mode=mode):
                obj = self.synth(mode)
                output = []
                for _ in range(15):
                    obj.last_input = time.monotonic()
                    raw = obj.render()
                    self.assertEqual(len(raw), 3840)
                    pcm = np.frombuffer(raw, dtype="<i4").reshape(-1, 2)
                    self.assertTrue(np.array_equal(pcm[:, 0], pcm[:, 1]))
                    self.assertLessEqual(int(np.max(np.abs(pcm.astype("int64")))),
                                         int(0.721 * 2147483647))
                    output.append(pcm[:, 0])
                self.assertTrue(any(np.any(x != 0) for x in output))

    def test_zero_volume_is_silent(self):
        obj = self.synth(volume=0)
        for _ in range(3):
            obj.last_input = time.monotonic()
            self.assertTrue(np.all(np.frombuffer(obj.render(), dtype="<i4") == 0))

    def test_reference_timeout_mutes_audio(self):
        obj = self.synth()
        obj.last_input = time.monotonic() - 2
        for _ in range(4):
            self.assertTrue(np.all(np.frombuffer(obj.render(), dtype="<i4") == 0))

    def test_reference_pcm_is_never_saved_or_routed(self):
        obj = self.synth("Robot")
        raw = (0.11 * np.sin(2 * np.pi * 900 * np.arange(BLOCK) / RATE)).astype("float32")
        obj.capture(raw.reshape(-1, 1), BLOCK, None, None)
        self.assertGreater(obj.target, 0)
        self.assertTrue(np.isfinite(obj.level_db))
        self.assertNotIn("captured_pcm", vars(obj))
        self.assertNotIn("voice_samples", vars(obj))


if __name__ == "__main__":
    unittest.main()
