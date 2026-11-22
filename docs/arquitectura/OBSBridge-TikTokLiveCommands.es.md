# SolarisPKN OBSBridge + TikTokLiveCommands — Diseño v0.1

Estado: **especificación de arquitectura, no implementado ni probado en máquinas reales**. Rama: `feat/obsbridge-tiktok-livecommands`.

## Objetivo
Controlar una sesión de TikTok LIVE Studio alojada en una segunda PC (posiblemente una PC corporativa **solo con autorización de su organización**) desde OBS y SolarisPKN-LiveTools en la PC principal. Integrar PreStream, escenas, acciones manuales, telemetría, mapeos de escenas y confirmaciones. La conectividad de comandos es independiente del transporte de audio/video.

## Componentes
1. **SolarisPKN-OBSBridge**: integración OBS Studio. Primera fase: OBS WebSocket (v5, OBS 28+) más dock local y script Lua para hotkeys/acciones que OBS no expone automáticamente. Fase posterior: plugin nativo si hace falta una UX más integrada. Detecta eventos de escenas/stream/recording y envía comandos al Core; recibe comandos de Core para OBS.
2. **SolarisPKN-LiveTools Core**: fuente de verdad para perfiles PreStream, reglas, asignación de dispositivos, autorización, deduplicación, historial y estado deseado/observado. Router de mensajes con IDs de correlación, logs de estados y confirmación.
3. **SolarisPKN-TikTokLiveCommands.exe**: agente Windows portable, sin administrador ni dependencia de Node/Python en producción; candidato Go como MultiChat. Enlace saliente autenticado/cifrado al Core mediante VPN autorizada o relay WSS, nunca publica la interfaz local de TikTok ni acceso genérico al equipo. Descubre la sesión LIVE Studio, identifica escenas, aplica acciones aprobadas, valida estado posterior y envía acuses. Icono en bandeja, pairing/revocación, diagnóstico y reconexión.
4. **TikTok LIVE Studio Adapter**: prioridad a canal local tipo Stream Deck Socket.IO (`streamdeck_ttls_v1`) detectando puerto propiedad del proceso local y comprobando compatibilidad; respaldo mediante hotkeys de LIVE Studio, y como último recurso automatización GUI asistida si el usuario la habilita. No hay una API pública general de control remoto garantizada; todo adaptador se versiona y se debe validar después de cada actualización.

## Contrato propuesto
- Protocolo `livetools.remote.v1` sobre WSS con identidad de dispositivo, pairing por código de uso único, permisos por acción.
- Mensaje de comando: `{id, origin, target, action, params, issued_at, expires_at, profile_revision}`; estado `queued | accepted | applied | rejected | timeout | unverified`.
- Acciones base: `scene.list`, `scene.set`, `source.visibility.set`, `mic.mute.set`, `desktop_audio.mute.set`, `recording.start`, `recording.stop`, `studio.status`, `live.status`.
- `live.start` y `live.stop` requieren confirmación **local explícita**, permisos elevados por acción y no se reintentan ciegamente. Nunca arranque del directo en una PC remota por una transición de escena.
- No enviar scripts, comandos shell, pulsaciones arbitrarias ni rutas de archivos remotas; permitir solo comandos tipados y listas de escenas válidas.
- `id` se deduplica en la ventana temporal y `expires_at` impide ejecutar eventos vencidos después de una reconexión. Tareas delicadas no se reproducen al reconectar.
- OBSBridge y agente remoto reportan `observed_state`; LiveTools compara `desired_state` con observado y marca divergencias, no finge éxito con solo recibir ACK.

## Ejemplo de mapeo por perfil
```json
{
  "profile": "roblox-terror",
  "obs_to_tiktok": {
    "Escena Inicial": "Starting Soon Vertical",
    "Terror": "Terror Vertical",
    "Pausa": "BRB Vertical",
    "Final": "Thanks Vertical"
  },
  "sync": {
    "obs_to_tiktok": true,
    "tiktok_to_obs": false,
    "debounce_ms": 500,
    "fallback_on_missing_scene": "alert_only"
  }
}
```
Control bidireccional opcional, sin bucles: comandos incluyen `origin`, `id` y estado esperado. En modo Studio, distinguir escena actual de escena de programa.

## Pruebas por fases
1. Simulador: dos procesos locales, fake OBS + fake TikTok, ACK, desconexiones, duplicados, comandos vencidos y reinicios.
2. OBS real: eventos de escena y hotkeys contra Core local, sin transmisión.
3. LIVE Studio real: detectar versión/puerto, consultar escenas y cambiar a una escena de prueba en modo offline; restaurar escena anterior.
4. Red remota: conectar por relay saliente o VPN autorizada, probar revocación, latencia, reconexión y sincronización.
5. Control de origen: prueba con Stream Deck/OBS, sincronización en cambios rápidos, casos de fallo.
6. Integración PreStream: perfiles y asignación de dispositivos persistentes.

## Red y PC corporativa
- Priorizar una segunda PC personal; en PC de trabajo, **requiere aprobación del empleador**. No tunelizar la red corporativa, desactivar antivirus, alterar firewall, instalar drivers sin permiso ni buscar saltarse políticas.
- Sin conexiones entrantes hacia la PC secundaria; de ser necesario, relay HTTPS/WSS con TLS y permisos restrictivos, no proxy general de red.
- Tokens por dispositivo almacenados con protección local del sistema y posibilidad de revocación.
- En caso de desconexión, no ejecutar órdenes atrasadas.

## Video y audio: problema diferente
Sincronizar escenas y controles **no transporta el video ni el audio** del OBS principal hacia TikTok Studio remoto. Para emitir la misma imagen se necesita un pipeline de media aparte (por ejemplo, OBS → SRT sobre una ruta permitida → receptor local/entrada virtual compatible con LIVE Studio), o fuentes locales equivalentes. Seleccionar el método en una fase separada con pruebas de latencia, carga y políticas de TI.

## Referencias de investigación
- OBS WebSocket integrado desde OBS 28: https://obsproject.com/kb/remote-control-guide
- Scripts/hotkeys de OBS: https://docs.obsproject.com/scripting
- TikTok LIVE Studio plugin oficial para Elgato Stream Deck: https://marketplace.elgato.com/product/tiktok-live-studio-ba05f173-6d92-4b6e-9f23-fbf0d61ee79e
- Exploración comunitaria local de Socket.IO, contrastada contra LIVE Studio 1.33.2 (no es API pública ni garantía de compatibilidad): https://github.com/dannguyen9x/tiktok-live-studio-mcp

## Criterios de aceptación MVP
- Cambiar escena OBS `Terror` produce una orden única y confirmada para TikTok `Terror Vertical`.
- Si TikTok está cerrado, se muestra desconectado; no bloquea OBS.
- Si falta una escena, alerta sin cambiar a una escena incorrecta.
- Un evento enviado dos veces no repite una acción de tipo toggle.
- Las transiciones offline/online no inician ni terminan LIVE por accidente.
- Un dispositivo revocado deja de aceptar órdenes.
- La aplicación identifica claramente el estado real y registra errores legibles.
