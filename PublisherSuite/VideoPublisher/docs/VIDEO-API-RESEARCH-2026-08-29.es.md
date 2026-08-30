# Investigación de APIs de video del Manifiesto — 2026-08-29

## Alcance

El Manifiesto declara diez perfiles en “Streaming y video”: YouTube, Twitch, Kick, TikTok, Rumble, Dailymotion, Vimeo, Nimo TV, SOOP Live e IMDb. También se evaluaron Facebook, Instagram, Threads, LinkedIn, Mastodon, Bluesky, Pinterest, Flickr, Pixelfed y X porque aceptan clips aunque estén catalogados como redes/comunidad.

Un perfil visible no demuestra que exista una API de escritura. Se usaron exclusivamente contratos oficiales o repositorios oficiales; un endpoint privado observado en una web no se considera autorización.

La instantánea HTTP del mismo día encontró nueve perfiles accesibles o respondiendo correctamente: YouTube, Twitch, Kick, TikTok, Dailymotion, Vimeo, Nimo TV y SOOP Live devolvieron 200, e IMDb devolvió 202. Rumble devolvió 403 y se clasifica como protegido por la plataforma, no roto. Ninguno de los diez quedó clasificado como enlace inexistente.

## Matriz de viabilidad

| Plataforma | Upload oficial | Fuente | Estado v0.1 |
|---|---|---|---|
| YouTube | Sí, `videos.insert`, reanudable | Archivo local; máximo documentado 256 GB | Implementado para largo y short |
| TikTok | Sí, Direct Post y Upload API | Archivo local o URL verificada | Implementado con `FILE_UPLOAD` |
| Vimeo | Sí, tus, formulario o pull | Archivo local/URL; app Free requiere aprobación | Implementado con tus |
| Dailymotion | Sí, sesión y multipart en API v2 | Archivo local | Asistido hasta validar cuenta/perfil v2 |
| Twitch | No para VOD pregrabado en Helix | La API pública lista/gestiona videos existentes | No automático |
| Kick | No en la API pública actual | Canales, live, chat, recompensas y eventos | No automático |
| Rumble | No se verificó upload público personal | El Help Center documenta carga web | Asistido |
| Nimo TV | No se encontró | SDK de player/broadcast | No automático |
| SOOP Live | No se encontró | Extension/chat SDK con acceso asociado | No automático |
| IMDb | No aplica | No es hosting editorial con API de perfil | No automático |
| Facebook Page | Sí, Reels Publishing | Archivo local o CDN; Page, no perfil personal | Implementado para Reel corto |
| Instagram profesional | Sí, Content Publishing | URL pública o flujo reanudable autorizado | Implementado por URL pública |
| Threads | Sí, contenedor VIDEO + publish | URL pública | Implementado por URL pública |
| LinkedIn | Sí, Videos API + Posts API | Multipart local, MP4 | Implementado |
| Mastodon | Sí, `/api/v2/media` + status | Multipart local; límites por instancia | Contrato preparado |
| Bluesky | Sí, servicio de video asíncrono | MP4 y job de procesamiento | Diferido por límites/flujo cambiante |
| Pinterest | Sí, media register + S3 + Pin | Video local más portada pública | Contrato preparado |
| Flickr | Sí, endpoint de upload | OAuth 1.0a; límites de video según cuenta | Asistido |
| Pixelfed | Variable | Compatibilidad Mastodon depende de instancia | Dependiente de instancia |
| X | Sí | Media y posts con escritura paga | Excluido de “gratis” |

## Hallazgos que cambian el diseño

### YouTube y Shorts

No existe un endpoint de upload exclusivo para Shorts. Ambos archivos usan `videos.insert`. YouTube clasifica automáticamente como Short un video cuadrado o vertical de hasta tres minutos. Una carga cuesta una operación del bucket de uploads; los proyectos no auditados creados desde julio de 2020 quedan restringidos a privado. Por seguridad, el editor comienza con visibilidad privada.

