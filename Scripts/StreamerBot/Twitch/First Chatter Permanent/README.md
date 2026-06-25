<p align="right">
🇺🇸 English | <a href="README.es.md">🇪🇸 Español</a>
</p>

# 🎉 First Chatter Logger for Streamer.bot

> 🚀 Part of the SolarisPKN-LiveTools collection for Streamer.bot

Track and manage first-time chatters in your Twitch community using a simple JSON database.

This script automatically records users who chat for the first time and stores them in a JSON file for future reference, statistics, and community engagement.

---

## ✨ Features

* 🎉 Automatically records first-time chatters.
* 📄 Stores data in a human-readable JSON file.
* 📂 Automatically creates folders and files when needed.
* 📊 Count all recorded first chatters.
* 🔍 Check whether a specific user is a first chatter.
* ⚙️ Configurable storage location.
* 🚀 Compatible with Streamer.bot 1.2+.

---

## 📦 Dependency

This script uses:

```text
Newtonsoft.Json
```

which is already included with Streamer.bot.

No additional installation is required.

---

## 📁 Default Storage Location

By default, the script stores data in:

```text
%InstallationPath%\TWITCH\firstchatters.json
```

Example:

```text
C:\Streamer.bot\TWITCH\firstchatters.json
```

---

## ⚙️ Configuration

Configuration can be provided using Global Variables or Action Arguments.

| Variable   | Default                          | Description                     |
| ---------- | -------------------------------- | ------------------------------- |
| basePath   | Streamer.bot installation folder | Base directory                  |
| jsonFolder | TWITCH                           | Folder containing the JSON file |
| jsonFile   | firstchatters.json               | JSON filename                   |

---

## 🎯 Available Methods

### Execute()

Adds the current chatter if they are not already recorded.

Trigger example:

```text
First Words Event
    └── Execute()
```

---

### GetFirstChattersCount()

Returns the total number of recorded first chatters.

Example output:

```text
🎉 We have 250 first chatters in the community!
```

---

### CheckFirstChatter()

Checks if a specific user exists in the database.

Example:

```text
!firstchatter username
```

Possible responses:

```text
✅ username is a first chatter!
```

or

```text
❌ username is not a first chatter.
```

---

## 📄 JSON Structure

Example file:

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

## 💡 Use Cases

* 🎉 Welcome first-time visitors.
* 📊 Community growth statistics.
* 🏆 Loyalty and reward systems.
* 🎁 First chatter giveaways.
* 📈 Stream analytics.

---

## 🚀 Recommended Integration

Use together with:

* 📂 Installation Path Recorder
* 💾 SQLite Points Migrator
* ⭐ StreamElements Points Fetcher

for a complete Streamer.bot community management setup.

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
