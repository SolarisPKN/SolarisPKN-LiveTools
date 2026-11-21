# 🔐 Privacy and licensing — AntiCensor
[Español](PRIVACIDAD.es.md) · [Home](../README.md)

## 🧠 Audio processing
- 🎤 Physical mic samples are processed temporarily in local memory for voice activity and loudness.
- 🤖 AntiCensor sends **only newly synthesized Lelele** to the virtual cable, never original mic PCM.
- 🧾 No speech transcripts or recordings are produced by this feature; WebRTC VAD does not need a cloud service.
- 🔄 Optional update checks contact the official repository for versions, not audio upload.

## ⚠️ Boundaries
- VB-CABLE is a **shared** playback/capture pair. Any other app playing into CABLE Input may reach CABLE Output. Expert Mode is not a system isolation feature.
- OBS, Discord or other apps explicitly configured to capture your physical mic can still access your original voice.
- Classical VAD is not acoustic echo cancellation and can mistake leaked game speech/music for the user's voice.
- AntiCensor cannot guarantee anonymity, compliance with platform rules or universal compatibility.

## ⚖️ Licenses
**AntiCensor:** GPL-3.0-only; source and terms in [LICENSE](../LICENSE). **Basic VB-CABLE:** independent proprietary donationware by **VB-Audio Software / Vincent Burel**, with its own terms: [official website](https://vb-cable.com/) and [license](https://vb-audio.com/Services/licensing.htm). Any optional vendor installer must preserve attribution. Extra A+B/C+D cables and altered drivers are not redistributed.

**Cover art:** supplied by the creator for AntiCensor; not automatically covered by the source code's GPL license unless the rights holder says so. SVG diagrams and GIF are original project illustrations.

🦊 **Software developed by [SolarisPKN-Labs](https://labs.solarispkn.com.ar/).**