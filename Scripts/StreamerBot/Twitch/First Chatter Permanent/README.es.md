<p align="right">
<a href="README.md">🇺🇸 English</a> | 🇪🇸 Español
</p>

# 🎉 First Chatter Logger para Streamer.bot

> 🚀 Parte de la colección SolarisPKN-LiveTools para Streamer.bot

Registra y administra los usuarios que escriben por primera vez en tu chat de Twitch utilizando una sencilla base de datos JSON.

El script almacena automáticamente los primeros mensajes de usuarios nuevos para futuras consultas, estadísticas y herramientas de comunidad.

---

## ✨ Características

* 🎉 Registra automáticamente a los usuarios que escriben por primera vez.
* 📄 Almacena la información en un archivo JSON legible.
* 📂 Crea carpetas y archivos automáticamente cuando es necesario.
* 📊 Permite obtener la cantidad total de usuarios registrados.
* 🔍 Permite verificar si un usuario forma parte de la lista.
* ⚙️ Ubicación de almacenamiento configurable.
* 🚀 Compatible con Streamer.bot 1.2+.

---

## 📦 Dependencia

Este script utiliza:

```text
Newtonsoft.Json
```

que ya viene incluido con Streamer.bot.

No requiere instalaciones adicionales.

---

## 📁 Ubicación predeterminada

Por defecto los datos se almacenan en:

```text
%InstallationPath%\TWITCH\firstchatters.json
```

Ejemplo:

```text
C:\Streamer.bot\TWITCH\firstchatters.json
```

---

## ⚙️ Configuración

La configuración puede realizarse mediante Variables Globales o Argumentos.

| Variable   | Valor por defecto                      | Descripción                       |
| ---------- | -------------------------------------- | --------------------------------- |
| basePath   | Carpeta de instalación de Streamer.bot | Directorio base                   |
| jsonFolder | TWITCH                                 | Carpeta donde se almacena el JSON |
| jsonFile   | firstchatters.json                     | Nombre del archivo JSON           |

---

## 🎯 Métodos disponibles

### Execute()

Agrega al usuario actual si aún no existe en la lista.

Ejemplo:

```text
Evento Primer Mensaje
    └── Execute()
```

---

### GetFirstChattersCount()

Obtiene la cantidad total de usuarios registrados.

Ejemplo:

```text
🎉 Tenemos 250 primeros visitantes registrados en la comunidad.
```

---

### CheckFirstChatter()

Verifica si un usuario existe en la lista.

Ejemplo:

```text
!firstchatter usuario
```

Posibles respuestas:

```text
✅ usuario es un primer visitante registrado.
```

o

```text
❌ usuario no se encuentra en la lista.
```

---

## 📄 Estructura JSON

Ejemplo:

```json
[
  {
    "Username": "viewer123",
    "FirstChatDate": "2026-06-25 12:00:00",
    "AddedDate": "2026-06-25 12:00:00",
    "Platform": "twitch"
  }
]
```

---

## 💡 Casos de uso

* 🎉 Dar la bienvenida a nuevos espectadores.
* 📊 Estadísticas de crecimiento de la comunidad.
* 🏆 Sistemas de recompensas y fidelización.
* 🎁 Sorteos para nuevos visitantes.
* 📈 Métricas y análisis de streams.

---

## 🚀 Integración recomendada

Funciona muy bien junto a:

* 📂 Installation Path Recorder
* 💾 SQLite Points Migrator
* ⭐ StreamElements Points Fetcher

para construir un ecosistema completo de automatización en Streamer.bot.

---

## ✅ Compatibilidad

Probado con:

```text
Streamer.bot 1.2+
```

---

## 📜 Licencia

Licencia MIT

Sentite libre de modificar, mejorar y compartir este proyecto.
