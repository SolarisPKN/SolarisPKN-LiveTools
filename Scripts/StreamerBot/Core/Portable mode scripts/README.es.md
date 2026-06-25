# 📂 Streamer.bot Installation Path Recorder

🌐 **Idiomas / Languages**

- 🇺🇸 [English](README.md)
- 🇪🇸 Español (Actual)

> 🚀 Parte de la colección SolarisPKN-LiveTools para Streamer.bot

Script liviano para Streamer.bot que guarda automáticamente la ruta de instalación de Streamer.bot en una Variable Global.

Esto permite que otros scripts y acciones utilicen rutas portables sin depender de directorios fijos.

---

## ✨ Características

* 📂 Detecta automáticamente la carpeta actual de Streamer.bot.
* 💾 Guarda la ruta en una Variable Global.
* ⚙️ Nombre de variable configurable.
* 🔄 Valor configurable.
* 🚀 Compatible con Streamer.bot 1.2+.
* 🛠️ Ideal para crear scripts y acciones portables.

---

## 🤔 ¿Por qué usar este script?

Muchos scripts de Streamer.bot necesitan acceder a archivos como:

```text
Bases de datos
DLLs
Imágenes
Archivos de configuración
Logs
```

Usar rutas fijas dificulta compartir proyectos.

En lugar de:

```text
D:\Usuarios\MiUsuario\Documents\StreamerBot\data.db
```

podés utilizar:

```text
%InstallationPath%\data.db
```

permitiendo que el mismo script funcione en distintas computadoras e instalaciones.

---

## ⚡ Comportamiento por defecto

Al ejecutarse, el script guarda:

```text
Environment.CurrentDirectory
```

en la variable global:

```text
InstallationPath
```

Ejemplo:

```text
C:\Streamer.bot
```

---

## ⚙️ Configuración

La configuración puede realizarse mediante Variables Globales o Argumentos.

| Variable         | Valor por defecto                 | Descripción                  |
| ---------------- | --------------------------------- | ---------------------------- |
| installPathVar   | InstallationPath                  | Nombre de la variable global |
| installPathValue | Directorio actual de Streamer.bot | Ruta personalizada           |

---

## 🛠️ Ejemplo de uso

Ejecutá el script una vez al iniciar Streamer.bot.

Luego de ejecutarlo:

```text
%InstallationPath%
```

contendrá:

```text
C:\Streamer.bot
```

A partir de ahí podés construir rutas portables:

```text
%InstallationPath%\dependencies\System.Data.SQLite.dll
```

```text
%InstallationPath%\database\PuntosRecompensas.db
```

```text
%InstallationPath%\images\overlay.png
```

---

## 🚀 Uso recomendado

Crear una acción de inicio:

```text
Streamer.bot Start
    └── Ejecutar Installation Path Recorder
```

De esta manera la variable estará disponible para cualquier otro script o acción.

---

## 💡 Casos de uso comunes

* 📦 Ubicación portable de bases de datos
* 🔌 Carga dinámica de DLLs
* 🖼️ Gestión de overlays e imágenes
* 📁 Carpetas compartidas entre proyectos
* ⚙️ Entornos con múltiples scripts

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
