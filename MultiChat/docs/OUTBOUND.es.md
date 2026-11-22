# SolarisPKN MultiChat — Centro de envíos multiplataforma

## Ventana

Abrí el visor de MultiChat en http://127.0.0.1:8765/ y pulsá **Enviar / Comandos**. Se abre una ventana independiente en `/compose`. También podés usar http://127.0.0.1:8765/compose directamente.

Cada plataforma tiene **checkbox** y un botón **Solo**. La ventana incluye **Seleccionar todas**, **Deseleccionar todas**, contador, vista previa del mensaje, control de hasta 200 caracteres Unicode, envío y resultados por plataforma.

Para probar sin publicar, iniciá MultiChat con `-demo`. Todos los destinos están disponibles en **simulación**, no hay llamadas a las redes ni se generan mensajes reales. Los mensajes de prueba se añaden al historial como `TU MENSAJE · DEMO`, tipo `outbound_demo`.

## Seguridad y permisos para envíos reales

Los envíos están **desactivados por defecto**. Para habilitarlos explícitamente se necesita:

```json
"panel_auth": {
  "enabled": true,
  "token_env": "MULTICHAT_PANEL_TOKEN",
  "obs_token_env": "MULTICHAT_OBS_TOKEN"
},
"outbound": {
  "enabled": true,
  "kick_token_env": "MULTICHAT_KICK_CHAT_WRITE_TOKEN",
  "kick_broadcaster_user_id": 0
}
```

`panel_auth.enabled` es obligatorio para permitir `outbound.enabled` en modo real. Se requiere una sesión del panel y un `POST /api/outbound/send` autenticado. El servidor valida el origen y exige una cabecera AJAX específica, contenido JSON, plataforma permitida, un máximo de cinco destinos, texto de 1–200 caracteres sin saltos de línea y dos segundos de espera mínima entre intentos. No hay reintentos automáticos, porque ante errores de conexión una plataforma puede haber recibido el mensaje.

La UI muestra los destinatarios y solicita confirmación antes de **enviar de verdad**. Una respuesta `accepted` significa **aceptado por la API**, no entrega garantizada a espectadores.

| Plataforma | Envío real | Requisitos |
| --- | --- | --- |
| YouTube | Endpoint oficial `liveChatMessages.insert` implementado | OAuth `youtube.force-ssl` (además de lectura), canal/emisión con chat activo y `youtube.enabled=true`. Ejecutar `-link-google` otra vez si autorizaste la aplicación antigua únicamente para leer |
| Twitch | Endpoint oficial `helix/chat/messages` implementado | OAuth de dispositivo `chat:read user:write:chat`, `twitch.use_saved_twitch_auth=true`, canal. Ejecutar `-link-twitch` otra vez si el token era solo de lectura |
| Facebook | API Graph `POST /{live-video-id}/comments` implementada | `facebook.enabled=true`, token de Página que autorice crear comentarios y `page_id` o `live_video_id` vigente. Meta determina acceso/revisión |
| Kick | API oficial `POST /public/v1/chat` implementada | `kick.enabled=true`, token OAuth con scope `chat:write` en `MULTICHAT_KICK_CHAT_WRITE_TOKEN` y `kick_broadcaster_user_id` positivo. Falta crear UI OAuth integrada de Kick |
| TikTok | **No disponible para enviar realmente** | Existe un lector Webcast de LIVE, pero el envío requiere autenticación/adaptador aparte. Se indica `unavailable`, nunca éxito falso |

Fuente YouTube: https://developers.google.com/youtube/data/live/reference/rest/v3/liveChatMessages/insert
Fuente Twitch: https://dev.twitch.tv/docs/api/reference#send-chat-message
Fuente Kick: https://github.com/KickEngineering/KickDevDocs/blob/main/apis/chat.md
Fuente Facebook: https://developers.facebook.com/docs/graph-api/reference/object/comments/

## Mensajes vs. comandos

La casilla de texto acepta `!comando` o `/comando` y envía el texto elegido. **Esto no equivale a ejecutar acciones de moderación oficiales**: los comandos solo funcionan si el chat o bot de esa plataforma los entiende. Para ban, timeout, borrar mensajes, gestionar títulos/escenas, etc., harán falta APIs de moderación y permisos independientes. No se implementa interpretación oculta ni ejecución de comandos del sistema.

## Observabilidad y garantías

`GET /api/outbound/status` devuelve `demo`, `live` o `disabled` más preparación aproximada de cada destino, no estado real garantizado. `POST /api/outbound/send` devuelve para cada objetivo: `simulated`, `accepted`, `unavailable` o `error` y su detalle. Los códigos HTTP 401/403/429 indican sesión, permiso o límite.

Tests locales: fanout de 5 plataformas en demo, simulación sin solicitudes externas, selección individual, rechazo de selección duplicada, bloqueo con origen ajeno, bloqueo de envíos reales por defecto, formatos HTTP oficiales simulados de YouTube/Facebook/Kick. Validación en streams reales: **pendiente**.

El módulo se integrará luego en la aplicación mayor **SolarisPKN-LiveTools**, con panel de cuentas, controles de moderación y cola de comandos. No se tocaron las transmisiones activas durante estas pruebas.
