# rdr Phase 1 — Step 5: Reader

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a third pane — the Reader. `enter` on an article opens it
full-screen with a scrollable viewport showing title, source/time/URL
meta line, and the (HTML-stripped) body. `esc`/`h` returns to the split
pane and marks the article as read, refreshing unread counters.

**Architecture:** New file `internal/ui/reader.go` with a
`bubbles/viewport.Model` field added to `Model`, a third `focus` value
`focusReader`, and a helper that builds the reader content string (title
line + meta line + divider + body). HTML → plain text via simple regexp
strip for Phase 1 (Phase 2 replaces with glamour). Mark-read fires on
enter; on exit we reload feeds + articles so counters update.

**Tech Stack:** `github.com/charmbracelet/bubbles/viewport`, existing
`internal/ui/`, `internal/db/`.

**Related docs:** `SPEC.md` (Reader section + key bindings),
`docs/plans/2026-04-13-rdr-phase1-design.md` (UI / Reader).

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

**Testing approach:** Manual visual verification, same as Step 4. No UI
unit tests.

---

## Task 1: Reader viewport + enter/esc integration

**Goal:** Pressing `enter` on a selected article opens a full-screen
Reader with title, meta, and scrollable body. `esc`/`h` returns to the
split pane. No mark-read yet — that lands in Task 2.

**Files:**
- Create: `internal/ui/reader.go`
- Modify: `internal/ui/model.go`

**Step 1.1: Create `reader.go`**

Create `rdr/internal/ui/reader.go`:

```go
package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

var (
	readerTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	readerMeta = lipgloss.NewStyle().
			Foreground(colorMuted)

	readerSource = lipgloss.NewStyle().Foreground(colorGreen)
	readerURL    = lipgloss.NewStyle().Foreground(colorTeal).Underline(true)
	readerBody   = lipgloss.NewStyle().Foreground(colorText)

	readerHint = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

// buildReaderContent renders the article into a width-aware string that
// the viewport can scroll.
func buildReaderContent(a db.Article, feedName string, width int) string {
	var b strings.Builder

	b.WriteString(readerTitle.Render(a.Title))
	b.WriteString("\n")

	metaParts := []string{readerSource.Render(feedName)}
	if ago := timeAgo(a.PublishedAt); ago != "" {
		metaParts = append(metaParts, readerMeta.Render(ago))
	}
	if a.URL != "" {
		metaParts = append(metaParts, readerURL.Render(a.URL))
	}
	b.WriteString(readerMeta.Render(strings.Join(metaParts, " · ")))
	b.WriteString("\n")

	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	body := stripHTML(a.Content)
	if body == "" {
		body = stripHTML(a.Description)
	}
	if body == "" {
		body = "(no content)"
	}
	b.WriteString(readerBody.Render(wrap(body, width)))
	b.WriteString("\n\n")

	b.WriteString(readerHint.Render("[f] load full article (Phase 2)"))
	return b.String()
}

var (
	reTag     = regexp.MustCompile(`<[^>]+>`)
	reEntity  = regexp.MustCompile(`&[a-zA-Z#0-9]+;`)
	reSpaces  = regexp.MustCompile(`[ \t]+`)
	reNewline = regexp.MustCompile(`\n{3,}`)
)

// stripHTML removes tags and collapses whitespace — Phase 1 MVP.
// Phase 2 replaces this with html-to-markdown + glamour.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	// Preserve paragraph breaks.
	s = strings.ReplaceAll(s, "</p>", "\n\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reTag.ReplaceAllString(s, "")
	s = reEntity.ReplaceAllStringFunc(s, decodeEntity)
	s = reSpaces.ReplaceAllString(s, " ")
	s = reNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func decodeEntity(e string) string {
	switch e {
	case "&amp;":
		return "&"
	case "&lt;":
		return "<"
	case "&gt;":
		return ">"
	case "&quot;":
		return `"`
	case "&#39;", "&apos;":
		return "'"
	case "&nbsp;":
		return " "
	case "&mdash;":
		return "—"
	case "&ndash;":
		return "–"
	case "&hellip;":
		return "…"
	}
	return e
}

// wrap breaks a string into lines no longer than width runes, respecting
// existing newlines and word boundaries.
func wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	for i, para := range strings.Split(s, "\n") {
		if i > 0 {
			out.WriteByte('\n')
		}
		if para == "" {
			continue
		}
		words := strings.Fields(para)
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if len(line)+1+len(w) > width {
				out.WriteString(line)
				out.WriteByte('\n')
				line = w
				continue
			}
			line += " " + w
		}
		if line != "" {
			out.WriteString(line)
		}
	}
	return out.String()
}

