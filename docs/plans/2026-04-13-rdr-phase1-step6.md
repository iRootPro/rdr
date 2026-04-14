# rdr Phase 1 — Step 6: Polish

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Round out the Phase 1 MVP — `?` help overlay, `o` open the
current article URL in the browser, and a per-feed fetch error indicator
so the user can tell which feed broke at a glance.

**Architecture:** Use `bubbles/help` with a KeyMap that implements
`help.KeyMap`. Browser open via `os/exec` with platform detection
(`open` / `xdg-open` / `start`). Fetch errors tracked in a
`map[int64]error` on the Model, populated from `fetchDoneMsg`, rendered
as a red `●` prefix in FeedList.

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Testing approach:** Manual visual verification.

---

## Task 1: `?` help overlay + `o` open URL

**Files:**
- Modify: `internal/ui/keys.go` (add Help + OpenURL bindings, add Short/Full help methods)
- Modify: `internal/ui/model.go` (help state, `o` handler)

**Step 1.1: Extend keyMap**

Add `Help` and `OpenURL` bindings with descriptive help text:

```go
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
	OpenURL    key.Binding
	Help       key.Binding
}
```

`defaultKeys()` adds both and sets `WithHelp`:

```go
Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
Left:       key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "back")),
Right:      key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "forward")),
Tab:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("^u", "page up")),
PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("^d", "page down")),
Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
RefreshOne: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh current")),
RefreshAll: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh all")),
OpenURL:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
```

Implement the `help.KeyMap` interface:

```go
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Back, k.RefreshAll, k.OpenURL, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown},
		{k.Left, k.Right, k.Tab, k.Enter, k.Back},
		{k.RefreshOne, k.RefreshAll, k.OpenURL},
		{k.Help, k.Quit},
	}
}
```

**Step 1.2: Add help model + help toggle**

In `model.go`, import `github.com/charmbracelet/bubbles/help` and add
a `help.Model` field plus a `showHelp` bool:

```go
type Model struct {
	// ... existing ...
	help     help.Model
	showHelp bool
}
```

Initialize in `New`:

```go
h := help.New()
h.Styles.ShortKey = lipgloss.NewStyle().Foreground(colorAccent)
h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(colorMuted)
h.Styles.FullKey = lipgloss.NewStyle().Foreground(colorAccent)
h.Styles.FullDesc = lipgloss.NewStyle().Foreground(colorMuted)
// ... return Model{ ... help: h, ... }
```

Handle the `?` toggle in Update:

```go
case key.Matches(msg, m.keys.Help):
	m.showHelp = !m.showHelp
	m.help.ShowAll = m.showHelp
	return m, nil
```

**Step 1.3: Render help in status bar**

Replace the status line construction in `View` to include help below
the normal status line when `showHelp` is off (short one-line form) or
replace the status line with a full help block when `showHelp` is on:

```go
var helpView string
if m.showHelp {
	helpView = m.help.View(m.keys)
} else {
	helpView = m.help.ShortHelpView(m.keys.ShortHelp())
}
```

Append `helpView` to the status line:

```go
statusLine := statusBar.Width(m.width).Render(statusText)
return lipgloss.JoinVertical(lipgloss.Top, row, statusLine, helpView)
```

Adjust `paneH` to account for the help line(s). Simpler: reserve 2 rows
for status + short help (3-4 for full help). Or just keep `paneH =
m.height - 2` and let terminal wrap — for Phase 1 polish that's
acceptable.

Apply the same `helpView` to the reader-focus branch of View.

**Step 1.4: Add `o` handler**

Create `internal/ui/browser.go`:

```go
package ui

import (
	"os/exec"
	"runtime"
)

func openInBrowser(url string) error {
	if url == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
```

In `Update`, add a branch before the Quit case:

