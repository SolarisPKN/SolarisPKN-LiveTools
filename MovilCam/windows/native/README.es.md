# MovilCam Windows Native Engine — prototipo optimizado

Diseñado para **Windows 10 LTSC 2021 (19044), x64**, i3-10105 y GPUs UHD 630 / GTX 1070.

## Arquitectura

- Un solo `MovilCamBridge.exe`, sin .NET runtime, Electron, Chromium ni dependencias de FFmpeg.
- TCP loopback mediante `adb forward tcp:27183 tcp:27183`.
- Un saludo binario `MCAMV002` solicita al Android frames H.264 completos con marca de tiempo (el modo crudo anterior de OBS sigue disponible).
- Media Foundation decodifica H.264 a NV12. Pide aceleración Direct3D 11 por defecto; si el driver no la admite correctamente, se debe revisar el log y probar `--software`. **No asumir que D3D11 activo implica hardware efectivo** hasta medir Video Decode por GPU.
- Un ring de **tres buffers NV12** compartidos por Windows (`Local\\SolarisPKN_MovilCam_NV12_v1`). Cada buffer se publica con *seqlock* de 64 bits para evitar lectores de frames parcialmente escritos.
- No se renderiza UI ni se convierte a RGB en el broker. Queda preparado para un filtro de cámara DirectShow que consuma NV12 directamente; el filtro de cámara virtual **aún no está implementado ni instalado**.
- Reintentos al perder conexión y límites de tamaño por frame. Sin colas de fotogramas ilimitadas.

## Compilación Windows x64

Necesita Visual Studio 2022 Build Tools con C++ y SDK Windows 10+:

    cmake -S MovilCam/windows/native -B build/movilcam-native -G "Visual Studio 17 2022" -A x64
    cmake --build build/movilcam-native --config Release
    ctest --test-dir build/movilcam-native -C Release --output-on-failure

CI en GitHub Actions: workflow `MovilCam Windows native` publica MovilCamBridge.exe.

## Ejecución

En el Note debe estar instalada la versión MovilCam con protocolo v2 (posterior a este cambio).

    adb forward tcp:27183 tcp:27183
    adb shell am start -n com.solarispkn.movilcam/.MainActivity --ez autostart true
    .\MovilCamBridge.exe

La cámara no está lista para aparecer automáticamente en otras aplicaciones: este EXE crea el transporte y los fotogramas compartidos; falta el dispositivo virtual de Windows.

## Memoria / trabajo de copia

1080p NV12 = 3.110.400 bytes por imagen. Tres buffers = 9.331.200 bytes, sin contar cabeceras y memoria del decodificador. El broker copia una vez al ring para compartir frames con distintas aplicaciones. Las transformaciones para formatos RGB, si alguna app las exige, deben ocurrir *solo* en su adaptador, no para todos.

La ruta ideal de cero copia total requeriría compartir directamente texturas de GPU; DirectShow de Windows 10 no garantiza que todos los consumidores acepten esa ruta. El ring NV12 es una base estable para compatibilidad, no una promesa de cero copia GPU/CPU.

## Seguridad pendiente

El protocolo actual por ADB y la memoria compartida de Windows **todavía no incluyen autenticación criptográfica de fotogramas ni protección contra inyección de un proceso local comprometido**. No habilitar puertos remotos ni usarlo como fuente confiable para autenticación/identidad hasta auditar ACL, emparejamiento dispositivo, transporte y autenticidad de frames.

## Pendientes obligatorios

- Compilación y pruebas del broker Windows en CI.
- Prueba física de Camera2 del Galaxy Note 10+.
- Confirmar decodificación D3D11/Intel y comportamiento de memoria en GPU/CPU.
- Implementar y probar filtro virtual Windows DirectShow y sus formatos NV12/RGB.
- Comparar FPS, calor, estabilidad, latencia y CPU contra Iriun.
