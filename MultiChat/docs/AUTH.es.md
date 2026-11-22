# SolarisPKN MultiChat — acceso seguro y autorización de plataformas

Estado: 21 de noviembre de 2026. Código probado con simulaciones locales en Windows. **No se ha autorizado ninguna cuenta real**, ni se deben ingresar contraseñas de redes sociales en MultiChat.

## A. Bloqueo del visor local

MultiChat escucha solo en `127.0.0.1:8765` por defecto. Configurá:

```json
"panel_auth": {
  "enabled": true,
  "token_env": "MULTICHAT_PANEL_TOKEN",
  "obs_token_env": "MULTICHAT_OBS_TOKEN"
}
```

Generá **dos** claves distintas de 256 bits con PowerShell:

```powershell
function NuevaClaveMultiChat {
  $bytes = New-Object byte[] 32
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  $rng.GetBytes($bytes)
  $rng.Dispose()
  [Convert]::ToBase64String($bytes)
}
$env:MULTICHAT_PANEL_TOKEN = NuevaClaveMultiChat
$env:MULTICHAT_OBS_TOKEN = NuevaClaveMultiChat
```

Las claves solo viven en la sesión de PowerShell que lanza el programa, a menos que el usuario **explícitamente** elija otro almacén. No pegar estos valores en GitHub, logs, capturas ni chats.

Al abrir `http://127.0.0.1:8765/` se solicita la clave del panel. Después se crea una cookie HttpOnly SameSite Strict con vida máxima de ocho horas, almacenada solo en memoria mientras corre MultiChat. El botón **Salir** invalida la sesión. La clave del panel protege mensajes, SSE y el visor; `/health` permanece sin contenido de chat y `POST /api/ingest` conserva su token independiente.

El complemento nativo experimental OBS usa **solo** `MULTICHAT_OBS_TOKEN` y `GET /obs/messages`, no puede leer el historial completo ni acceder al panel. El proceso OBS debe heredar esa variable del entorno al iniciarse. **La DLL nativa aún debe compilarse/probarse.** La fuente Browser Source puede iniciar sesión mediante la ventana de interacción de OBS; hay que verificar la persistencia de su cookie con la versión instalada.

Un proceso local con el mismo usuario y suficiente privilegio puede acceder a la memoria/entorno de otros procesos; el bloqueo protege al visor de accesos accidentales y procesos locales sin credenciales, **no** reemplaza el aislamiento del sistema operativo. No exponer el puerto en redes, proxies o túneles sin una capa de autenticación adicional y HTTPS.

## B. Google / YouTube — OAuth de solo lectura

Crear credenciales de tipo **Desktop app** en Google Cloud y habilitar YouTube Data API v3. Se usa el scopes de lectura `https://www.googleapis.com/auth/youtube.readonly` y envío de chat `https://www.googleapis.com/auth/youtube.force-ssl`, `state` aleatorio y PKCE `S256`, con un callback local temporal.

```powershell
$env:MULTICHAT_GOOGLE_CLIENT_ID = "ID.apps.googleusercontent.com"
$env:MULTICHAT_GOOGLE_CLIENT_SECRET = "CLIENT_SECRET_DE_GOOGLE_CLOUD"
.\MultiChat.exe -link-google
```

Solo ese comando abre la autorización en el navegador. El consentimiento pertenece al usuario. El token de actualización se cifra mediante DPAPI del **usuario Windows que inició el programa**, no con una contraseña de la red social.

En `config.json`:

```json
"youtube": {
  "enabled": true,
  "live_chat_id": "",
  "video_id": "",
  "use_saved_google_auth": true,
  "api_key_env": "MULTICHAT_YOUTUBE_API_KEY",
  "oauth_token_env": "MULTICHAT_YOUTUBE_OAUTH_TOKEN",
  "poll_seconds": 20,
  "daily_quota_units": 6000
}
```

Con `use_saved_google_auth:true` **no necesitás** configurar API key ni un token pegado en entorno. `liveBroadcasts.list(mine=true,broadcastStatus=active)` localiza el chat; MultiChat refresca el token cuando queda menos de dos minutos de vigencia. Para revocar **localmente** la credencial: `.\MultiChat.exe -unlink-google`; para revocar permisos reales, usar la sección de aplicaciones de la Cuenta Google.

**No se inicia ninguna transmisión, no se publica contenido y no se envían mensajes.** No se ha probado todavía contra la cuenta real ni aprobado la aplicación OAuth en Google Cloud.

Fuentes oficiales:
- https://developers.google.com/identity/protocols/oauth2/native-app
- https://developers.google.com/youtube/data/live/authentication

## C. Twitch — Device Authorization

Twitch permite una aplicación OAuth pública por **Device Code Grant**, sin copiar contraseñas ni usar un secreto de cliente para ese flujo. Registrar una aplicación de Twitch y obtener su Client ID; pedir `chat:read` para recibir mensajes y `user:write:chat` para enviar mensajes explícitos.

```powershell
$env:MULTICHAT_TWITCH_CLIENT_ID = "CLIENT_ID_DE_TWITCH"
.\MultiChat.exe -link-twitch
```

El programa muestra una URL de activación oficial y un código temporal que debés autorizar en Twitch. Valida que la autorización obtenida incluya `chat:read` y guarda el acceso/refresh cifrado con DPAPI para el mismo usuario de Windows. Los refresh tokens de Twitch pueden rotar y el programa almacena el nuevo token en cada renovación.

En `config.json`:

```json
"twitch": {
  "enabled": true,
  "channel": "solarispkn",
  "nick": "solarispkn",
  "use_saved_twitch_auth": true,
  "token_env": "MULTICHAT_TWITCH_TOKEN"
}
```

Con autenticación guardada, el programa utiliza el nombre de la cuenta realmente autorizada, aun si difiere del `nick` configurado. Para borrar el vínculo local: `.\MultiChat.exe -unlink-twitch`. Para revocarlo también en Twitch, usar la configuración de conexiones de Twitch.

Fuente oficial: https://dev.twitch.tv/docs/authentication/getting-tokens-oauth/#device-code-grant-flow

## D. Meta/Facebook, Kick y TikTok

- **Facebook Live**: por ahora necesita un **token autorizado de Página** (`MULTICHAT_FACEBOOK_PAGE_TOKEN`) y `page_id` para detectar la emisión activa o `live_video_id` específico. Configuración de Meta Login, permisos y renovación automática: pendiente. **No** usar contraseñas ni cookies personales.
- **Kick**: el lector de chat público Centrifugo no requiere login; integración no oficial pendiente de prueba real.
- **TikTok**: el lector Webcast público usa el nombre de usuario sin iniciar sesión; integración no oficial pendiente de prueba real.

## E. Estado y verificación

Se prueban, en local: bloqueo de history/SSE/OBS, token OBS de solo lectura, login/logout, antirrobo de sesión por origen cruzado, cifrado DPAPI, rechazo de vinculaciones sin Client ID y compilación. **No se ha completado OAuth real, autorización real de Meta ni instalación/verificación de la DLL de OBS**. Todo permiso se pide únicamente cuando el usuario lo inicia expresamente.

La aplicación principal SolarisPKN-LiveTools podrá integrar en el futuro un gestor central de identidades, botones de conexión/desconexión y estado de cada plataforma sin reimplementar todo el código del chat.
