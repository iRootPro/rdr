# rdr Phase 1 — Step 4: TUI skeleton (split-pane)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the CLI `--fetch` scaffolding with a real Bubble Tea UI:
split-pane FeedList + ArticleList with Tokyo Night styling, vim-style
navigation, auto-fetch on startup, `r`/`R` refresh, status bar with
spinner, and a live unread counter that updates when the fetcher returns.

**Architecture:** New package `internal/ui/` with files split by
responsibility: `styles.go` (palette + lipgloss styles), `keys.go`
(KeyMap), `messages.go` (tea.Msg types), `model.go` (Model + Update +
View + Init), `feedlist.go` (left-pane render helper), `articlelist.go`
(right-pane render helper). Async work uses `tea.Cmd` — feeds/articles
load from DB, fetcher runs via `FetchAll`. One focus state machine:
`focusFeeds` ↔ `focusArticles` (Reader focus is Step 5). `main.go` drops
the `--fetch` flag and becomes `tea.NewProgram(ui.New(db, fetcher)).Run()`.

**Tech Stack:** `github.com/charmbracelet/bubbletea`,
`github.com/charmbracelet/bubbles` (spinner, key), `github.com/charmbracelet/lipgloss`,
existing `internal/db` / `internal/feed`.

**Related docs:** `SPEC.md` (key bindings, Tokyo Night palette),
`docs/plans/2026-04-13-rdr-phase1-design.md` (UI section).

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Testing approach:** No unit tests in `internal/ui/`. UI is verified
manually by running `go build ./...` and `RDR_HOME=./dev go run .` between
tasks. Each task leaves the code in a compilable, visually-testable
state. Per the design doc: "UI руками. TUI snapshot-тесты хрупкие, дают
мало пользы."

**Git note:** parent repo at `/Users/sasha/Code/github.com/iRootPro`,
commit with `git add -f rdr/...`.

---

## Tokyo Night palette (reference for all tasks)

From `SPEC.md`:

| Name | Hex | Use |
|---|---|---|
| Background | `#1a1b26` | pane background |
| AltBG | `#24283b` | selected row background |
| Border | `#3b4261` | pane borders |
| Muted | `#565f89` | read items, inactive panes |
| Text | `#c0caf5` | default text |
| Accent | `#7aa2f7` | active pane border, headers |
| Secondary | `#bb9af7` | selected feed |
| Green | `#9ece6a` | unread counters, source |
| Orange | `#ff9e64` | time ago |
| Red | `#f7768e` | errors |
| Yellow | `#e0af68` | unread articles |
| Teal | `#2ac3de` | URLs |

---

## Task 1: UI skeleton — deps + styles + keys + messages + minimal Model

**Goal:** Get a runnable bubbletea app on screen. At the end of this task
`go run .` opens a window that shows "rdr — loading..." and responds to
`q`. No data yet.

**Files:**
- Modify: `go.mod` / `go.sum` (via `go get`)
- Create: `internal/ui/styles.go`
- Create: `internal/ui/keys.go`
- Create: `internal/ui/messages.go`
- Create: `internal/ui/model.go`
- Modify: `main.go`

