# MultiChat — SolarisPKN LiveTools

**Un solo motor de consola, un único chat unificado y nada de navegadores por plataforma.** Aplicación Go 1.22+ con librería estándar, sin Chromium/Electron. Los conectores comparten memoria y un visor local opcional para OBS.

## Conectores

| Plataforma | Implementación | Estado |
|---|---|---|
| Twitch | IRC con OAuth sobre TLS, lectura de chat y reconexión | Código listo; falta probar con cuenta real |
| YouTube Live | API oficial `liveChatMessages.list`; intervalos y presupuesto local de cuota | Código listo; falta probar en directo |
| Rumble | API oficial de livestream, consulta de mensajes recientes | Código listo; falta probar en directo |
| Integraciones adicionales | `POST /api/ingest` autenticado en loopback | Probado localmente |
| Kick, TikTok, CHZZK, Bilibili, SOOP, FC2, Vaughn Live y otras | — | **Aún no implementadas** |

## Instalar en Windows

1. Instalá [Go](https://go.dev/dl/) 1.22+ (necesario para compilar; un ejecutable compilado no requiere Go).
2. En PowerShell, desde la carpeta `MultiChat`:

```powershell
Copy-Item config.example.json config.json
go test ./...
go build -trimpath -ldflags="-s -w" -o MultiChat.exe .
.\MultiChat.exe -config config.json -demo
```

También se incluye `iniciar-multichat.cmd`, que usa el EXE local o `go run`. El modo `-demo` **solo muestra mensajes simulados**, no se conecta a servicios reales. Visor: **http://127.0.0.1:8765**. Con `-quiet` dejás el motor sin imprimir mensajes en terminal. El historial es únicamente en memoria, no se guarda en disco.

## Configuración

Todos los conectores empiezan con `enabled: false`. Copiá y editá `config.example.json` como `config.json` (ignorado por Git). Definí las credenciales en variables de entorno:

```powershell
$env:MULTICHAT_TWITCH_TOKEN = 'token-con-chat-read'
$env:MULTICHAT_YOUTUBE_API_KEY = 'clave-api'
$env:MULTICHAT_RUMBLE_API_URL = 'https://rumble.com/TU-URL-PRIVADA-API'
$env:MULTICHAT_INGEST_TOKEN = 'clave-aleatoria-muy-larga-de-24-caracteres-o-mas'
```

- **Twitch:** canal, nombre de usuario y OAuth con permiso de lectura `chat:read`.
- **YouTube:** `liveChatId` de una emisión en vivo, API key y intervalo de consulta; el costo y límites los determina Google. El presupuesto local no persiste entre reinicios; migrar a `streamList` es una mejora pendiente.
- **Rumble:** URL privada que proporciona la plataforma. No publicarla.
- **Integraciones:** `POST /api/ingest` apagado por defecto, únicamente para programas locales autorizados.

## API local

`GET /health` — estado sin secretos; `GET /history` — historial JSON; `GET /events` — SSE en tiempo real; `POST /api/ingest` — ingesta opcional con Bearer y JSON:

```json
{"platform":"custom","id":"mensaje-1","user":"Gamer","text":"¡Hola!","kind":"chat"}
```

Para activar ingesta: `"ingest":{"enabled":true,"token_env":"MULTICHAT_INGEST_TOKEN"}`. El receptor tiene tamaño limitado y exige Bearer con token de 24+ caracteres.

## Documentación completa

- [Arquitectura y consumo de recursos](docs/ARCHITECTURE.es.md)
- [Estado y hoja de ruta de los conectores](docs/CONNECTORS.es.md)
- [Seguridad, privacidad y límites](docs/SECURITY.es.md)
- [OBS y solución de problemas](docs/OBS.es.md)

## Tests

```powershell
go test ./...
go vet ./...
```

**Importante:** no se versionan binarios, claves, logs ni configuraciones locales. El programa no envía respuestas a chats ni realiza moderación remota. Las pruebas con cuentas reales y la medición prolongada de CPU/RAM están pendientes.
