# MultiChat: estado de conectores (21/11/2026)

**Prioridad acordada:** YouTube > TikTok > Twitch > Kick > Facebook.

| Plataforma | Implementación | Verificado | Pendiente |
|---|---|---|---|
| YouTube | `liveChatMessages.list` con opción de descubrir `liveChatId` desde `videos.list` usando `video_id` | Compilación y pruebas unitarias | API key y directo real; evaluar `streamList` para mejorar latencia/cuota |
| TikTok | Cliente Go público Webcast, sin login; dependencia `github.com/PirateTok/live-go` (0BSD) | Parser y build local | Evento real, resistencia a cambios de TikTok |
| Twitch | IRC SSL/TLS con OAuth, PING/PONG, tags y reconexión | Parser y build local | Cuenta y directo real |
| Kick | Cliente Centrifugo por WebSocket, con JWT de invitado y búsqueda de chatroom | Parser y build local | Negociación guest y mensajes reales, posible bloqueo por CDN |
| Facebook | Meta Graph API para comentarios de video LIVE de Página | Parser y build local | Token de Página con permisos, chat real y cuota |
| Rumble | API Live Stream privada | Código y tests base existentes | Cuenta y directo real |
| Adaptadores | Ingesta Bearer loopback | Test local | Puentes adicionales |

## Fuentes técnicas

- YouTube: https://developers.google.com/youtube/data/live/reference/rest/v3/liveChatMessages/list
- Video ID a chat ID: https://developers.google.com/youtube/data/reference/rest/v3/videos
- Twitch IRC: https://dev.twitch.tv/docs/chat/irc/
- TikTok Go público no oficial: https://github.com/PirateTok/live-go
- Kick migración Pusher->Centrifugo desde octubre de 2026: https://github.com/AnisAbdellatif/kicksdk/blob/main/docs/KICK.md
- Kick webhook **oficial** `chat.message.sent` si se diseña un relay público HTTPS con firma: https://github.com/KickEngineering/KickDevDocs/blob/main/events/webhook-security.md
- Meta Graph LIVE comments: https://developers.facebook.com/docs/graph-api/reference/live-video/comments/

## Seguridad y políticas

1. Todo está desactivado salvo habilitación explícita en configuración.
2. El servidor local solo escucha en loopback.
3. YouTube, Facebook y Twitch requieren credenciales proporcionadas por el usuario mediante variables de entorno; nunca incluirlas en Git.
4. TikTok y Kick son **no oficiales**: no simulan una garantía de compatibilidad ni usan cookies de sesión guardadas.
5. No hay automatización de publicación ni acciones de moderación en estos conectores.
6. Tests de parsers, deduplicación, HTTP y compilación **no equivalen** a un directo real validado.
