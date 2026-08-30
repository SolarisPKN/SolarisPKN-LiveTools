# ADR-002: Dos archivos maestros y una cola editorial serial

**Estado:** Aceptado  
**Fecha:** 2026-08-29  
**Decisor:** Pablo KoutaNya

## Contexto

Se quiere distribuir una o dos piezas por día, no producir spam. Un video largo horizontal y un short vertical tienen codecs y narrativa relacionados, pero contratos de publicación diferentes. Algunas APIs aceptan binario local, otras exigen URL pública y varias no ofrecen upload oficial.

## Decisión

El editor conserva dos activos maestros ya editados: `long` y `short`. Calcula SHA-256, inspecciona con `ffprobe` cuando está disponible y crea adaptaciones de metadatos inmutables. La publicación exige revisión humana, confirmación literal y se ejecuta en una cola serial con resultados independientes. El límite local predeterminado es dos trabajos editoriales por día.

## Opciones consideradas

| Opción | Complejidad | Riesgo | Decisión |
|---|---:|---:|---|
| Reutilizar el mismo archivo en todas las redes | Baja | Alto: recortes, límites y narrativa incorrectos | Rechazada |
| Transcodificar automáticamente con FFmpeg | Media/alta | Puede degradar o alterar la obra sin revisión | Diferida |
| Dos archivos finales más adaptación editorial | Media | Bajo y auditable | Elegida |
| Bots que operan editores web | Alta | Suspensión, cookies y roturas | Rechazada |

## Consecuencias

- YouTube largo puede publicarse antes del short para obtener una URL reutilizable.
- Los reintentos son por plataforma y no duplican deliberadamente toda la campaña.
- Instagram y Threads requieren hosting temporal HTTPS o upload reanudable autorizado.
- Las APIs no documentadas permanecen asistidas.
- OAuth interactivo, scheduler y transcodificación pueden añadirse sin cambiar el contrato editorial.
