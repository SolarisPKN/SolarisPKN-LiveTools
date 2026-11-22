# OBS + SolarisPKN MultiChat: inicio/parada y búsqueda automática de chats

## Opción recomendada: OBS WebSocket 5 integrado (sin nueva DLL)

El motor Go soporta OBS WebSocket v5, incluido en OBS Studio moderno, mediante conexión **exclusiva a 127.0.0.1**, no expuesta a la red.

1. En OBS, abrí **Herramientas → Ajustes del servidor WebSocket** (el nombre puede variar según la versión). Habilitá el servidor, anotá el puerto (habitualmente `4455`) y definí una contraseña segura.
2. En Windows, cargá la contraseña en la variable de entorno del **mismo usuario que ejecuta MultiChat**, por ejemplo `MULTICHAT_OBS_WS_PASSWORD`. Nunca la guardes en JSON o dentro de Git. Para una consola PowerShell de prueba: `$env:MULTICHAT_OBS_WS_PASSWORD = Read-Host "Clave OBS WebSocket"`.
3. Para funcionamiento real, utilizá una **copia privada de configuración**, no el archivo `demo-config.json`:

```json
{
  "listen": "127.0.0.1:8766",
  "obs": {
    "enabled": true,
    "websocket_enabled": true,
    "websocket_port": 4455,
    "websocket_password_env": "MULTICHAT_OBS_WS_PASSWORD",
    "start_collectors_on_stream": true,
    "scan_all_configured": false
  }
}
```

4. Habilitá expresamente las plataformas y sus canales, OAuth/token/IDs en la misma configuración. No actives `outbound.enabled` hasta tener el login del panel y permisos de escritura. Arrancá `MultiChat-native-engine.exe -config NativeUI/config.local.json -quiet` **sin** `-demo`. Luego abrí `MultiChat-Native.exe`, que se conectará al motor ya iniciado en 8766.
5. Si `panel_auth.enabled` está activo, configurá un secreto de al menos 32 caracteres exclusivo para `panel_auth.native_token_env: "MULTICHAT_NATIVE_TOKEN"` y definilo también en el entorno de la interfaz. El secreto web, el secreto OBS para leer mensajes y el secreto de UI nativa deben ser distintos.

### Qué detecta realmente

Al conectarse, OBS WebSocket 5 realiza handshake con desafío/contraseña si corresponde, recibe el estado actual del directo (**incluso si MultiChat abrió después**) y consulta **el servicio del stream principal de OBS**. Solo extrae la plataforma reconocida del nombre del servicio; **no guarda ni registra claves RTMP/stream keys ni direcciones privadas**.

- OBS está apagado/desconectado: reintenta la conexión local con backoff de 2 a 30 segundos.
- OBS empieza directo: activa los **lectores explícitamente habilitados** y asociados con el servicio detectado, solo para esa plataforma por defecto. Cada lector busca el LIVE/chat correspondiente según su usuario, canal o Página; tiene reconexión y reintentos propios. El indicador verde se activa solo si el lector confirma conexión o una solicitud API exitosa.
- OBS termina directo: cancela los lectores de esa sesión y apaga los LEDs; no detiene OBS ni el stream.
- OBS cambia el servicio al iniciar otro directo: recalcula la plataforma asociada, sin reutilizar IDs o estados de otro stream.
- Para OBS con **múltiples salidas** (plugins múltiples RTMP, TikTok simultáneo, etc.), configurá `"scan_all_configured": true` para revisar **todas las plataformas habilitadas**. No significa que estén todas en directo: solo el conector correspondiente, tras confirmar un LIVE, mostrará verde. El WebSocket del OBS principal no garantiza descubrir las salidas gestionadas por plugins externos.

El cliente de OBS WebSocket está **probado con un servidor simulado** (autenticación, estado inicial, plataforma y evento de finalización). **No se probó con OBS real en la PC del usuario**.

## Opción alternativa: módulo nativo de OBS (código fuente)

`MultiChat/OBSPlugin/obs-multichat.cpp` es un módulo C++ que implementa una fuente de chat **GDI+ sin Browser Source** y ahora escucha eventos `OBS_FRONTEND_EVENT_STREAMING_STARTED` / `OBS_FRONTEND_EVENT_STREAMING_STOPPED`. Puede enviar eventos mediante POST local a `/api/obs/event` con una credencial diferente definida en `MULTICHAT_OBS_EVENT_TOKEN`; no transmite stream keys. Envía un pequeño heartbeat cada 15 s mientras hay directo para recuperarse si MultiChat se abre después.

Si se elige esta opción, en la configuración privada del motor:
```json
"obs": {
  "enabled": true,
  "token_env": "MULTICHAT_OBS_EVENT_TOKEN",
  "start_collectors_on_stream": true,
  "scan_all_configured": false
}
```

El compilado requiere **libobs + OBS Frontend API** de desarrollo y CMake. El código fuente y el enlace de CMake están preparados, pero **no se compiló ni instaló la DLL en OBS**, porque no se encontró el SDK en las rutas habituales. No confundas esta fuente con el cliente WebSocket integrado, que es una alternativa ya compilada dentro del motor Go.

## Seguridad

Los endpoints locales no aceptan Host ajenos a localhost. El endpoint `/api/obs/event` exige token de evento exclusivo de más de 32 caracteres, formato JSON y cabecera de acción. La UI nativa utiliza un token distinto y sólo para estado/lectura/envío. El motor demo ignora todas las conexiones externas.

El proceso no toma decisiones de transmitir, cambiar escenas, crear directos, iniciar OBS ni publicar en plataformas sin una acción explícita de la persona. El usuario controla plataformas habilitadas y permisos. Un error HTTP 200 de lectura de comentarios demuestra acceso a ese chat, no necesariamente que haya espectadores o que el usuario tenga permiso de publicar.

Referencia: [API frontend oficial de OBS](https://docs.obsproject.com/reference-frontend-api).
