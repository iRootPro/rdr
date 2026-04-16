# rdr

English | [Русский](README.md)

Terminal RSS/Atom feed reader built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

Vim-style navigation, full article reading, smart folders, query language search, 4 color themes, Russian keyboard layout support.

![rdr demo](demo/demo_en.gif)

## Features

- Split-pane interface: feeds on the left, articles on the right
- Full article reading in the terminal (readability + glamour)
- Feed categories with collapse/expand
- Smart folders (saved search queries)
- Search with query language (`title:rust unread newer:1w`)
- Batch operations: mark read, star, copy by query
- OPML import and export
- 4 themes: Dark (Tokyo Night), Light (Catppuccin Latte), Catppuccin Mocha, Rose Pine
- Nerd Font icons for sources (GitHub, HN, Habr, Lobsters, etc.)
- Powerline status bar
- Localization: English / Russian
- Russian keyboard layout works without switching
- Auto-refresh on a timer
- URL copying via OSC 52 (works over SSH and tmux)

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap iRootPro/tap
brew install rdr
```

### Go

```bash
go install github.com/iRootPro/rdr@latest
```

### From source

```bash
git clone https://github.com/iRootPro/rdr.git
cd rdr
go build -o rdr .
```

### Binaries

Pre-built binaries for macOS (arm64, amd64) and Linux (arm64, amd64) are available on the [Releases](https://github.com/iRootPro/rdr/releases) page.

### Requirements

- Go 1.22+
- Terminal with true color support (Kitty, iTerm2, WezTerm, Ghostty, etc.)
- [Nerd Fonts](https://www.nerdfonts.com/) patched font (for icons)

## Quick Start

```bash
rdr
```

No config file required. Everything can be configured from the UI:

1. Press `s` to open settings
2. Press `a` to add a feed (name + URL)
3. Press `esc` to close settings
4. Press `R` to sync feeds

Feeds, smart folders, categories, language, theme — all configurable via settings (`s`).

## Configuration (optional)

The config file `~/.config/rdr/config.yaml` is optional — everything can be set up from the UI (`s`). The config is handy for bootstrapping feeds on first launch.

Path: `~/.config/rdr/config.yaml` (or `$RDR_HOME/config.yaml`):

```yaml
# Auto-refresh interval in minutes (0 = disabled)
refresh_interval_minutes: 15

# Feeds with optional categories
feeds:
  - name: Hacker News
    url: https://hnrss.org/frontpage
    category: IT
  - name: Lobsters
    url: https://lobste.rs/rss
    category: IT
  - name: Go Blog
    url: https://go.dev/blog/feed.atom
    category: Programming

# Smart folders — saved queries shown in the feed pane
smart_folders:
  - name: Inbox
    query: unread
  - name: Today
    query: today
  - name: This Week
    query: newer:1w unread
  - name: Starred
    query: starred

# Commands run automatically after every sync
after_sync_commands:
  - read title:sponsor
  - read feed:habr title:ad
