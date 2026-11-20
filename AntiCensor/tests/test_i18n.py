# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Bilingual UI coverage and persistence; no audio driver needed."""
import os
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)
from i18n import (LANGUAGES, STRINGS, get_language, save_language,
                  t, normalize_language)


class I18nTests(unittest.TestCase):
    def test_all_translations_present(self):
        self.assertEqual(set(STRINGS["es"]), set(STRINGS["en"]))
        for language in LANGUAGES:
            for key, value in STRINGS[language].items():
                self.assertTrue(value.strip(), (language, key))
                self.assertEqual(t(key, language), value)

    def test_format_placeholders_are_identical(self):
        import string
        formatter = string.Formatter()
        for key in STRINGS["es"]:
            a = set(p for _, p, _, _ in formatter.parse(STRINGS["es"][key]) if p)
            b = set(p for _, p, _, _ in formatter.parse(STRINGS["en"][key]) if p)
            self.assertEqual(a, b, key)

    def test_language_persistence(self):
        with tempfile.TemporaryDirectory() as tmp, patch.dict(os.environ, {"LOCALAPPDATA": tmp}):
            self.assertEqual(get_language(), "es")
            save_language("en")
            self.assertEqual(get_language(), "en")
            save_language("es")
            self.assertEqual(get_language(), "es")
        self.assertEqual(normalize_language("nonsense"), "es")

    def test_key_ui_texts(self):
        self.assertIn("calibration", t("tab_calibration", "en").lower())
        self.assertIn("Calibración", t("tab_calibration", "es"))
        self.assertIn("check", t("tray_check", "en").lower())
        self.assertEqual(t("mode_Pitidos", "en"), "Beeps")
        self.assertEqual(t("mode_Ruido", "en"), "Noise")


if __name__ == "__main__":
    unittest.main()
