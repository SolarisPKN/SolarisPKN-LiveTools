# Seguridad, privacidad y límites

- El servidor solo permite `127.0.0.1`, `localhost` o `[::1]` y valida el encabezado `Host` para reducir ataques de DNS rebinding. **No habilitar redirecciones de puertos ni túneles públicos sin seguridad adicional.**
- Los endpoints `/history` y `/events` son legibles por procesos locales: no hay aislamiento de privacidad ante otros programas instalados en el mismo equipo.
- La entrada de mensajes `/api/ingest` está deshabilitada por defecto; usa Bearer de 24+ caracteres y limita el tamaño de cada petición a 16 KiB. El token prueba autorización del puente, **no autentica a la plataforma de origen**.
- La UI usa `textContent` para mensajes: no interpreta el HTML suministrado por el chat. Aplicar revisión de seguridad al agregar nuevos elementos.
- Los secretos se pasan por variables de entorno. `config.json`, `*.exe` y logs se excluyen de Git. No publicar URLs privadas de Rumble, OAuth tokens ni API keys.
- No se almacenan mensajes en disco. El historial y el presupuesto local de consultas de YouTube se reinician al reiniciar el proceso.
- No hay respuestas enviadas a chats, moderación, automatismos de cuentas ni verificación de webhooks externos en esta versión.

Antes de activar una plataforma real: confirmar permisos mínimos, condiciones de uso, autenticación y costo de las APIs. No asumir compatibilidad completa sin pruebas autenticadas.
