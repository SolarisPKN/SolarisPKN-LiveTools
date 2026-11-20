# AntiCensor Setup / Installation (EN)

[Español](README_INSTALADOR.es.md) | [Project README](README.md)

Version: **0.7.0** · Windows 10/11 x64.

The AntiCensor Setup installs the **GPLv3 application** and offers **VB-CABLE Basic by VB-Audio Software** as an optional third-party dependency. The driver is **proprietary donationware**, not GPLv3, and remains entirely under its manufacturer's license. We ship the original unmodified vendor ZIP inside the installer. The vendor's Setup requires admin permission and a **Windows restart**. We do not disable Secure Boot, install test certificates, or change default microphones.

**To install:** run `SolarisPKN-AntiCensor-Setup-v0.7.0.exe`, choose English or Spanish, accept the license, and enable the VB-CABLE task if you need virtual microphone output. Let the original VB-Audio installer complete (UAC prompt), then **restart Windows**. AntiCensor's Settings and whisper/normal/shout calibration work even if you skip the dependency.

**To use:** open AntiCensor via its system-tray icon, choose the reference microphone, select the language, calibrate, and enable the microphone. In Roblox/Discord select **CABLE Output** (recording). AntiCensor sends only generated sounds to **CABLE Input** (playback). The original voice never travels through the virtual cable. The audio output is not yet verified end-to-end in Roblox.

**Uninstall:** remove AntiCensor via Windows Apps. VB-CABLE remains installed intentionally; use the vendor's uninstall procedure if you no longer need it. The local calibration and language preferences can be removed separately under `%LOCALAPPDATA%\SolarisPKN\AntiCensor`.

**Automatic updates:** check GitHub weekly for Releases tagged `anticensor-vX.Y.Z` inside the LiveTools repository; notifications only, no silent EXE execution.

**Build from source:** run `INSTALAR.ps1`, install PyInstaller in the venv, run `packaging\BUILD_EXE.ps1` (which fetches a verified unchanged official VB-CABLE ZIP when missing), using Inno Setup 6.

**Legal:** GPLv3 applies to AntiCensor code; VB-CABLE is a separate donationware dependency. Manufacturer: VB-Audio Software / Vincent Burel. Official download: https://vb-audio.com/Cable/ · Terms: https://vb-audio.com/Services/licensing.htm. Professional redistribution may require paid licenses.
