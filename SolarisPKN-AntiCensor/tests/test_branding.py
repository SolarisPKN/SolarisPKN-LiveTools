# -*- coding: utf-8 -*-
"""Bilingual creator branding, user-supplied cover and About tab smoke tests."""
import unittest
from pathlib import Path
import tkinter as tk
from mic import SolarisMic
from i18n import t

ROOT=Path(__file__).resolve().parents[1]

class BrandingTests(unittest.TestCase):
    def test_cover_is_valid(self):
        cover=SolarisMic._cover_path()
        self.assertIsNotNone(cover)
        self.assertTrue(cover.is_file())
        from PIL import Image
        with Image.open(cover) as im:
            self.assertEqual(im.format, "WEBP")
            self.assertGreaterEqual(im.width, 200)

    def test_translations(self):
        self.assertEqual(t("about_developer", "es"), "Software desarrollado por SolarisPKN-Labs")
        self.assertEqual(t("about_developer", "en"), "Software developed by SolarisPKN-Labs")
        self.assertIn("Acerca", t("tab_about", "es"))
        self.assertEqual(t("tab_about", "en"), "💚 About")

    def test_project_origin_is_explained_in_both_languages(self):
        es = t("about_description", "es").lower()
        en = t("about_description", "en").lower()
        for marker in ("roblox", "terror", "sancion", "lelele"):
            self.assertIn(marker, es)
        for marker in ("roblox", "horror", "sanction", "lelele"):
            self.assertIn(marker, en)
        for filename, keywords in (
            ("README.es.md", ("puteada", "sancion", "chat de voz", "obs")),
            ("README.md", ("swear", "sanction", "voice chat", "obs")),
        ):
            content = (ROOT / filename).read_text(encoding="utf-8").lower()
            for keyword in keywords:
                self.assertIn(keyword, content)

    def test_public_docs_and_media(self):
        assert (ROOT/"docs/assets/cover.webp").exists()
        for f in ("README.md","README.es.md","docs/GUIA-DE-USO.es.md",
                  "docs/USER-GUIDE.en.md","docs/PRIVACIDAD.es.md",
                  "docs/PRIVACY.en.md","docs/FAQ.es.md","docs/FAQ.en.md",
                  "docs/RELEASE-NOTES.es.md","docs/RELEASE-NOTES.en.md"):
            t=(ROOT/f).read_text(encoding="utf-8")
            self.assertIn("SolarisPKN-Labs",t)
            self.assertIn("https://labs.solarispkn.com.ar/",t)
        for f in ("docs/assets/audio-flow.es.svg","docs/assets/audio-flow.en.svg",
                  "docs/assets/features.es.svg","docs/assets/features.en.svg",
                  "docs/assets/voice-gate-demo.gif"):
            self.assertTrue((ROOT/f).is_file(),f)

    def test_gui_about_image_keeps_reference(self):
        root=tk.Tk()
        root.withdraw()
        try:
            a=SolarisMic(root)
            self.assertIsNotNone(a._about_image)
            self.assertIn("About",t("tab_about","en"))
            self.assertEqual(a.tabs.tab(a.about_tab,"state"),"normal")
        finally:
            root.destroy()

if __name__=="__main__":
    unittest.main()
