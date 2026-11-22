# SolarisPKN-LiveTools — aplicación principal (diseño futuro)

Este documento solo propone una arquitectura. **No crea, instala ni reemplaza actualmente la aplicación principal.** MultiChat se mantiene como módulo independiente y ligero.

## Objetivo

Un solo programa y un solo panel para administrar el streaming de SolarisPKN:

1. Inicio y estado de conexiones a YouTube, TikTok, Twitch, Kick, Facebook y otras redes.
2. **MultiChat**: recepción, unificación, filtros y overlay OBS.
3. **Publisher**: subir shorts y videos largos donde se admita, con confirmación y registro.
4. **ClipDetector / editor de momentos**: revisión, origen y exportación de clips.
5. **AntiCensor**: audio virtual, perfiles y controles.
6. **OBS/SceneSync**: sincronización opcional de escenas OBS ↔ TikTok LIVE Studio, sin encender/empezar LIVE silenciosamente.
7. **Configuración y diagnóstico**: servicios, permisos, actualizaciones y logs sin secretos.

## Arquitectura sugerida

```text
SolarisPKN-LiveTools (aplicación principal futura)
  ├── AppShell / UI (diseño a decidir: escritorio nativo o un único WebView)
  ├── AccountBroker (autorización OAuth, DPAPI/llavero, scopes y revocación)
  ├── MultiChat (módulo Go existente; HTTP loopback/SSE)
  ├── PublisherSuite (módulo actual; tareas y cola de publicación)
  ├── ClipDetector (módulo actual; análisis y exportación)
  ├── AntiCensor (módulo actual; audio y perfiles)
  └── OBS / TikTok SceneSync (módulo aislado, permisos explícitos)
```

**Principios:** cada módulo usa su propia lógica de negocio, un contrato local versionado de IPC/HTTP, permisos mínimos, operaciones auditables, sin cinco ventanas de navegador, sin reescribir funcionalidad verificada. No compartir cookies/contraseñas entre módulos. Los tokens de acceso se obtienen mediante OAuth y se cifran por usuario; la aplicación central evita duplicar credenciales.

## Progresión

- **Ahora**: login de MultiChat, protección de historial/SSE/OBS, autorización explícita Google OAuth y Twitch Device Code, Meta Page Token manual y lectura pública Kick/TikTok.
- **Próxima etapa**: panel de estado de conexiones, OAuth de Meta con permisos aprobados, autoarranque de módulos, y pruebas de LIVE real por plataforma.
- **App principal**: interfaz unificada con pestañas, centralización de credenciales, orquestación del ciclo de vida de procesos y observabilidad.
- **Después**: instalador único, extensiones, actualizador con rollback y distribución en Windows/Linux según la capacidad real de cada módulo (audio drivers Windows pueden requerir componentes propios).

## Límites

No se presupone que Facebook, Instagram o TikTok permitan los mismos scopes para chat, publicación y LIVE. Un permiso de lectura del chat no autoriza a publicar videos ni comenzar emisiones. Las acciones de publicación/streaming requerirán consentimiento separado.

Módulos actuales ajenos a MultiChat permanecen sin tocar en esta fase.
