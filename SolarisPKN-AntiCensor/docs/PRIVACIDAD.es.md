# 🔐 Privacidad y licencias — AntiCensor
[English](PRIVACY.en.md) · [Inicio](../README.es.md)

## 🧠 Tratamiento del audio
- 🎤 El micrófono físico se procesa localmente en memoria para obtener intensidad y actividad vocal.
- 🤖 **Solo la síntesis Lelele** se envía al cable virtual por AntiCensor, no el contenido original del micrófono.
- 🧾 No se transcriben palabras ni se guardan grabaciones de voz por esta función; WebRTC VAD no requiere servicios remotos.
- 🔄 La función de actualizaciones opcional consulta versiones oficiales; no transmite audio.

## ⚠️ Límites
- VB-CABLE es un cable de audio **compartido**. Cualquier aplicación que reproduzca audio a CABLE Input puede enviarlo a CABLE Output. Modo experto no bloquea esta posibilidad.
- OBS, Discord u otras aplicaciones que capten **directamente** tu micrófono físico conservan esa capacidad.
- El VAD clásico no es cancelación de eco y puede clasificar falsamente audio musical o voces de juegos.
- Las funciones de AntiCensor no garantizan anonimato, cumplimiento de normas de juegos ni compatibilidad universal.

## ⚖️ Licencias
**AntiCensor:** GPL-3.0-only, con fuente y condiciones en [LICENSE](../LICENSE). **VB-CABLE Basic:** software externo de **VB-Audio Software / Vincent Burel** sujeto a condiciones y licencias independientes: [sitio oficial](https://vb-cable.com/) y [términos](https://vb-audio.com/Services/licensing.htm). El instalador opcional, cuando se utilice, debe reconocer al titular. No se distribuyen cables adicionales A+B/C+D ni controladores modificados.

**Portada:** obra gráfica aportada por el creador para AntiCensor; no se incluye en la licencia GPL del código a menos que su titular indique lo contrario. Diagramas SVG y GIF fueron creados para esta documentación.

🦊 **Software desarrollado por [SolarisPKN-Labs](https://labs.solarispkn.com.ar/).**