# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Pruebas automatizadas de niveles RMS; nunca abre el micrófono real."""
import math
import os
import sys
import tempfile
import unittest
from unittest.mock import patch

import numpy as np

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)

from calibration import (CalibrationProfile, sample_stage, percentile,
                         WHISPER_LEVEL, NORMAL_LEVEL, RED_START)
from mic import SolarisMic, load_settings, save_settings, settings_file


class CalibrationMathTests(unittest.TestCase):
    def setUp(self):
        self.profile = CalibrationProfile.from_measurements(
            noise_db=-64, whisper_db=-45, normal_db=-29)

    def test_whisper_normal_and_auto_shout_anchors(self):
        p = self.profile
        self.assertFalse(p.measured_shout)
        self.assertGreater(p.shout_db, p.normal_db)
        self.assertAlmostEqual(p.map_level(p.whisper_db), WHISPER_LEVEL)
        self.assertAlmostEqual(p.map_level(p.normal_db), NORMAL_LEVEL)
        self.assertAlmostEqual(p.map_level(p.shout_db), 1)
        self.assertAlmostEqual(p.map_level(p.shout_db + 10), 1)

    def test_optional_measured_shout_changes_upper_anchor(self):
        measured = CalibrationProfile.from_measurements(-64, -45, -29, -12)
        self.assertTrue(measured.measured_shout)
        self.assertEqual(measured.shout_db, -12)
        self.assertAlmostEqual(measured.map_level(-12), 1.0)
        self.assertAlmostEqual(measured.map_level(-29), 0.5)
        self.assertGreater(measured.map_level(-16), 0.85)

    def test_noise_gate_silent(self):
        for db in (-98, -64, -62, float("-inf"), float("nan")):
            with self.subTest(db=db):
                self.assertEqual(self.profile.map_level(db), 0.0)

    def test_monotonic_growth_and_bounds(self):
        values = [self.profile.map_level(v / 10) for v in range(-1000, 1)]
        self.assertTrue(all(math.isfinite(v) and 0 <= v <= 1 for v in values))
        self.assertTrue(all(values[i] <= values[i + 1] for i in range(len(values) - 1)))
        self.assertLess(self.profile.map_level(-46), 0.075)
        self.assertGreater(self.profile.map_level(-20), .50)

    def test_red_zone(self):
        self.assertEqual(self.profile.color(85), "rojo")
        self.assertEqual(self.profile.color(60), "amarillo")
        self.assertEqual(self.profile.color(50), "verde")
        self.assertAlmostEqual(RED_START, .85)

    def test_reject_bad_calibration_and_nonfinite(self):
        with self.assertRaises(ValueError):
            CalibrationProfile.from_measurements(-60, -59, -30)
        with self.assertRaises(ValueError):
            CalibrationProfile.from_measurements(-60, -40, -38)
        with self.assertRaises(ValueError):
            CalibrationProfile.from_measurements(-60, -45, -30, -29)
        with self.assertRaises(ValueError):
            CalibrationProfile.from_measurements(-60, float("nan"), -30)

    def test_outliers_do_not_break_percentile(self):
        samples = [-60] * 10 + [-45] * 240 + [-12] * 3 + [0] * 2
        self.assertAlmostEqual(sample_stage("whisper", samples), -45.0)
        self.assertAlmostEqual(percentile([-70, -50, -30], 0.5), -50)
        with self.assertRaises(ValueError):
            sample_stage("normal", [1, 2])

    def test_profile_round_trip(self):
        p = CalibrationProfile.from_dict(self.profile.to_dict())
        self.assertEqual(p, self.profile)

    def test_capture_maps_normal_to_half_without_buffering_pcm(self):
        obj = SolarisMic.__new__(SolarisMic)
        obj.profile = self.profile
        obj.calibrating = False
        obj.threshold_value = -44.0
        obj.target = 0.0
        obj.level_db = -96.0
        obj.last_input = 0
        amplitude = 10 ** (self.profile.normal_db / 20.0)
        samples = np.full((480, 1), amplitude, dtype=np.float32)
        obj.capture(samples, 480, None, None)
        self.assertAlmostEqual(obj.target, .5, places=4)
        self.assertNotIn("raw_audio", obj.__dict__)
        self.assertNotIn("voice_samples", obj.__dict__)

    def test_calibration_settings_are_only_numbers_and_per_device(self):
        with tempfile.TemporaryDirectory() as tmp:
            with patch.dict(os.environ, {"LOCALAPPDATA": tmp}):
                a = self.profile.to_dict()
                b = CalibrationProfile.from_measurements(-55, -42, -25).to_dict()
                save_settings({"last_source": "WASAPI|usb-audio|1",
                               "profiles": {"WASAPI|usb-audio|1": a, "WASAPI|realtek|2": b}})
                with open(settings_file(), encoding="utf-8") as file:
                    saved = file.read()
                self.assertNotIn("pcm", saved.lower())
                self.assertNotIn("wav", saved.lower())
                loaded = load_settings()
                self.assertEqual(len(loaded["profiles"]), 2)
                self.assertEqual(CalibrationProfile.from_dict(
                    loaded["profiles"]["WASAPI|usb-audio|1"]), self.profile)


if __name__ == "__main__":
    unittest.main()
