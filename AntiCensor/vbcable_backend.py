# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Backend de sonido VB-CABLE independiente mediante PortAudio/WASAPI.

La aplicación solo usa las APIs estándar de Windows y no contiene código del
controlador de VB-Audio. Su instalación es externa y opcional.
"""
from __future__ import annotations

import numpy as np
import sounddevice as sd
from i18n import get_language, t

VBCABLE_VENDOR = "VB-Audio Software"
VBCABLE_URL = "https://vb-audio.com/Cable/"
VBCABLE_LICENSE_URL = "https://vb-audio.com/Services/licensing.htm"
SOURCE_CHANNEL_RATE = 48000


def _match(name: str, term: str) -> bool:
    return term in name.casefold().replace("-", " ")


def find_cable_pair(device_list=None, api_list=None):
    """Devuelve (playback PortAudio index, recording label) o None.

    Priorizamos WASAPI. Nunca se usa el extremo de captura como referencia de
    voz ni enviamos audio a altavoces físicos por error.
    """
    devices = sd.query_devices() if device_list is None else device_list
    apis = sd.query_hostapis() if api_list is None else api_list
    inputs = []
    outputs = []
    for i, d in enumerate(devices):
        name = str(d["name"])
        api = str(apis[d["hostapi"]]["name"])
        lower = name.casefold()
        vendor = "vb-audio" in lower or "vb audio" in lower or "cable input" in lower or "cable output" in lower
        if not vendor:
            continue
        # El nombre histórico es contraintuitivo: Input es dispositivo render.
        if "cable input" in lower and int(d["max_output_channels"]) >= 1:
            outputs.append((i, name, api))
        elif "cable output" in lower and int(d["max_input_channels"]) >= 1:
            inputs.append((i, name, api))
    matches = []
    for index, name, api in outputs:
        for _, mic, cap_api in inputs:
            if api == cap_api:
                matches.append((index, mic, api))
    if not matches:
        return None
    matches.sort(key=lambda d: ("wasapi" not in d[2].casefold(), d[0]))
    index, mic, _ = matches[0]
    return (index, mic)


def is_vbcable_capture_device(name: str) -> bool:
    lower = name.casefold()
    return "cable output" in lower or "vb-audio virtual cable" in lower


def pcm_to_float32(raw: bytes, frames: int = 480) -> np.ndarray:
    """PCM estéreo signed 32 bits del sintetizador -> float32 WASAPI."""
    value = np.frombuffer(raw, dtype="<i4")
    if value.size != frames * 2:
        raise ValueError(t("err_no_pcm", get_language()))
    return (value.reshape(frames, 2).astype(np.float32) / np.float32(2147483647.0))


def make_output_stream(output_index: int, callback):
    """Toma VB-CABLE playback exclusivamente a 48k; nunca el default speaker."""
    info = sd.query_devices(output_index, kind="output")
    channels = 2 if int(info["max_output_channels"]) >= 2 else 1
    sd.check_output_settings(device=output_index, channels=channels,
                             samplerate=SOURCE_CHANNEL_RATE, dtype="float32")
    return sd.OutputStream(device=output_index, samplerate=SOURCE_CHANNEL_RATE,
                           blocksize=480, channels=channels, dtype="float32",
                           latency="low", callback=callback)
