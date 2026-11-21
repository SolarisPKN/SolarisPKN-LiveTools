# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, ROOT)
from tray_app import tray_icon_image

icon_file = os.path.join(ROOT, "packaging", "anticensor.ico")
tray_icon_image().save(icon_file, format="ICO", sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128)])
print("ICO_CREATED=" + icon_file)