```go
case key.Matches(msg, m.keys.OpenURL):
	var url string
	switch m.focus {
	case focusArticles:
		if len(m.articles) > 0 {
			url = m.articles[m.selArt].URL
		}
	case focusReader:
		if m.readerArt != nil {
			url = m.readerArt.URL
		}
	}
	if url != "" {
		if err := openInBrowser(url); err != nil {
			m.err = err
		}
	}
	return m, nil
```

**Step 1.5: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Checklist:
- [ ] Bottom of screen shows a short help line with key hints
- [ ] `?` expands to a multi-line help overlay; `?` again collapses
- [ ] Navigate to an article, press `o` — your browser opens the article URL
- [ ] In the reader, `o` opens the current article URL
- [ ] `q` still quits

**Step 1.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/keys.go rdr/internal/ui/model.go rdr/internal/ui/browser.go
git commit -m "feat(rdr): add help overlay and open-in-browser"
```

---

## Task 2: Per-feed fetch error indicator

**Goal:** If a feed fails to fetch (network error, parse error, HTTP
500), show a red `●` prefix next to its name in FeedList so the user
sees at a glance which feed is broken. Error clears on a successful
re-fetch.

**Files:**
- Modify: `internal/ui/model.go` (feedErrors field)
- Modify: `internal/ui/feedlist.go` (render indicator)

**Step 2.1: Track per-feed errors in Model**

```go
type Model struct {
	// ... existing ...
	feedErrors map[int64]error
}
```

Initialize in `New`:

```go
feedErrors: map[int64]error{},
```

In the `fetchDoneMsg` handler, replace the simple failure counter with
per-feed error tracking:

```go
case fetchDoneMsg:
	m.fetching = false
	m.feedErrors = map[int64]error{}
	var failed int
	for _, r := range msg.results {
		if r.Err != nil {
			m.feedErrors[r.Feed.ID] = r.Err
			failed++
		}
	}
	if failed > 0 {
		m.status = fmt.Sprintf("fetched · %d error(s)", failed)
	} else {
		m.status = "fetched"
	}
	// ... reload cmds as before ...
```

**Step 2.2: Pass errors to FeedList renderer**

Change `renderFeedList` signature:

```go
func renderFeedList(feeds []db.Feed, errors map[int64]error, selected int, active bool, width, height int) string {
```

Before rendering each feed, check `errors[f.ID]` — if non-nil, prepend a
red `●`:

```go
errMark := ""
if _, ok := errors[f.ID]; ok {
	errMark = errStyle.Render("● ")
}
```

Then compose the line as `errMark + name` (inside the selection bracket
logic as needed). Keep the name truncation math adjusted by 2 runes.

Update the caller in `model.View`:

```go
left := renderFeedList(m.feeds, m.feedErrors, m.selFeed, m.focus == focusFeeds, leftW, paneH)
```

**Step 2.3: Clear errors on successful re-fetch**

The `fetchDoneMsg` handler already reassigns `m.feedErrors = map[int64]error{}`
so this is automatic — feeds that pass on re-fetch silently drop their
indicator, only feeds still broken keep theirs.

**Step 2.4: Build + smoke test**

```bash
go build ./...
```

To exercise the error path, edit `dev/config.yaml` to include a garbage
URL:

```yaml
feeds:
  - name: Broken
    url: https://definitely-not-a-feed.invalid/rss
  - name: Hacker News
    url: https://hnrss.org/frontpage
```

Then:

```bash
RDR_HOME=./dev go run .
```

Expected: after fetch completes, the "Broken" feed shows a red `●`
prefix and status bar says `fetched · 1 error(s)`. HN still works.

**Step 2.5: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go rdr/internal/ui/feedlist.go
git commit -m "feat(rdr): show per-feed fetch error indicator"
```

---

## What's next

Phase 1 complete: full MVP with help, browser integration, and error
visibility. Next Phase 2 starts with `go-readability` + `html-to-markdown`
+ `glamour` for proper article rendering and the `f` keybind to load
full articles.
