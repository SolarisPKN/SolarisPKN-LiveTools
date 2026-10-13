# Windows, OBS y resolución de problemas

1. Desde `MultiChat`, copiá `config.example.json` a `config.json`. Sin credenciales podés usar `go run . -config config.json -demo`.
2. Abrí `http://127.0.0.1:8765` en un navegador local para validar que la UI muestra mensajes.
3. En OBS agregá **una Fuente de navegador**, con URL `http://127.0.0.1:8765/`, por ejemplo 380 × 700 píxeles.
4. Esa fuente renderiza **un único chat para todas las plataformas**; no carga Twitch, Kick, YouTube, etc. La página de esta versión es oscura, todavía no transparente.
5. Cerrá con Ctrl+C. Si querés minimizar salida por consola, ejecutá `MultiChat.exe -config config.json -quiet`.

## Problemas frecuentes

- **No aparecen mensajes reales:** `-demo` solo produce ejemplos. Revisá credenciales, canal, liveChatId o API URL y `enabled:true`.
- **No se abre la página:** verificá que el proceso siga ejecutándose y que el puerto 8765 no esté ocupado. Podés cambiar `listen` a otro puerto de loopback.
- **YouTube deja de consultar:** revisar presupuesto, cuota externa y permisos del API.
- **OBS usa CPU:** la Fuente de navegador renderiza la UI; para evaluar solo el recolector probá sin abrir el visor.
- **Falta el EXE:** compilá con Go 1.22+ (`go build -o MultiChat.exe .`) o usá `iniciar-multichat.cmd` cuando Go esté instalado.
