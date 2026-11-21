# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Safe expert diagnostics for VB-CABLE; never changes Windows endpoint state."""
from __future__ import annotations

import json
import os
from pathlib import Path


def config_path() -> Path:
    base = os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")
    return Path(base) / "SolarisPKN" / "AntiCensor" / "ui_settings.json"


def load_expert_mode() -> bool:
    try:
        value = json.loads(config_path().read_text(encoding="utf-8"))
        return isinstance(value, dict) and value.get("expert_mode") is True
    except (OSError, ValueError, TypeError):
        return False


def _load_ui_config() -> dict:
    try:
        result = json.loads(config_path().read_text(encoding="utf-8"))
        return result if isinstance(result, dict) else {}
    except (OSError, ValueError, TypeError):
        return {}


def _save_ui_config(config: dict) -> None:
    path = config_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(".json.tmp")
    try:
        tmp.write_text(json.dumps(config, indent=2), encoding="utf-8")
        os.replace(tmp, path)
    finally:
        tmp.unlink(missing_ok=True)


def save_expert_mode(enabled: bool) -> None:
    config = _load_ui_config()
    config["expert_mode"] = bool(enabled)
    _save_ui_config(config)


def load_voice_filter() -> tuple[bool, int]:
    config = _load_ui_config()
    enabled = config.get("voice_filter_enabled", True) is True
    try:
        aggressiveness = int(config.get("voice_filter_aggressiveness", 2))
    except (ValueError, TypeError):
        aggressiveness = 2
    return enabled, max(0, min(3, aggressiveness))


def save_voice_filter(enabled: bool, aggressiveness: int) -> None:
    level = int(aggressiveness)
    if level < 0 or level > 3:
        raise ValueError("VAD aggressiveness must be in range 0..3")
    config = _load_ui_config()
    config["voice_filter_enabled"] = bool(enabled)
    config["voice_filter_aggressiveness"] = level
    _save_ui_config(config)


def is_vb_cable_endpoint(name: str) -> bool:
    """Recognize VB-Audio's basic / 16ch / Point endpoints, including renamed mic.

    Avoid generic 'Cable' matches: never hide physical devices from other makers.
    """
    lower = str(name).casefold()
    if lower.startswith("solarispkn-mic lelele"):
        return True
    if "vb-audio" not in lower and "vb audio" not in lower:
        return False
    return (
        "cable input" in lower
        or "cable output" in lower
        or "cable in 16ch" in lower
        or "cable out 16ch" in lower
        or "vb-audio virtual cable" in lower
        or "vb-audio point" in lower
    )


def reference_microphones(device_list, api_list):
    """Show each Windows capture endpoint only once, using native WASAPI.

    PortAudio enumerates one endpoint via multiple host APIs (MME, DirectSound,
    WASAPI and WDM-KS). The UI should use native WASAPI, like Windows Sound.
    If WASAPI is missing altogether, select one backend per name, without
    collapsing two identically named microphones within the *same* backend.
    Return (PortAudio index, endpoint name, API name, input channels).
    """
    candidates = []
    for index, device in enumerate(device_list):
        channels = int(device.get("max_input_channels", 0))
        if channels <= 0:
            continue
        name = str(device.get("name", "")).strip()
        if not name or is_vb_cable_endpoint(name) or any(
            marker in name.casefold() for marker in
            ("solaris anti-cheat", "solarispkn-mic lelele", "solaris virtual mic")
        ):
            continue
        api = str(api_list[device["hostapi"]]["name"])
        candidates.append((index, name, api, channels))

    wasapi = [row for row in candidates if "wasapi" in row[2].casefold()]
    if wasapi:
        return sorted(wasapi, key=lambda row: (row[1].casefold(), row[0]))

    # Fallback for older / non-WASAPI hosts. Deduplicate backend *copies*,
    # but preserve physical devices that legitimately share a display name.
    rank = lambda api: (
        0 if "directsound" in api.casefold()
        else 1 if "wdm-ks" in api.casefold()
        else 2 if "mme" in api.casefold()
        else 3
    )
    by_name = {}
    for row in candidates:
        by_name.setdefault(row[1].casefold(), {}).setdefault(row[2], []).append(row)
    selected = []
    for apis in by_name.values():
        best = min(apis, key=rank)
        selected.extend(apis[best])
    return sorted(selected, key=lambda row: (row[1].casefold(), row[0]))


def cable_inventory(device_list, api_list):
    """Human-readable, read-only inventory; WASAPI first, no stream is opened."""
    rows = []
    for device in device_list:
        name = str(device.get("name", ""))
        if not is_vb_cable_endpoint(name):
            continue
        api = str(api_list[device["hostapi"]]["name"])
        inputs = int(device.get("max_input_channels", 0)) > 0
        outputs = int(device.get("max_output_channels", 0)) > 0
        if not inputs and not outputs:
            continue
        kind = "capture" if inputs and not outputs else "playback" if outputs and not inputs else "duplex"
        rows.append((api, kind, name))
    if any("wasapi" in api.casefold() for api, _, _ in rows):
        rows = [row for row in rows if "wasapi" in row[0].casefold()]
    # A single entry per name+direction, regardless of backend.
    unique = {}
    for api, kind, name in rows:
        unique.setdefault((kind, name.casefold()), (kind, name))
    return sorted(unique.values(), key=lambda item: (item[0], item[1].casefold()))