```

Feeds and smart folders from the config are synced into the database on every launch. All other settings (language, theme, sort, preview) are stored in SQLite and changed via the UI (`s`).

## Navigation

### Global Keys

| Key | Action |
|-----|--------|
| `j` / `k` | Down / up |
| `g` / `G` | Top / bottom |
| `Ctrl+d` / `Ctrl+u` | Page down / up |
| `tab` | Switch pane |
| `enter` | Open |
| `esc` | Back |
| `R` | Refresh all feeds |
| `r` | Refresh current feed |
| `s` | Settings |
| `/` | Search |
| `:` | Command mode |
| `?` | Help |
| `z` | Zen mode (single pane) |
| `q` | Quit |

### Article List

| Key | Action |
|-----|--------|
| `enter` | Open article |
| `x` | Toggle read/unread |
| `X` | Mark all read |
| `m` | Toggle star |
| `n` | Next unread |
| `y` | Copy URL |
| `Y` | Copy as markdown link |
| `o` | Open in browser |
| `a` | Show all |
| `u` | Unread only |
| `S` | Starred only |
| `p` | Toggle preview |

### Reader

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll line by line |
| `space` | Page down |
| `J` / `K` | Next / previous article |
| `f` | Fetch full article |
| `L` | Link picker |
| `o` | Open URL in browser |
| `y` / `Y` | Copy URL / markdown |
| `t` | Translate article (AI) |
| `Ctrl+s` | Summarize article (AI) |
| `x` | Toggle read |
| `m` | Toggle star |
| `esc` | Back to list |

### Settings

Tabs: Feeds · General · Folders · Smart Folders · Auto-commands · AI

| Key | Action |
|-----|--------|
| `tab` | Switch tab |
| `a` | Add feed / folder / auto-command |
| `d` | Delete |
| `e` | Rename / edit |
| `c` | Assign folder to feed |
| `i` | Import OPML |
| `E` | Export OPML |
| `enter` / `space` | Toggle value (General tab) |
| `esc` | Close settings |

**General** tab: language, images, sort, preview, theme, auto-refresh (0/5/15/30/60 min).

**Auto-commands** tab: commands run after every sync (query syntax, e.g. `read title:sponsor`).

> All keys also work with Russian keyboard layout.

## Query Language

Used in search (`/`), smart folders, and batch commands.

```
word                substring match in title or feed name
title:rust          title contains "rust"
feed:habr           feed name contains "habr"
description:python  description contains "python"
unread              unread articles only
read                read articles only
starred             starred articles only
unstarred           not starred
today               published today
yesterday           published yesterday
newer:1w            newer than 1 week (1d, 3h, 45m, 1mo, 1y)
older:2w            older than 2 weeks
~title:ad           negation (does NOT contain "ad")
```

Atoms are combined with AND: `unread title:rust newer:1w` = unread articles with "rust" in the title from the last week.

## Commands

Invoked via `:` (command mode). Tab autocomplete available.

| Command | Description |
|---------|-------------|
| `:sync` | Refresh all feeds |
| `:sort date` / `:sort title` | Change sort |
| `:sortreverse` | Reverse sort order |
| `:filter all\|unread\|starred` | Set filter |
| `:read <query>` | Batch mark as read |
| `:unread <query>` | Batch mark as unread |
| `:star <query>` | Batch star |
| `:unstar <query>` | Batch unstar |
| `:copy url <query>` | Copy URLs of matching articles |
| `:copy md <query>` | Copy as markdown links |
| `:import <path>` | Import feeds from OPML |
| `:export <path>` | Export feeds to OPML |
| `:images` | Toggle image display |
| `:zen` | Toggle zen mode |
| `:collapseall` | Collapse all categories |
| `:expandall` | Expand all categories |
| `:q` | Quit |

## AI: Translation & Summarization

rdr supports article translation and summarization via any OpenAI-compatible API.

### Setup

Open Settings (`s`) > **AI** tab and configure:

| Parameter | Description |
|-----------|-------------|
| Endpoint | API URL (e.g. `http://localhost:11434/v1`) |
| API Key | API key (optional for local models) |
| Model | Model name |

### Usage

In the reader:
- `t` — translate article to the UI language
- `Ctrl+s` — summarize (3-5 key bullet points)

### Provider Examples

**Apple Intelligence** (macOS, free, on-device):
```bash
brew install apfel
brew services start apfel
```
Endpoint: `http://localhost:11434/v1`, Model: `apple-foundationmodel`

**Ollama** (macOS/Linux, free, local):
```bash
brew install ollama && ollama serve
ollama pull llama3
```
Endpoint: `http://localhost:11434/v1`, Model: `llama3`

**OpenAI** (cloud, paid):
Endpoint: `https://api.openai.com/v1`, API Key: `sk-...`, Model: `gpt-4o-mini`

## Feed Catalog

Built-in curated RSS feed directory. Opens automatically on first launch or via `:discover`.

Categories: Tech News, Programming, AI/ML, Security, Linux/Open Source, Science, Health & Fitness, RU Tech, Design.

## Themes

Switch via Settings (`s`) > General > Theme:

- **dark** — Tokyo Night (default)
- **light** — Catppuccin Latte
- **catppuccin** — Catppuccin Mocha
- **rose-pine** — Rose Pine

Light theme works correctly on dark terminals and vice versa.

## Data

- Database: `~/.config/rdr/rdr.db` (SQLite)
- Config: `~/.config/rdr/config.yaml`
- Command history: `~/.config/rdr/history`
- Folder state: `~/.config/rdr/collapsed_categories`

The `RDR_HOME` environment variable overrides the directory.

## License

MIT
