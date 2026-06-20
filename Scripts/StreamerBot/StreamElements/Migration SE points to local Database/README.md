<div align="center">

# StreamElements to Nyans Migration for Streamer.bot

**Building the future of content production workflows**

[🇺🇸 English](README.md) | [🇪🇸 Español](README.es.md)

</div>

Migrate loyalty points from StreamElements to the Nyans point system using Streamer.bot.

This package includes both a complete Streamer.bot action and the standalone migration script, allowing creators to quickly transition from StreamElements points to a local Nyans database.

## Included Files

| File                          | Description                                  |
| ----------------------------- | -------------------------------------------- |
| `Migration Action 1-0.import` | Complete Streamer.bot action ready to import |
| `Migration Script 1-0.cs`     | Standalone migration script                  |

---

## Features

* Migrates user points from StreamElements to Nyans.
* Prevents migrating users with zero points.
* Automatically transfers the exact balance.
* Removes migrated points from StreamElements.
* User notifications included.
* Fully compatible with Streamer.bot.
* Designed for one-time migration workflows.

---

## Workflow

1. Retrieves the user's StreamElements points.
2. Stores the value in:

```text
%sepoints%
```

3. Waits 5 seconds to ensure the API request has completed.
4. Checks whether `%sepoints%` is greater than 0.

### If points are available

* Executes the migration script.
* Adds points to the Nyans database.
* Removes the migrated balance from StreamElements:

```text
!addpoints %username% -%sepoints%
```

* Notifies the user that the migration was successful.

### If no points are available

* Notifies the user that they have no points available to migrate.

---

## Requirements

* Streamer.bot
* StreamElements account
* StreamElements Points Fetcher
* Nyans database system
* StreamElements JWT Token
* StreamElements Channel ID

---

## Variables Used

| Variable     | Description                               |
| ------------ | ----------------------------------------- |
| `%username%` | Twitch username                           |
| `%sepoints%` | User points retrieved from StreamElements |

---

## Notifications

Successful migration:

```text
%username%, you have successfully moved all your %sepoints% points to the new Nyans system!
```

No points available:

```text
%username%, you don't have any Nyans in the old system.
```

---

## Recommended Usage

This migration action is intended to be used once per user during the transition from StreamElements points to the Nyans system.

After all users have migrated their balances, the action can be safely removed.

---

## License

This project is released under the MIT License.
