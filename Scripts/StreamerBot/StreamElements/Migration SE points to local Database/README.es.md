<div align="center">

# Migración de StreamElements a Nyans para Streamer.bot

**Construyendo el futuro de los flujos de trabajo para creadores de contenido**

[🇺🇸 English](README.md) | [🇪🇸 Español](README.es.md)

</div>

Migra puntos de lealtad desde StreamElements al sistema de puntos Nyans utilizando Streamer.bot.

Este paquete incluye tanto una acción completa para Streamer.bot como el script de migración por separado, permitiendo una transición rápida desde StreamElements hacia una base de datos local de Nyans.

## Archivos Incluidos

| Archivo                       | Descripción                                         |
| ----------------------------- | --------------------------------------------------- |
| `Migration Action 1-0.import` | Acción completa lista para importar en Streamer.bot |
| `Migration Script 1-0.cs`     | Script de migración independiente                   |

---

## Características

* Migra puntos desde StreamElements hacia Nyans.
* Evita migraciones de usuarios sin puntos.
* Transfiere automáticamente el saldo exacto.
* Descuenta los puntos migrados de StreamElements.
* Incluye notificaciones al usuario.
* Totalmente compatible con Streamer.bot.
* Diseñado para migraciones únicas.

---

## Flujo de Trabajo

1. Obtiene los puntos del usuario desde StreamElements.
2. Guarda el resultado en:

```text
%sepoints%
```

3. Espera 5 segundos para asegurar que la consulta finalice correctamente.
4. Comprueba si `%sepoints%` es mayor que 0.

### Si existen puntos disponibles

* Ejecuta el script de migración.
* Agrega los puntos a la base de datos de Nyans.
* Descuenta los puntos migrados de StreamElements:

```text
!addpoints %username% -%sepoints%
```

* Notifica al usuario que la migración fue exitosa.

### Si no existen puntos disponibles

* Notifica al usuario que no tiene puntos para migrar.

---

## Requisitos

* Streamer.bot
* Cuenta de StreamElements
* StreamElements Points Fetcher
* Sistema de base de datos Nyans
* JWT Token de StreamElements
* Channel ID de StreamElements

---

## Variables Utilizadas

| Variable     | Descripción                           |
| ------------ | ------------------------------------- |
| `%username%` | Nombre de usuario de Twitch           |
| `%sepoints%` | Puntos obtenidos desde StreamElements |

---

## Notificaciones

Migración exitosa:

```text
%username%, has movido correctamente todos tus %sepoints% puntos al nuevo sistema de Nyans.
```

Sin puntos disponibles:

```text
%username%, no tienes puntos en el sistema antiguo para migrar.
```

---

## Uso Recomendado

Esta acción está diseñada para utilizarse una sola vez por usuario durante la transición desde StreamElements hacia el sistema Nyans.

Una vez que toda la comunidad haya migrado sus puntos, la acción puede eliminarse sin inconvenientes.

---

## Licencia

Este proyecto se distribuye bajo la licencia MIT.
