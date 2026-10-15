# SolarisPKN MovilCam - Android USB camera para OBS

Version 0.1 experimental. Proyecto Android puro (Java + Camera2 + MediaCodec) sin Iriun, scrcpy ni navegador. Pensado para Galaxy Note 10+ sin pantalla fisica, conectado por ADB.

**Estado:** fuentes implementadas y pruebas unitarias preparadas. Falta compilacion Android en CI o PC, instalacion y pruebas reales en Note 10+. No se garantiza aun mejor temperatura ni calidad que Iriun.

## Caracteristicas

- Camera2 escribe directamente a la Surface de MediaCodec H.264: no se muestran frames en Android.
- Un servidor TCP escucha SOLO en 127.0.0.1:27183 del telefono; ADB hace el tunel USB.
- La camara y el codificador permanecen APAGADOS si no hay cliente OBS.
- Al abrirse OBS, se habilita captura y flujo H.264 Annex-B.
- Al desconectar el cliente, se libera la camara y MediaCodec.
- Solo video, sin microfono, sin subida a Internet, sin WebView, sin vista previa.
- Rechaza codificadores H.264 por software para evitar calor adicional.
- Configurable: camara trasera o frontal, ancho, alto, FPS (5..60), bitrate.

## Compilar APK

Requisitos: Android SDK API 35, Build Tools 35.0.0, JDK 17 y Gradle 8.9.
En PowerShell, desde MovilCam:

    gradle --no-daemon :app:testDebugUnitTest :app:assembleDebug

Salida: app/build/outputs/apk/debug/app-debug.apk.

Si no tenes Gradle/SDK, el workflow de GitHub Actions llamado "MovilCam Android" puede compilar y publicar un artefacto APK de depuracion despues de subirse al repo. Es una build de desarrollo; no es un release firmado para tienda.

## Preparar el Note sin pantalla (USB + ADB)

Verifica que indique "device" (no "unauthorized"):

    adb devices

Instala el APK y concede el permiso de camara desde ADB:

    cd windows
    .\install-movilcam.ps1 -Apk "C:\ruta\app-debug.apk"

Arranca desde Windows (no necesitas manejar ninguna UI Android):

    .\start-movilcam.ps1 -Lens back -Width 1920 -Height 1080 -Fps 30 -BitrateMbps 8

Alternativamente, comandos ADB puros:

    adb forward tcp:27183 tcp:27183
    adb shell pm grant com.solarispkn.movilcam android.permission.CAMERA
    adb shell am start -n com.solarispkn.movilcam/.MainActivity --ez autostart true --es lens back --ei width 1920 --ei height 1080 --ei fps 30 --ei bitrate 8000000

Se inicia una Activity solo para autorizar el inicio legal de un servicio de camara en primer plano. Despues no se necesita previsualizacion. Android 12+ restringe servicios de camara iniciados directamente desde segundo plano.

## Conectar OBS sin plugin ni Browser Source

En OBS: Fuentes > + > Fuente multimedia (Media Source).
- Desmarcar "Archivo local".
- Entrada/Input: tcp://127.0.0.1:27183
- Formato de entrada/Input Format: h264
- Audio: sin audio desde MovilCam.
- Probar buffering de red minimo y decodificacion por hardware, si es compatible.

IMPORTANTE: El soporte de H.264 sin contenedor sobre TCP depende de FFmpeg y tu version de OBS; aun falta verificarlo en OBS 31. Si la fuente no funciona, revisar diagnostico con FFplay antes de concluir que falla el Android.

Prueba opcional con FFplay (solo uno conectado a la vez; cerrar fuente OBS):

    ffplay -fflags nobuffer -flags low_delay -f h264 -framerate 30 -i tcp://127.0.0.1:27183

OBS puede reutilizar su unica fuente MovilCam en escenas de streaming y grabacion para evitar crear capturas dobles. No conectar dos clientes TCP simultaneos.

## Diagnostico y parada

    adb logcat -s MovilCam:V

Mensajes esperados: "Listening...", "OBS client connected", "Camera started" y "H264 frames sent".
Para detener desde Windows:

    .\windows\stop-movilcam.ps1

## Rendimiento y limites

1080p30 y 8 Mbps son valores de partida. MediaCodec usa el encoder H.264 de hardware del telefono si declara capacidad. El sensor, ISP, autoexposicion, autofocus y encoder siguen consumiendo energia. Desactivar la pantalla o la previsualizacion no garantiza menos calor que Iriun.

Sin pantalla fisica, Android aun requiere permisos CAMERA previamente otorgados, una Activity para el inicio de Foreground Service, y una fuente de alimentacion electrica estable.
Si la fuente se congela, el proceso se puede diagnosticar con logcat y ADB. Si la alimentacion externa provoca caidas de voltaje, ningun cambio de software lo corrige.

**Seguridad:** servidor del telefono solo loopback, ADB requiere PC autorizada. No exponer el puerto a red local ni Internet. No almacenar secretos, tokens ni claves ADB en el repo.

Pendiente: probar codec real del Note, calidad vs Iriun, temperatura de 1h+, desconexiones USB, mejores ajustes Camera2/ISP, integracion nativa OBS si Media Source no sirve.
