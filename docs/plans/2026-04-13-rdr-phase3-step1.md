# rdr Phase 3 — Step 1: Settings TUI (add / delete / rename feeds)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Press `s` to open a full-screen Settings view. List all feeds,
add a new one (name + URL), delete the selected one, rename it — all in
the TUI. config.yaml becomes import-only.

**Architecture:**
- New `internal/ui/settings.go` — render helper for the full-screen
  settings view + small state machine for sub-modes.
- New `focusSettings` focus value + `settingsMode` int enum with values
  `smList`, `smAddName`, `smAddURL`, `smRename`.
- `bubbles/textinput` for all three text prompts. One shared
  `textinput.Model` field on the Model; its value is repurposed by
  mode.
- New `db.RenameFeed(id int64, name string)` method (the lightest
  addition on the DB side; delete/upsert already exist from Step 1).

**Tech Stack:** existing deps + `github.com/charmbracelet/bubbles/textinput`.

**Working directory:** `/Users/sasha/Code/github.com/iRootPro/rdr`

---

## Task 1: Settings skeleton — open / close / list

**Files:**
- Create: `internal/ui/settings.go`
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/keys.go`

**Step 1.1: Add `Settings` key binding**

Extend `keyMap`:

```go
Settings key.Binding
```

In `defaultKeys()`:
```go
Settings: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
```

Add to `FullHelp`'s last row:
```go
{k.Help, k.Settings, k.Quit},
```

**Step 1.2: Add `focusSettings` + `settingsMode`**

In `model.go`:

```go
const (
	focusFeeds focus = iota
	focusArticles
	focusReader
	focusSettings
)

type settingsMode int

const (
	smList settingsMode = iota
	smAddName
	smAddURL
	smRename
)
```

Add fields to `Model`:
```go
settingsMode  settingsMode
settingsSel   int
settingsInput textinput.Model
pendingName   string // used to carry name between smAddName -> smAddURL
```

Import `github.com/charmbracelet/bubbles/textinput`.

Initialize in `New`:
```go
ti := textinput.New()
ti.Placeholder = ""
ti.CharLimit = 200
ti.Prompt = "› "
// ... return Model{ ... settingsInput: ti, ... }
```

**Step 1.3: Handle `s` to enter Settings**

In the key switch (before Quit handler):

```go
case key.Matches(msg, m.keys.Settings):
	if m.focus != focusSettings {
		m.focus = focusSettings
		m.settingsMode = smList
		m.settingsSel = 0
		return m, nil
	}
	// s closes settings too
	m.focus = focusFeeds
	return m, nil
```

Handle `esc`/`h` (Back) to close Settings from any sub-mode:
Update the Left/Back case:
```go
case focusSettings:
	if m.settingsMode != smList {
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, nil
	}
	m.focus = focusFeeds
	return m, nil
```

Forward `j`/`k` to settings list when in `smList`:
```go
case key.Matches(msg, m.keys.Down):
	if m.focus == focusSettings && m.settingsMode == smList {
		if m.settingsSel < len(m.feeds)-1 {
			m.settingsSel++
		}
		return m, nil
	}
	// ... existing ...
```

Same for `Up`, `Top`, `Bottom`, `PageDown`, `PageUp`.

**Step 1.4: Render settings view**

Create `internal/ui/settings.go`:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

var (
	settingsTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Padding(0, 0, 1, 0)

	settingsKeyHint = lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true)
)

func renderSettings(feeds []db.Feed, selected int, mode settingsMode, input string, width, height int) string {
	var b strings.Builder
	b.WriteString(settingsTitle.Render("Settings · Feeds"))
	b.WriteString("\n\n")

	switch mode {
	case smAddName:
		b.WriteString("New feed name:\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to continue · esc to cancel"))
	case smAddURL:
		b.WriteString("New feed URL:\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to save · esc to cancel"))
	case smRename:
		b.WriteString("Rename feed:\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to save · esc to cancel"))
	default: // smList
		if len(feeds) == 0 {
			b.WriteString(readStyle.Render("(no feeds) — press a to add"))
		} else {
			for i, f := range feeds {
				prefix := "  "
				style := lipgloss.NewStyle().Foreground(colorText)
				if i == selected {
					prefix = "› "
					style = itemSelected
				}
				line := fmt.Sprintf("%s%s  %s",
					prefix,
					style.Render(f.Name),
					readStyle.Render(f.URL),
				)
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(settingsKeyHint.Render("a add · d delete · e rename · esc close"))
	}

	return paneActive.Width(width - 2).Height(height - 2).Render(b.String())
}
```

