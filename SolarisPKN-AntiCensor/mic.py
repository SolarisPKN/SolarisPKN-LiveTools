# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
# Copyright (c) 2026 SolarisPKN contributors
"""SolarisPKN AntiCensor: mic virtual SolarisPKN-Mic Lelele.

Solo el nivel RMS del micrófono de referencia controla un sintetizador.
El audio sintético se entrega a VB-CABLE mediante PortAudio/WASAPI. La voz
original no se reenvía. VB-CABLE es dependencia externa propietaria donationware;
el código de esta aplicación es GPLv3.
"""
from __future__ import annotations

import json
import math
import os
import threading
import time
import webbrowser
import tkinter as tk
from tkinter import ttk, messagebox
import subprocess
import sys
from theme import apply_dark, BG
from version import APP_VERSION
from updater import auto_enabled, set_auto_enabled

import numpy as np
import sounddevice as sd

from calibration import CalibrationProfile, sample_stage
from i18n import LANGUAGES, get_language, save_language, t
from vbcable_backend import (find_cable_pair, is_vbcable_capture_device,
                             pcm_to_float32, make_output_stream, VBCABLE_URL)

RATE, BLOCK = 48000, 480
VIRTUAL_NAME = "SolarisPKN-Mic Lelele"
PRODUCT_NAME = "SolarisPKN AntiCensor"

# Tiempo de preparación y medición por etapa. El usuario ve un contador.
STAGES = {
    "noise": ("stage_noise", 0.0, 5.0),
    "whisper": ("stage_whisper", 0.0, 5.0),
    "normal": ("stage_normal", 0.0, 5.0),
    "shout": ("stage_shout", 0.0, 5.0),
}


def settings_file() -> str:
    base = os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")
    return os.path.join(base, "SolarisPKN", "AntiCensor", "calibration.json")


def load_settings() -> dict:
    try:
        with open(settings_file(), "r", encoding="utf-8") as file:
            data = json.load(file)
        if not isinstance(data, dict):
            return {"profiles": {}}
        # Un JSON editado o corrupto no debe impedir abrir el micrófono.
        raw = data.get("profiles", {})
        if not isinstance(raw, dict):
            raw = {}
        profiles = {}
        for name, values in raw.items():
            if not isinstance(name, str) or not isinstance(values, dict):
                continue
            try:
                profiles[name] = CalibrationProfile.from_dict(values).to_dict()
            except (ValueError, TypeError, KeyError):
                continue
        return {"profiles": profiles, "last_source": str(data.get("last_source", ""))}
    except (OSError, ValueError, TypeError):
        return {"profiles": {}}


def save_settings(settings: dict) -> None:
    """Escribir solo valores dBFS; nunca guardamos samples ni grabaciones."""
    dest = settings_file()
    os.makedirs(os.path.dirname(dest), exist_ok=True)
    temp = dest + ".tmp"
    try:
        with open(temp, "w", encoding="utf-8") as file:
            json.dump(settings, file, indent=2, ensure_ascii=False, allow_nan=False)
        os.replace(temp, dest)
    finally:
        if os.path.exists(temp):
            try:
                os.remove(temp)
            except OSError:
                pass


