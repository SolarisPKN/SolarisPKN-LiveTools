# MultiChat y OBS

## Opción A: visor web local (ya incluido)

1. Configurar `config.json` sin subir secretos a Git.
2. Iniciar `MultiChat.exe -config config.json` (o `-demo` para simular).
3. Agregar una **Fuente de navegador** en OBS con `http://127.0.0.1:8765/` (por ejemplo, 380x700).
4. El visor es una única página local, no cinco navegadores de plataformas.

## Opción B: plugin nativo GDI+ (fuente experimental)

En la copia local existe `OBSPlugin/obs-multichat.cpp`, con `CMakeLists.txt`. Su diseño evita CEF en la fuente de chat: consulta `GET /obs/messages?limit=N` y pinta texto con GDI+.

**Corregido en el servidor Go:** ahora existe `/obs/messages`, limitado a 1-16 entradas. El test `TestOBSMessagesRoute` comprueba el JSON. Falta compilar la DLL con el SDK/libobs compatible con OBS instalado y probar la carga y render real. No copiar ni inyectar módulos sin validación.

## SceneSync para TikTok LIVE Studio

Se clonó para **auditoría** el proyecto externo MIT https://github.com/dannguyen9x/tiktok-live-studio-mcp bajo
`D:\SolarisPKN-IA\Proyectos-Locales\TikTok-LIVE-Studio-MCP-Research`.

El programa externo permite detectar escenas y cambiarlas mediante el canal local Socket.IO de Stream Deck. `npm run test:ci` pasó con 12 tests; el smoke MCP real informa `STUDIO_NOT_RUNNING` mientras LIVE Studio esté cerrado. Eso no demuestra control de escenas en la PC todavía.

Futuro enlace: OBS WebSocket evento `CurrentProgramSceneChanged` -> mapeo de nombres -> `studio_switch_scene`, con confirmación posterior y sin arrancar/detener emisiones automáticamente.
