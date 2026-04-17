# Title

I built a terminal RSS reader with vim keybindings, AI translation, and a built-in feed catalog

# Body

Hey everyone! I've been working on **rdr** — a TUI RSS/Atom reader built with Go (Bubble Tea + Lip Gloss). I wanted something fast, keyboard-driven, and that works well over SSH.

## What it does

- **Split-pane layout** — feeds on the left, articles on the right, full-screen reader
- **Vim-style navigation** — j/k, g/G, Ctrl+d/u, counts (5j), the works
- **Full article rendering** — fetches the full article via readability and renders it with glamour (markdown in your terminal)
- **Smart folders** — saved queries like `unread newer:1w title:rust`
- **Query language** — search with `title:`, `feed:`, `unread`, `starred`, `today`, `newer:1w`, negation with `~`
- **AI translation & summarization** — works with Claude Code (subscription), Apple Intelligence (apfel), Ollama, or any OpenAI-compatible API. Press `t` to translate, `Ctrl+s` to summarize
- **Feed catalog** — built-in curated directory with 37 feeds in 9 categories. Opens on first launch as onboarding
- **4 themes** — Dark (Tokyo Night), Light (Catppuccin Latte), Catppuccin Mocha, Rose Pine. Light theme works correctly on dark terminals and vice versa
- **Nerd Font icons** — source-specific icons (GitHub, HN, Habr, Reddit, Go, etc.), powerline status bar
- **Russian keyboard layout** — all keybindings work without switching to English
- **OPML import/export**
- **Batch operations** — `:read title:sponsor`, `:star feed:go`, `:copy md unread`
- **OSC 52 clipboard** — copy URLs over SSH/tmux
- **Auto-refresh** with configurable interval
- **After-sync commands** — auto-mark ads as read

## Install

```bash
brew tap iRootPro/tap && brew install rdr
```

Or: `go install github.com/iRootPro/rdr@latest`

No config needed — first launch opens a language picker, then a feed catalog where you pick what to subscribe to. Smart folders (Inbox, Today, This Week, Starred) are created automatically.

## Tech stack

- Go 1.22+, Bubble Tea, Lip Gloss, glamour
- SQLite (modernc.org/sqlite, pure Go, no CGO)
- go-readability for full article extraction

## Links

- GitHub: https://github.com/iRootPro/rdr
- Homebrew: `brew tap iRootPro/tap && brew install rdr`

Would love feedback! What features would you want in a terminal RSS reader?

# Subreddits to post

- r/commandline — main audience
- r/golang — Go community
- r/linux — terminal users
- r/selfhosted — RSS reader users
