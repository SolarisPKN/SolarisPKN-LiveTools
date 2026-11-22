# SolarisPKN MultiChat — interfaz Win32 nativa 16:9

## Objetivo exacto

Una sola ventana **Windows Win32 nativa** (compilada con MSVC, GDI+ solo para íconos), con área cliente fija 16:9. **Cero Electron, Chromium, CEF y WebView2** en tiempo de ejecución.

Distribución acordada:
- **Tercio izquierdo**: encabezado, chat vertical de varias plataformas, botones Todas / Ninguna, cinco íconos vectoriales (YouTube, TikTok, Twitch, Kick y Facebook), una sola caja de comando/mensaje justo debajo y un botón de envío. Gris si el backend no permite enviar y a color si el destino está habilitado en el modo actual. La selección se destaca con un borde y marca.
- **Dos tercios derechos**: **intencionalmente vacíos**. No se agregarán widgets hasta que el usuario decida qué colocar.
- **Un único panel**: la consola de comandos no abre una segunda ventana.

Los íconos tienen fuentes de referencia en `assets/*.svg`, y se dibujan por primitivas vectoriales GDI+ en el shell para evitar un motor web o un conversor SVG residente.

La interfaz es un ejecutable pequeño en C++, separado del motor Go existente. El motor Go sigue haciendo recepción de mensajes y control de permisos. La interfaz consulta `/native/feed` para evitar interpretar HTML o cargar librerías de navegador y `/api/outbound/status` para saber qué plataformas habilitar. El botón de enviar utiliza el endpoint JSON original `/api/outbound/send`, con su cabecera de acción, control de límites, resultados por plataforma y confirmación ante envíos reales.

## Prueba local

El ejecutable `MultiChat-Native.exe` está al lado de los ejecutables del motor dentro de la carpeta `MultiChat`.

Al iniciar en modo demo, conecta con `127.0.0.1:8766` y, si no hay un motor escuchando, lanza oculto `MultiChat-native-engine.exe -config NativeUI\\demo-config.json -demo -quiet`. La demostración no autentica plataformas ni envía mensajes a Internet. No utiliza el anterior puerto 8765, para no modificar la sesión antigua del visor web.

**No activar envíos reales mediante este archivo de demo.** Los botones se colorean en demo porque las simulaciones están habilitadas, pero no significan que exista autorización real en las cuentas. La API puede marcar no disponible un destino si faltan permisos.

## Compilación

Desde una consola de Windows con Visual Studio 2026 Build Tools (MSVC y Windows SDK):

```bat
cd D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-LiveTools\MultiChat\NativeUI
build-native.cmd
```

El script compila `..\\MultiChat-Native.exe` usando exclusivamente Windows SDK, Win32, WinHTTP y GDI+. Requiere compilar el backend Go `MultiChat-native-engine.exe` por separado:

```powershell
cd D:\SolarisPKN-IA\Proyectos-Locales\SolarisPKN-LiveTools\MultiChat
go test ./...
go build -trimpath -ldflags="-s -w" -o MultiChat-native-engine.exe .
```

## Límites

Se priorizó la implementación del layout y el modo de prueba local. Conectar una sesión real requiere configurar y verificar OAuth y los adaptadores de escritura en el motor Go; no hubo cambios de permisos ni de tokens en las redes sociales. La integración futura en un único ejecutable/instalador se definirá dentro de SolarisPKN-LiveTools completo. El panel web viejo queda como diagnóstico opcional; **no forma parte del nuevo flujo nativo**.
