# SolarisPKN MultiChat — cómo identificar el chat de cada directo

Auditoría del 21/11/2026. Estos enlaces sirven para configurar el chat; no prueban que el directo siga activo.

## Matriz de enlaces y estabilidad

| Plataforma | Enlace entregado | Identificador útil | Duración |
| --- | --- | --- | --- |
| YouTube | `studio.youtube.com/live_chat?is_popout=1&v=hgszxU2CTEs` | `video_id=hgszxU2CTEs`; a partir de él se obtiene `activeLiveChatId` | **Por emisión**, cambia en cada video |
| Twitch | `dashboard.twitch.tv/popout/u/solarispkn/stream-manager/chat?uuid=...` | `channel=solarispkn`; no usar `uuid` del panel | **Por canal**, estable |
| Kick | `kick.com/popout/solarispkn/chat` | `channel=solarispkn`, resolver chatroom dinámicamente | **Por canal**, normalmente estable |
| Facebook Live Producer | `facebook.com/live/producer/dashboard/964285429841619/COMMENTS/` | `964285429841619` es **candidato** a `live_video_id`: validar por Graph API | **Por video LIVE** |
| TikTok | El usuario no proporcionó una URL | `username` de la cuenta LIVE, por ejemplo a partir de `tiktok.com/@USUARIO/live` | **Username estable; room_id dinámico** |

La URL del administrador de Twitch contiene un identificador de la ventana/dock; no es necesaria para el chat IRC y no debe guardarse como un identificador permanente de emisión.

## YouTube: manual y automático

**Por transmisión:** especificar `youtube.video_id="hgszxU2CTEs"` y una API key en `MULTICHAT_YOUTUBE_API_KEY`. El recolector obtiene `activeLiveChatId` usando `videos.list`. Esta modalidad no detecta automáticamente el **siguiente** video.

**Auto-cambio entre transmisiones:** dejar `youtube.video_id=""` y `youtube.live_chat_id=""`, con `youtube.oauth_token_env="MULTICHAT_YOUTUBE_OAUTH_TOKEN"` apuntando a un OAuth válido del propietario del canal. Se consulta `liveBroadcasts.list?broadcastStatus=active&mine=true` cada ~60 s y se actualiza el chat y el cursor al cambiar de emisión. Esto exige autorización OAuth; no basta la API key. **La renovación de tokens OAuth todavía no está implementada**, por lo que el modo automático no será permanente sin renovar el token.

Referencia: https://developers.google.com/youtube/data/live/reference/rest/v3/liveBroadcasts/list

## Facebook: manual y automático

**Por transmisión:** configurar `facebook.live_video_id="964285429841619"` **solo si Graph API confirma que corresponde al video**; requiere token autorizado de Página.

**Por Página, autodescubrimiento:** dejar `facebook.live_video_id=""` y configurar `facebook.page_id="ID_NUMERICO_DE_LA_PAGINA"`. El recolector solicita `/{page-id}/live_videos`, elige un objeto con estado `LIVE_NOW` y usa su ID para `/{live-video-id}/comments`. Reconsulta aproximadamente cada minuto para reconocer otro directo. Requiere permisos vigentes del token de Página. El número de la URL de Producer **no está verificado como Page ID**; no usarlo en `page_id` sin comprobarlo.

Referencias:
- https://developers.facebook.com/docs/graph-api/reference/page/live_videos/
- https://developers.facebook.com/docs/graph-api/reference/live-video/comments/

## Twitch / Kick

Twitch: usar `twitch.channel="solarispkn"` con cuenta y OAuth de lectura IRC. No necesita UUID de dashboard. Su sala persiste cuando terminás y reiniciás un directo.

Kick: usar `kick.channel="solarispkn"`, cuya popout es fija. El puente experimental resuelve el ID de chatroom y se conecta a Centrifugo; es **no oficial** y requiere verificación en un directo real.

Referencias:
- https://dev.twitch.tv/docs/chat/irc/
- https://help.kick.com/en/articles/7978574-how-to-create-a-kick-chat-dock-in-obs

## TikTok

Usar el nombre de usuario visible en el perfil, **sin arroba**. El conector Webcast de `live-go` intenta localizar la sala que esté transmitiendo y se reconecta tras cierre/error. No necesita pegar una URL de cada LIVE. Como este acceso es **no oficial**, TikTok puede cambiarlo.

Código de referencia: https://github.com/PirateTok/live-go

## Archivo listo para completar

`config.solarispkn.example.json` incorpora los datos no secretos de los enlaces anteriores con todos los conectores apagados. El nombre de TikTok queda como marcador hasta confirmarlo. No iniciar el motor de producción con IDs de ejemplo ni subir tokens, cookies o configuraciones `config.json` a Git.

Los conectores existentes siguen siendo de **solo lectura**. Integración real, autorización y continuidad de acceso siguen pendientes.
