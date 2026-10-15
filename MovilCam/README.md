# SolarisPKN MovilCam

Experimental dedicated Android USB webcam for OBS. Native Camera2 -> H.264 hardware MediaCodec encoder -> loopback TCP over ADB -> OBS Media Source. No Iriun, scrcpy, WebView, camera preview, microphone or Internet service.

Camera and encoder start **only after OBS connects** and are released after disconnect. Android 12 camera foreground service is started from a visible Activity via ADB.

**Status:** source implementation; hardware test and APK build not yet verified on the Galaxy Note 10+. Build through Android SDK/Gradle or GitHub Actions.

For full installation, commands, debugging, power/thermal limitations and OBS settings: [README.es.md](README.es.md).
