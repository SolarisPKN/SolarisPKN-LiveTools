# SolarisPKN Multiplatform Publisher

Local Markdown/MDX publisher that creates one canonical content model and adapts it to articles, social posts, captions, messages, and SolarisPKN Labs bundles. Previewing stays local. A real API write requires reviewed targets, the unchanged SHA-256 plan, and the literal `PUBLICAR` confirmation.

Double-click `abrir-publicador-multiplataforma.cmd` on Windows. Node.js 20+ is required. On first use the launcher installs both the shared `PublisherSuite/Linkedin` converter dependencies and this module's dependencies, builds the UI, binds only to `127.0.0.1`, and opens the browser.

Implemented official connectors: DEV, Blogger, Tumblr, LinkedIn feed posts, Facebook Pages, professional Instagram image posts, Threads, Mastodon, Bluesky, Reddit self posts, Pinterest Pins, and Discord incoming webhooks. SolarisPKN Labs produces a local draft bundle. Medium, Hashnode, and X are assisted because Medium rejects new integrations, Hashnode publishing requires Pro, and X writes are paid. Video platforms only receive prepared metadata in this Markdown-first release.

See [README.es.md](README.es.md) for setup, environment variables, the complete capability matrix, security boundaries, and commands. See [docs/API-RESEARCH-2026-08-29.es.md](docs/API-RESEARCH-2026-08-29.es.md) for the dated research, [config/profiles.json](config/profiles.json) for the 109-profile inventory, and [docs/PROFILE-AUDIT-2026-08-29.es.md](docs/PROFILE-AUDIT-2026-08-29.es.md) for the HTTP findings.
