# Instalación de AntiCensor (ES)

[English](README_INSTALADOR.md) | [README del proyecto](README.es.md)

Versión: **0.7.1** · Windows 10/11 x64.

El instalador contiene **la aplicación GPLv3** y ofrece **VB-CABLE básico de VB-Audio Software** como complemento externo opcional. El driver es **propietario donationware**, no GPLv3, y conserva las condiciones del fabricante. Dentro del instalador se incluye el ZIP original sin modificar. Su instalación requiere permisos de administrador y **reiniciar Windows**. No desactivamos Secure Boot, no agregamos certificados de prueba ni cambiamos el micrófono predeterminado.

**Para instalar:** ejecutá `SolarisPKN-AntiCensor-Setup-v0.7.1.exe`, elegí español o inglés, aceptá la licencia y marcá VB-CABLE si querés la salida de micrófono virtual. Terminá la instalación original de VB-Audio (solicitud UAC) y **reiniciá Windows**. La configuración y la calibración siguen funcionando si omitís la dependencia, pero Roblox no recibirá audio hasta tener VB-CABLE.

**Para usar:** abrí AntiCensor desde el icono de la bandeja, seleccioná tu micrófono físico, configurá idioma, calibrá y activá. AntiCensor intenta poner automáticamente al dispositivo de grabación el nombre **SolarisPKN-Mic Lelele**, al instalar y después del reinicio de Windows. En Roblox/Discord elegí ese nombre (o **CABLE Output** si Windows no permite cambiarlo). AntiCensor envía únicamente Lelele sintético a **CABLE Input** (reproducción). Tus palabras reales nunca se envían al cable. Falta validar el circuito completo con una prueba real en Roblox.

**Para desinstalar:** quitá AntiCensor desde Aplicaciones de Windows. VB-CABLE queda instalado intencionalmente, ya que otros programas pueden utilizarlo. Los niveles de calibración e idioma se guardan en `%LOCALAPPDATA%\SolarisPKN\AntiCensor`.

**Actualizaciones semanales:** revisamos Releases de LiveTools etiquetadas `anticensor-vX.Y.Z` y avisamos si existe una versión nueva; no instalamos ejecutables descargados en silencio.

**Compilar desde el código:** `INSTALAR.ps1` y después `packaging\BUILD_EXE.ps1`; si no existe el ZIP de VB-CABLE, el script lo descarga del fabricante y verifica su SHA-256. Requiere Python 3.12 x64, PyInstaller e Inno Setup 6.

**Legal:** GPLv3 para nuestro código, VB-CABLE bajo licencia separada donationware. Fabricante: VB-Audio Software / Vincent Burel. Descarga: https://vb-audio.com/Cable/ · Licencias: https://vb-audio.com/Services/licensing.htm. Las distribuciones profesionales pueden exigir licencias abonadas.
