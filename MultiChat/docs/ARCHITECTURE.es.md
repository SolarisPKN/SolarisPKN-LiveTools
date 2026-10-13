# Arquitectura sin navegadores

```text
Twitch IRC ──┐
YouTube API ─┼──> Hub (validación, deduplicación, historial acotado)
Rumble API ──┤                ├── consola
Ingest local ┘                ├── GET /history (JSON)
                              └── GET /events (SSE) ──> único visor/OBS
```

Un solo proceso Go, con goroutines independientes por plataforma y cola no bloqueante por consumidor. Los conectores **no** ejecutan navegadores, reproductores ni páginas de streaming. El visor HTML local es totalmente opcional; OBS puede abrir **una fuente de navegador** para un chat compartido.

- El historial tiene un máximo de 1000 mensajes; valor predeterminado: 250.
- La tabla de IDs deduplicados se reconstruye periódicamente para no crecer indefinidamente.
- Cada cliente SSE usa un buffer de 128 eventos. Ante una UI lenta, se descartan eventos para no bloquear a los conectores.
- El visor limita el DOM y los IDs locales a 250 mensajes.
- Los mensajes tienen los campos `id`, `platform`, `user`, `text`, `kind` y `time`.
- No hay base de datos ni persistencia. Cerrar el programa borra el historial.

**Rendimiento aún no medido en el equipo del usuario:** ejecutar `MultiChat.exe -config config.json -quiet`, observar RAM y CPU en Administrador de tareas o `Get-Process MultiChat | Select CPU,WorkingSet64,PrivateMemorySize64`, comparar sin visor, con visor y con fuente OBS activa, y registrar picos durante una transmisión larga. No presentar presupuestos de RAM como benchmarks.

Mejoras posteriores: YouTube `streamList`, Twitch EventSub, telemetría de eventos descartados, GUI nativa, integraciones autorizadas, SQLite opcional.