**Step 1.1: Install bubbletea deps**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go mod tidy
```

Expected: `go.mod` gains the three direct requires.

**Step 1.2: Create `styles.go`**

Create `rdr/internal/ui/styles.go`:

```go
package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorBG        = lipgloss.Color("#1a1b26")
	colorAltBG     = lipgloss.Color("#24283b")
	colorBorder    = lipgloss.Color("#3b4261")
	colorMuted     = lipgloss.Color("#565f89")
	colorText      = lipgloss.Color("#c0caf5")
	colorAccent    = lipgloss.Color("#7aa2f7")
	colorSecondary = lipgloss.Color("#bb9af7")
	colorGreen     = lipgloss.Color("#9ece6a")
	colorOrange    = lipgloss.Color("#ff9e64")
	colorRed       = lipgloss.Color("#f7768e")
	colorYellow    = lipgloss.Color("#e0af68")
	colorTeal      = lipgloss.Color("#2ac3de")

	paneActive = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	paneInactive = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	paneTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)

	itemSelected = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	itemSelectedInactive = lipgloss.NewStyle().
				Foreground(colorMuted)

	unreadStyle = lipgloss.NewStyle().Foreground(colorYellow)
	readStyle   = lipgloss.NewStyle().Foreground(colorMuted)

	counterStyle = lipgloss.NewStyle().Foreground(colorGreen)
	timeAgoStyle = lipgloss.NewStyle().Foreground(colorOrange)
	errStyle     = lipgloss.NewStyle().Foreground(colorRed)

	statusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)
)
```

**Step 1.3: Create `keys.go`**

Create `rdr/internal/ui/keys.go`:

```go
package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Tab        key.Binding
	Top        key.Binding
	Bottom     key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Enter      key.Binding
	Back       key.Binding
	RefreshOne key.Binding
	RefreshAll key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
		Up:         key.NewBinding(key.WithKeys("k", "up")),
		Down:       key.NewBinding(key.WithKeys("j", "down")),
		Left:       key.NewBinding(key.WithKeys("h", "left")),
		Right:      key.NewBinding(key.WithKeys("l", "right")),
		Tab:        key.NewBinding(key.WithKeys("tab")),
		Top:        key.NewBinding(key.WithKeys("g", "home")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end")),
		PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
		PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown")),
		Enter:      key.NewBinding(key.WithKeys("enter")),
		Back:       key.NewBinding(key.WithKeys("esc")),
		RefreshOne: key.NewBinding(key.WithKeys("r")),
		RefreshAll: key.NewBinding(key.WithKeys("R")),
	}
}
```

**Step 1.4: Create `messages.go`**

Create `rdr/internal/ui/messages.go`:

```go
package ui

import (
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

type feedsLoadedMsg struct {
	feeds []db.Feed
}

type articlesLoadedMsg struct {
	feedID   int64
	articles []db.Article
}

type fetchStartedMsg struct{}

type fetchDoneMsg struct {
	results []feed.FetchResult
}

type errMsg struct {
	err error
}
```

**Step 1.5: Create minimal `model.go`**

Create `rdr/internal/ui/model.go`:

```go
package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

type focus int

const (
	focusFeeds focus = iota
	focusArticles
)

type Model struct {
	db      *db.DB
	fetcher *feed.Fetcher
	keys    keyMap

	feeds    []db.Feed
	articles []db.Article
	selFeed  int
	selArt   int
	focus    focus

	width  int
	height int

	status string
	err    error
}

func New(database *db.DB, fetcher *feed.Fetcher) Model {
	return Model{
		db:      database,
		fetcher: fetcher,
		keys:    defaultKeys(),
		status:  "loading…",
	}
}

func (m Model) Init() tea.Cmd {
	return loadFeedsCmd(m.db)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}

	case feedsLoadedMsg:
		m.feeds = msg.feeds
		m.status = "ready"
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errStyle.Render("error: " + m.err.Error()) + "\n"
	}
	return "rdr — " + m.status + "\n"
}

func loadFeedsCmd(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		feeds, err := d.ListFeeds()
		if err != nil {
			return errMsg{err}
		}
		return feedsLoadedMsg{feeds: feeds}
	}
}

