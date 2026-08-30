# ADR-001: Fuente canónica y adaptadores oficiales con revisión humana

**Estado:** Aceptada  
**Fecha:** 2026-08-29  
**Decisor:** Pablo / SolarisPKN

## Contexto

El Manifiesto declara 109 perfiles, pero éstos mezclan redes sociales, blogs, repositorios, video, formación, gaming, trabajo y tiendas. Sólo una fracción permite crear publicaciones mediante una API oficial; incluso dentro de esa fracción cambian el formato, las credenciales, el costo y la revisión exigida.

El objetivo es escribir una sola vez en Markdown/MDX, mantener limpio el contenido y distribuirlo sin depender de bots de navegador ni ocultar límites reales.

## Decisión

Usar una arquitectura `fuente canónica → plan inmutable → adaptadores → publicadores`:

1. El núcleo existente de LinkedIn analiza Markdown/MDX sin ejecutar JSX.
2. Un modelo canónico reúne título, descripción, Markdown saneado, HTML, texto plano, URL canónica, portada, idioma, tags y activos.
3. Cada adaptador crea un payload específico y declara si es API, API con media, bundle local, asistido o sólo multimedia.
4. El servidor loopback conserva los secretos fuera del bundle web.
5. El SHA-256 cubre contenido y destinos planeados. La escritura exige que el digest no cambie y una confirmación explícita.
6. Los errores de un destino no ocultan los resultados de los demás; se escribe un recibo local.

## Opciones consideradas

### A. API oficiales modulares — elegida

| Dimensión | Evaluación |
|---|---|
| Complejidad | Media/alta inicial; baja al sumar adaptadores |
| Costo | Gratis donde la plataforma lo permite |
| Riesgo de cuenta | Bajo si se respetan permisos y políticas |
| Fidelidad | Alta para artículos; controlada para posts cortos |
| Mantenimiento | Versiones y permisos deben revisarse periódicamente |

### B. Automatización de navegador

| Dimensión | Evaluación |
|---|---|
| Complejidad | Parece baja al inicio, alta y frágil en operación |
| Costo | Bajo |
| Riesgo de cuenta | Alto en plataformas que prohíben bots |
| Fidelidad | Depende del DOM y cambia sin contrato |
| Mantenimiento | Muy alto |

Se descarta: no es un contrato oficial, puede violar políticas y obliga a almacenar sesiones/cookies.

### C. SaaS agregador externo

| Dimensión | Evaluación |
|---|---|
| Complejidad | Baja |
| Costo | Recurrente y normalmente por cantidad de canales |
| Privacidad | El contenido y tokens pasan por un tercero |
| Control | Dependencia de cuotas y roadmap ajeno |

Se mantiene como alternativa futura, no como base de una herramienta abierta y local-first.

## Consecuencias

- Es fácil sumar una plataforma sin reescribir el núcleo.
- “Automático” significa API oficial configurada, no copiar/pegar disfrazado.
- Instagram/Pinterest quedan bloqueados si falta media; YouTube/TikTok/Vimeo no fingen aceptar Markdown como publicación.
- Medium, Hashnode y X se preparan correctamente, pero no se marcan gratis/automáticos.
- La obtención/renovación OAuth todavía es una fase posterior; esta versión consume tokens ya autorizados desde el entorno.
- La sincronización del blog produce un bundle de borrador: integrar directamente con el repo de Labs requerirá un contrato separado de no sobrescritura y validación de Git.

