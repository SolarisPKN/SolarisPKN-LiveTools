# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Iniciador de Windows con menú AntiCensor en bandeja del sistema (junto al reloj).

Clic derecho -> Configuración, Calibrar, Buscar updates, Ajustes, Salir. Repositorio FIJO.
Cierre de ventana oculta al tray; "Salir" hace el cierre real.
"""
from __future__ import annotations

import ctypes
import os
import queue
import subprocess
import sys
import threading
import tkinter as tk
from tkinter import messagebox
import webbrowser

from PIL import Image, ImageDraw
import pystray

from mic import SolarisMic
from vbcable_backend import find_cable_pair
from updater import UpdateChecker, UpdateResult, auto_enabled, pending_update
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
        self.rename_complete = False
        self.rename_running = False
        self.icon = pystray.Icon(
            "SolarisPKN-AntiCensor",
            tray_icon_image(),
            APP_NAME + " v" + APP_VERSION,
            menu=pystray.Menu(
                pystray.MenuItem(lambda item: self.app.T("tray_open"), self.enqueue_open, default=True),
                pystray.MenuItem(lambda item: self.app.T("tray_cal"), self.enqueue_calibration),
                pystray.Menu.SEPARATOR,
                pystray.MenuItem(lambda item: self.app.T("tray_check"), self.enqueue_check),
                pystray.MenuItem(lambda item: self.app.T("tray_settings"), self.enqueue_settings),
                pystray.MenuItem(lambda item: self.app.T("tray_release"), self.enqueue_release,
                                 enabled=lambda _: bool(self.latest and self.latest.release_url)),
                pystray.Menu.SEPARATOR,
                pystray.MenuItem(lambda item: self.app.T("tray_exit"), self.enqueue_exit),
            ),
        )
        self.app.on_language_changed = lambda _language: self.icon.update_menu()
        self.app.on_check_updates = lambda: self.start_check(manual=True)

    def enqueue_open(self, _icon=None, _item=None):
        self.queue.put(("show", None))

    def enqueue_calibration(self, _icon=None, _item=None):
        self.queue.put(("calibration", None))

    def enqueue_check(self, _icon=None, _item=None):
        self.queue.put(("manual_check", None))

    def enqueue_settings(self, _icon=None, _item=None):
        self.queue.put(("settings", None))

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

    def stage_release(self, result: UpdateResult, automatic: bool):
        self.app.update_status.configure(text=self.app.T("update_downloading"))
        def background():
            try:
                downloaded = self.updater.stage(result, automatic=automatic)
            except Exception as exc:
                downloaded = UpdateResult("download_error",
                    self.app.T("update_download_error", error=exc))
            self.queue.put(("download_result", downloaded))
        threading.Thread(target=background, daemon=True).start()

    def handle_download(self, result: UpdateResult):
        self.app.update_status.configure(text=result.message)
        if result.status == "ready":
            self._notify(self.app.T("update_title"), result.message)
        elif result.status not in ("disabled", "busy"):
            self._notify(self.app.T("update_dialog"), result.message)

    def handle_update(self, result: UpdateResult, manual: bool):
        self.app.update_status.configure(text=result.message)
        if result.status == "available":
            self.latest = result
            self.icon.update_menu()
            if manual:
                if messagebox.askyesno(self.app.T("update_dialog"),
                        self.app.T("update_manual_download", message=result.message)):
                    self.stage_release(result, automatic=False)
            elif auto_enabled():
                self.stage_release(result, automatic=True)
        elif manual:
            messagebox.showinfo(self.app.T("update_dialog"), result.message)

    def open_release(self):
        if not self.latest or not self.latest.release_url:
            return
        # URL HTTPS validada por Updater; no ejecutar archivos descargados.
        webbrowser.open(self.latest.release_url)

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
                elif cmd == "settings":
                    self.show(self.app.settings_tab)
                elif cmd == "open_release":
                    self.open_release()
                elif cmd == "update_result":
                    self.handle_update(*data)
                elif cmd == "download_result":
                    self.handle_download(data)
                elif cmd == "rename_result":
                    self.handle_rename(data)
                elif cmd == "exit":
                    self.quit()
                    return
        except queue.Empty:
            pass
        if not self.closed:
            self.window.after(200, self.poll)

    @staticmethod
    def mic_rename_script():
        """Find the GPLv3 helper in either installed Setup or source checkout."""
        base = os.path.dirname(sys.executable) if getattr(sys, "frozen", False) else os.path.dirname(__file__)
        for path in (
            os.path.join(base, "scripts", "Renombrar-Mic.ps1"),
            os.path.join(base, "packaging", "Renombrar-Mic.ps1"),
        ):
            if os.path.isfile(path):
                return path
        return None

    def start_rename(self):
        """Rename only the VB-CABLE recording endpoint; never modify driver files."""
        if self.closed or self.rename_complete or self.rename_running or self.app.active or self.app.calibrating:
            return
        script = self.mic_rename_script()
        if script is None:
            return
        try:
            pair = find_cable_pair()
        except Exception:
            return
        if pair is None:
            return  # VB-CABLE is not installed or not enumerated after reboot yet
        self.rename_running = True

        def background():
            try:
                result = subprocess.run(
                    [
                        os.path.join(os.environ.get("SystemRoot", r"C:\Windows"),
                                     "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
                        "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
                        "-File", script,
                    ],
                    capture_output=True, encoding="utf-8", errors="replace",
                    timeout=25, check=False,
                    creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
                )
                output = result.stdout.strip()
                code = output.rsplit("ANTICENSOR_ENDPOINT_NAME=", 1)[-1].splitlines()[0] if (
                    "ANTICENSOR_ENDPOINT_NAME=" in output
                ) else "ERROR"
                if result.returncode != 0:
                    code = "ERROR"
                self.queue.put(("rename_result", code))
            except (OSError, subprocess.SubprocessError):
                self.queue.put(("rename_result", "ERROR"))

        threading.Thread(target=background, daemon=True).start()

    def handle_rename(self, status: str):
        self.rename_running = False
        if status in ("RENAMED", "ALREADY", "NOT_PERMITTED"):
            # The property store can be read-only on Win10: do not hammer
            # it every minute. Manual sound settings remain available.
            self.rename_complete = True
            if status == "NOT_PERMITTED":
                self.app.settings_driver.configure(text=self.app.T("rename_not_permitted"))
            elif status == "RENAMED" and not self.app.active and not self.app.calibrating:
                self.app.refresh()

    def periodic_rename(self):
        if self.closed:
            return
        self.start_rename()
        # A first login may precede driver enumeration. Retry without UAC;
        # do not change any microphone other than VB-CABLE Basic capture.
        self.window.after(60_000, self.periodic_rename)

    def periodic_check(self):
        if self.closed:
            return
        if auto_enabled() and not pending_update() and self.updater.due():
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
        # Install only AFTER this process exits. Never interrupt a stream.
        if pending_update():
            helper = os.path.join(os.path.dirname(sys.executable) if
                getattr(sys, "frozen", False) else os.path.dirname(__file__),
                "scripts" if getattr(sys, "frozen", False) else "packaging",
                "Apply-PendingUpdate.ps1")
            if os.path.isfile(helper):
                try:
                    subprocess.Popen([
                        os.path.join(os.environ.get("SystemRoot", r"C:\Windows"),
                                     "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
                        "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
                        "-File", helper, "-ParentPid", str(os.getpid())
                    ], creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0))
                except OSError:
                    pass

    def run(self):
        self.window.after(100, self.poll)
        self.window.after(2000, self.periodic_check)
        self.window.after(3500, self.periodic_rename)
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
