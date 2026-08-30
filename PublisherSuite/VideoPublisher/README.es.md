# SolarisPKN Video Publisher

Editor local de bajo volumen para distribuir un video largo y un short vertical desde una sola revisión. No recorta, recomprime ni añade marcas de agua: se cargan dos archivos ya editados y se adaptan los metadatos por plataforma.

## Uso

1. Hacé doble clic en `abrir-publicador-video.cmd`.
2. Cargá el video largo y el short vertical.
3. Escribí título, descripción larga, caption corto, tags, visibilidad y URLs opcionales.
4. Elegí destinos y prepará la vista previa.
5. Revisá el SHA-256 y cada payload.
6. Marcá sólo las APIs deseadas, confirmá la revisión y escribí `PUBLICAR VIDEO`.

La primera ejecución instala dependencias locales. Requiere Node.js 20 o superior. `ffprobe` es opcional pero recomendado para verificar duración, dimensiones y codec.

## Conectores implementados

- YouTube video largo y YouTube Short mediante el mismo `videos.insert` reanudable.
- Facebook Page Reel desde archivo local.
- TikTok Direct Post con `FILE_UPLOAD` por bloques y consulta previa de opciones del creador.
- Vimeo mediante tus.
- LinkedIn Videos API más Posts API.
- Instagram Reel y Threads video mediante URL HTTPS pública.

Los conectores se habilitan sólo con todas sus variables de entorno. No fueron probados contra cuentas reales en esta entrega; los tests validan contratos, adaptación, digest y bloqueos, no autorización externa.

## Credenciales

- `SOLARIS_YOUTUBE_ACCESS_TOKEN`
- `SOLARIS_FACEBOOK_PAGE_TOKEN`, `SOLARIS_META_GRAPH_VERSION` opcional
- `SOLARIS_INSTAGRAM_ACCESS_TOKEN`, `SOLARIS_INSTAGRAM_USER_ID`
- `SOLARIS_TIKTOK_ACCESS_TOKEN`
- `SOLARIS_THREADS_ACCESS_TOKEN`, `SOLARIS_THREADS_USER_ID`, `SOLARIS_THREADS_API_BASE` opcional
- `SOLARIS_VIMEO_ACCESS_TOKEN`
- `SOLARIS_LINKEDIN_ACCESS_TOKEN`, `SOLARIS_LINKEDIN_AUTHOR_URN`, `SOLARIS_LINKEDIN_VERSION` opcional
- `SOLARIS_VIDEO_DAILY_JOB_LIMIT` opcional; predeterminado `2`
- `SOLARIS_VIDEO_MAX_BYTES` opcional; predeterminado 20 GiB
- `SOLARIS_VIDEO_PORT` opcional

Los tokens sólo se leen en el servidor local. No vuelven al navegador, no se escriben en recibos y no deben guardarse en el repositorio.

## Límites deliberados

- Los archivos se copian a `media/`, que está ignorado por Git. Se pueden borrar desde la interfaz al reemplazarlos o manualmente con el publicador cerrado.
- La cola publica secuencialmente y genera recibos independientes.
- El máximo local predeterminado es de dos trabajos editoriales por día, no dos publicaciones por plataforma.
- La visibilidad inicial es privada. Cambiarla a pública es una decisión explícita.
- Instagram y Threads necesitan una URL pública; el servidor loopback no es accesible desde Meta.
- No hay bots de navegador, cookies, scraping, commit, push ni deploy.
- YouTube debe autorizarse con OAuth y los proyectos no auditados pueden quedar restringidos a privado.
- TikTok exige `video.publish`, consentimiento explícito y auditoría para contenido público.
- Vimeo Free exige aprobación de la aplicación para uploads por API.

Investigación: [docs/VIDEO-API-RESEARCH-2026-08-29.es.md](docs/VIDEO-API-RESEARCH-2026-08-29.es.md). Arquitectura: [docs/ADR-002-video-multiplataforma.md](docs/ADR-002-video-multiplataforma.md).

## Diagnóstico

```powershell
npm ci --cache .npm-cache
npm run check
```
