# SolarisPKN MovilCam

Experimental dedicated Android USB webcam for OBS. Native Camera2 -> H.264 hardware MediaCodec encoder -> loopback TCP over ADB -> OBS Media Source. No Iriun, scrcpy, WebView, camera preview, microphone or Internet service.

Camera and encoder start **only after OBS connects** and are released after disconnect. Android 12 camera foreground service is started from a visible Activity via ADB.

**Status (2026-10-15):** GitHub Actions built the debug APK and passed the H264 unit tests. Real video capture, thermals and OBS playback still need validation on the Galaxy Note 10+.

For Android setup, commands, debugging and OBS settings: [README.es.md](README.es.md). For the native Windows broker, DirectShow virtual camera and Intel/NVIDIA/AMD decode selection, see [Windows camera guide](windows/native/README.es.md). CI verifies COM camera registration and enumeration; real-device streaming and third-party application support remain untested.