**Step 1.5: Wire into View**

In `model.View`, before the reader branch, add:

```go
if m.focus == focusSettings {
	helpView := m.helpView()
	body := renderSettings(
		m.feeds,
		m.settingsSel,
		m.settingsMode,
		m.settingsInput.View(),
		m.width,
		m.height-lipgloss.Height(helpView)-1,
	)
	statusText := "rdr · settings"
	if m.err != nil {
		statusText += "  " + errStyle.Render("! "+m.err.Error())
	}
	status := statusBar.Width(m.width).Render(statusText)
	return lipgloss.JoinVertical(lipgloss.Top, body, status, helpView)
}
```

**Step 1.6: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Checklist:
- [ ] `s` opens Settings, shows all feeds with names and URLs
- [ ] `j`/`k` navigate the selection
- [ ] `esc` and `s` close
- [ ] Hint line at bottom: `a add · d delete · e rename · esc close`

**Step 1.7: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/settings.go rdr/internal/ui/model.go rdr/internal/ui/keys.go
git commit -m "feat(rdr): add settings overlay with feed list"
```

---

## Task 2: Add feed flow — `a` key + two-step textinput

**Files:**
- Modify: `internal/ui/model.go`

**Step 2.1: Handle `a` to start add**

In the key switch, add inside the settings-focus branch (or as a
top-level key when `m.focus == focusSettings && m.settingsMode == smList`):

The cleanest approach: special-case settings keys at the very top of
the `tea.KeyMsg` handler when `focus == focusSettings`:

```go
if m.focus == focusSettings {
	return m.updateSettings(msg)
}
```

Add the helper:

```go
func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Text input modes consume most keys.
	switch m.settingsMode {
	case smAddName, smAddURL, smRename:
		switch {
		case key.Matches(msg, m.keys.Back):
			m.settingsMode = smList
			m.settingsInput.Blur()
			m.settingsInput.SetValue("")
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			return m.settingsSubmit()
		}
		var cmd tea.Cmd
		m.settingsInput, cmd = m.settingsInput.Update(msg)
		return m, cmd
	}

	// smList mode
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Settings), key.Matches(msg, m.keys.Back):
		m.focus = focusFeeds
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.settingsSel < len(m.feeds)-1 {
			m.settingsSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.settingsSel > 0 {
			m.settingsSel--
		}
		return m, nil
	case msg.String() == "a":
		m.settingsMode = smAddName
		m.settingsInput.SetValue("")
		m.settingsInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}
```

**Step 2.2: Implement `settingsSubmit`**

```go
func (m Model) settingsSubmit() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.settingsInput.Value())
	if value == "" {
		return m, nil
	}
	switch m.settingsMode {
	case smAddName:
		m.pendingName = value
		m.settingsMode = smAddURL
		m.settingsInput.SetValue("")
		return m, textinput.Blink
	case smAddURL:
		if _, err := m.db.UpsertFeed(m.pendingName, value); err != nil {
			m.err = err
			return m, nil
		}
		m.pendingName = ""
		m.settingsMode = smList
		m.settingsInput.Blur()
		m.settingsInput.SetValue("")
		return m, loadFeedsCmd(m.db)
	}
	return m, nil
}
```

Add `"strings"` to imports if missing.

**Step 2.3: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Checklist:
- [ ] `s` → Settings. Press `a`
- [ ] Prompt: "New feed name:" — type `Test Feed`, press `enter`
- [ ] Prompt: "New feed URL:" — type `https://hnrss.org/newest`, press `enter`
- [ ] New feed appears in the list + in the main Feed List on the left
- [ ] `esc` during name or URL prompt cancels the flow

