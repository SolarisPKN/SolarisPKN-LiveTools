# Installing SolarisPKN-AntiCensor v0.8.5

[Español](README_INSTALADOR.es.md) · [Module README](README.md)

Windows 10/11 x64. **SolarisPKN-AntiCensor-Setup-v0.8.5.exe** installs the GPLv3 application with **Solaris Tray-style dark mode**, bilingual settings and **LELELE only**.

**VB-CABLE Basic** from VB-Audio Software is a proprietary donationware external dependency. If missing, Setup offers the original vendor installer (UAC permission and restart). If already installed, **Install is hidden**, leaving optional **Clean driver (troubleshooting only)** with a warning and confirmation. It opens the original vendor utility and does not silently uninstall anything. Secure Boot remains enabled.

AntiCensor attempts to rename the Windows recording endpoint **SolarisPKN-Mic Lelele** on setup/login without changing driver binaries; some apps may still display **CABLE Output**.

**Microphone selector:** Windows WASAPI endpoints without duplicates from MME, DirectSound or WDM-KS; retains the previous choice when possible.

**Expert mode (Settings):** off hides VB-CABLE names and maintenance controls inside AntiCensor while keeping the virtual microphone usable; on exposes the advanced read-only cable selector and diagnostics. It never hides or disables Windows endpoints or prevents third-party sound from entering CABLE Input.

**Manual calibration:** silence 5 s → whisper 5 s → normal speech 5 s → optional shout 5 s. Every step requires an explicit click. Repeat, Previous and Cancel are supported; without shout the maximum is estimated.

**Updates:** optional weekly automatic updates checkbox or manual Check. Official packages are SHA-256 checked and **installed only after AntiCensor closes or at the next Windows login**, never while streaming. Fixed source: **SolarisPKN/SolarisPKN-LiveTools/SolarisPKN-AntiCensor**.

Uninstalling AntiCensor does not remove VB-CABLE. Manufacturer license terms: https://vb-audio.com/Services/licensing.htm. Real Roblox/OBS tests are pending.

**Automatic name v0.8.5:** if Windows blocks the audio property API, the tray helper uses the built-in Sound Properties dialog to rename only the verified VB-CABLE endpoint. No driver signing changes. The update repository is fixed and NOT editable.
