<div align="center">

# `mdpush`

**Beautiful markdown, shared fast.**
The zero-knowledge CLI for sharing markdown with humans and AI agents.

[![License: MIT](https://img.shields.io/badge/license-MIT-3ddc97.svg)](LICENSE)
[![Go Report](https://img.shields.io/badge/go-1.26-3ddc97.svg)](go.mod)
[![Built with](https://img.shields.io/badge/built%20with-bubbletea%20%C2%B7%20glamour-3ddc97.svg)](https://charm.sh)

</div>

---

```bash
$ mdpush share spec.md

  ✓ encrypted client-side
  ✓ uploaded
  🔗 https://mdpush.io/d/k7f2x9
  🔒 password: arctic-fox
```

One command in. One link out. The server never sees your content — not the title, not the body, not the project name. The decryption key rides in the URL fragment, which browsers keep client-side. We literally can't read your docs.

## Why it exists

Sharing a markdown file usually means picking the wrong tool:

- **Slack** strips the formatting and buries it in scrollback
- **Gist** is public-by-default and stripped of polish
- **Notion / wikis** require accounts, permissions, and a project to set up
- **Email** is, well, email

`mdpush` is a tiny tool with one job: take a `.md` file and turn it into a beautifully rendered, end-to-end encrypted, throwaway link in one command. It feels as fast as `scp`, looks as nice as a docs site, and the operators of the service can't read a word of it.

## Features

- 🔒 **Zero-knowledge encryption** — AES-256-GCM, key never leaves your machine
- 🪶 **Light & strong locks** — "Who sent you this?" or memorable themed passwords (`arctic-fox`, `brave-new-world`, `cerulean-midnight`)
- ⏳ **Expiration & view limits** — `--expires 7d --views 10`
- 🎨 **Themed reading experience** — clean, dark, github, technical
- 📥 **Inbox & library** — fetch, list, export docs people share with you
- 🤖 **Agent-friendly** — `mdpush agent-skill` returns a paste-ready skill spec for Claude Code, Cursor, and friends
- 🖥️ **TUI** — built on [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Glamour](https://github.com/charmbracelet/glamour)
- 🪪 **Open source** — auditable encryption, no hidden behavior

## Install

```bash
brew tap mdpush-io/mdpush
brew install mdpush
```

Or build from source:

```bash
go install github.com/mdpush-io/cli@latest
```

## Quick start

### 1. Log in

```bash
mdpush login
```

You'll get a 6-digit code by email. The CLI then asks you to set a passphrase that protects your encryption keys — **this passphrase never leaves your machine**, and we never see it. You'll get a recovery code; save it somewhere safe.

### 2. Share a file

```bash
mdpush share notes.md
```

That's it. You get a link and (if strong-locked) a password. Paste the link into Slack, an email, or wherever — only people with the link (and the lock credential) can read it.

### 3. Fancier shares

```bash
# strong lock with a memorable themed password
mdpush share spec.md --lock strong --pw-theme books
#   🔒 password: brave-new-world

# expiration + view limit
mdpush share draft.md --expires 3d --views 5

# tag with category & project
mdpush share spec.md --category new-feature --project mdpush

# pipe from stdin
cat report.md | mdpush share - --title "Q3 report"
```

### 4. Read what people send you

```bash
# fetch + render in your terminal
mdpush fetch https://mdpush.io/d/k7f2x9#k=...

# list everything in your inbox
mdpush list
```

### 5. Manage your stuff

```bash
mdpush list                          # what you've shared
mdpush extend abc123 --expires 7d    # add more time
mdpush revoke abc123                 # kill access right now
mdpush update abc123 spec.md         # replace contents
mdpush delete abc123                 # purge it
```

## All commands

| Command | What it does |
|---|---|
| `mdpush share <file.md>` | Encrypt and share a markdown document |
| `mdpush fetch <url-or-id>` | Fetch, decrypt, and render a doc |
| `mdpush list` | List your sent docs and inbox |
| `mdpush open <title>` | Open a locally-saved doc by title |
| `mdpush copy <id>` | Copy a doc's share URL to your clipboard |
| `mdpush update <id> <file.md>` | Replace a doc's contents |
| `mdpush extend <id>` | Extend expiration or view limit |
| `mdpush revoke <id>` | Revoke access |
| `mdpush delete <id>` | Permanently delete a doc |
| `mdpush export <id>` | Export decrypted content to a file |
| `mdpush login` | Authenticate (email code + passphrase) |
| `mdpush logout` | Clear local session |
| `mdpush whoami` | Show the current authenticated user |
| `mdpush config set <k> <v>` | Set a default for share flags |
| `mdpush config list` | View current defaults |
| `mdpush config reset` | Reset all defaults |
| `mdpush agent-skill` | Print a paste-ready skill spec for AI agents |

Run `mdpush <command> --help` for full flag documentation.

## Defaults

Tired of typing the same flags every time? Set defaults once:

```bash
mdpush config set lock light
mdpush config set expires 7d
mdpush config set theme dark
mdpush config set project my-project
mdpush config set pw-theme books
```

Stored in `~/.config/mdpush/config.json`. Explicit flags always win.

## How the encryption works (in plain English)

1. **Encryption happens on your device.** Before your markdown leaves your machine, the CLI encrypts it with `AES-256-GCM` using a fresh key generated just for that document.
2. **The key never touches the server.** It rides in the URL fragment (`https://mdpush.io/d/abc123#k=...`) — browsers are required to keep that part client-side.
3. **The server only stores ciphertext.** Title, content, category, and project name are all encrypted. Our database holds an opaque blob and a few operational fields (expiration, view count, lock type).
4. **The reader decrypts in their browser.** Their browser pulls the ciphertext, lifts the key out of the fragment, and renders the markdown locally. Plaintext never reaches the server, on either end.

The CLI is open source precisely so you can audit this for yourself. Read [`internal/crypto`](internal/crypto), run it locally, watch what gets uploaded.

For the full architecture, see the [mdpush.io privacy model](https://mdpush.io/privacy-model).

## Working with AI agents

Run:

```bash
mdpush agent-skill
```

You'll get a markdown skill spec you can paste into Claude Code, Cursor, or any agent that reads skills. Once installed, your agent can:

- Push documents on your behalf (`mdpush share`)
- Query your inbox (`mdpush list`)
- Fetch and read shared docs (`mdpush fetch`)
- Tag, extend, revoke, and clean up

The CLI is designed to be small, scriptable, and predictable — exactly what an agent wants from a tool.

## Building from source

```bash
git clone https://github.com/mdpush-io/cli.git
cd cli
go build -o mdpush
./mdpush --help
```

Requires Go 1.26+.

## Contributing

Issues, ideas, and PRs are very welcome — especially for:

- New password themes
- Additional reading themes
- Platform-specific tweaks (Windows, Linux distro packaging)
- Agent skill integrations for more environments
- Bug fixes for the TUI

Open an issue first if you're planning a substantial change so we can discuss the shape.

## License

[MIT](LICENSE) — do what you want with it.

## Built by

[Gabriel Medeiros](https://www.linkedin.com/in/gabriel-medeiros-do-nascimento/) — Senior Data Engineer, terminal enthusiast, and the only human in the inbox at [mdpush.io](https://mdpush.io).

Reach out to talk software engineering, photography, books, videogames, or yorkshires.
