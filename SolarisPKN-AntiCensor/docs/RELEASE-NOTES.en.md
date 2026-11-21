# 🚀 SolarisPKN AntiCensor v0.8.5 — Release notes
🇬🇧 English · [🇦🇷 Español](RELEASE-NOTES.es.md)

### 🎮 Why the project exists
While streaming **Roblox horror games**, an unexpected jump scare sometimes made me swear. I was tired of the **sanctions** and losing access to **voice chat** in voice-driven horror experiences. The goal is to keep real commentary in OBS separate from the synthetic **Lelele** effect sent to the game, not to guarantee exemption from Roblox moderation.

### 🎙️ What's new
- 🧠 **Classical WebRTC VAD**, configurable 0–3, enabled by default (2). No word transcription or LLM.
- 🖱️ Fewer accidental isolated-click triggers, using temporal voting and release hold.
- 🎚️ Clean WASAPI microphone selector, explicit source choice and no duplicated PortAudio APIs.
- 🛠️ **Expert Mode**: read-only VB-CABLE diagnostics without changing Windows endpoints.
- 🦊 **Original creator-supplied cover** displayed in README and About tab.
- 💚 **Software developed by SolarisPKN-Labs**, https://labs.solarispkn.com.ar/
- 🌍 Complete bilingual guides, SVG figures, animated GIF and examples involving Roblox, Gadnic, SteelSeries Sonar, Redragon and OBS.
- ✅ 68 automated tests passed in the development environment.

### ⚠️ Limitations
No advanced echo cancellation. VB-CABLE is shared and other applications may play audio into it. Game-specific audio routing needs real-world verification.

### 📦 Installer
Module release assets: **SolarisPKN-AntiCensor-Setup-v0.8.5.exe** (Windows x64 installer), **SolarisPKN-AntiCensor-Portable-v0.8.5.exe** (Windows x64 standalone executable), and **SHA256SUMS-AntiCensor-v0.8.5.txt** (integrity verification). The module has its own **anticensor-v0.8.5** tag in LiveTools. The built-in updater selects only the exact setup name for a strictly newer semantic version, ignoring other modules. No functional Debian/Linux package exists for this Windows-only release.

### 🔐 Security
No credentials, private IP addresses, personal OBS profiles, private diagnostics or captured audio are included in this public documentation.

[📘 Documentation](../README.md) · [🌐 SolarisPKN-Labs](https://labs.solarispkn.com.ar/)