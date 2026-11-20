# 🎙️ SolarisPKN AntiCensor

**Español** · [English](README.md)

SolarisPKN AntiCensor es una **aplicación de Windows bajo GPLv3** que usa el *micrófono físico solamente para medir la intensidad de la voz*. En vez de reenviar tus palabras, genera sonidos sintéticos (Lelele, Robot, Pitidos y Ruido) cuyo volumen sigue tu nivel calibrado. Forma parte de [SolarisPKN-LiveTools](https://github.com/SolarisPKN/SolarisPKN-LiveTools).

**Estado actual: vista previa v0.7.1.** El código, la interfaz bilingüe, el menú de bandeja y el instalador cuentan con pruebas automáticas. Todavía faltan las pruebas completas de audio real VB-CABLE → Roblox.

[English](README.md) | [Instalación (ES)](README_INSTALADOR.es.md) | [Installation (EN)](README_INSTALADOR.md)

## 🎧 Cómo funciona

```text
Micrófono físico ── SOLO intensidad RMS ──▶ AntiCensor
                                              │ genera PCM sintético
                                              ▼
                                       CABLE Input (reproducción)
                                              │
                                       CABLE Output (grabación)
                                              ▼
                                           Roblox / Discord
```

No guarda muestras de tu voz ni las reenvía al cable virtual. OBS u otro programa pueden usar el micrófono físico por separado si así lo elegís. No cambia los dispositivos predeterminados de Windows.

## Características

- Calibración guiada: ruido ambiente, susurro (≈7,5%), voz normal (≈50%), grito opcional (≈100%). Si no calibrás grito, calcula automáticamente el límite superior.
- Selección del micrófono físico de referencia y perfil por dispositivo, almacenado solo como valores dBFS, nunca como audio grabado.
- Modos **Lelele**, **Robot**, **Pitidos** y **Ruido**. La grabación original de Lelele todavía no está incluida. Formato recomendado: WAV PCM mono de 24 bits, 48 kHz y 8–12 segundos, sin saturación.
- Medidor de intensidad en vivo (verde → amarillo → rojo).
- Selector **Español / English** desde la configuración, sin reiniciar. También traduce el menú del icono junto al reloj.
- Icono de bandeja para abrir configuración, calibrar, revisar actualizaciones, configurar repositorio o salir completamente.
- Búsqueda semanal **solo informativa** de GitHub Releases; reconoce exclusivamente etiquetas `anticensor-vMAYOR.MENOR.PARCHE` del repositorio `SolarisPKN/SolarisPKN-LiveTools`. Nunca ejecuta instaladores descargados automáticamente.

## Dependencia externa obligatoria para transmitir: VB-CABLE

VB-CABLE básico de **VB-Audio Software / Vincent Burel** es un componente **propietario donationware**. **NO está bajo GPLv3**, aunque el código original de nuestra aplicación sí lo está. Nuestro instalador puede ofrecer instalar el paquete oficial sin modificar, ejecutando el instalador del fabricante con permisos de administrador. Después se debe **reiniciar Windows**. No hay que desactivar Secure Boot ni usar firmas de prueba.

- Descarga oficial: https://vb-audio.com/Cable/
- Condiciones de licencia y redistribución: https://vb-audio.com/Services/licensing.htm
- Para uso profesional o distribución comercial, cumplir las licencias requeridas por VB-Audio.
- AntiCensor intenta poner **automáticamente el nombre visible SolarisPKN-Mic Lelele** al instalarse o en el siguiente inicio de Windows, cuando aparezca el dispositivo de VB-Audio. No altera el controlador firmado. Falta comprobar el resultado en una máquina con VB-CABLE instalado.
- Desinstalar AntiCensor **no desinstala automáticamente VB-CABLE**, porque podrían usarlo otros programas.

## Instalación rápida (Windows 10/11 x64)

1. Ejecutá `SolarisPKN-AntiCensor-Setup-v0.7.1.exe` desde una compilación o Release verificada.
2. Cuando se ofrezca, instalá la dependencia VB-CABLE mediante el asistente del fabricante y **reiniciá Windows**.
3. Abrí AntiCensor desde el icono junto al reloj y seleccioná el micrófono físico de referencia.
4. Elegí **Idioma / Language**, calibrá susurro y voz normal, y pulsá **Activar micrófono**.
5. En Roblox o Discord seleccioná **SolarisPKN-Mic Lelele** cuando Windows aplique el nombre. Si Windows bloquea el cambio, se conserva **CABLE Output** como respaldo. Reiniciá el juego para actualizar dispositivos.

Para compilar desde el código: `powershell -NoProfile -ExecutionPolicy Bypass -File .\INSTALAR.ps1`, luego `powershell -NoProfile -ExecutionPolicy Bypass -File .\packaging\BUILD_EXE.ps1`. Requiere Python 3.12 x64, PyInstaller e Inno Setup 6. El ZIP propietario no se sube a Git: `packaging\FETCH_VBCABLE.ps1` lo descarga directamente del fabricante y comprueba su SHA-256.

## Licencias, privacidad y seguridad

- Código original de AntiCensor, traducciones, pruebas y scripts: **GPL-3.0-only**, ver [LICENSE](LICENSE).
- VB-CABLE: **dependencia independiente propietaria donationware** de VB-Audio Software. Ver [aviso de terceros](packaging/third_party/VBCABLE-NOTICE.txt); **no se relicencia**.
- El driver experimental anterior MicyWMD/SysVAD, con código Microsoft MS-PL, **no se incluye en el módulo de LiveTools** ni se instala.
- No se graban palabras, no se hace reconocimiento de voz ni se sube audio a internet. Solo se guardan mediciones locales dBFS. La única conexión de red es la consulta opcional de metadatos de actualizaciones.
- El EXE de AntiCensor aún no tiene firma comercial y SmartScreen podría advertir; eso es **distinto de la firma Microsoft del controlador de VB-Audio**.

Es una versión preliminar: no afirmamos compatibilidad completa con Roblox hasta validar el circuito de audio en una sesión real.
