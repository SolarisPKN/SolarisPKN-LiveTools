# ❓ Frequently asked questions — AntiCensor
[Español](FAQ.es.md) · [Home](../README.md)

### 🎮 Does Roblox receive my real speech?
**Not through AntiCensor.** It generates synthetic Lelele instead of forwarding your original mic PCM. **Other apps** that you configure to capture the physical mic can still do so. Choose CABLE Output/SolarisPKN-Mic Lelele in Roblox.

### 🎙️ Why is CABLE Input a playback device?
VB-Audio names endpoints from the cable's perspective: **CABLE Input** receives playback from AntiCensor, and **CABLE Output** delivers virtual microphone capture to the game.

### 🧠 Does it recognize words?
No. Classical WebRTC VAD estimates probable speech activity; RMS controls synthetic volume. No words, transcription or language model are involved.

### 🚚 Does it remove mouse clicks, traffic or music?
It reduces some false triggers but is neither a professional noise suppressor nor echo canceller. Loud traffic, music and game dialog can still trigger the gate; correct audio routing first.

### 🔁 What does Expert Mode do?
Shows technical VB-CABLE endpoints **inside AntiCensor**. Does not disable Windows devices.

### 🎧 Why is the Lelele meter moving with music?
The physical mic may detect game playback and trigger synthesis, or another app may be playing directly to CABLE Input. Briefly stop AntiCensor while music plays to distinguish. Avoid unnecessary OBS monitoring of Lelele.

### 📺 Can OBS use my real voice while Roblox receives Lelele?
Yes, with separate OBS physical and game virtual microphone inputs. Always verify both app selectors.

### 🔐 Is this intended to bypass voice restrictions?
No. Respect platform rules and voice-chat requirements.

### 🧩 Licensing?
AntiCensor source is GPLv3. Basic VB-CABLE is independent donationware licensed by its vendor.

💚 [SolarisPKN-Labs](https://labs.solarispkn.com.ar/)