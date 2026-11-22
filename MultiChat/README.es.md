# SolarisPKN MultiChat — LiveTools

Chat unificado local en Go, sin Electron ni un navegador por plataforma. Conectores independientes, deduplicación acotada, API en loopback, visor web opcional y soporte para una fuente nativa de OBS (código C++ separado, todavía sin binario verificado).

## Plataformas (prioridad 21/11/2026)

| Plataforma | Integración local | Validación |
| --- | --- | --- |
| YouTube Live | API oficial `liveChatMessages.list`; ID de chat manual o descubrimiento automático mediante `videos.list` por ID del directo | Tests Go y compilación OK; falta directo real |
| TikTok LIVE | Webcast público no oficial por `github.com/PirateTok/live-go`, sin cookies ni navegador | Tests de normalización OK; falta directo real |
| Twitch | IRC/TLS OAuth `chat:read`, tags, reconexión y respuesta a PING | Tests de parsing OK; falta directo real |
| Kick | Realtime **Centrifugo** anónimo (nuevo protocolo desde octubre de 2026), sin clave Pusher vieja | Tests de parsing OK; handshake real pendiente |
| Facebook Live | API Graph de Meta `/{live-video-id}/comments` con Page Access Token | Tests de parsing OK; permisos y directo real pendientes |
| Rumble | API de livestream original | No probado contra una cuenta real |
| Adaptadores externos | `POST /api/ingest` con Bearer, protegido y solo en loopback | Test HTTP local OK |

Todos los conectores empiezan con `enabled: false`. **Compilar y superar los tests no equivale a tener cuentas conectadas.** No se instalaron tokens ni se iniciaron transmisiones.

### Descubrimiento automático de directos

- **YouTube:** si dejás `youtube.video_id` y `youtube.live_chat_id` vacíos y proporcionás `youtube.oauth_token_env`, MultiChat consulta la emisión activa de la cuenta y cambia de chat cuando cambiás de directo (aproximadamente cada minuto). Un token OAuth de propietario es indispensable; una API key sola no puede consultar emisiones privadas del dueño. **Renovación automática del token: pendiente.**
- **Facebook:** si proporcionás `facebook.page_id` (el identificador de tu Página, no del directo) y dejás `facebook.live_video_id` vacío, MultiChat detecta el video con estado LIVE_NOW y utiliza sus comentarios. Exige un token de Página con permisos válidos.
- **Twitch y Kick:** `solarispkn` es el canal estable; no se guardan UUID de panel ni URLs por directo.
- **TikTok:** el usuario es estable, la sala LIVE cambia; el conector público intenta descubrirla sin pegar una URL nueva.

`config.solarispkn.example.json` contiene los datos de tus URLs como ejemplo, **desactivado** y sin secretos. El ID de Facebook extraído del panel de Producer es un candidato no validado por Graph API.

Más detalles: [Mapa de URLs e IDs](docs/CHAT-IDS-SOLARISPKN.es.md).

## Instalar/ejecutar (Windows)

Requiere Go **1.23+** para compilar; el `.exe` ya compilado no requiere Go.

```powershell
cd D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-LiveTools\MultiChat
Copy-Item config.example.json config.json
go test ./...
go vet ./...
go build -trimpath -ldflags="-s -w" -o MultiChat.exe .
.\MultiChat.exe -config config.json -demo
```

Modo `-demo` solo crea mensajes ficticios y **no se conecta** a servicios reales. Para live, quitar `-demo` y configurar `enabled: true` únicamente en los conectores que quieras.

Visor web local: `http://127.0.0.1:8765/`.

## Configurar los cinco conectores principales

1. **YouTube:** `youtube.video_id` = ID de 11 caracteres del video LIVE, o `youtube.live_chat_id` explícito; `MULTICHAT_YOUTUBE_API_KEY` en entorno. El video debe tener chat activo. Puede consumir cuota y los permisos los determina Google.
2. **TikTok:** `tiktok.username` = tu usuario público, sin `@`. No requiere login; Webcast es un protocolo **no oficial** y puede dejar de funcionar si TikTok cambia su sistema.
3. **Twitch:** `twitch.channel`, `twitch.nick` y token OAuth de lectura en `MULTICHAT_TWITCH_TOKEN`.
4. **Kick:** `kick.channel` para resolver el ID de chat, o `kick.chatroom_id` numérico si el endpoint del canal está restringido. La conexión guest Centrifugo es **no oficial** y sujeta a cambios.
5. **Facebook:** `facebook.live_video_id` es el ID numérico del video en vivo de una Página; `MULTICHAT_FACEBOOK_PAGE_TOKEN` es un token de Página válido, con permisos para leer los comentarios. No soporta perfiles personales sin acceso concedido por Meta.

