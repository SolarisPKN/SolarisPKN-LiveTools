# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Smoke test manual: construir GUI Tkinter sin abrir micrófonos ni driver.

Ejecutar solo en Windows con escritorio interactivo o sesión apta para Tk.
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import tkinter as tk
from mic import SolarisMic

root = tk.Tk()
root.withdraw()
try:
    app = SolarisMic(root)
    root.update_idletasks()
    assert app.tabs.index("end") == 2
    assert app.calibrating is False
    assert app.active is False
    print("GUI_OK", root.title(), "tabs", app.tabs.index("end"),
          "sources", len(app.devices))
finally:
    root.destroy()
