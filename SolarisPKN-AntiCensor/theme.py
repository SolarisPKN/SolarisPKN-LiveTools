# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Solaris Tray inspired always-dark palette shared by AntiCensor UI.

Original SolarisTray.cs: menu background (29,29,39), light text (235,242,245),
soft mint status (165,222,198). No external themes or framework required.
"""
from __future__ import annotations

import tkinter as tk
from tkinter import ttk

BG = "#1d1d27"
PANEL = "#23232f"
INPUT = "#292a38"
HOVER = "#363849"
BORDER = "#424557"
FG = "#ebf2f5"
MUTED = "#a5afbe"
MINT = "#a5dec6"
GREEN = "#42cb72"
RED = "#ff646d"
GOLD = "#f2c245"


def apply_dark(root: tk.Misc) -> ttk.Style:
    root.configure(background=BG)
    style = ttk.Style(root)
    try:
        style.theme_use("clam")
    except tk.TclError:
        pass
    style.configure(".", background=BG, foreground=FG, font=("Segoe UI", 10))
    style.configure("TFrame", background=BG)
    style.configure("Card.TFrame", background=PANEL)
    style.configure("TLabel", background=BG, foreground=FG)
    style.configure("Muted.TLabel", foreground=MUTED)
    style.configure("Header.TLabel", foreground=MINT, font=("Segoe UI Semibold", 15))
    style.configure("TButton", background=INPUT, foreground=FG, borderwidth=1,
                    relief="flat", padding=(12, 8), bordercolor=BORDER)
    style.map("TButton", background=[("pressed", HOVER), ("active", HOVER)],
              foreground=[("disabled", MUTED)])
    style.configure("Accent.TButton", background="#235342", foreground="#f0fff7",
                    bordercolor="#34755c", padding=(12, 10),
                    font=("Segoe UI Semibold", 10))
    style.map("Accent.TButton", background=[("pressed", "#376b56"),
                                            ("active", "#337a5d")])
    style.configure("Danger.TButton", background="#522e3a", foreground="#ffeef0",
                    bordercolor="#6e4050")
    style.map("Danger.TButton", background=[("active", "#693849")])
    style.configure("TNotebook", background=BG, borderwidth=0)
    style.configure("TNotebook.Tab", background=INPUT, foreground=MUTED,
                    padding=(15, 9), borderwidth=0)
    style.map("TNotebook.Tab",
              background=[("selected", PANEL), ("active", HOVER)],
              foreground=[("selected", MINT), ("active", FG)])
    style.configure("TCombobox", fieldbackground=INPUT, background=INPUT,
                    foreground=FG, arrowcolor=MINT, selectbackground=HOVER,
                    selectforeground=FG, bordercolor=BORDER, padding=(5, 7))
    style.map("TCombobox", fieldbackground=[("readonly", INPUT)],
              foreground=[("readonly", FG)])
    style.configure("TCheckbutton", background=BG, foreground=FG, padding=5)
    style.map("TCheckbutton", foreground=[("disabled", MUTED)],
              background=[("active", BG)])
    style.configure("TScale", background=BG, troughcolor=INPUT)
    style.configure("Horizontal.TProgressbar", troughcolor=INPUT,
                    background=MINT, lightcolor=MINT, darkcolor=MINT,
                    bordercolor=BORDER)
    style.configure("TSeparator", background=BORDER)
    try:
        root.option_add("*TCombobox*Listbox.background", INPUT)
        root.option_add("*TCombobox*Listbox.foreground", FG)
        root.option_add("*TCombobox*Listbox.selectBackground", HOVER)
        root.option_add("*TCombobox*Listbox.selectForeground", FG)
    except tk.TclError:
        pass
    return style