// readerFeedName returns the feed name for the currently-open article, or
// a blank string if the feed is not in the slice.
func readerFeedName(feeds []db.Feed, feedID int64) string {
	for _, f := range feeds {
		if f.ID == feedID {
			return f.Name
		}
	}
	return ""
}

// unused import guard
var _ = fmt.Sprintf
var _ = time.Now
```

The two `_ =` lines at the bottom exist only if you don't end up using
`fmt` or `time`. If you do, delete them. The file compiles either way —
just keep imports in sync.

**Step 1.2: Extend `Model` with viewport + focusReader**

Edit `rdr/internal/ui/model.go` imports to add:

```go
"github.com/charmbracelet/bubbles/viewport"
```

Extend the `focus` enum:

```go
const (
	focusFeeds focus = iota
	focusArticles
	focusReader
)
```

Extend the `Model` struct with a viewport and a "current open article"
index snapshot:

```go
type Model struct {
	// ... existing ...

	reader    viewport.Model
	readerArt *db.Article // pointer to the article currently in the reader, nil when closed
}
```

Update `New` to initialize an empty viewport:

```go
func New(database *db.DB, fetcher *feed.Fetcher) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorAccent)
	return Model{
		db:       database,
		fetcher:  fetcher,
		keys:     defaultKeys(),
		status:   "fetching…",
		spin:     s,
		fetching: true,
		reader:   viewport.New(0, 0),
	}
}
```

**Step 1.3: Handle `enter` to open reader, `esc`/`h` to close**

In the key switch in `Update`, change the `Right`/`Enter` branch:

```go
case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Enter):
	switch m.focus {
	case focusFeeds:
		if len(m.articles) > 0 {
			m.focus = focusArticles
		}
	case focusArticles:
		if len(m.articles) > 0 {
			return m.openReader()
		}
	}
	return m, nil
```

And the `Left`/`Back` branch:

```go
case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
	switch m.focus {
	case focusArticles:
		m.focus = focusFeeds
	case focusReader:
		m.focus = focusArticles
		m.readerArt = nil
	}
	return m, nil
```

Add the `openReader` helper:

```go
func (m Model) openReader() (tea.Model, tea.Cmd) {
	a := m.articles[m.selArt]
	m.readerArt = &a
	m.focus = focusReader
	m.reader.Width = m.width - 4
	m.reader.Height = m.height - 2
	feedName := readerFeedName(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, m.reader.Width-4))
	m.reader.GotoTop()
	return m, nil
}
```

**Step 1.4: Forward scroll keys to the viewport when focus is reader**

Above the navigation key branches, special-case the reader focus. The
cleanest approach: in the `Down`/`Up`/`Top`/`Bottom`/`PageDown`/`PageUp`
branches, when `m.focus == focusReader`, delegate to the viewport
`Update` rather than the feed/article nav helpers. Example for `Down`:

```go
case key.Matches(msg, m.keys.Down):
	if m.focus == focusReader {
		var cmd tea.Cmd
		m.reader, cmd = m.reader.Update(msg)
		return m, cmd
	}
	return m.moveDown()
```

Apply the same pattern to `Up`, `Top`, `Bottom`, `PageDown`, `PageUp`.

Also handle `tea.WindowSizeMsg` for the reader:

```go
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	m.reader.Width = m.width - 4
	m.reader.Height = m.height - 2
	if m.readerArt != nil {
		feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
		m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, m.reader.Width-4))
	}
	return m, nil
```

**Step 1.5: Render the reader in `View`**

At the top of `View`, after the width/height guards, add:

```go
if m.focus == focusReader && m.readerArt != nil {
	statusText := "rdr · reader"
	if m.err != nil {
		statusText += "  " + errStyle.Render("! "+m.err.Error())
	}
	status := statusBar.Width(m.width).Render(statusText)
	body := paneActive.Width(m.width - 2).Height(m.height - 2).Render(m.reader.View())
	return lipgloss.JoinVertical(lipgloss.Top, body, status)
}
```

This short-circuits the split-pane render when the reader is active.

**Step 1.6: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Expected:
- Navigate to an article (tab or `l`/enter), press `enter` → full-screen reader opens with title, meta, body
- `j`/`k`/`^d`/`^u`/`g`/`G` scroll the viewport
- `esc`/`h` returns to the split pane
- `q` quits from any focus

**Step 1.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/reader.go rdr/internal/ui/model.go
git commit -m "feat(rdr): add scrollable reader with HTML strip"
```

