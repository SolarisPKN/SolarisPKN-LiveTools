# SolarisPKN LinkedIn Publisher

A local Markdown/MDX converter that prepares content for review and pasting into LinkedIn's article editor.

It preserves headings, lists, links, bold and italic text, quotes, separators, and code while flagging tables, relative links, raw HTML, MDX expressions, and JSX components for review.

## Features

- Processes files entirely in the browser; MDX content is never uploaded.
- Does not request LinkedIn credentials, cookies, or tokens.
- Never executes MDX imports, exports, JSX, or JavaScript expressions.
- Accepts optional JSON metadata with `title`, `description`, `tags`, and `id`.
- Provides a rich clipboard copy, plain text, preview, image queue, diagnostics, and exports.
- Includes a command-line converter for repeatable workflows.

## Open with a Windows double-click

Run `abrir-linkedin-publisher.cmd`. It checks Node.js and npm, installs local dependencies on the first run, builds the interface, starts the loopback server, and opens the browser automatically.

Keep the command window open while using the publisher. Close it or press `Ctrl+C` to stop the session. If port `4179` is busy, the launcher automatically chooses another local port.

## Start from a terminal

Requirements: Node.js 20 or newer and npm.

```powershell
npm install
npm start
```

Open `http://127.0.0.1:4179`. The server binds only to loopback, and the browser performs the conversion locally.

## Command-line conversion

```powershell
npm run convert -- "D:\path\index-en.mdx" `
  --metadata "D:\path\en\posts\article.json" `
  --base-url "https://solarispkn.dev/en/blog/article/" `
  --article-url "https://solarispkn.dev/en/blog/article/"
```

Use `--out "D:\output"` to choose an output directory.

## Validate

```powershell
npm run check
```

For the complete workflow, compatibility boundaries, and security model, see [README.es.md](README.es.md).
