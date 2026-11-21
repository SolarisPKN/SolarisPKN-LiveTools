# 🚀 SolarisPKN AntiCensor v0.8.5 — Novedades
🇦🇷 Español · [🇬🇧 English](RELEASE-NOTES.en.md)

### 🎮 Origen del proyecto
AntiCensor nació porque, al streamear **juegos de terror en Roblox**, un susto podía hacerme soltar alguna palabrota. Me cansé de las **sanciones** y de perder la posibilidad de participar en experiencias con **chat de voz**. La propuesta es separar mi voz real para OBS del efecto sintético **Lelele** que recibe el juego, sin prometer que evita la moderación de Roblox.

### 🎙️ Qué hay de nuevo
- 🧠 **WebRTC VAD clásico** con nivel 0–3, activado por defecto (2). No hay transcripción de palabras ni LLM.
- 🖱️ Menos falsas activaciones por clics aislados, con umbral temporal y cola suave.
- 🎚️ Selección WASAPI sin duplicados de PortAudio, con selección explícita del micrófono.
- 🛠️ **Modo experto** para consultar endpoints VB-CABLE sin modificar Windows.
- 🦊 **Portada ilustrada oficial** visible en README y pestaña Acerca de.
- 💚 **Software desarrollado por SolarisPKN-Labs**, https://labs.solarispkn.com.ar/
- 🌍 Manuales bilingües, SVG, GIF y ejemplos de uso con Roblox, Gadnic, SteelSeries Sonar, Redragon y OBS.
- ✅ 68 pruebas automatizadas aprobadas en el entorno de desarrollo.

### ⚠️ Limitaciones
No hay cancelación de eco avanzada. El cable es compartido y otras apps pueden escribir audio a CABLE Input. Las pruebas en algunos juegos, así como regresiones de rutas específicas, deben repetirse por el usuario.

### 📦 Instalador
Paquetes de la release del módulo: **SolarisPKN-AntiCensor-Setup-v0.8.5.exe** (instalador Windows x64), **SolarisPKN-AntiCensor-Portable-v0.8.5.exe** (ejecutable portable Windows x64) y **SHA256SUMS-AntiCensor-v0.8.5.txt** (verificación de integridad). Tag propio: **anticensor-v0.8.5**, alojado en LiveTools sin interferir con otros módulos. El actualizador usa solo el instalador, rechaza los assets de otros productos y requiere una versión semántica mayor que la instalada. No existe paquete Debian/Linux funcional en esta versión; no se anuncia uno inexistente.

### 🔐 Seguridad
No se publican credenciales, IP privadas, diagnósticos del usuario, archivos locales de OBS ni audios capturados.

[📘 Documentación](../README.es.md) · [🌐 SolarisPKN-Labs](https://labs.solarispkn.com.ar/)