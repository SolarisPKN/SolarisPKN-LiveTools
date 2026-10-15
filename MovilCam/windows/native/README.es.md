# MovilCam Windows Native Engine — prototipo optimizado

Diseñado para **Windows 10 LTSC 2021 (19044), x64**, i3-10105 y GPUs UHD 630 / GTX 1070.

## Arquitectura

- Un solo `MovilCamBridge.exe`, sin .NET runtime, Electron, Chromium ni dependencias de FFmpeg.
- TCP loopback mediante `adb forward tcp:27183 tcp:27183`.
- Un saludo binario `MCAMV002` solicita al Android frames H.264 completos con marca de tiempo (el modo crudo anterior de OBS sigue disponible).
- Media Foundation decodifica H.264 a NV12. Pide aceleración Direct3D 11 por defecto; si el driver no la admite correctamente, se debe revisar el log y probar `--software`. **No asumir que D3D11 activo implica hardware efectivo** hasta medir Video Decode por GPU.
- Un ring de **tres buffers NV12** compartidos por Windows (`Local\\SolarisPKN_MovilCam_NV12_v1`). Cada buffer se publica con *seqlock* de 64 bits para evitar lectores de frames parcialmente escritos.
- No se renderiza UI ni se convierte a RGB en el broker. La **cámara virtual DirectShow** consume NV12 directamente; se puede registrar aparte con el script de Windows (todavía sin validar con el Note real).
- Reintentos al perder conexión y límites de tamaño por frame. Sin colas de fotogramas ilimitadas.

## Compilación Windows x64

Necesita Visual Studio 2022 Build Tools con C++ y SDK Windows 10+:

    cmake -S MovilCam/windows/native -B build/movilcam-native -G "Visual Studio 17 2022" -A x64
    cmake --build build/movilcam-native --config Release
    ctest --test-dir build/movilcam-native -C Release --output-on-failure

CI en GitHub Actions: el workflow `MovilCam Windows native` construye `MovilCamBridge.exe` y `MovilCamVirtualCamera.dll`, ejecuta pruebas reales en Release y valida el registro/enumeración DirectShow cuando el entorno lo permite.

## Ejecución

En el Note debe estar instalada la versión MovilCam con protocolo v2 (posterior a este cambio).

    adb forward tcp:27183 tcp:27183
    adb shell am start -n com.solarispkn.movilcam/.MainActivity --ez autostart true
    .\MovilCamBridge.exe

Para exponer la fuente a otras aplicaciones, también hay que registrar la DLL DirectShow con `windows/install-virtualcam.ps1`. Un programa que solo use Media Foundation podría no enumerarla.


## Selección de GPU para el decodificador

`windows/start-native.ps1 -Gpu auto` usa la GPU predeterminada de Windows.
En equipos con Intel UHD y NVIDIA, probá `-Gpu intel` o `-Gpu nvidia`, y compará CPU, Video Decode, memoria y estabilidad. También admite `-Gpu amd` y `-Gpu software`.

El selector enumera adaptadores por DXGI y pasa el seleccionado al Media Foundation H.264 Decoder mediante D3D11. Si se solicita un fabricante no disponible, el puente rechaza la selección para que no haya cambios silenciosos. **Que el decodificador acepte D3D11 no garantiza que todos sus pasos ejecuten en GPU**; comprobar con el Administrador de tareas / GPU Engine. La decodificación final se copia a memoria NV12 para ser consumida por DirectShow.

## Memoria / trabajo de copia

1080p NV12 = 3.110.400 bytes por imagen. Tres buffers = 9.331.200 bytes, sin contar cabeceras y memoria del decodificador. El broker copia una vez al ring para compartir frames con distintas aplicaciones. Las transformaciones para formatos RGB, si alguna app las exige, deben ocurrir *solo* en su adaptador, no para todos.

La ruta ideal de cero copia total requeriría compartir directamente texturas de GPU; DirectShow de Windows 10 no garantiza que todos los consumidores acepten esa ruta. El ring NV12 es una base estable para compatibilidad, no una promesa de cero copia GPU/CPU.

## Seguridad pendiente

El protocolo actual por ADB y la memoria compartida de Windows **todavía no incluyen autenticación criptográfica de fotogramas ni protección contra inyección de un proceso local comprometido**. No habilitar puertos remotos ni usarlo como fuente confiable para autenticación/identidad hasta auditar ACL, emparejamiento dispositivo, transporte y autenticidad de frames.

## Pendientes obligatorios

- Pruebas de registro, enumeración y vídeo de la cámara DirectShow en Windows 10.
- Prueba física de Camera2 del Galaxy Note 10+.
- Confirmar decodificación D3D11/Intel y comportamiento de memoria en GPU/CPU.
- Probar el filtro virtual DirectShow implementado con consumidores reales; por ahora expone NV12 y YUY2 (no RGB directo).
- Comparar FPS, calor, estabilidad, latencia y CPU contra Iriun.

## Cámara virtual DirectShow (Windows 10)

La build de Windows también genera `MovilCamVirtualCamera.dll`. Es un filtro COM en modo usuario, no un driver de kernel. Su registro permite que las aplicaciones DirectShow de 64 bits intenten abrir `SolarisPKN MovilCam`; **la compatibilidad con Discord y apps modernas todavía no está verificada**.

1. Descargá el artefacto `MovilCamBridge-win-x64` del workflow Windows y extraé ambos archivos.
2. Desde PowerShell de 64 bits **con permisos de administrador**, ejecutá `windows/install-virtualcam.ps1 -Dll "C:\ruta\MovilCamVirtualCamera.dll"`.
3. Desde PowerShell normal: `windows/start-native.ps1 -BridgeExe "C:\ruta\MovilCamBridge.exe"`. Debés tener la versión Android compatible con protocolo v2 instalada y autorizada por ADB.
4. Abrí una aplicación que soporte DirectShow y buscá la fuente `SolarisPKN MovilCam`. Cuando el programa realmente empieza a capturar, el broker activa la cámara de Android.
5. Para revertirlo, cerrá todas las aplicaciones que utilicen la fuente y ejecutá `windows/uninstall-virtualcam.ps1` desde PowerShell elevado.

**Limitaciones:** En Windows 10 el filtro DirectShow no garantiza aparecer en todo software que solo utilice Media Foundation. El formato actual está orientado a 1080p30 y necesita que el lector y la resolución del broker coincidan. No modifiques una escena en producción sin haber probado la señal primero. La compilación CI verifica la instalación temporal y enumeración del filtro, pero no sustituye una prueba real de vídeo.
