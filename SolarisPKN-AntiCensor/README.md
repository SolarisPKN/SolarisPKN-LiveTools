# 🎙️ SolarisPKN-AntiCensor

**English** · [Español](README.es.md) · [Installation guide](README_INSTALADOR.md)

Official [SolarisPKN-LiveTools](https://github.com/SolarisPKN/SolarisPKN-LiveTools/tree/main/SolarisPKN-AntiCensor) module, a **GPLv3 Windows application** that uses a physical microphone to measure **RMS loudness only**, without routing speech. It outputs **LELELE exclusively**. The original recorded Lelele WAV has not been supplied yet; a local synthesizer is currently used.

**v0.8.0 preview.** Dark UI, guided calibration, update staging and setup are implemented; final end-to-end Roblox testing is still pending.

## Features
- **Mandatory dark mode**, matching Solaris Tray: background #1D1D27, pale text and mint accents.
- **LELELE only**; Robot, Beeps and Noise modes have been removed.
- Select a physical reference microphone; green/yellow/red output intensity meter.
- Persistent **Español / English** settings and tray menu.
- Three tabs: **Microphone, Calibration, Settings**.

## Guided five-second calibration
1. Click **Start: Silence** and remain silent for 5 seconds.
2. After the stage completes choose **Repeat**, **Previous**, **Next**, or **Cancel**. It **never advances on its own**.
3. **Whisper**: explicitly click Start Stage and whisper for 5 seconds (approximately 7.5% output).
4. **Normal speech**: explicitly click Start Stage and speak for 5 seconds (approximately 50% output).
5. **Optional shout**: when enabled, add 5 seconds to calibrate 100%. Otherwise, extrapolate the maximum from previous measurements.

Only aggregate dBFS levels and per-device profiles are stored, never audio. Repeating one stage does not delete the other stages' measurements.

## VB-CABLE external dependency
**VB-CABLE Basic** by **VB-Audio Software / Vincent Burel** is proprietary donationware, **not GPLv3**. Its original Microsoft-signed kernel driver is used unchanged.
- **If missing**, the installer offers to install VB-CABLE.
- **If already installed**, Install is hidden; **Clean driver (troubleshooting only)** is offered with a warning and explicit confirmation, launching the vendor maintenance utility, **never silently deleting drivers**.
- AntiCensor tries to display the recording microphone as **SolarisPKN-Mic Lelele** by changing the Windows endpoint label, not the signed driver. Some apps may still show **CABLE Output**.
- Synthetic Lelele goes to **CABLE Input** for playback and applications capture the corresponding recording endpoint.
- Uninstalling AntiCensor does not uninstall VB-CABLE.

Vendor website: https://vb-audio.com/Cable/ · License terms: https://vb-audio.com/Services/licensing.htm

## Updates — only when closed or next Windows sign-in
The fixed source path is **SolarisPKN/SolarisPKN-LiveTools/SolarisPKN-AntiCensor**. The GitHub repository is SolarisPKN/SolarisPKN-LiveTools and the module release tags are **solarispkn-anticensor-vX.Y.Z** (legacy anticensor-vX.Y.Z also accepted). The installer asset name is **SolarisPKN-AntiCensor-Setup-vX.Y.Z.exe**.
- Opt in via the **Automatic weekly updates** checkbox in Settings. Once every seven days it downloads and verifies the official installer when a Release with SHA-256 exists.
- When disabled, use the **Check for updates now** button and manually approve any download.
- Installation is **deferred until AntiCensor has closed**, or until the next Windows sign-in before the app starts. The update helper never interrupts a stream, closes another application or restarts Windows.
- No GitHub Release with a verifiable installer means **no download and no execution**. The repository cannot be configured to an external source.

## Build and legal
On Windows 10/11 x64 use INSTALAR.ps1 and packaging/BUILD_EXE.ps1 with Python 3.12, PyInstaller and Inno Setup 6. The unchanged VB-CABLE ZIP is fetched from the vendor and SHA-256 checked; **vendor binaries are not committed to GitHub**.

Original source code is GPL-3.0-only ([LICENSE](LICENSE)). VB-CABLE remains separately licensed proprietary software. The old experimental MicyWMD driver is excluded. Roblox, OBS, latency, endpoint naming and a real GitHub Release update still need live validation.
