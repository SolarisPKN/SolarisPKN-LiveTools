# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Iniciador de Windows con menú AntiCensor en bandeja del sistema (junto al reloj).

Clic derecho -> Configuración, Calibrar, Buscar updates, Repo, Salir.
Cierre de ventana oculta al tray; "Salir" hace el cierre real.
"""
from __future__ import annotations

import ctypes
import os
import queue
import sys
import threading
import tkinter as tk
from tkinter import messagebox, simpledialog
import webbrowser

from PIL import Image, ImageDraw
import pystray

from mic import SolarisMic
from updater import UpdateChecker, UpdateResult
from version import APP_NAME, APP_VERSION


def tray_icon_image() -> Image.Image:
    """Ícono original dibujado con Pillow, sin assets externos privativos."""
    im = Image.new("RGBA", (128, 128), (0, 0, 0, 0))
    draw = ImageDraw.Draw(im)
    draw.ellipse((3, 3, 125, 125), fill=(18, 27, 35, 255))
    draw.ellipse((18, 18, 110, 110), outline=(53, 211, 125, 255), width=7)
    draw.rounded_rectangle((53, 29, 75, 72), radius=11, fill=(237, 244, 240, 255))
    draw.arc((40, 42, 88, 92), 0, 180, fill=(237, 244, 240, 255), width=7)
    draw.line((64, 88, 64, 99), fill=(237, 244, 240, 255), width=7)
    draw.line((48, 100, 80, 100), fill=(237, 244, 240, 255), width=7)
    return im


class TrayAntiCensor:
    def __init__(self):
        self.queue = queue.Queue()
        self.window = tk.Tk()
        self.window.withdraw()
        self.app = SolarisMic(self.window)
        self.window.protocol("WM_DELETE_WINDOW", self.hide)
        self.updater = UpdateChecker()
        self.latest: UpdateResult | None = None
        self.closed = False
        self.icon = pystray.Icon(
            "SolarisPKN-AntiCensor",
            tray_icon_image(),
            APP_NAME + " v" + APP_VERSION,
            menu=pystray.Menu(
                pystray.MenuItem(lambda item: self.app.T("tray_open"), self.enqueue_open, default=True),
                pystray.MenuItem(lambda item: self.app.T("tray_cal"), self.enqueue_calibration),
                pystray.Menu.SEPARATOR,
                pystray.MenuItem(lambda item: self.app.T("tray_check"), self.enqueue_check),
                pystray.MenuItem(lambda item: self.app.T("tray_repo"), self.enqueue_repo),
                pystray.MenuItem(lambda item: self.app.T("tray_release"), self.enqueue_release,
                                 enabled=lambda _: bool(self.latest and self.latest.release_url)),
                pystray.Menu.SEPARATOR,
                pystray.MenuItem(lambda item: self.app.T("tray_exit"), self.enqueue_exit),
            ),
        )
        self.app.on_language_changed = lambda _language: self.icon.update_menu()

    def enqueue_open(self, _icon=None, _item=None):
        self.queue.put(("show", None))

    def enqueue_calibration(self, _icon=None, _item=None):
        self.queue.put(("calibration", None))

    def enqueue_check(self, _icon=None, _item=None):
        self.queue.put(("manual_check", None))

    def enqueue_repo(self, _icon=None, _item=None):
        self.queue.put(("repo", None))

    def enqueue_release(self, _icon=None, _item=None):
        self.queue.put(("open_release", None))

    def enqueue_exit(self, _icon=None, _item=None):
        self.queue.put(("exit", None))

    def hide(self):
        self.window.withdraw()

    def show(self, tab=None):
        self.window.deiconify()
        self.window.state("normal")
        if tab is not None:
            self.app.tabs.select(tab)
        self.window.lift()
        try:
            self.window.focus_force()
        except tk.TclError:
            pass

    def _notify(self, title, text):
        try:
            self.icon.notify(text, title)
        except (OSError, NotImplementedError):
            pass

    def start_check(self, manual: bool):
        def background():
            try:
                result = self.updater.check(force=manual)
            except Exception as exc:
                result = UpdateResult("offline", self.app.T("update_worker_error", error=exc))
            self.queue.put(("update_result", (result, manual)))
        threading.Thread(target=background, daemon=True).start()

    def handle_update(self, result: UpdateResult, manual: bool):
        if result.status == "available":
            self.latest = result
            self._notify(self.app.T("update_title"), result.message)
            self.icon.update_menu()
            if manual:
                if messagebox.askyesno(self.app.T("update_dialog"),
                                      self.app.T("update_question", message=result.message)):
                    self.open_release()
        elif manual:
            messagebox.showinfo(self.app.T("update_dialog"), result.message)

    def open_release(self):
        if not self.latest or not self.latest.release_url:
            return
        # URL HTTPS validada por Updater; no ejecutar archivos descargados.
        webbrowser.open(self.latest.release_url)

    def configure_repository(self):
        self.show()
        current = self.updater.repository()
        new = simpledialog.askstring(
            self.app.T("repo_title"),
            self.app.T("repo_question"),
            initialvalue=current,
            parent=self.window,
        )
        if new is None:
            return
        try:
            self.updater.set_repository(new)
            self.latest = None
            self.icon.update_menu()
            self._notify(self.app.T("repo_updated"), self.updater.repository())
            self.start_check(manual=True)
        except (OSError, ValueError) as exc:
            messagebox.showerror(self.app.T("repo_invalid"), str(exc), parent=self.window)

    def poll(self):
        if self.closed:
            return
        try:
            while True:
                cmd, data = self.queue.get_nowait()
                if cmd == "show":
                    self.show()
                elif cmd == "calibration":
                    self.show(self.app.cal_tab)
                elif cmd == "manual_check":
                    self.start_check(manual=True)
                elif cmd == "repo":
                    self.configure_repository()
                elif cmd == "open_release":
                    self.open_release()
                elif cmd == "update_result":
                    self.handle_update(*data)
                elif cmd == "exit":
                    self.quit()
                    return
        except queue.Empty:
            pass
        if not self.closed:
            self.window.after(200, self.poll)

    def periodic_check(self):
        if self.closed:
            return
        if self.updater.due():
            self.start_check(manual=False)
        self.window.after(60 * 60 * 1000, self.periodic_check)

    def quit(self):
        if self.closed:
            return
        self.closed = True
        try:
            self.icon.stop()
        except (OSError, RuntimeError):
            pass
        try:
            self.app.close()
        except tk.TclError:
            self.window.destroy()

    def run(self):
        self.window.after(100, self.poll)
        self.window.after(2000, self.periodic_check)
        # pystray Windows backend puede usar su propio message loop en un hilo.
        threading.Thread(target=self.icon.run, daemon=True).start()
        self.window.mainloop()


if __name__ == "__main__":
    if "--self-test" in sys.argv:
        # Apto para smoke test del ejecutable Windows en sesión SYSTEM:
        # no lanza Tk ni icono, ni abre audio o drivers.
        import json
        from calibration import CalibrationProfile
        p = CalibrationProfile.from_measurements(-65.0, -45.0, -28.0)
        result = {
            "status": "ok",
            "app": APP_NAME,
            "version": APP_VERSION,
            "tray_icon": tray_icon_image().size,
            "calibration_whisper": p.map_level(-45),
            "calibration_normal": p.map_level(-28),
            "calibration_shout": p.map_level(p.shout_db),
            "weekly_period_days": 7,
        }
        if "--report" in sys.argv:
            i = sys.argv.index("--report")
            if i + 1 < len(sys.argv):
                with open(sys.argv[i + 1], "w", encoding="utf-8") as f:
                    json.dump(result, f)
        else:
            print(json.dumps(result))
        raise SystemExit(0)
    TrayAntiCensor().run()
