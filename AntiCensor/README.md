# 🎙️ SolarisPKN AntiCensor

**English** · [Español](README.es.md)

SolarisPKN AntiCensor is a **GPLv3-licensed Windows desktop application** that uses a *physical microphone only as a loudness reference*. Instead of forwarding speech, it synthesizes a Lelele/robot/beep/noise sound whose amplitude follows the calibrated microphone level. It is part of [SolarisPKN-LiveTools](https://github.com/SolarisPKN/SolarisPKN-LiveTools).

**Current state: v0.7.1 preview.** The code, bilingual UI, tray and installer have automated tests. A full interactive end-to-end VB-CABLE → Roblox test is still pending.

[Español](README.es.md) | [Installation (EN)](README_INSTALADOR.md) | [Instalación (ES)](README_INSTALADOR.es.md)

## 🎧 How it works

```text
Physical microphone ── RMS loudness ONLY ──▶ AntiCensor
                                              │ synthesize original PCM
                                              ▼
                                     VB-CABLE Input (playback)
                                              │
                                     VB-CABLE Output (recording)
                                              ▼
                                         Roblox / Discord
```

No voice samples are stored or routed through the virtual cable. Your recording or streaming software may capture the **real physical microphone separately** if you explicitly select it. No global default recording/playback device is changed.

## Features

- Guided calibration: ambient noise, whisper (≈7.5%), conversational speech (≈50%), optional shout (≈100%). Without a shout, the upper limit is estimated.
- A selectable physical microphone as the reference device; per-device profile is saved as aggregated dBFS readings, never raw audio.
- Synthetic modes: **Lelele**, **Robot**, **Beeps**, **Noise**. The user-recorded Lelele WAV is not supplied yet. Suggested source: 8–12 seconds, mono, 24-bit PCM WAV at 48 kHz, with no clipping.
- Live intensity meter (green → yellow → red).
- **Español / English** live language switch in Settings. The choice persists between launches and updates the system-tray menu too.
- Windows system-tray icon to open Settings, Calibrate, Check for updates, choose repository or fully Quit.
- Weekly *read-only* checks of GitHub Releases; only tags matching `anticensor-vMAJOR.MINOR.PATCH` in `SolarisPKN/SolarisPKN-LiveTools` count. No downloaded executable is silently run.

## Required third-party driver: VB-CABLE

VB-CABLE Basic by **VB-Audio Software / Vincent Burel** is a **proprietary donationware** external dependency. It is **NOT GPLv3**, even though all original AntiCensor application code is GPLv3. The unmodified VB-CABLE package can be installed by the user from its official vendor installer as an optional step in our Windows Setup. Installation requires administrator access and a Windows restart. No test-signing, Secure Boot changes, or custom kernel driver are required.

- Official download: https://vb-audio.com/Cable/
- License/distribution conditions: https://vb-audio.com/Services/licensing.htm
- For commercial/professional distribution, confirm and obtain the licenses required by VB-Audio.
- AntiCensor automatically sets the recording endpoint's **Windows display name** to **SolarisPKN-Mic Lelele** during setup or at the next Windows login after the VB-Audio driver becomes available. The signed driver remains unchanged. This needs validation on a machine with VB-CABLE installed.
- Uninstalling AntiCensor **does not automatically remove VB-CABLE**, as other apps may need it.

## Quick start (Windows 10/11 x64)

1. Run `SolarisPKN-AntiCensor-Setup-v0.7.1.exe` from a verified build or Release.
2. If prompted, enable the VB-CABLE dependency, let the manufacturer's installer complete, and **restart Windows**.
3. Open AntiCensor from the notification-area icon and select your real reference microphone.
4. Choose **Language / Idioma**, calibrate whisper and normal speech, then press **Enable Microphone**.
5. In Roblox/Discord, choose **SolarisPKN-Mic Lelele** as the input device once Windows applies the name. If the rename is blocked, the original **CABLE Output** remains the fallback. Restart the game to refresh its device list.

Source-only build: `powershell -NoProfile -ExecutionPolicy Bypass -File .\INSTALAR.ps1`, then `powershell -NoProfile -ExecutionPolicy Bypass -File .\packaging\BUILD_EXE.ps1`. Build tools: Python 3.12 x64, PyInstaller and Inno Setup 6. The vendor archive is excluded from Git; `packaging\FETCH_VBCABLE.ps1` fetches the official unchanged ZIP and checks its pinned SHA-256.

## Licensing, privacy, and safety

- Original AntiCensor application, translations, tests and packaging helpers: **GPL-3.0-only**, see [LICENSE](LICENSE).
- VB-CABLE: **external, proprietary donationware**, copyright VB-Audio Software. See [third-party notice](packaging/third_party/VBCABLE-NOTICE.txt); **not relicensed**.
- Legacy experimental MicyWMD/SysVAD code, including Microsoft samples under MS-PL, is **not part of this LiveTools module**, and is not installed by its Setup.
- No microphone recordings, speech analysis, speech-to-text or internet uploading. Only local aggregated levels (dBFS) are stored. The only network activity is optional scheduled release metadata retrieval.
- The AntiCensor EXE itself is not commercially code-signed yet; SmartScreen may display a warning. This is **separate from the Microsoft-signed vendor kernel driver**.

This module is a preview, not a claim of working Roblox compatibility until a full interactive audio loopback test is completed.
