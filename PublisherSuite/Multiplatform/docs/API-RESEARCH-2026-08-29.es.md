# Investigación de publicación por API — 2026-08-29

## Alcance auditado

- Dominio válido: `https://manifiesto.solarispkn.com.ar/` respondió 200 y publicó el título “SolarisPKN — Manifiesto de Ingeniería Transparente”.
- Dominio alternativo solicitado: `manifiesto.solaris.pkn.com.ar` no existe en DNS al 29 de agosto de 2026.
- El bloque **Presencia** declara 109 perfiles: 26 código/desarrollo, 12 infraestructura/IA, 10 streaming/video, 18 redes/comunidad, 13 gaming/mods, 7 diseño/arte, 15 trabajo/apoyo y 8 formación/credenciales.
- El inventario exacto y reproducible está en `config/profiles.json`; no se interpreta que un perfil público equivale a una API de escritura.

## Resultado ejecutivo

“API gratis” describe el acceso al endpoint, no una promesa de cero fricción. Casi todas exigen cuenta de desarrollador, OAuth/tokens, cuotas y respeto de políticas. Meta, LinkedIn, TikTok, YouTube y Vimeo pueden exigir revisión/auditoría o limitar contenido público. Algunas APIs gratuitas sólo publican en Pages/cuentas profesionales, no en perfiles personales.

| Plataforma del Manifiesto | Escritura oficial | Costo API relevante | Contrato útil para Markdown | Estado en v0.1 |
|---|---|---|---|---|
| DEV | Sí, `POST /api/articles` | Gratis con API key; 10 req/30 s documentadas | Artículo Markdown, draft/public, canonical, portada | Automático |
| Blogger | Sí, `posts.insert` | Gratis sujeto a cuota de Google | Artículo HTML y borrador/publicación | Automático |
| Tumblr | Sí, `/v2/blog/{id}/posts` NPF | Gratis con OAuth y límites | Artículo en bloques NPF | Automático |
| Medium | Endpoint histórico de posts | No disponible para integraciones nuevas; API sin soporte | Markdown/HTML y canonical | Asistido |
| Hashnode | Mutación `publishPost` | Requiere publicación Pro según documentación actual | Artículo Markdown | Asistido |
| LinkedIn | Sí, Posts API | Sin cargo por request; permisos/tier de producto | Post de feed o tarjeta de enlace; no artículo largo nativo | Automático para feed |
| Facebook | Sí, Graph API para Pages | Sin cargo por request; permisos/app review | Texto/enlace/foto de Page; no perfil personal | Automático |
| Instagram | Sí, Content Publishing | Sin cargo por request; cuenta profesional/permisos | Caption más imagen/video; no texto puro | Automático si hay imagen |
| Threads | Sí, Threads API | Sin cargo por request; app y permisos | Texto, imagen, video o carrusel | Automático textual |
| Mastodon | Sí, `POST /api/v1/statuses` | Gratis; límite depende de instancia | Status, media, idioma, programación | Automático |
| Bluesky | Sí, AT Protocol `createRecord` | Gratis | Post de 300 caracteres, idiomas, embeds | Automático textual |
| Reddit | Sí, submit/Devvit | Gratis con OAuth y reglas del subreddit | Self post Markdown | Automático |
| Pinterest | Sí, `POST /v5/pins` | Gratis con app/token | Pin con título, descripción, link y media | Automático si hay imagen |
| Discord | Sí, incoming webhook | Gratis | Mensaje Markdown limitado al canal | Automático |
| X | Sí, `POST /2/tweets` | Pago por uso: Content Create tiene precio por request | Texto corto | Asistido, excluido de “gratis” |
| YouTube | Sí, `videos.insert` | Cuota gratis; proyectos no auditados publican privado | Video + metadatos, no post Markdown | Sólo prepara metadatos |
| TikTok | Sí, Content Posting API | Sin cargo directo; scope y auditoría | Foto/video + caption; cliente no auditado queda privado | Sólo prepara metadatos |
| Vimeo | Sí, video uploads | Cuenta Free limitada y app de upload aprobada | Video + metadatos | Sólo prepara metadatos |
| Dailymotion | Sí, video upload | Cuenta/permisos/cuotas | Video + metadatos | Candidato futuro |
| Flickr/Pixelfed | APIs de media/ActivityPub según servicio | Depende del servicio/instancia | Foto + caption | Candidatos futuros |

## Perfiles que no son destinos editoriales equivalentes

Los otros perfiles sí pueden tener APIs, pero crean recursos especializados: repositorios, issues, releases, sandboxes, modelos, datasets, mods, productos, citas, pagos, cursos o portfolios. Publicar el mismo artículo allí sería semánticamente incorrecto o spam. Por eso GitHub/GitLab/Codeberg, Docker Hub, Hugging Face, CurseForge/Modrinth, tiendas, freelancing y formación permanecen en el inventario, no en el selector editorial v0.1.

## Fuentes oficiales consultadas

- LinkedIn Posts API: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api
- Meta Pages API posts: https://developers.facebook.com/docs/pages-api/posts/
- Meta Instagram Content Publishing: https://developers.facebook.com/docs/instagram-platform/instagram-api-with-facebook-login/content-publishing/
- Meta Threads API: https://developers.facebook.com/docs/threads/posts/
- Colecciones oficiales Meta en Postman: https://www.postman.com/meta/
- DEV / Forem API: https://developers.forem.com/api/v0 y https://developers.forem.com/api/
- Medium API oficial archivada: https://github.com/Medium/medium-api-docs
- Hashnode GraphQL: https://docs.hashnode.com/quickstart/introduction y https://github.com/Hashnode/gql-skill
- Mastodon statuses: https://docs.joinmastodon.org/methods/statuses/
- Bluesky posting: https://docs.bsky.app/blog/create-post
- Tumblr API: https://github.com/tumblr/docs
- Blogger `posts.insert`: https://developers.google.com/blogger/docs/3.0/reference/posts/insert
- Reddit Developer Platform: https://developers.reddit.com/docs/capabilities/server/reddit-api
- Pinterest Pins: https://developers.pinterest.com/docs/work-with-organic-content-and-users/create-boards-and-pins/
- Discord webhooks: https://docs.discord.com/developers/resources/webhook
- X pricing y creación de posts: https://docs.x.com/x-api/getting-started/pricing y https://docs.x.com/x-api/posts/create-post
- YouTube video uploads: https://developers.google.com/youtube/v3/docs/videos/insert
- TikTok Direct Post: https://developers.tiktok.com/docs/en/content-posting-api-get-started
- Vimeo uploads: https://developer.vimeo.com/api/upload/videos
- Dailymotion uploads: https://developers.dailymotion.com/v0/docs/upload-videos

## Límites que deben revalidarse antes de activar credenciales

1. Versión vigente de Meta Graph API y permisos aprobados para la app.
2. Versión mensual soportada de LinkedIn; se permite `SOLARIS_LINKEDIN_VERSION` para fijarla.
3. Límite de caracteres real de la instancia Mastodon; puede configurarse.
4. Visibilidad/auditoría de proyectos YouTube y TikTok.
5. Precios actuales de X y estado Pro de Hashnode.
6. Renovación y expiración de cada token. El publicador no implementa todavía OAuth interactivo.

