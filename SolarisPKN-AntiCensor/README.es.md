# 🎙️ SolarisPKN-AntiCensor

**Español** · [English](README.md) · [Guía de instalación](README_INSTALADOR.es.md)

Módulo oficial de [SolarisPKN-LiveTools](https://github.com/SolarisPKN/SolarisPKN-LiveTools/tree/main/SolarisPKN-AntiCensor), aplicación **GPLv3** para Windows. Solo mide la intensidad RMS del micrófono físico seleccionado y envía **LELELE**, nunca la voz real. Por ahora usa una síntesis Lelele temporal, pendiente de incorporar la grabación WAV original.

**Versión preliminar 0.8.0.** Interfaz oscura, calibración, actualizador e instalador implementados; falta prueba real completa en Roblox.

## Características
- **Modo oscuro obligatorio como Solaris Tray:** fondo #1D1D27, texto claro y acentos verde menta.
- **LELELE exclusivamente**; eliminados Robot, Pitidos, Ruido y otros modos.
- Selección de micrófono físico para medir únicamente intensidad, con medidor de salida verde/amarillo/rojo.
- Ajustes y menú de bandeja **Español / English**, guardando la preferencia.
- Tres pestañas: **Micrófono, Calibración y Ajustes**.

## Calibración, 5 segundos por etapa
1. Pulsá **Empezar: Silencio**, permanecé en silencio 5 segundos.
2. Cuando termina, elegís **Repetir**, **Anterior**, **Siguiente** o **Cancelar**; nunca pasa de etapa sin tu acción.
3. **Susurro**, pulsá **Iniciar etapa** y hablá bajo 5 s: ancla aproximada 7,5 %.
4. **Voz normal**, pulsá **Iniciar etapa** y hablá 5 s: ancla aproximada 50 %.
5. **Grito opcional**, si marcaste el checkbox: 5 s para fijar 100 %. Sin grito, la amplitud máxima se extrapola automáticamente de las mediciones anteriores.

No guarda audio, solo agregados dBFS y el perfil por dispositivo. Repetir una etapa no borra los resultados de las demás.

## Dependencia VB-CABLE
Se utiliza el **VB-CABLE básico de VB-Audio Software / Vincent Burel**, software **propietario donationware, no GPLv3**, instalado como componente externo sin modificar su firma Microsoft.
- **Si falta**: el instalador muestra **Instalar VB-CABLE**.
- **Si ya está**: desaparece Instalar; solo se ofrece **Limpiar driver (solo por errores)**, con advertencia y confirmación. Abre la utilidad oficial de mantenimiento; **no borra el driver automáticamente**.
- AntiCensor intenta mostrar el micrófono de grabación como **SolarisPKN-Mic Lelele** sin modificar el controlador firmado. Algunas aplicaciones pueden mostrar el nombre original **CABLE Output**.
- AntiCensor reproduce Lelele por **CABLE Input** y Roblox/Discord lo reciben por el dispositivo de grabación.
- Desinstalar AntiCensor no desinstala VB-CABLE.

Web del fabricante: https://vb-audio.com/Cable/ · Licencias: https://vb-audio.com/Services/licensing.htm

## Actualizaciones
La ruta oficial es **SolarisPKN/SolarisPKN-LiveTools/SolarisPKN-AntiCensor**, fija y no configurable. GitHub utiliza el repositorio SolarisPKN/SolarisPKN-LiveTools con tags del módulo **solarispkn-anticensor-vX.Y.Z** y nombre de instalador **SolarisPKN-AntiCensor-Setup-vX.Y.Z.exe**.
- Checkbox **Habilitar actualizaciones automáticas semanales**: al marcarlo revisa cada 7 días, descarga y verifica el SHA-256 del instalador oficial cuando exista una Release válida.
- Checkbox desactivado: únicamente **Buscar actualizaciones ahora** y confirmación manual de la descarga.
- **Nunca instala con AntiCensor abierto ni interrumpe el stream.** El auxiliar de Windows instala después de que cierres la app, o antes de lanzarla en el próximo inicio de sesión; no reinicia Windows ni detiene otro programa.
- Si no hay Release con instalador y SHA-256, no ejecuta nada.

## Compilación y licencia
En Windows 10/11 x64: ejecutar INSTALAR.ps1 y packaging/BUILD_EXE.ps1 con Python 3.12, PyInstaller e Inno Setup 6. Se obtienen el EXE portátil y el Setup v0.8.0. El ZIP del proveedor se descarga desde VB-Audio y se valida su SHA-256; **no se sube a Git**.

Código original GPL-3.0-only ([LICENSE](LICENSE)); VB-CABLE mantiene su licencia independiente. No se incluye el controlador experimental MicyWMD. Quedan pendientes pruebas de latencia, Roblox, OBS, persistencia del nombre visible y una actualización real desde GitHub Releases.