func fetchAllCmd(f *feed.Fetcher) tea.Cmd {
	return func() tea.Msg {
		results, err := f.FetchAll(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return fetchDoneMsg{results: results}
	}
}
```

Note: the import block needs `github.com/charmbracelet/bubbles/key`. Add
it.

**Step 1.6: Rewrite `main.go`**

Replace `rdr/main.go`:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iRootPro/rdr/internal/config"
	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/ui"
)

func main() {
	home, err := config.ResolveHome()
	if err != nil {
		log.Fatalf("resolve home: %v", err)
	}

	database, err := db.Open(filepath.Join(home, "rdr.db"))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	cfg, err := config.Load(home)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := config.Sync(database, cfg); err != nil {
		log.Fatalf("sync feeds: %v", err)
	}

	fetcher := feed.New(database)
	program := tea.NewProgram(ui.New(database, fetcher), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
```

The `--fetch` flag is gone. The UI owns fetching from here on.

**Step 1.7: Build**

```bash
cd /Users/sasha/Code/github.com/iRootPro/rdr
go build ./...
```

Expected: exits 0. If the import for `key` is missing in `model.go`, add
it.

**Step 1.8: Smoke test**

```bash
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected: the terminal switches to alt-screen, shows either `rdr — loading…`
briefly then `rdr — ready`, and `q` quits cleanly (returns to your
shell). If the DB is empty or the terminal is weird, the app should still
quit on `q`. Clean up: `rm -rf dev`.

**Step 1.9: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/go.mod rdr/go.sum \
    rdr/internal/ui/styles.go \
    rdr/internal/ui/keys.go \
    rdr/internal/ui/messages.go \
    rdr/internal/ui/model.go \
    rdr/main.go
git commit -m "feat(rdr): add ui package skeleton and bubbletea entrypoint"
```

---

## Task 2: Render FeedList (left pane) with real data

**Goal:** Draw the feed list with Tokyo Night colors, unread counters,
and a selected item indicator. Still a single-pane view at this point.

**Files:**
- Create: `internal/ui/feedlist.go`
- Modify: `internal/ui/model.go` (View method)

**Step 2.1: Create `feedlist.go`**

Create `rdr/internal/ui/feedlist.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

func renderFeedList(feeds []db.Feed, selected int, active bool, width, height int) string {
	var b strings.Builder
	b.WriteString(paneTitle.Render("Feeds"))
	b.WriteString("\n")

	if len(feeds) == 0 {
		b.WriteString(readStyle.Render("(no feeds)"))
		return framePane(b.String(), active, width, height)
	}

	for i, f := range feeds {
		name := f.Name
		counter := ""
		if f.UnreadCount > 0 {
			counter = counterStyle.Render(fmt.Sprintf("%d", f.UnreadCount))
		}

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(width-6).Render(name),
			counter,
		)
		if i == selected {
			if active {
				line = itemSelected.Render("› " + name)
			} else {
				line = itemSelectedInactive.Render("› " + name)
			}
			if f.UnreadCount > 0 {
				line = lipgloss.JoinHorizontal(lipgloss.Top,
					lipgloss.NewStyle().Width(width-6).Render(line),
					counter,
				)
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return framePane(b.String(), active, width, height)
}

func framePane(content string, active bool, width, height int) string {
	style := paneInactive
	if active {
		style = paneActive
	}
	return style.Width(width).Height(height).Render(content)
}
```

**Step 2.2: Update `model.View`**

Replace the `View` method in `rdr/internal/ui/model.go` with:

```go
func (m Model) View() string {
	if m.err != nil {
		return errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	if m.width == 0 || m.height == 0 {
		return "rdr — " + m.status
	}

	leftW := m.width/3 - 2
	paneH := m.height - 2 // leave room for status bar

	left := renderFeedList(m.feeds, m.selFeed, m.focus == focusFeeds, leftW, paneH)

	status := statusBar.Width(m.width).Render("rdr · " + m.status)

	return lipgloss.JoinVertical(lipgloss.Top, left, status)
}
```

**Step 2.3: Build + smoke test**

```bash
go build ./...
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected: terminal shows a left-side pane titled "Feeds" with the 3
seeded feeds (Hacker News, Go Blog, Lobsters), an active blue border, and
the first feed marked with `›`. Counters show 0 (no articles until fetch
runs). `q` quits.

Clean up: `rm -rf dev`.

**Step 2.4: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/feedlist.go rdr/internal/ui/model.go
git commit -m "feat(rdr): render feed list with Tokyo Night styles"
```

---

## Task 3: Render ArticleList (right pane) + focus switching

**Goal:** Two-pane layout. Tab switches focus between FeedList and
ArticleList. Selecting a different feed reloads its articles.

**Files:**
- Create: `internal/ui/articlelist.go`
- Modify: `internal/ui/model.go`

**Step 3.1: Create `articlelist.go`**

Create `rdr/internal/ui/articlelist.go`:

```go
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

func renderArticleList(articles []db.Article, selected int, active bool, width, height int) string {
	var b strings.Builder
	b.WriteString(paneTitle.Render("Articles"))
	b.WriteString("\n")

	if len(articles) == 0 {
		b.WriteString(readStyle.Render("(no articles)"))
		return framePane(b.String(), active, width, height)
	}

	for i, a := range articles {
		style := unreadStyle
		if a.ReadAt != nil {
			style = readStyle
		}
		title := truncate(a.Title, width-14)
		when := timeAgoStyle.Render(timeAgo(a.PublishedAt))

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			style.Width(width-12).Render(title),
			when,
		)
		if i == selected {
			prefix := "› "
			if !active {
				prefix = "  "
			}
			line = lipgloss.JoinHorizontal(
				lipgloss.Top,
				itemSelected.Render(prefix+truncate(a.Title, width-14)),
				" ",
				when,
			)
			if !active {
				line = itemSelectedInactive.Render(prefix+truncate(a.Title, width-14)) + " " + when
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	return framePane(b.String(), active, width, height)
}

func truncate(s string, max int) string {
	if max <= 1 {
		return "…"
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
```

**Step 3.2: Update model — add loadArticlesCmd + tab handling**

Add to `rdr/internal/ui/model.go`, alongside `loadFeedsCmd` and
`fetchAllCmd`:

```go
func loadArticlesCmd(d *db.DB, feedID int64) tea.Cmd {
	return func() tea.Msg {
		articles, err := d.ListArticles(feedID, 100)
		if err != nil {
			return errMsg{err}
		}
		return articlesLoadedMsg{feedID: feedID, articles: articles}
	}
}
```

Update `Update` to handle `articlesLoadedMsg`, `Tab`, and feed-selection
changes. Replace the key/Message handling block with:

```go
case tea.KeyMsg:
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Tab):
		if m.focus == focusFeeds {
			m.focus = focusArticles
		} else {
			m.focus = focusFeeds
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		return m.moveDown()
	case key.Matches(msg, m.keys.Up):
		return m.moveUp()
	}

case feedsLoadedMsg:
	m.feeds = msg.feeds
	m.status = "ready"
	if len(m.feeds) > 0 {
		return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
	}
	return m, nil

case articlesLoadedMsg:
	if len(m.feeds) > 0 && m.feeds[m.selFeed].ID == msg.feedID {
		m.articles = msg.articles
		m.selArt = 0
	}
	return m, nil
```

Add navigation helpers to model.go:

```go
func (m Model) moveDown() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if m.selFeed < len(m.feeds)-1 {
			m.selFeed++
			return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
		}
	case focusArticles:
		if m.selArt < len(m.articles)-1 {
			m.selArt++
		}
	}
	return m, nil
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if m.selFeed > 0 {
			m.selFeed--
			return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
		}
	case focusArticles:
		if m.selArt > 0 {
			m.selArt--
		}
	}
	return m, nil
}
```

Update `View` to render both panes side by side:

```go
func (m Model) View() string {
	if m.err != nil {
		return errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	if m.width == 0 || m.height == 0 {
		return "rdr — " + m.status
	}

	leftW := m.width/3 - 2
	rightW := m.width - leftW - 4
	paneH := m.height - 2

	left := renderFeedList(m.feeds, m.selFeed, m.focus == focusFeeds, leftW, paneH)
	right := renderArticleList(m.articles, m.selArt, m.focus == focusArticles, rightW, paneH)

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	status := statusBar.Width(m.width).Render("rdr · " + m.status)
	return lipgloss.JoinVertical(lipgloss.Top, row, status)
}
```

**Step 3.3: Build + smoke test**

```bash
go build ./...
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected:
- Two panes side by side: Feeds left, Articles right
- Left has 3 feeds, first highlighted with blue border active
- Right says "(no articles)" (feeds have no articles yet, fetch hasn't happened)
- `j`/`k` moves feed selection (articles pane re-loads)
- `tab` switches the active border to the right pane
- `q` quits

Clean up: `rm -rf dev`.

**Step 3.4: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/articlelist.go rdr/internal/ui/model.go
git commit -m "feat(rdr): render article list and add tab focus switching"
```

---

## Task 4: Auto-fetch on `Init()` + `r`/`R` refresh + spinner status bar

**Goal:** When the app starts, kick off `FetchAll` in the background.
`r` refetches the current feed; `R` refetches all feeds. A spinner in
the status bar animates while a fetch is in flight.

**Files:**
- Modify: `internal/ui/model.go`

**Step 4.1: Add spinner field**

Add to imports:
```go
"github.com/charmbracelet/bubbles/spinner"
```

Extend the `Model` struct:

```go
type Model struct {
	// ... existing fields ...
	spin     spinner.Model
	fetching bool
}
```

Update `New`:

```go
func New(database *db.DB, fetcher *feed.Fetcher) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAccent)
	return Model{
		db:      database,
		fetcher: fetcher,
		keys:    defaultKeys(),
		status:  "loading…",
		spin:    s,
	}
}
```

**Step 4.2: Kick off auto-fetch in `Init`**

```go
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadFeedsCmd(m.db),
		fetchAllCmd(m.fetcher),
		m.spin.Tick,
	)
}
```

Also handle `fetchStartedMsg` and `fetchDoneMsg` in Update, and route
spinner ticks to the spinner. Inside `fetchAllCmd`, send a
`fetchStartedMsg` too — simplest is to have `Init()` flip `fetching`
directly since the batch runs synchronously, OR dispatch via a sentinel.
Go with the simpler approach: in `Init`, set `fetching=true` via a
`tea.Cmd` that emits `fetchStartedMsg` first:

Actually simpler: set `m.fetching = true` in `New()` and clear it in
`fetchDoneMsg`. The spinner animates while `fetching` is true.

Update `New`:
```go
return Model{
    // ... existing ...
    fetching: true,
}
```

**Step 4.3: Handle fetch messages + refresh keys in Update**

Add to the `Update` switch:

```go
case spinner.TickMsg:
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd

case fetchDoneMsg:
	m.fetching = false
	var failed int
	for _, r := range msg.results {
		if r.Err != nil {
			failed++
		}
	}
	if failed > 0 {
		m.status = fmt.Sprintf("fetched · %d error(s)", failed)
	} else {
		m.status = "fetched"
	}
	// Reload feeds (for unread counters) and the current feed's articles.
	cmds := []tea.Cmd{loadFeedsCmd(m.db)}
	if len(m.feeds) > 0 {
		cmds = append(cmds, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID))
	}
	return m, tea.Batch(cmds...)
```

And in the key handler:

```go
case key.Matches(msg, m.keys.RefreshAll):
	if !m.fetching {
		m.fetching = true
		m.status = "fetching…"
		return m, fetchAllCmd(m.fetcher)
	}
	return m, nil
case key.Matches(msg, m.keys.RefreshOne):
	// Step 4: reuse FetchAll for simplicity. Single-feed fetch
	// gets its own command in Step 5 when Reader lands.
	if !m.fetching && len(m.feeds) > 0 {
		m.fetching = true
		m.status = "fetching…"
		return m, fetchAllCmd(m.fetcher)
	}
	return m, nil
```

**Step 4.4: Include spinner in the status bar**

Update the `status` line in `View`:

```go
statusText := "rdr · " + m.status
if m.fetching {
	statusText = "rdr · " + m.spin.View() + " " + m.status
}
status := statusBar.Width(m.width).Render(statusText)
```

Add `"fmt"` to imports if not already present.

**Step 4.5: Build + smoke test**

```bash
go build ./...
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected:
- App starts, status bar shows a spinner and `fetching…`
- After a few seconds, status changes to `fetched` (or `fetched · N error(s)`)
- The right pane now shows real articles for the current feed
- Feed unread counters reflect the just-fetched articles
- Navigation works: `j`/`k` scroll, `tab` switches focus, `R` re-fetches
- `q` quits

Clean up: `rm -rf dev`.

**Step 4.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go
git commit -m "feat(rdr): auto-fetch on startup with spinner status bar"
```

---

## Task 5: Navigation polish — `g`/`G`, `^d`/`^u`, `h`/`l`, enter/esc

**Goal:** Round out vim-style navigation so the keybinds advertised in
SPEC.md actually work. `l`/`enter` moves focus right (FeedList →
ArticleList). `h`/`esc` moves focus left (ArticleList → FeedList). `g`/`G`
jump to top/bottom. `^d`/`^u` page.

**Files:**
- Modify: `internal/ui/model.go`

**Step 5.1: Extend Update key handling**

Add to the key switch, next to Tab:

```go
case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
	if m.focus == focusFeeds && len(m.articles) > 0 {
		m.focus = focusArticles
	}
	// focusArticles + enter → Reader, lands in Step 5.
	return m, nil

case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
	if m.focus == focusArticles {
		m.focus = focusFeeds
	}
	return m, nil

case key.Matches(msg, m.keys.Top):
	return m.moveTo(0)
case key.Matches(msg, m.keys.Bottom):
	return m.moveToEnd()
case key.Matches(msg, m.keys.PageDown):
	return m.moveByPage(+1)
case key.Matches(msg, m.keys.PageUp):
	return m.moveByPage(-1)
```

Add the helpers:

```go
func (m Model) moveTo(idx int) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		if idx < 0 || idx >= len(m.feeds) {
			return m, nil
		}
		m.selFeed = idx
		return m, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID)
	case focusArticles:
		if idx < 0 || idx >= len(m.articles) {
			return m, nil
		}
		m.selArt = idx
	}
	return m, nil
}

func (m Model) moveToEnd() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusFeeds:
		return m.moveTo(len(m.feeds) - 1)
	case focusArticles:
		return m.moveTo(len(m.articles) - 1)
	}
	return m, nil
}

func (m Model) moveByPage(dir int) (tea.Model, tea.Cmd) {
	step := m.height - 4
	if step < 1 {
		step = 1
	}
	switch m.focus {
	case focusFeeds:
		return m.moveTo(clamp(m.selFeed+dir*step, 0, len(m.feeds)-1))
	case focusArticles:
		return m.moveTo(clamp(m.selArt+dir*step, 0, len(m.articles)-1))
	}
	return m, nil
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
```

**Step 5.2: Build + smoke test**

```bash
go build ./...
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Expected:
- After fetch, `l`/`enter` moves focus from Feeds to Articles pane
- `h`/`esc` moves it back
- `G` jumps to the last item in the focused pane; `g` to the first
- `^d`/`^u` paginate
- `j`/`k` still step one

Clean up: `rm -rf dev`.

**Step 5.3: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go
git commit -m "feat(rdr): vim page/top/bottom navigation + enter/esc focus"
```

---

## Task 6: Resize + error indicator + empty states + final verification

**Goal:** The app should look sane on small terminals, show an error line
at the bottom when something fails, and not crash when the DB has zero
feeds.

**Files:**
- Modify: `internal/ui/model.go`

**Step 6.1: Error line in the status bar**

Change the error handling — instead of replacing the whole View with an
error message, show errors in the status bar so the layout is preserved:

Delete this from View:
```go
if m.err != nil {
	return errStyle.Render("error: "+m.err.Error()) + "\n"
}
```

Then append error display to the status line construction in View:

```go
statusText := "rdr · " + m.status
if m.fetching {
	statusText = "rdr · " + m.spin.View() + " " + m.status
}
if m.err != nil {
	statusText += "  " + errStyle.Render("! "+m.err.Error())
}
status := statusBar.Width(m.width).Render(statusText)
```

Also clear `m.err` when a successful load message arrives, to avoid stale
errors sticking around forever:

```go
case feedsLoadedMsg:
	m.feeds = msg.feeds
	m.err = nil
	// ... existing handling ...

case articlesLoadedMsg:
	// ... existing handling ...
	m.err = nil
```

**Step 6.2: Guard against zero-feed state**

The current code already returns `nil` from `moveUp`/`moveDown` when the
slice is empty, but `View` passes an empty slice through `renderFeedList`
which handles `len(feeds) == 0` already. Verify by running with a fresh
empty home:

```bash
go build ./...
rm -rf /tmp/rdr-empty
RDR_HOME=/tmp/rdr-empty go run .
```

Expected: both panes draw, FeedList shows `(no feeds)`, status bar shows
the fetch errors (since there's nothing to fetch — should still say
`fetched`, possibly `fetched · 0 error(s)`), and `q` still quits.

Clean up: `rm -rf /tmp/rdr-empty`.

**Step 6.3: Tiny terminal guard**

In `View`, if the terminal is very small (e.g. `width < 40` or
`height < 10`), draw a fallback line:

```go
if m.width < 40 || m.height < 10 {
	return "rdr: terminal too small"
}
```

This slots right after the `m.width == 0` guard.

**Step 6.4: Build + full smoke test**

```bash
go build ./...
rm -rf dev && mkdir dev && cp config.yaml dev/config.yaml
RDR_HOME=./dev go run .
```

Run through the full checklist manually:
- [ ] Spinner animates on startup
- [ ] After fetch, articles appear
- [ ] Unread counters show real numbers
- [ ] `j`/`k` step, `g`/`G` jump, `^d`/`^u` paginate
- [ ] `tab`, `l`/`enter` → Articles; `h`/`esc` → Feeds
- [ ] `R` refetches all
- [ ] Resize: shrink/grow the terminal; panes re-flow (resize triggers `tea.WindowSizeMsg`)
- [ ] Very small terminal shows `rdr: terminal too small`
- [ ] `q` quits cleanly

Clean up: `rm -rf dev`.

**Step 6.5: Run all module tests**

```bash
go test ./... -race
```

Expected: all 31 tests from Step 3 still pass, nothing broken.

**Step 6.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go
git commit -m "feat(rdr): handle resize, errors, and empty states in ui model"
```

---

## What's next

Step 4 delivers a working split-pane TUI: browse feeds, see articles,
fetch from the network, navigate with vim keys. Step 5 adds the Reader
overlay — press `enter` on an article, see its content in a scrollable
viewport, mark it read. Step 6 rounds out polish: `?` help, `o` open in
browser, better empty/error states.

The `--fetch` CLI flag is gone forever at this point. `go run .` is the
only entry point.
