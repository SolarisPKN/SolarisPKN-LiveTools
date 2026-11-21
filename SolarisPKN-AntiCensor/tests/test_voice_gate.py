# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""WebRTC classic VAD: no recordings or speech-to-text in AntiCensor."""
import os
import tempfile
import tkinter as tk
import unittest
from unittest.mock import patch

import numpy as np

from voice_gate import ClassicalVoiceGate
from device_visibility import (config_path, load_voice_filter, save_voice_filter,
                               load_expert_mode, save_expert_mode)
from mic import SolarisMic


class FakeVad:
    def __init__(self, responses):
        self.responses = iter(responses)
        self.frame_sizes = []
    def is_speech(self, pcm, rate):
        self.frame_sizes.append((len(pcm), rate))
        return bool(next(self.responses, False))


class ClassicalVadTests(unittest.TestCase):
    def test_web_rtc_vad_silent_frames(self):
        gate = ClassicalVoiceGate(2)
        for _ in range(12):
            self.assertFalse(gate.feed(np.zeros((480, 1), dtype=np.float32), 48000))
        self.assertFalse(gate.active)

    def test_reject_one_click_and_require_two_voice_windows(self):
        gate = ClassicalVoiceGate(2)
        fake = FakeVad([False, False, True, False, False, False, True, True])
        gate.vad = fake
        data = np.zeros((960, 1), dtype=np.float32)
        for _ in range(6):
            self.assertFalse(gate.feed(data, 48000))
        self.assertFalse(gate.feed(data, 48000))
        self.assertTrue(gate.feed(data, 48000))
        self.assertTrue(all(item == (640, 16000) for item in fake.frame_sizes))
        for _ in range(20):
            gate.feed(data, 48000)
        self.assertFalse(gate.active)

    def test_44100_and_stereo_silent_channel(self):
        gate = ClassicalVoiceGate(3)
        gate.vad = FakeVad([True, True, True])
        # Left silent, right active; 44100 frame size is 882.
        frame = np.column_stack((np.zeros(441), np.ones(441) * .2)).astype(np.float32)
        self.assertFalse(gate.feed(frame, 44100))
        self.assertFalse(gate.feed(frame, 44100))  # first 20ms VAD frame
        self.assertFalse(gate.feed(frame, 44100))
        self.assertTrue(gate.feed(frame, 44100))   # second 20ms VAD frame
        self.assertEqual(gate.vad.frame_sizes, [(640, 16000), (640, 16000)])

    def test_no_audio_buffers_accumulate_after_complete_frames(self):
        gate = ClassicalVoiceGate(2)
        for _ in range(100):
            gate.feed(np.zeros((960, 2), dtype=np.float32), 48000)
        self.assertEqual(len(gate._pending), 0)
        self.assertFalse(hasattr(gate, "audio_path"))
        self.assertFalse(hasattr(gate, "transcript"))

    def test_settings_preserve_expert_mode_and_voice_filter(self):
        with tempfile.TemporaryDirectory() as folder:
            with patch.dict(os.environ, {"LOCALAPPDATA": folder}):
                self.assertEqual(load_voice_filter(), (True, 2))
                save_expert_mode(True)
                save_voice_filter(True, 3)
                self.assertTrue(load_expert_mode())
                self.assertEqual(load_voice_filter(), (True, 3))
                save_expert_mode(False)
                self.assertEqual(load_voice_filter(), (True, 3))
                save_voice_filter(False, 0)
                self.assertFalse(load_expert_mode())
                self.assertEqual(load_voice_filter(), (False, 0))

    def test_capture_mutes_synthetic_output_without_voice(self):
        class ClosedGate:
            def feed(self, _audio, _rate):
                return False
        obj = SolarisMic.__new__(SolarisMic)
        obj.profile = None
        obj.threshold_value = -44.0
        obj.voice_gate = ClosedGate()
        obj.capture_rate = 48000
        obj.calibrating = False
        obj.target = 0.0
        obj.level_db = -96.0
        obj.last_input = 0
        obj.capture(np.ones((480, 1), dtype=np.float32) * .08, 480, None, None)
        self.assertGreater(obj.level_db, -30.0)
        self.assertEqual(obj.target, 0.0)
        self.assertFalse(obj.voice_detected)


if __name__ == "__main__":
    unittest.main()
