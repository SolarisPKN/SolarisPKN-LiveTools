# Auditoría de perfiles del Manifiesto — 2026-08-29

## Resumen

El inventario del Manifiesto contiene exactamente 109 URLs únicas y todas usan HTTPS. La comprobación automatizada hace `HEAD` y, cuando el servidor no lo admite o devuelve un 404 dudoso, repite con un `GET` parcial. Los resultados se clasifican así:

- **Accesible/redirige:** respuesta menor a 400.
- **Protegido:** 401, 403, 429 o redirección reconocible a CAPTCHA. No implica que el perfil esté roto.
- **Roto/no verificable:** 404, 5xx persistente, timeout o fallo de conexión.

La última ejecución produjo **78 accesibles/redirecciones, 24 protegidos y 7 rotos/no verificables**. El caso eBay fue una redirección a CAPTCHA y se cuenta correctamente como protegido.

## Enlaces que requieren revisión humana

| Perfil | Resultado observado | Lectura recomendada |
|---|---:|---|
| JSFiddle | 404 en la URL declarada y en `/user/SolarisPKN/` | Probable perfil inexistente, privado o username diferente |
| SourceForge | 404 | Probable perfil inexistente o enlace obsoleto |
| Dribbble | 404 | Probable perfil inexistente o username diferente |
| freeCodeCamp | 404 | Perfil/ruta no pública en la URL declarada |
| ADPList | 404 | Enlace de miembro probablemente obsoleto |
| Salesforce Trailblazer | 500 | No concluyente; revisar sesión y la URL actual desde el avatar del perfil |
| Docker Hub | Fallo de fetch en `/repositories/solarispkn` | Reemplazar por `https://hub.docker.com/u/solarispkn`, verificada con HTTP 200 |
| eBay | 500 después de redirigir a CAPTCHA | Perfil protegido por anti-bot; no marcarlo roto sin revisión visual |

## Redirecciones relevantes

- Bitbucket terminó en login de Atlassian; la URL puede ser válida pero la vista no es pública en esta sesión.
- Discord redirigió correctamente de `discord.gg` a `discord.com/invite/...`.
- Facebook, Instagram, TikTok, Twitch y YouTube normalizaron al host `www`.
- Goodreads resolvió el alias a un perfil numérico público.

## Reproducción

```powershell
cd PublisherSuite\Multiplatform
npm run audit:profiles
```

El JSON completo se genera en `audit-output/profiles-latest.json` y se mantiene fuera de Git por ser una instantánea temporal. Un código HTTP por sí solo no demuestra propiedad de la cuenta, actividad, identidad ni acceso a una API de publicación.
