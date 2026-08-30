# SolarisPKN Multiplatform Publisher

Publicador local de Markdown/MDX que genera una versión canónica y la adapta a artículos, posts, captions, mensajes y bundles. La vista previa no envía contenido a terceros. La publicación real sólo ocurre después de seleccionar destinos, revisar el SHA-256 y escribir `PUBLICAR`.

## Uso normal en Windows

1. Hacé doble clic en `abrir-publicador-multiplataforma.cmd`.
2. Cargá un `.md` o `.mdx` y, si existe, su JSON de metadatos.
3. Completá URL canónica, portada HTTPS, hashtags e idioma.
4. Elegí destinos y pulsá **Preparar vista previa**.
5. Revisá cada payload. Los destinos sin credenciales, pagos o incompatibles quedan claramente bloqueados/asistidos.
6. Para publicar de verdad, marcá sólo los destinos deseados, confirmá la revisión y escribí `PUBLICAR`.

La primera ejecución instala tanto las dependencias del conversor compartido de `PublisherSuite/Linkedin` como las de este módulo. Requiere Node.js 20 o superior.

## Qué está implementado

| Destino | Resultado | Publicación |
|---|---|---|
| SolarisPKN Labs Blog | Bundle `index-es.mdx` + `post.json` en `output/` | Local; no hace commit, push ni deploy |
| DEV | Artículo Markdown, tags, portada y canonical | API oficial |
| Blogger | Artículo HTML | API oficial |
| Tumblr | Post NPF | API oficial |
| LinkedIn | Post de feed y tarjeta de enlace | Posts API; no crea un artículo/newsletter nativo largo |
| Facebook | Post de una Page | Graph API; no publica en perfiles personales |
| Instagram | Caption + imagen | API oficial; cuenta profesional e imagen pública obligatorias |
| Threads | Post textual | API oficial |
| Mastodon | Status, idioma e idempotencia | API oficial de la instancia |
| Bluesky | Record `app.bsky.feed.post` | AT Protocol oficial |
| Reddit | Self post en un subreddit | API OAuth oficial |
| Pinterest | Pin con enlace e imagen | API oficial; imagen pública obligatoria |
| Discord | Mensaje a un canal | Incoming Webhook oficial |
| Medium | Markdown listo para importar/pegar | Asistido: API sin soporte y cerrada a integraciones nuevas |
| Hashnode | Markdown listo | Asistido: publicación API actual requiere Pro |
| X | Texto de hasta 280 caracteres | Asistido: escritura API paga por uso |
| YouTube, TikTok, Vimeo | Título/descripción | Sólo media; faltan archivo, OAuth y/o auditoría de app |

La investigación completa está en [docs/API-RESEARCH-2026-08-29.es.md](docs/API-RESEARCH-2026-08-29.es.md). El inventario auditable de los 109 perfiles del Manifiesto está en [config/profiles.json](config/profiles.json) y los hallazgos HTTP en [docs/PROFILE-AUDIT-2026-08-29.es.md](docs/PROFILE-AUDIT-2026-08-29.es.md).

## Credenciales

Los secretos se leen exclusivamente desde variables de entorno del proceso del servidor. No se incrustan en `dist`, no vuelven al navegador y no se escriben en los recibos.

Ejemplo de sesión PowerShell (completá sólo los destinos que vayas a usar):

```powershell
$env:SOLARIS_DEVTO_API_KEY = '...'
$env:SOLARIS_MASTODON_BASE_URL = 'https://mastodon.social'
$env:SOLARIS_MASTODON_ACCESS_TOKEN = '...'
$env:SOLARIS_BLUESKY_HANDLE = 'solarispkn.com.ar'
$env:SOLARIS_BLUESKY_APP_PASSWORD = '...'
& '.\abrir-publicador-multiplataforma.cmd'
```

Variables reconocidas:

- `SOLARIS_DEVTO_API_KEY`
- `SOLARIS_BLOGGER_ACCESS_TOKEN`, `SOLARIS_BLOGGER_BLOG_ID`
- `SOLARIS_TUMBLR_ACCESS_TOKEN`, `SOLARIS_TUMBLR_BLOG_ID`
- `SOLARIS_LINKEDIN_ACCESS_TOKEN`, `SOLARIS_LINKEDIN_AUTHOR_URN`, `SOLARIS_LINKEDIN_VERSION` opcional (`YYYYMM`)
- `SOLARIS_FACEBOOK_PAGE_TOKEN`, `SOLARIS_FACEBOOK_PAGE_ID`, `SOLARIS_META_GRAPH_VERSION` opcional
- `SOLARIS_INSTAGRAM_ACCESS_TOKEN`, `SOLARIS_INSTAGRAM_USER_ID`
- `SOLARIS_THREADS_ACCESS_TOKEN`, `SOLARIS_THREADS_USER_ID`, `SOLARIS_THREADS_API_BASE` opcional
- `SOLARIS_MASTODON_ACCESS_TOKEN`, `SOLARIS_MASTODON_BASE_URL`, `SOLARIS_MASTODON_CHARACTER_LIMIT` opcional
- `SOLARIS_BLUESKY_HANDLE`, `SOLARIS_BLUESKY_APP_PASSWORD`, `SOLARIS_BLUESKY_PDS` opcional
- `SOLARIS_REDDIT_ACCESS_TOKEN`, `SOLARIS_REDDIT_SUBREDDIT`, `SOLARIS_REDDIT_USER_AGENT` opcional
- `SOLARIS_PINTEREST_ACCESS_TOKEN`, `SOLARIS_PINTEREST_BOARD_ID`
- `SOLARIS_DISCORD_WEBHOOK_URL`

No pongas tokens en Markdown, metadatos, commits, capturas ni archivos `config.local.json`. Para OAuth de larga duración conviene sumar después un asistente con Windows Credential Manager; esta primera versión falla cerrado si falta una credencial.

## CLI y diagnóstico

```powershell
npm --prefix ..\Linkedin ci --cache ..\Linkedin\.npm-cache
npm ci --cache .npm-cache
npm run check
npm run plan -- articulo.mdx --targets devto,linkedin,mastodon --canonical-url https://example.com/articulo
npm run audit:profiles
```

La primera línea es necesaria en un clon limpio porque el publicador reutiliza el parser MDX probado del módulo de LinkedIn. El `.cmd` hace ambas instalaciones automáticamente.

`npm run plan` sólo genera artefactos JSON. No publica. `npm run audit:profiles` hace comprobaciones HTTP prudentes: un `401`, `403` o `429` se clasifica como protegido, no como perfil inexistente.

## Seguridad y límites deliberados

- Servidor sólo en `127.0.0.1`, Hosts loopback, Origin de mismo puerto y token de sesión anti-CSRF.
- Límite de 2 MiB para Markdown/MDX y 2,5 MiB para solicitudes.
- Imports, exports, JSX y expresiones MDX no se ejecutan.
- Cada plan tiene SHA-256; si cambia contenido o destinos, la publicación se rechaza.
- Publicación secuencial con resultado independiente y recibo JSON local.
- No usa Playwright/Selenium para operar editores, no guarda cookies y no simula clics en redes.
- No hace commit, push, deploy ni publicación externa durante build o tests.
- Las APIs pueden cambiar versiones, permisos, revisión y precios; verificá el documento de investigación antes de activar un conector nuevo.