Fuentes: https://developers.google.com/youtube/v3/docs/videos/insert, https://developers.google.com/youtube/v3/guides/using_resumable_upload_protocol y https://support.google.com/youtube/answer/15424877

### Meta

Facebook Page Reels permite iniciar sesión, subir binario local al `upload_url` y finalizar como borrador o publicado. El contrato consultado documenta 9:16, 4–60 segundos y mínimo 540×960. Instagram profesional crea un contenedor `REELS`, procesa y publica; el flujo clásico requiere una URL pública. Threads usa también contenedor y publicación, y descarga `video_url` desde un servidor público.

Fuentes: https://www.postman.com/meta/facebook/documentation/r56bjfd/facebook-api, https://www.postman.com/meta/instagram/documentation/6yqw8pt/instagram-api y https://www.postman.com/meta/threads/documentation/dht3nzz/threads-api

### TikTok

Direct Post exige consultar primero `creator_info`, mostrar las opciones de privacidad reales, obtener consentimiento explícito, inicializar `/v2/post/publish/video/init/` y cargar el archivo. El scope es `video.publish`. Clientes no auditados sólo pueden publicar en privado. TikTok impone límites anti-spam propios; el volumen solicitado de una o dos piezas diarias no elimina la necesidad de auditoría.

Fuentes: https://developers.tiktok.com/docs/en/content-posting-api-get-started, https://developers.tiktok.com/docs/en/content-posting-api-reference-direct-post y https://developers.tiktok.com/docs/en/content-sharing-guidelines

### Vimeo

La API ofrece tus reanudable, formulario y pull. Vimeo recomienda tus. Una cuenta Free necesita pedir aprobación para upload API y mantiene cuotas de almacenamiento/carga del plan.

Fuente: https://developer.vimeo.com/api/upload/videos

### LinkedIn

Videos API reemplaza Assets API: inicializa instrucciones multipart, sube partes, finaliza y referencia el URN en Posts API. El permiso personal es `w_member_social`. La documentación consultada especifica MP4, 3 segundos a 30 minutos y 75 KB a 500 MB.

Fuentes: https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/videos-api y https://learn.microsoft.com/en-us/linkedin/marketing/community-management/shares/posts-api

### Plataformas que no deben automatizarse con endpoints privados

La API pública de Kick no muestra upload de VOD. Twitch Helix expone lectura de videos, no carga de un MP4 arbitrario. Rumble ofrece una carga web documentada, pero no se verificó un contrato público oficial de upload personal. Nimo y SOOP publican SDKs orientados a reproducción, extensiones, chat o broadcast. IMDb no es un destino editorial equivalente. Usar endpoints internos, cookies o automatización del navegador convertiría una integración estable en un bot frágil y con riesgo de cuenta.

Fuentes: https://api.kick.com/swagger/index.html, https://dev.twitch.tv/docs/api/videos, https://rumble.support/help/how-to-upload-a-video y https://developers.sooplive.co.kr/

## Política de bajo volumen

El publicador aplica un máximo local predeterminado de dos trabajos editoriales por día. Un trabajo puede distribuir la misma pieza a varios destinos; no cuenta cada destino como una pieza nueva. El límite no reemplaza las cuotas ni políticas externas. La cola es serial, requiere confirmación humana y guarda recibos por plataforma para evitar reintentos ciegos.

## Pendientes antes de producción real

1. Crear aplicaciones y completar OAuth de YouTube, Meta, TikTok, Vimeo y LinkedIn.
2. Auditar YouTube y TikTok si se necesita visibilidad pública.
3. Elegir hosting temporal HTTPS para Instagram/Threads o completar el flujo reanudable de Instagram.
4. Probar cada conector inicialmente con visibilidad privada y archivos pequeños sin copyright ajeno.
5. Confirmar permisos de Page/Professional Account y versiones vigentes de Graph/LinkedIn.
6. Incorporar scheduler sólo después de que los recibos e idempotencia real estén validados.
