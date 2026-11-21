# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Classical WebRTC VAD activity gate for SolarisPKN-AntiCensor.

Only temporary 20 ms PCM buffers in RAM; no transcription, files, uploads,
recognition of words, or recording of the user's speech. The *only* output
is a Boolean telling the synthesizer whether speech is likely present.

This is NOT acoustic echo cancellation: game dialogue or music leaking into
the mic can sometimes be classified as speech and must be fixed at the source.
"""
from __future__ import annotations

from collections import deque
import math

import numpy as np


class ClassicalVoiceGate:
    """Lightweight 20 ms WebRTC voice detector with anti-click hysteresis."""

    TARGET_RATE = 16000
    FRAME_MS = 20
    FRAME_SAMPLES = TARGET_RATE * FRAME_MS // 1000  # 320

    def __init__(self, aggressiveness: int = 2):
        try:
            import webrtcvad
        except ImportError as exc:
            raise RuntimeError(
                "Falta webrtcvad-wheels. Instalá dependencias o el Setup oficial "
                "de AntiCensor para usar el filtro clásico de voz."
            ) from exc
        if not 0 <= int(aggressiveness) <= 3:
            raise ValueError("WebRTC VAD aggressiveness must be 0..3")
        self.vad = webrtcvad.Vad(int(aggressiveness))
        self._pending = np.empty(0, dtype=np.float32)
        self._rate = 0
        self._votes: deque[bool] = deque(maxlen=4)
        self._hold_frames = 0
        self.active = False

    def feed(self, data: np.ndarray, sample_rate: int) -> bool:
        """Consume live input frames and return whether speech gate is open.

        Handle PortAudio's variable block sizes and 44.1/48 kHz microphones
        by reformatting to valid 20 ms / 16 kHz WebRTC VAD frames.
        """
        rate = int(sample_rate)
        if rate < 8000 or rate > 192000:
            raise ValueError("Unsupported microphone sample rate")
        if rate != self._rate:
            self._rate = rate
            self._pending = np.empty(0, dtype=np.float32)
            self._votes.clear()
            self._hold_frames = 0
            self.active = False

        incoming = np.asarray(data, dtype=np.float32)
        if incoming.ndim == 2:
            if incoming.shape[1] == 0:
                return self.active
            # Keep the strongest live channel: analog mics sometimes deliver
            # one silent channel in a stereo Realtek stream.
            channel = int(np.argmax(np.mean(np.square(incoming), axis=0)))
            incoming = incoming[:, channel]
        if incoming.ndim != 1 or len(incoming) == 0:
            return self.active
        if not np.all(np.isfinite(incoming)):
            incoming = np.nan_to_num(incoming, copy=True)

        self._pending = np.concatenate((self._pending, incoming))
        frame_size = int(round(rate * self.FRAME_MS / 1000))
        if frame_size < 1:
            return False

        while len(self._pending) >= frame_size:
            frame = self._pending[:frame_size]
            self._pending = self._pending[frame_size:]
            if frame_size != self.FRAME_SAMPLES:
                # Linear resampling is sufficient for VAD analysis;
                # the original waveform is NEVER sent to any output.
                frame = np.interp(
                    np.linspace(0.0, frame_size - 1.0, self.FRAME_SAMPLES),
                    np.arange(frame_size),
                    frame,
                )
            pcm = (np.clip(frame, -1.0, 1.0) * 32767.0).astype("<i2").tobytes()
            is_voice = bool(self.vad.is_speech(pcm, self.TARGET_RATE))
            self._votes.append(is_voice)
            # Two positive 20-ms frames avoid clicks or isolated impulses
            # opening the synthesizer. Brief hold keeps word tails natural.
            if len(self._votes) >= 2 and sum(self._votes) >= 2:
                self.active = True
                self._hold_frames = 7  # ~140 ms
            elif self._hold_frames > 0:
                self._hold_frames -= 1
            else:
                self.active = False
        return self.active