**Step 2.4: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go
git commit -m "feat(rdr): add feed via settings TUI"
```

---

## Task 3: Delete + rename

**Files:**
- Modify: `internal/db/feeds.go` (new `RenameFeed` method)
- Modify: `internal/db/feeds_test.go` (TDD)
- Modify: `internal/ui/model.go` (`d` + `e` key handling, rename submit)

**Step 3.1: TDD `db.RenameFeed`**

Append to `feeds_test.go`:

```go
func TestRenameFeed_ChangesNameKeepsURL(t *testing.T) {
	d := openTestDB(t)
	f, err := d.UpsertFeed("Old", "https://a.example/rss")
	if err != nil {
		t.Fatalf("UpsertFeed: %v", err)
	}
	if err := d.RenameFeed(f.ID, "New"); err != nil {
		t.Fatalf("RenameFeed: %v", err)
	}
	feeds, _ := d.ListFeeds()
	if len(feeds) != 1 {
		t.Fatalf("want 1 feed, got %d", len(feeds))
	}
	if feeds[0].Name != "New" {
		t.Fatalf("name: got %q, want %q", feeds[0].Name, "New")
	}
	if feeds[0].URL != "https://a.example/rss" {
		t.Fatalf("url changed: %q", feeds[0].URL)
	}
}
```

Run: `go test ./internal/db/... -run TestRenameFeed -v` — expect compile error.

Implement in `feeds.go`:

```go
func (d *DB) RenameFeed(id int64, name string) error {
	_, err := d.sql.Exec(`UPDATE feeds SET name = ? WHERE id = ?`, name, id)
	return err
}
```

Run: `go test ./internal/db/... -v` — expect all pass.

Commit:
```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/db/feeds.go rdr/internal/db/feeds_test.go
git commit -m "feat(rdr): add db.RenameFeed for settings TUI"
```

**Step 3.2: `d` and `e` handlers in `updateSettings`**

Add cases inside the `smList` branch:

```go
case msg.String() == "d":
	if len(m.feeds) == 0 {
		return m, nil
	}
	id := m.feeds[m.settingsSel].ID
	if err := m.db.DeleteFeed(id); err != nil {
		m.err = err
		return m, nil
	}
	if m.settingsSel > 0 {
		m.settingsSel--
	}
	return m, loadFeedsCmd(m.db)
case msg.String() == "e":
	if len(m.feeds) == 0 {
		return m, nil
	}
	m.settingsMode = smRename
	m.settingsInput.SetValue(m.feeds[m.settingsSel].Name)
	m.settingsInput.Focus()
	return m, textinput.Blink
```

**Step 3.3: Rename submit**

Extend `settingsSubmit`:

```go
case smRename:
	if len(m.feeds) == 0 {
		return m, nil
	}
	id := m.feeds[m.settingsSel].ID
	if err := m.db.RenameFeed(id, value); err != nil {
		m.err = err
		return m, nil
	}
	m.settingsMode = smList
	m.settingsInput.Blur()
	m.settingsInput.SetValue("")
	return m, loadFeedsCmd(m.db)
```

**Step 3.4: Build + smoke test**

```bash
go build ./...
RDR_HOME=./dev go run .
```

Checklist:
- [ ] `s` → Settings. Navigate to a feed
- [ ] `d` deletes it. Selection moves one up if at the end. The main Feed List on the left updates after closing Settings
- [ ] `e` prompts for a new name (pre-filled with current name), `enter` saves
- [ ] Deleted feeds don't come back after fetch (they would if config.yaml re-adds them — that's expected; user can edit yaml or we revisit)
- [ ] `esc` mid-rename cancels

**Step 3.5: Final verification**

```bash
go test ./... -race
go build ./...
```

Expected: all tests still pass (33 from Phase 2 Step 1 + 1 RenameFeed = 34).

**Step 3.6: Commit**

```bash
cd /Users/sasha/Code/github.com/iRootPro
git add -f rdr/internal/ui/model.go
git commit -m "feat(rdr): delete and rename feeds from settings TUI"
```

---

## What's next

Settings Step 1 delivers feed management. Future steps can add general
settings editing (refresh interval, max articles per feed, theme
toggle), OPML import/export, and a confirm dialog for destructive
operations. The config.yaml stays as import-only for first-run seeding.
