# Estado de las plataformas

**Implementado en el MVP (sin validación con cuentas reales):** Twitch IRC/TLS, YouTube Live `liveChatMessages.list`, Rumble Live Stream API. Entrada local de adaptadores disponible mediante Bearer `/api/ingest` (no equivale a un conector completo).

**Pendiente de desarrollar:** Kick (webhooks oficiales y verificación de firma), TikTok LIVE (alternativas no oficiales sujetas a cambios y condiciones), CHZZK/Naver (Socket.IO y autorización), Bilibili (API de Live), FC2 Live (comentarios), Vaughn Live (WebSocket/IRC según acceso), SOOP (API/SDK), Nimo TV, Steam Broadcast, Dailymotion, Naver TV clásico, Facebook Live e Instagram Live. Nonolive se posterga hasta confirmar que el servicio siga operativo.

## Reglas para cada conector

1. Usar transportes oficiales HTTPS, WebSocket, IRC o SSE antes que una instancia de navegador.
2. Respetar intervalos, cuotas y `Retry-After`, con reconexión progresiva.
3. No registrar ni subir tokens, cookies o endpoints privados.
4. Validar eventos de origen antes de inyectarlos al hub; los webhooks de Kick requieren verificación de firma HTTPS pública en un puente separado.
5. Probar primero payloads sintéticos y después un live real autorizado. Documentar límites y consumo.

Documentación de implementaciones actuales: [Twitch IRC](https://dev.twitch.tv/docs/chat/irc/), [YouTube](https://developers.google.com/youtube/v3/live/docs/liveChatMessages/list), [Rumble](https://rumble.support/help/how-to-use-rumble-s-live-stream-api).
