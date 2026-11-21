# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
# Copyright (c) 2026 SolarisPKN contributors
"""Calibración RMS en dBFS de AntiCensor, sin almacenar muestras de voz.

Cada perfil queda asociado al dispositivo físico seleccionado. Un susurro se
traduce a 7,5 % de intensidad de síntesis; voz normal a 50 %; y voz fuerte a
100 %. El nivel del grito puede medirse o inferirse desde los dos primeros.
"""
from __future__ import annotations

from dataclasses import asdict, dataclass
from math import isfinite
from i18n import get_language, t


WHISPER_LEVEL = 0.075
NORMAL_LEVEL = 0.50
RED_START = 0.85
SHOUT_LEVEL = 1.0


def percentile(values, p: float) -> float:
    """Percentil lineal robusto para ventanas de muestras RMS ya agregadas."""
    ordered = sorted(float(x) for x in values if isfinite(float(x)))
    if not ordered:
        raise ValueError(t("err_no_rms", get_language()))
    at = (len(ordered) - 1) * max(0.0, min(1.0, p))
    low = int(at)
    high = min(len(ordered) - 1, low + 1)
    ratio = at - low
    return ordered[low] * (1.0 - ratio) + ordered[high] * ratio


def infer_shout_db(whisper_db: float, normal_db: float) -> float:
    """Con dos lecturas, extrapolar sin obligar al usuario a gritar."""
    gap = normal_db - whisper_db
    return min(-1.0, normal_db + max(12.0, min(21.0, gap * 1.25)))


@dataclass(frozen=True)
class CalibrationProfile:
    noise_db: float
    whisper_db: float
    normal_db: float
    shout_db: float
    measured_shout: bool = False

    def __post_init__(self):
        all_values = (self.noise_db, self.whisper_db, self.normal_db, self.shout_db)
        if not all(isfinite(float(x)) and -100.0 <= float(x) <= 0.0 for x in all_values):
            raise ValueError(t("err_db_range", get_language()))
        if self.whisper_db - self.noise_db < 4.0:
            raise ValueError(t("err_whisper", get_language()))
        if self.normal_db - self.whisper_db < 4.0:
            raise ValueError(t("err_normal", get_language()))
        if self.shout_db - self.normal_db < 4.0:
            raise ValueError(t("err_shout", get_language()))

    @classmethod
    def from_measurements(cls, noise_db: float, whisper_db: float,
                          normal_db: float, shout_db: float | None = None) -> "CalibrationProfile":
        if shout_db is None:
            candidate = infer_shout_db(whisper_db, normal_db)
            return cls(noise_db, whisper_db, normal_db, candidate, False)
        return cls(noise_db, whisper_db, normal_db, shout_db, True)

    @classmethod
    def from_dict(cls, data: dict) -> "CalibrationProfile":
        return cls(
            noise_db=float(data["noise_db"]),
            whisper_db=float(data["whisper_db"]),
            normal_db=float(data["normal_db"]),
            shout_db=float(data["shout_db"]),
            measured_shout=bool(data.get("measured_shout", False)),
        )

    def to_dict(self) -> dict:
        return asdict(self)

    def map_level(self, db: float) -> float:
        """De dBFS del mic físico a ganancia de síntesis [0,1].

        Anchors:
         - ruido ambiente -> silencio (0)
         - susurro -> 0.075
         - voz normal -> 0.5
         - voz fuerte/grito -> 1.0
        """
        if not isfinite(db):
            return 0.0
        gate = max(self.noise_db + 2.5,
                   self.whisper_db - min(8.0, (self.whisper_db - self.noise_db) * .7))
        if db <= gate:
            return 0.0
        if db < self.whisper_db:
            return WHISPER_LEVEL * (db - gate) / max(self.whisper_db - gate, 1e-3)
        if db < self.normal_db:
            x = (db - self.whisper_db) / (self.normal_db - self.whisper_db)
            return WHISPER_LEVEL + x * (NORMAL_LEVEL - WHISPER_LEVEL)
        if db < self.shout_db:
            x = (db - self.normal_db) / (self.shout_db - self.normal_db)
            return NORMAL_LEVEL + x * (SHOUT_LEVEL - NORMAL_LEVEL)
        return SHOUT_LEVEL

    def color(self, percent: float) -> str:
        if percent >= RED_START * 100:
            return "rojo"
        if percent >= 60:
            return "amarillo"
        return "verde"


def sample_stage(stage: str, values: list[float]) -> float:
    """Percentil representativo; rechazar audio casi sin muestras."""
    # Con blocksize=0 algunos dispositivos WASAPI entregan bloques largos.
    # 12 callbacks bastan para una etapa de ~2-3 segundos sin depender del buffer.
    if len(values) < 12:
        raise ValueError(t("err_too_few", get_language()))
    q = {"noise": .80, "whisper": .65, "normal": .65, "shout": .72}
    if stage not in q:
        raise ValueError(t("err_stage", get_language()))
    result = percentile(values, q[stage])
    if not -100 <= result <= 0:
        raise ValueError(t("err_out_of_range", get_language()))
    return result