---

## Task 2: Mark read + refresh counters on reader close

**Goal:** Opening an article marks it read in the DB; closing the reader
reloads feeds + articles so the unread counter drops and the article
renders muted in the list.

**Files:**
- Modify: `internal/ui/model.go`

**Step 2.1: Add `markReadCmd`**

Add alongside `loadFeedsCmd`/`loadArticlesCmd`:

```go
func markReadCmd(d *db.DB, articleID int64) tea.Cmd {
	return func() tea.Msg {
		if err := d.MarkRead(articleID); err != nil {
			return errMsg{err}
		}
		return articleMarkedMsg{articleID: articleID}
	}
}
```

Add `articleMarkedMsg` to `messages.go`:

```go
type articleMarkedMsg struct {
	articleID int64
}
```

**Step 2.2: Fire `markReadCmd` when the reader opens**

Update `openReader`:

```go
func (m Model) openReader() (tea.Model, tea.Cmd) {
	a := m.articles[m.selArt]
	m.readerArt = &a
	m.focus = focusReader
	m.reader.Width = m.width - 4
	m.reader.Height = m.height - 2
	feedName := readerFeedName(m.feeds, a.FeedID)
	m.reader.SetContent(buildReaderContent(a, feedName, m.reader.Width-4))
	m.reader.GotoTop()
	if a.ReadAt == nil {
		return m, markReadCmd(m.db, a.ID)
	}
	return m, nil
}
```

**Step 2.3: Reload feeds + articles when the reader closes**

Update the `Left`/`Back` case:

```go
case key.Matches(msg, m.keys.Left), key.Matches(msg, m.keys.Back):
	switch m.focus {
	case focusArticles:
		m.focus = focusFeeds
		return m, nil
	case focusReader:
		m.focus = focusArticles
		m.readerArt = nil
		cmds := []tea.Cmd{loadFeedsCmd(m.db)}
		if len(m.feeds) > 0 {
			cmds = append(cmds, loadArticlesCmd(m.db, m.feeds[m.selFeed].ID))
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
```

**Step 2.4: Handle `articleMarkedMsg`**

Inside `Update`, between the article-loaded and err cases, add a no-op
handler (we reload on reader close, so nothing else to do here — but
catching the message keeps errors visible):

```go
case articleMarkedMsg:
	m.err = nil
	return m, nil
```

**Step 2.5: Preserve article selection index on reload**

When `feedsLoadedMsg`/`articlesLoadedMsg` arrive after close, the current
code resets `selArt = 0`. That's jarring — the user loses their place in
the article list. Fix `articlesLoadedMsg`:

```go
case articlesLoadedMsg:
	m.err = nil
	if len(m.feeds) > 0 && m.feeds[m.selFeed].ID == msg.feedID {
		m.articles = msg.articles
		if m.selArt >= len(m.articles) {
			m.selArt = 0
		}
	}
	return m, nil
```

(Removed the unconditional `m.selArt = 0`.)

**Step 2.6: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Expected walk-through:
1. Fetch completes, articles populate (e.g. HN shows 20 unread)
2. `tab` to Articles, pick an article, press `enter`
3. Reader opens — verify article is visible, body is readable
4. `esc` back → the list shows that article now muted (read style), and
   the feed's unread counter dropped by 1
5. Reopen the same article — reader still works, no extra MarkRead runs
6. `q` quits

**Step 2.7: Final full-module verification**

```bash
go test ./... -race
go build ./...
```

Expected: 31 tests still green, build clean.

**Step 2.8: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go rdr/internal/ui/messages.go
git commit -m "feat(rdr): mark articles read on reader open, refresh on close"
```

---

## What's next

Step 5 delivers the full Phase 1 MVP: browse feeds, fetch from the
network, read articles inline with a scrollable viewport, see unread
counters update live. Phase 1 Step 6 is polish — `?` help overlay, `o`
open in browser, tighter empty/error states, resize edge cases.

Phase 2 (separate plan) swaps the regex HTML strip for a real
html-to-markdown + glamour pipeline and wires up `f` to load the full
article through go-readability.