Ejemplo de secretos solo en la sesión actual (no escribir claves en el repo):

```powershell
$env:MULTICHAT_YOUTUBE_API_KEY = '...'
$env:MULTICHAT_TWITCH_TOKEN = 'oauth:...'
$env:MULTICHAT_FACEBOOK_PAGE_TOKEN = '...'
```

## Autenticación del visor y de las cuentas

La configuración opcional `panel_auth.enabled` añade un login local (cookie de sesión HttpOnly, SameSite Strict, sin almacenar la clave en el repo) y un token separado para que OBS lea solamente `/obs/messages`. La muestra `config.solarispkn.example.json` activa el bloqueo de forma intencional: requiere configurar previamente dos variables de entorno con claves distintas, de al menos 32 caracteres.

Con OAuth, **YouTube** dispone de `-link-google` (escritorio Google, PKCE, permiso `youtube.readonly`) y **Twitch** de `-link-twitch` (código de dispositivo, permiso `chat:read`). Ambos guardan el token cifrado para el usuario Windows con DPAPI y tienen renovación automática; jamás guardan contraseñas de las plataformas. En configuración podés habilitar `youtube.use_saved_google_auth` y `twitch.use_saved_twitch_auth` después de autorizar las cuentas. Para borrar los vínculos locales existen `-unlink-google` y `-unlink-twitch`.

La autorización con Google y Twitch necesita registrar sus propias aplicaciones OAuth/Client ID y consentir expresamente desde el navegador; las rutas se verificaron con pruebas aisladas, **pero todavía no se probaron sobre cuentas reales**. Facebook sigue usando un token autorizado de Página; los chats públicos de Kick/TikTok no necesitan login actualmente.

**Instrucciones completas, comandos para crear las claves y límites:** [Autenticación de MultiChat](docs/AUTH.es.md). La aplicación principal futura está descrita en [Plan SolarisPKN-LiveTools](docs/LIVETOOLS-APP-PLAN.es.md).

## Consola de envío de mensajes y comandos

Desde el chat podés abrir la ventana **Enviar / Comandos** (o directamente `/compose`). Incluye cinco destinos —YouTube, TikTok, Twitch, Kick y Facebook— con checkboxes, **Seleccionar todas**, **Deseleccionar todas** y **Solo** para dirigir un mensaje a una plataforma. El resultado de cada destino aparece por separado.

El modo `-demo` simula todos los envíos sin publicar nada, y muestra el resultado de prueba en el historial. El envío real está **desactivado por defecto** y exige `outbound.enabled=true`, autenticación del panel y los permisos de escritura de cada red. YouTube, Twitch, Facebook y Kick tienen adaptadores HTTP oficiales para publicar mensajes; TikTok aún se mantiene solo en lectura. Los comandos enviados como texto (por ejemplo `!redes`) **no son acciones administrativas garantizadas**.

Google ahora solicita expresamente el alcance `youtube.force-ssl` para publicar junto con el de lectura; Twitch requiere `user:write:chat` además de `chat:read`. Si tenés un OAuth de solo lectura, debe autorizarse nuevamente. No se creó ninguna autorización de cuenta real durante el desarrollo.

Detalles completos: [Consola de envío](docs/OUTBOUND.es.md).

## API y OBS

- `GET /health`: conectores habilitados, historial y salud básica (no confirma autenticación remota)
- `GET /history`: mensajes en memoria
- `GET /events`: eventos SSE
- `GET /obs/messages?limit=8`: JSON limitado a 1–16 mensajes para `OBSPlugin/obs-multichat.cpp`
- `POST /api/ingest`: ingesta autenticada por Bearer si se habilita

OBS puede abrir la fuente de navegador del chat local; además hay una fuente **GDI+ nativa** en `OBSPlugin/` (falta compilarla y probarla con el SDK/versión instalada de OBS). No instalar una DLL sin compatibilidad verificada.

## Auditoría y límites

- No persistimos historial: se pierde cuando se cierra el proceso.
- Los mensajes se limitan, deduplican y distribuyen sin bloquear los recolectores.
- Todavía faltan tests de extremo a extremo con las cinco plataformas, telemetría de desconexiones, UI de configuración y validación del plugin nativo.
- **SceneSync** (control de escenas de TikTok LIVE Studio desde OBS) es un componente separado y experimental: se descargó para revisión el proyecto MIT `dannguyen9x/tiktok-live-studio-mcp` junto a LiveTools, sin conectarlo aún al LIVE Studio real.

Detalles: `docs/ARCHITECTURE.es.md`, `docs/OBS.es.md`, `docs/CONNECTORS.es.md`.
