# SolarisPKN MultiChat — interfaz nativa compacta

MultiChat es **una ventana lateral vertical** Win32/GDI+ (sin Chromium, Electron, CEF o WebView2). NO es el futuro dashboard 16:9 de SolarisPKN-LiveTools. Se inicia con una anchura aproximada de **un tercio de la pantalla**, ajustada al área de trabajo de Windows, y se puede redimensionar.

## Diseño compacto

- Encabezado pequeño con botones **A−** y **A+** para modificar el tamaño de fuente entre 10–19 píxeles. La preferencia se guarda en `NativeUI/preferences.ini` (archivo local no versionado).
- **Chat unificado** en un control RichEdit nativo, fondo oscuro, nombres de plataforma coloreados y hasta 80 mensajes en el feed local.
- Debajo del chat: botones **Todas** y **Ninguna**, cinco íconos vectoriales para YouTube, TikTok, Twitch, Kick y Facebook, **engrane de configuración sobre cada ícono**, indicador circular de estado, caja de mensaje/comando y botón de envío; no hay ventana de envío separada.
- **Ícono a color** cuando el backend habilita enviar para ese destino (en `-demo`, únicamente simular). **Gris** cuando no se puede enviar.
- **LED verde intermitente** exclusivamente cuando el motor confirmó la conexión con ese chat (IRC JOIN/Twitch, HTTP 200 YouTube/Facebook, sesión Webcast TikTok, suscripción Kick). **Ámbar** configurado pero todavía sin conexión verificada. En modo demo nunca se atribuye un chat real a un LED verde.
- El símbolo de engranaje abre una configuración nativa por plataforma. Permite abrir una copia de la plantilla `NativeUI/config.local.json` con Notepad y, en YouTube/Twitch, iniciar las respectivas vinculaciones OAuth existentes mediante `-link-google` o `-link-twitch` en la sesión del usuario. La edición del archivo no reinicia automáticamente conectores ni equivale a verificar credenciales. Para Facebook/Kick/TikTok todavía faltan asistentes de conexión completos.

**Fuentes SVG:** `NativeUI/assets/*.svg`. El ejecutable dibuja logotipos equivalentes con GDI+ (sin motor SVG residente).

## Ejecutable de prueba

`MultiChat-Native.exe` está en la carpeta `MultiChat`. Inicia un motor Go local oculto `MultiChat-native-engine.exe` si no encuentra uno escuchando en `127.0.0.1:8766`. El arranque automático usa exclusivamente `NativeUI/demo-config.json -demo -quiet`: **no abre ni publica nada en redes sociales**.

Los mensajes y comandos enviados a las cinco redes son simulados. No se piden tokens en modo demo. El antiguo visor web en `8765` es opcional para diagnóstico.

### Modo real (no configurado por defecto)

El motor real se inicia por separado, previamente, en `127.0.0.1:8766`, con configuración privada propia y sin `-demo`. Debe tener credenciales autorizadas por cada plataforma y, para envíos reales, `panel_auth.enabled` más `outbound.enabled`. La UI Win32 puede usar un **token nativo independiente** definido en `panel_auth.native_token_env` (por defecto usar `MULTICHAT_NATIVE_TOKEN`, mínimo 32 caracteres). No comparte el secreto del login web ni los del OBS plugin. El arranque de demo no configura ni habilita cuentas reales.

## Compilación

```bat
cd D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-LiveTools\MultiChat\NativeUI
build-native.cmd
```

Requiere Visual Studio Build Tools y Windows SDK. El batch usa MSVC `/utf-8`, lo que corrige los textos mojibake `Ã...` / `â...` del primer prototipo. Compila una ventana Win32/GDI+ independiente. Motor Go:

```powershell
cd D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-LiveTools\MultiChat
go test ./...
go vet ./...
go build -trimpath -ldflags="-s -w" -o MultiChat-native-engine.exe .
```

La integración de inicio/parada de streaming en OBS está explicada en [OBS-INTEGRATION.es.md](../docs/OBS-INTEGRATION.es.md). No se instaló ningún plugin OBS en el programa real; el motor sí trae un cliente opcional para el obs-websocket integrado y el código fuente del módulo nativo.
