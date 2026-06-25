# 📂 Streamer.bot Installation Path Recorder

🌐 **Languages / Idiomas**

- 🇺🇸 English (Current)
- 🇪🇸 [Español](README.es.md)

> 🚀 Part of the SolarisPKN-LiveTools collection for Streamer.bot

A lightweight utility script for Streamer.bot that automatically stores the Streamer.bot installation directory into a Global Variable.

This allows other scripts and actions to build portable paths without requiring hardcoded directories.

---

## ✨ Features

* 📂 Automatically detects the current Streamer.bot installation directory.
* 💾 Stores the path in a Global Variable.
* ⚙️ Configurable variable name.
* 🔄 Configurable path value.
* 🚀 Fully compatible with Streamer.bot 1.2+.
* 🛠️ Ideal for creating portable scripts and actions.

---

## 🤔 Why Use This Script?

Many Streamer.bot scripts require access to files such as:

```text
Databases
DLL files
Images
Configuration files
Logs
```

Using hardcoded paths makes scripts difficult to share.

Instead of:

```text
D:\Users\MyUser\Documents\StreamerBot\data.db
```

you can use:

```text
%InstallationPath%\data.db
```

making the same script work across different computers and installations.

---

## ⚡ Default Behavior

When executed, the script stores:

```text
Environment.CurrentDirectory
```

inside the global variable:

```text
InstallationPath
```

Example:

```text
C:\Streamer.bot
```

---

## ⚙️ Configuration

Configuration can be provided through Global Variables or Action Arguments.

| Variable         | Default Value                  | Description          |
| ---------------- | ------------------------------ | -------------------- |
| installPathVar   | InstallationPath               | Global variable name |
| installPathValue | Current Streamer.bot directory | Custom path value    |

---

## 🛠️ Usage Example

Run the script once during startup.

After execution:

```text
%InstallationPath%
```

will contain:

```text
C:\Streamer.bot
```

You can then build portable paths:

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

## 🚀 Recommended Startup Usage

Create a Streamer.bot startup action:

```text
Streamer.bot Start
    └── Run Installation Path Recorder
```

This ensures the variable is always available before other actions run.

---

## 💡 Common Use Cases

* 📦 Portable database locations
* 🔌 Dynamic DLL loading
* 🖼️ Overlay and image management
* 📁 Shared project folders
* ⚙️ Multi-script environments

---

## ✅ Compatibility

Tested with:

```text
Streamer.bot 1.2+
```

---

## 📜 License

MIT License

Feel free to modify, improve, and share this project.