class SolarisMic:
    def __init__(self, root):
        self.root = root
        apply_dark(root)
        self.language = get_language()
        self._localized_widgets = []
        self.on_language_changed = None
        root.title(self.T("app_title", version=APP_VERSION))
        root.geometry("830x730")
        root.minsize(710, 640)

        self.active = False
        self.stream = None
        self.output_stream = None
        self.output_pending = np.empty((0, 2), dtype=np.float32)
        self.cable_mic_name = None
        self.level_db = -96.0
        self.target = 0.0
        self.outlevel = 0.0
        self.last_input = 0.0
        self.pos = 0
        self.error = ""
        self.devices = []  # (PortAudio index, displayed name, stable-ish name+API key)
        self.rng = np.random.default_rng()

        self.settings = load_settings()
        self.profile: CalibrationProfile | None = None
        self.calibrating = False
        self.cal_stream = None
        self.cal_lock = threading.Lock()
        self.cal_steps: list[str] = []
        self.cal_step_idx = 0
        self.cal_step_name = ""
        self.cal_samples: list[float] = []  # SOLO mediciones dBFS
        self.cal_step_ready = 0.0
        self.cal_step_end = 0.0
        self.cal_results: dict[str, float] = {}
        self.cal_running = False
        self.on_check_updates = None
        self.auto_updates = tk.BooleanVar(value=auto_enabled())

        self.threshold = tk.DoubleVar(value=-44)
        self.volume = tk.DoubleVar(value=100)
        self.response = tk.DoubleVar(value=65)
        self.include_shout = tk.BooleanVar(value=False)
        self.threshold_value = -44.0
        self.volume_value = 100.0
        self.response_value = 65.0
        # El hilo PortAudio jamás invoca métodos Tk: valores cacheados simples.
        self.threshold.trace_add("write", lambda *_: setattr(self, "threshold_value", float(self.threshold.get())))
        self.volume.trace_add("write", lambda *_: setattr(self, "volume_value", float(self.volume.get())))
        self.response.trace_add("write", lambda *_: setattr(self, "response_value", float(self.response.get())))

        tabs = ttk.Notebook(root)
        tabs.pack(fill="both", expand=True, padx=10, pady=10)
        main = ttk.Frame(tabs, padding=15)
        calibration = ttk.Frame(tabs, padding=15)
        settings = ttk.Frame(tabs, padding=15)
        tabs.add(main, text=self.T("tab_audio"))
        tabs.add(calibration, text=self.T("tab_calibration"))
        tabs.add(settings, text=self.T("tab_settings"))
        self.tabs = tabs
        self.main_tab = main
        self.cal_tab = calibration
        self.settings_tab = settings

        header = ttk.Frame(main)
        header.pack(fill="x")
        self._label_key(header, "product_identity", font=("Segoe UI", 15, "bold")).pack(side="left", anchor="w")
        lang_frame = ttk.Frame(main)
        lang_frame.pack(fill="x", pady=(2, 12))
        self._label_key(lang_frame, "language").pack(side="left", padx=(0, 8))
        self.language_selector = ttk.Combobox(lang_frame, state="readonly", width=12,
                                               values=list(LANGUAGES.values()))
        self.language_selector.pack(side="left")
        self.language_selector.set(LANGUAGES[self.language])
        self.language_selector.bind("<<ComboboxSelected>>", self.change_language)

        self._label_key(main, "cable_explanation", wraplength=680).pack(anchor="w", pady=(0, 7))
        self.dependency_status = self._label_key(main, "dependency_checking")
        self.dependency_status.pack(anchor="w")
        self.btn_vendor = self._button_key(main, "vendor_install", self.open_vbcable_website)
        self.btn_vendor.pack(anchor="w", pady=(3, 10))
        self.btn_clean = self._button_key(main, "driver_clean", self.clean_driver)
        # Only one of these two buttons may be visible.
        self.btn_clean.pack_forget()
        self._label_key(main, "reference_mic").pack(anchor="w")
        self.in_selector = ttk.Combobox(main, state="readonly", width=79)
        self.in_selector.pack(fill="x", pady=(4, 6))
        self.in_selector.bind("<<ComboboxSelected>>", self.source_changed)
        self.btn_refresh = self._button_key(main, "refresh_devices", self.refresh)
        self.btn_refresh.pack(anchor="w", pady=(0, 8))
        self._label_key(main, "only_lelele", font=("Segoe UI Semibold", 12)).pack(anchor="w", pady=(4, 5))
        self.manual_row = ttk.Frame(main)
        self.manual_row.pack(fill="x")
        self.slider(self.manual_row, "manual_threshold", self.threshold, -70, -15)
        self.slider(main, "volume", self.volume, 0, 100)
        self.slider(main, "response", self.response, 0, 100)
        self._label_key(main, "intensity").pack(anchor="w", pady=(8, 3))
        self.meter = tk.Canvas(main, height=26, bg=BG, highlightthickness=0)
        self.meter.pack(fill="x")
        self.meter_segments = []
        for i in range(50):
            color = self._meter_color(i, False)
            x = (i * 13) + 1
            self.meter_segments.append(self.meter.create_rectangle(
                x, 2, x + 11, 24, fill=color, outline=""
            ))
        self._label_key(main, "anchors").pack(anchor="w", pady=(4, 3))
        self.readout = self._label_key(main, "meter_idle")
        self.readout.pack(anchor="w", pady=(2, 10))
        action_row = ttk.Frame(main)
        action_row.pack(fill="x")
        self.btn_on = self._button_key(action_row, "start", self.start)
        self.btn_on.pack(side="left", expand=True, fill="x")
        self.btn_off = self._button_key(action_row, "stop", self.stop, state="disabled")
        self.btn_off.pack(side="left", expand=True, fill="x", padx=(7, 0))
        self.status = self._label_key(main, "initial_status", wraplength=680)
        self.status.pack(anchor="w", pady=(10, 4))
        self._label_key(main, "obs_note", wraplength=680).pack(anchor="w")

        self._label_key(calibration, "cal_title", font=("Segoe UI", 16, "bold")).pack(anchor="w")
        self._label_key(calibration, "cal_explanation", wraplength=660).pack(anchor="w", pady=(5, 15))
        self.cal_source = self._label_key(calibration, "device_label", wraplength=660)
        self.cal_source.pack(anchor="w", pady=(0, 12))
        self.chk_shout = ttk.Checkbutton(calibration, variable=self.include_shout,
                                         text=self.T("shout_checkbox"))
        self._localized_widgets.append((self.chk_shout, "shout_checkbox"))
        self.chk_shout.pack(anchor="w")
        self._label_key(calibration, "shout_explanation", wraplength=660).pack(anchor="w", pady=(4, 14))
        self.btn_cal = self._button_key(calibration, "cal_start", self.start_calibration)
        self.btn_cal.pack(fill="x", pady=(1, 5))
        nav = ttk.Frame(calibration)
        nav.pack(fill="x", pady=(0, 8))
        self.btn_cal_prev = self._button_key(nav, "cal_previous", self.previous_stage, state="disabled")
        self.btn_cal_prev.pack(side="left", fill="x", expand=True)
        self.btn_cal_next = self._button_key(nav, "cal_next", self.next_stage, state="disabled")
        self.btn_cal_next.pack(side="left", fill="x", expand=True, padx=(8, 0))
        self.btn_cal_cancel = self._button_key(calibration, "cal_cancel",
                                               self.cancel_calibration, state="disabled")
        self.btn_cal_cancel.pack(fill="x", pady=(0, 15))
        self.cal_stage_label = self._label_key(calibration, "cal_ready",
                                                 font=("Segoe UI", 11, "bold"), wraplength=660)
        self.cal_stage_label.pack(anchor="w", pady=(5, 6))
        self.cal_progress = ttk.Progressbar(calibration, maximum=100)
        self.cal_progress.pack(fill="x")
        self.cal_status = self._label_key(calibration, "cal_privacy", wraplength=660)
        self.cal_status.pack(anchor="w", pady=(8, 16))
        self._label_key(calibration, "cal_profile", font=("Segoe UI", 10, "bold")).pack(anchor="w")
        self.cal_summary = self._label_key(calibration, "cal_not_saved", wraplength=660)
        self.cal_summary.pack(anchor="w", pady=(6, 15))
        self.btn_clear_cal = self._button_key(calibration, "cal_clear", self.clear_calibration)
        self.btn_clear_cal.pack(anchor="w", pady=(0, 3))

        # Updates and dependencies have their own Settings tab.
        self._label_key(settings, "settings_header", font=("Segoe UI Semibold", 16)).pack(anchor="w", pady=(3, 14))
        self._label_key(settings, "repo_fixed", wraplength=680).pack(anchor="w", pady=(2, 15))
        self.auto_box = ttk.Checkbutton(settings, variable=self.auto_updates,
                                         text=self.T("auto_updates"), command=self.set_auto_updates)
        self._localized_widgets.append((self.auto_box, "auto_updates"))
        self.auto_box.pack(anchor="w", pady=5)
        self._label_key(settings, "auto_description", wraplength=680).pack(anchor="w", pady=(0, 14))
        self.btn_check_updates = self._button_key(settings, "update_check_button", self.check_updates_now)
        self.btn_check_updates.pack(anchor="w", pady=(0, 9))
        self.update_status = self._label_key(settings, "update_idle", wraplength=680)
        self.update_status.pack(anchor="w", pady=(3, 12))
        self._label_key(settings, "driver_section", font=("Segoe UI Semibold", 12)).pack(anchor="w", pady=(12, 5))
        self.settings_driver = self._label_key(settings, "dependency_checking", wraplength=680)
        self.settings_driver.pack(anchor="w")

        self.apply_language()
        self.refresh()
        root.after(100, self.tick)
        root.protocol("WM_DELETE_WINDOW", self.close)

    def T(self, key, **values):
        return t(key, self.language, **values)

    def _label_key(self, parent, key, **options):
        widget = ttk.Label(parent, text=self.T(key), **options)
        self._localized_widgets.append((widget, key))
        return widget

    def _button_key(self, parent, key, command, **options):
        widget = ttk.Button(parent, text=self.T(key), command=command, **options)
        self._localized_widgets.append((widget, key))
        return widget

    def set_auto_updates(self):
        set_auto_enabled(self.auto_updates.get())
        self.update_status.configure(text=self.T("auto_on" if self.auto_updates.get() else "auto_off"))

    def check_updates_now(self):
        if self.on_check_updates is not None:
            self.on_check_updates()
        else:
            self.update_status.configure(text=self.T("update_manual_requires_tray"))

    def clean_driver(self):
        if self.active or self.calibrating:
            messagebox.showwarning(self.T("driver_clean"), self.T("driver_clean_in_use"))
            return
        if not messagebox.askyesno(self.T("driver_clean"), self.T("driver_clean_warning")):
            return
        base = os.path.dirname(sys.executable) if getattr(sys, "frozen", False) else os.path.dirname(__file__)
        helper = os.path.join(base, "scripts", "Instalar-VBCable.ps1")
        if not os.path.isfile(helper):
            helper = os.path.join(base, "packaging", "Instalar-VBCable.ps1")
        if not os.path.isfile(helper):
            messagebox.showerror(self.T("driver_clean"), self.T("driver_clean_missing"))
            return
        try:
            subprocess.Popen(
                [os.path.join(os.environ.get("SystemRoot", r"C:\Windows"),
                              "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
                 "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", helper,
                 "-Clean", "-Language", self.language],
                creationflags=getattr(subprocess, "CREATE_NO_WINDOW", 0),
            )
        except OSError as exc:
            messagebox.showerror(self.T("driver_clean"), str(exc))

    def change_language(self, _event=None):
        chosen = self.language_selector.get()
        lang = next((code for code, name in LANGUAGES.items() if name == chosen), self.language)
        if lang == self.language:
            return
        self.language = lang
        try:
            save_language(lang)
        except OSError:
            pass
        self.apply_language()
        callback = self.on_language_changed
        if callback is not None:
            callback(lang)

    def apply_language(self):
        self.root.title(self.T("app_title", version=APP_VERSION))
        self.tabs.tab(self.main_tab, text=self.T("tab_audio"))
        self.tabs.tab(self.cal_tab, text=self.T("tab_calibration"))
        self.tabs.tab(self.settings_tab, text=self.T("tab_settings"))
        for widget, key in self._localized_widgets:
            widget.configure(text=self.T(key))
        self.language_selector.set(LANGUAGES[self.language])
        self._show_dependency(bool(getattr(self, "cable_mic_name", None)))
        try:
            _, device, _ = self.selected_device()
            self.cal_source.configure(text=self.T("selected_source", device=device))
        except (RuntimeError, AttributeError):
            pass
        self.update_profile_info()
        if self.calibrating:
            self.btn_cal.configure(text=self.T(
                "cal_repeat" if self.cal_step_name in self.cal_results else "cal_begin"))
            self.btn_cal_next.configure(text=self.T(
                "cal_finish" if self.cal_step_idx == len(self.cal_steps) - 1 else "cal_next"))
        if self.active:
            self.status.configure(text=self.T("active_status",
                                 device=self.cable_mic_name or "CABLE Output"))
        else:
            self.status.configure(text=self.T("initial_status"))

    @staticmethod
    def _meter_color(index: int, active: bool) -> str:
        pct = index * 2
        if pct >= 85:
            return "#ee424c" if active else "#53252a"
        if pct >= 60:
            return "#f2c245" if active else "#574b29"
        return "#42cb72" if active else "#244b33"

    def _draw_meter(self, percent: float):
        active = max(0, min(50, int(math.ceil(percent / 2))))
        for i, item in enumerate(self.meter_segments):
            self.meter.itemconfigure(item, fill=self._meter_color(i, i < active))

    def slider(self, parent, title, var, low, high):
        self._label_key(parent, title).pack(anchor="w", pady=(7, 1))
        row = ttk.Frame(parent)
        row.pack(fill="x")
        ttk.Scale(row, from_=low, to=high, variable=var).pack(side="left", fill="x", expand=True)
        label = ttk.Label(row, width=5)
        label.pack(side="right")
        var.trace_add("write", lambda *_: label.configure(text=str(round(var.get()))))
        label.configure(text=str(round(var.get())))

    @staticmethod
    def open_vbcable_website():
        """No instala software; abre la web original de VB-Audio."""
        webbrowser.open(VBCABLE_URL)

    @staticmethod
    def source_key(name, host_api, channels):
        # Identidad legible persistente que sigue siendo estable si PortAudio cambia índice.
        return f"{host_api.casefold()}|{name.casefold()}|{int(channels)}"

    def refresh(self):
        if self.active or self.calibrating:
            return
        previous = self.selected_source_key() or self.settings.get("last_source", "")
        try:
            apis = sd.query_hostapis()
            candidates = []
            for index, info in enumerate(sd.query_devices()):
                name = info["name"]
                if not info["max_input_channels"]:
                    continue
                if is_vbcable_capture_device(name) or any(term in name.casefold() for term in
                       ("solaris anti-cheat", "solarispkn-mic lelele", "solaris virtual mic")):
                    continue
                api_name = apis[info["hostapi"]]["name"]
                key = self.source_key(name, api_name, info["max_input_channels"])
                candidates.append((index, f"{name} [{api_name}]", key))
            # WASAPI primero y después otros backends Windows.
            self.devices = sorted(candidates, key=lambda x: ("WASAPI" not in x[1].upper(), x[1]))
            self.in_selector["values"] = [d[1] for d in self.devices]
            selected = next((i for i, d in enumerate(self.devices) if d[2] == previous), None)
            if selected is None:
                selected = next((i for i, d in enumerate(self.devices)
                                 if d[0] == sd.default.device[0]), None)
            if selected is None:
                selected = next((i for i, d in enumerate(self.devices)
                                 if "USB Audio" in d[1] or "Realtek" in d[1]), 0)
            if self.devices:
                self.in_selector.current(selected)
            self.source_changed()
            pair = find_cable_pair()
            self.cable_mic_name = pair[1] if pair else None
            self._show_dependency(bool(pair))
        except Exception as exc:
            self.status.configure(text=self.T("device_enumeration_failed", error=exc))

    def _show_dependency(self, present: bool):
        key = "dependency_found" if present else "dependency_missing"
        text = self.T(key, name=self.cable_mic_name) if present else self.T(key)
        self.dependency_status.configure(text=text)
        self.settings_driver.configure(text=text)
        if present:
            self.btn_vendor.pack_forget()
            if not self.btn_clean.winfo_manager():
                self.btn_clean.pack(anchor="w", pady=(3, 10))
        else:
            self.btn_clean.pack_forget()
            if not self.btn_vendor.winfo_manager():
                self.btn_vendor.pack(anchor="w", pady=(3, 10))

    def selected_source_key(self):
        if hasattr(self, "in_selector"):
            idx = self.in_selector.current()
            if 0 <= idx < len(self.devices):
                return self.devices[idx][2]
        return None

    def selected_device(self):
        idx = self.in_selector.current()
        if 0 <= idx < len(self.devices):
            return self.devices[idx]
        raise RuntimeError(self.T("no_input"))

    def source_changed(self, _event=None):
        if self.active or self.calibrating:
            return
        try:
            _, display, key = self.selected_device()
        except RuntimeError:
            self.profile = None
            return
        try:
            self.profile = CalibrationProfile.from_dict(self.settings.get("profiles", {})[key])
        except (KeyError, ValueError, TypeError):
            self.profile = None
        self.settings["last_source"] = key
        self.cal_source.configure(text=self.T("selected_source", device=display))
        self.update_profile_info()
        # La selección no reconfigura ni abre el micrófono físico.
        try:
            save_settings(self.settings)
        except OSError:
            pass

    def update_profile_info(self):
        p = self.profile
        if p is None:
            self.cal_summary.configure(text=self.T("cal_not_calibrated"))
            self.cal_stage_label.configure(text=self.T("cal_press_start"))
            return
        method = self.T("measured") if p.measured_shout else self.T("automatic")
        self.cal_summary.configure(text=self.T(
            "cal_summary", noise=p.noise_db, whisper=p.whisper_db,
            normal=p.normal_db, shout=p.shout_db, method=method))
        self.cal_stage_label.configure(text=self.T("cal_saved_title"))

    def clear_calibration(self):
        if self.active or self.calibrating:
            return
        key = self.selected_source_key()
        if not key:
            return
        self.settings.setdefault("profiles", {}).pop(key, None)
        self.profile = None
        try:
            save_settings(self.settings)
        except OSError as exc:
            self.cal_status.configure(text=self.T("cal_deletion_failed", error=exc))
        self.update_profile_info()
        self.cal_status.configure(text=self.T("cal_deleted"))

    @staticmethod
    def open_reference_stream(device_index, callback):
        info = sd.query_devices(device_index, kind="input")
        rates = list(dict.fromkeys([RATE, int(info["default_samplerate"])]))
        channels = [1]
        if int(info["max_input_channels"]) >= 2:
            channels.append(2)
        last_error = None
        for rate in rates:
            for channel_count in channels:
                try:
                    sd.check_input_settings(device=device_index, channels=channel_count,
                                            samplerate=rate, dtype="float32")
                    stream = sd.InputStream(device=device_index, samplerate=rate,
                                            blocksize=0, channels=channel_count, dtype="float32",
                                            callback=callback, latency="low")
                    return stream
                except (sd.PortAudioError, ValueError) as error:
                    last_error = error
        raise RuntimeError(f"El dispositivo no admite los formatos de captura probados: {last_error}")

    def capture(self, data, frames, timing, status):
        """Solo RMS; no guardar PCM ni enviar la voz física al driver."""
        # Puede ser micrófono mono o estéreo; detectar la voz aunque llegue
        # solamente por el canal derecho.
        rms = float(np.max(np.sqrt(np.mean(np.square(data, dtype=np.float64), axis=0))))
        self.level_db = max(-96.0, 20 * math.log10(max(rms, 1e-9)))
        profile = getattr(self, "profile", None)
        if profile is not None:
            self.target = profile.map_level(self.level_db)
        else:
            # Fallback manual, para desarrollo y compatibilidad con versiones previas.
            threshold = self.threshold_value
            self.target = float(np.clip((self.level_db - threshold) /
                                        max(3.0, -14 - threshold), 0, 1)) ** .78
        now = time.monotonic()
        self.last_input = now
        if getattr(self, "calibrating", False):
            with self.cal_lock:
                if self.cal_step_ready <= now < self.cal_step_end:
                    # Solo floats dBFS; descartar muestras de voz inmediatamente.
                    self.cal_samples.append(self.level_db)

    def start_calibration(self):
        """Start the first, 5-second SILENCE stage explicitly on user click."""
        if self.active:
            return
        if self.calibrating:
            self.repeat_stage()
            return
        try:
            device_index, _, _ = self.selected_device()
            self.cal_steps = ["noise", "whisper", "normal"]
            if self.include_shout.get():
                self.cal_steps.append("shout")
            self.cal_results = {}
            self.calibrating = True
            self.cal_running = False
            self.cal_stream = self.open_reference_stream(device_index, self.capture)
            self.cal_stream.start()
        except Exception as exc:
            self.cancel_calibration(silent=True)
            messagebox.showerror(self.T("tab_calibration"), self.T("recording_failed", error=exc))
            return
        self.in_selector.configure(state="disabled")
        self.btn_refresh.configure(state="disabled")
        self.btn_on.configure(state="disabled")
        self.btn_clear_cal.configure(state="disabled")
        self.chk_shout.configure(state="disabled")
        self.btn_cal_cancel.configure(state="normal")
        self.tabs.select(self.cal_tab)
        self._prepare_step(0, run=True)

    def _prepare_step(self, index: int, *, run: bool = False):
        """Arm a stage; it only records for 5 seconds after a button press."""
        name = self.cal_steps[index]
        label, _prep, seconds = STAGES[name]
        self.cal_step_idx = index
        self.cal_step_name = name
        self.cal_running = run
        now = time.monotonic()
        with self.cal_lock:
            self.cal_step_ready = now if run else float("inf")
            self.cal_step_end = now + seconds if run else float("inf")
            self.cal_samples = []
        self.cal_stage_label.configure(text=self.T(
            "cal_step", step=index + 1, total=len(self.cal_steps),
            description=self.T(label)))
        self.cal_progress["value"] = 0
        self.btn_cal.configure(
            text=self.T("cal_repeat" if name in self.cal_results else "cal_begin"),
            state="disabled" if run else "normal")
        self.btn_cal_prev.configure(state="disabled" if run or index == 0 else "normal")
        self.btn_cal_next.configure(
            text=self.T("cal_finish" if index == len(self.cal_steps)-1 else "cal_next"),
            state="normal" if not run and name in self.cal_results else "disabled")
        self.cal_status.configure(
            text=self.T("cal_record_timer", seconds=seconds, db=self.level_db)
            if run else self.T("cal_stage_waiting", seconds=seconds))

    def repeat_stage(self):
        if not self.calibrating or self.cal_running:
            return
        self._prepare_step(self.cal_step_idx, run=True)

    def previous_stage(self):
        if not self.calibrating or self.cal_running or self.cal_step_idx == 0:
            return
        self._prepare_step(self.cal_step_idx-1)

    def next_stage(self):
        if not self.calibrating or self.cal_running:
            return
        if self.cal_step_name not in self.cal_results:
            return
        nxt = self.cal_step_idx + 1
        if nxt < len(self.cal_steps):
            self._prepare_step(nxt)
            return
        self._complete_calibration()

    def _finish_step(self):
        """Freeze RMS measurements, then WAIT for next user click."""
        if not self.calibrating or not self.cal_running:
            return
        name = self.cal_step_name
        with self.cal_lock:
            samples = self.cal_samples[:]
            self.cal_samples = []
            self.cal_step_ready = float("inf")
            self.cal_step_end = float("inf")
        self.cal_running = False
        try:
            self.cal_results[name] = sample_stage(name, samples)
        except ValueError as exc:
            self.cal_results.pop(name, None)
            self.btn_cal.configure(state="normal", text=self.T("cal_repeat"))
            self.btn_cal_prev.configure(
                state="normal" if self.cal_step_idx > 0 else "disabled")
            self.btn_cal_next.configure(state="disabled")
            self.cal_status.configure(text=self.T("cal_retry", error=exc))
            return
        self.cal_progress["value"] = 100
        self.btn_cal.configure(state="normal", text=self.T("cal_repeat"))
        self.btn_cal_prev.configure(
            state="normal" if self.cal_step_idx > 0 else "disabled")
        self.btn_cal_next.configure(state="normal")
        self.cal_status.configure(text=self.T(
            "cal_stage_done", db=self.cal_results[name], seconds=STAGES[name][2]))

    def _complete_calibration(self):
        try:
            result = CalibrationProfile.from_measurements(
                noise_db=self.cal_results["noise"],
                whisper_db=self.cal_results["whisper"],
                normal_db=self.cal_results["normal"],
                shout_db=self.cal_results.get("shout"),
            )
        except (KeyError, ValueError) as exc:
            messagebox.showwarning(self.T("cal_bad"), self.T("cal_retry", error=exc))
            return
        self.cancel_calibration(silent=True)
        self.profile = result
        key = self.selected_source_key()
        if key:
            self.settings.setdefault("profiles", {})[key] = result.to_dict()
            self.settings["last_source"] = key
        try:
            save_settings(self.settings)
            saved = True
        except OSError as exc:
            saved = False
            self.cal_status.configure(text=self.T("cal_memory_only", error=exc))
        self.update_profile_info()
        if saved:
            self.cal_status.configure(text=self.T("cal_completed"))
        self.cal_progress["value"] = 100

    def cancel_calibration(self, silent=False):
        self.calibrating = False
        self.cal_running = False
        with self.cal_lock:
            self.cal_step_ready = float("inf")
            self.cal_step_end = float("inf")
            self.cal_samples = []
        if self.cal_stream is not None:
            stream, self.cal_stream = self.cal_stream, None
            try:
                stream.stop()
            except Exception:
                pass
            try:
                stream.close()
            except Exception:
                pass
        self.in_selector.configure(state="readonly")
        self.btn_refresh.configure(state="normal")
        self.btn_clear_cal.configure(state="normal")
        self.chk_shout.configure(state="normal")
        self.btn_on.configure(state="normal")
        self.btn_cal.configure(state="normal", text=self.T("cal_start"))
        self.btn_cal_cancel.configure(state="disabled")
        self.btn_cal_prev.configure(state="disabled")
        self.btn_cal_next.configure(state="disabled")
        if not silent:
            self.cal_status.configure(text=self.T("cal_cancelled"))
            self.cal_progress["value"] = 0

    def render(self):
        """Generate Lelele only, PCM 48k stereo signed int32."""
        target = self.target if time.monotonic() - self.last_input < .4 else 0.0
        speed = self.response_value / 100
        alpha = math.exp(-.010 / (.025 + (1 - speed) * .14))
        end = alpha * self.outlevel + (1 - alpha) * target
        amplitude = np.linspace(self.outlevel, end, BLOCK, dtype=np.float32)
        self.outlevel = end
        t = (self.pos + np.arange(BLOCK)) / RATE
        self.pos += BLOCK
        # Only Lelele. No robot, beeps, noise or alternate audio modes.
        phase = 2 * np.pi * (165 * t - 1.5 * np.cos(2 * np.pi * 2.1 * t))
        wave = (.64 * np.sin(phase) + .27 * np.sin(2 * phase + .4)) * (
            .25 + .75 * np.maximum(0, np.sin(2 * np.pi * 6 * t)))
        samples = np.clip(wave * amplitude * self.volume_value / 100, -.88, .88)
        pcm = np.rint(samples * 2147483647).astype("<i4")
        return np.repeat(pcm[:, None], 2, axis=1).tobytes()

    def cable_callback(self, outdata, frames, timing, status):
        """PortAudio output callback: solo samples sintéticos, jamás voz de referencia."""
        try:
            if status and getattr(status, "output_underflow", False):
                # El flujo continúa; se informará solo si también se detiene.
                pass
            offset = 0
            while offset < frames:
                if self.output_pending.shape[0] == 0:
                    self.output_pending = pcm_to_float32(self.render(), BLOCK)
                size = min(frames - offset, self.output_pending.shape[0])
                samples = self.output_pending[:size]
                if outdata.shape[1] == 1:
                    outdata[offset:offset + size, 0] = samples[:, 0]
                else:
                    outdata[offset:offset + size, :2] = samples
                self.output_pending = self.output_pending[size:]
                offset += size
        except Exception as exc:
            outdata.fill(0)
            self.error = f"Salida VB-CABLE: {exc}"
            raise sd.CallbackAbort

    def start(self):
        if self.active or self.calibrating:
            return
        if self.profile is None and not messagebox.askyesno(
            self.T("without_cal"),
            self.T("without_cal_ask")
        ):
            self.tabs.select(self.cal_tab)
            return
        self.error = ""
        pair = find_cable_pair()
        if pair is None:
            self.dependency_status.configure(text=self.T("dependency_missing"))
            if messagebox.askyesno(
                self.T("missing_dep_title"),
                self.T("missing_dep_ask")
            ):
                self.open_vbcable_website()
            return
        try:
            source_index, _, _ = self.selected_device()
            self.stream = self.open_reference_stream(source_index, self.capture)
            self.output_pending = np.empty((0, 2), dtype=np.float32)
            self.output_stream = make_output_stream(pair[0], self.cable_callback)
            self.stream.start()
            self.output_stream.start()
        except Exception as exc:
            if self.output_stream is not None:
                try:
                    self.output_stream.close()
                except Exception:
                    pass
                self.output_stream = None
            if self.stream is not None:
                try:
                    self.stream.close()
                except Exception:
                    pass
                self.stream = None
            messagebox.showerror(self.T("cannot_start"), str(exc))
            return
        self.active = True
        self.btn_on.configure(state="disabled")
        self.btn_off.configure(state="normal")
        self.in_selector.configure(state="disabled")
        self.btn_refresh.configure(state="disabled")
        self.btn_cal.configure(state="disabled")
        self.btn_clear_cal.configure(state="disabled")
        self.status.configure(text=self.T("active_status", device=pair[1]))

    def stop(self):
        self.active = False
        if self.output_stream is not None:
            try:
                self.output_stream.stop()
            except Exception:
                pass
            try:
                self.output_stream.close()
            except Exception:
                pass
            self.output_stream = None
        self.output_pending = np.empty((0, 2), dtype=np.float32)
        if self.stream:
            try:
                self.stream.stop()
            except Exception:
                pass
            try:
                self.stream.close()
            except Exception:
                pass
            self.stream = None
        self.target = self.outlevel = 0.0
        self.btn_on.configure(state="normal")
        self.btn_off.configure(state="disabled")
        self.in_selector.configure(state="readonly")
        self.btn_refresh.configure(state="normal")
        self.btn_cal.configure(state="normal")
        self.btn_clear_cal.configure(state="normal")
        self.status.configure(text=self.T("stopped_status"))

    def tick(self):
        if self.active or self.calibrating:
            signal = (self.outlevel if self.active else self.target) * 100.0
            self._draw_meter(signal)
            color = self.T("red") if signal >= 85 else (self.T("yellow") if signal >= 60 else self.T("green"))
            self.readout.configure(text=self.T("meter_readout", db=self.level_db, percent=signal, color=color))
        if self.calibrating and self.cal_running:
            now = time.monotonic()
            _, _, seconds = STAGES[self.cal_step_name]
            remaining = max(0.0, self.cal_step_end - now)
            self.cal_status.configure(text=self.T("cal_record_timer",
                                                  seconds=remaining, db=self.level_db))
            self.cal_progress["value"] = max(0.0, min(100.0,
                (1 - remaining / seconds) * 100.0))
            if now >= self.cal_step_end:
                self._finish_step()
        if self.active and self.error:
            reason = self.error
            self.stop()
            self.status.configure(text=self.T("stream_error", error=reason))
        self.root.after(100, self.tick)

    def close(self):
        if self.calibrating:
            self.cancel_calibration(silent=True)
        self.stop()
        self.root.destroy()


if __name__ == "__main__":
    window = tk.Tk()
    SolarisMic(window)
    window.mainloop()
