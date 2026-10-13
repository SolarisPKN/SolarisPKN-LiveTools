# MultiChat — SolarisPKN LiveTools

**Browser-free collectors, one shared chat.** A small Go 1.22+ command-line app with no external Go modules. Collectors run in one process; an optional **single local** HTML/SSE view is reusable in OBS. No Chromium, Electron, per-platform tabs or video streams.

## Available in this MVP

| Connector | Transport | Status |
|---|---|---|
| Twitch | OAuth IRC over TLS 6697 | Coded; live credentials not yet tested |
| YouTube Live | official liveChatMessages.list | Coded; live credentials not yet tested |
| Rumble | HTTPS private Live Stream API | Coded; live credentials not yet tested |
| External adapters | authenticated loopback POST /api/ingest | Local tests passed |
| Kick, TikTok, CHZZK, Bilibili, SOOP and others | — | Not implemented yet |

### Quick start

```powershell
cd MultiChat
Copy-Item config.example.json config.json
go test ./...
go build -trimpath -ldflags="-s -w" -o MultiChat.exe .
.\MultiChat.exe -config config.json -demo
```

Open **http://127.0.0.1:8765** or add it to **one OBS Browser Source**. The `-demo` flag uses synthetic messages, NOT real streams. `-quiet` disables terminal message printing. On Windows, `iniciar-multichat.cmd` runs a local binary or `go run` if Go is installed.

Connectors are OFF by default. Copy and edit `config.example.json` into `config.json`, set the desired `enabled` fields, and configure secret environment variables: `MULTICHAT_TWITCH_TOKEN`, `MULTICHAT_YOUTUBE_API_KEY`, `MULTICHAT_RUMBLE_API_URL`. Optional local ingestion requires `MULTICHAT_INGEST_TOKEN` (24+ characters). **Never commit credentials, URLs with private keys, or config.json**.

### API and design

- `GET /health`: health and enabled connectors (no secrets).
- `GET /history`: bounded recent messages as JSON.
- `GET /events`: Server-Sent Events for the one shared viewer.
- `POST /api/ingest`: optional token-authenticated local entry point.

The localhost server never serves external platform pages. Messages are in memory only. YouTube polling has a minimum 20-second interval, local per-process budget and honors server polling guidance; **official quotas still apply and restart resets the local counter**. Prefer the official `streamList` upgrade later.

Full Spanish instructions: [README.es.md](README.es.md). Technical documentation: [Architecture](docs/ARCHITECTURE.es.md), [Connectors](docs/CONNECTORS.es.md), [Security](docs/SECURITY.es.md), [Windows & OBS](docs/OBS.es.md).

Official docs: [Twitch IRC](https://dev.twitch.tv/docs/chat/irc/), [YouTube liveChatMessages](https://developers.google.com/youtube/v3/live/docs/liveChatMessages/list), [Rumble API](https://rumble.support/help/how-to-use-rumble-s-live-stream-api).

**Scope:** This is an MVP, not a claim that every listed streaming platform already works. Real API connectivity and performance targets need verification on user hardware.
