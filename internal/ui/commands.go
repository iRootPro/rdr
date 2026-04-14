package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
)

type commandSuggestion struct {
	Complete string
	Help     string
}

var commandCompletions = []commandSuggestion{
	{"sync", "Fetch all feeds"},
	{"refresh", "Fetch all feeds (alias for sync)"},
	{"sort date", "Sort articles by publish date"},
	{"sort title", "Sort articles alphabetically"},
	{"sortreverse", "Toggle sort direction"},
	{"filter all", "Show all articles"},
	{"filter unread", "Show only unread articles"},
	{"filter starred", "Show only starred articles"},
	{"star", "Toggle star on current article"},
	{"read", "Mark matching articles read (:read <query>)"},
	{"unread", "Mark matching articles unread (:unread <query>)"},
	{"unstar", "Unstar matching articles (:unstar <query>)"},
	{"import", "Import feeds from OPML file (:import <path>)"},
	{"export", "Export feeds to OPML file (:export <path>)"},
	{"zen", "Toggle zen mode"},
	{"help", "Toggle help overlay"},
	{"settings", "Open feed settings"},
	{"search", "Open search picker"},
	{"quit", "Exit rdr"},
	{"q", "Exit rdr (alias for quit)"},
}

func commandSuggestionsFor(input string) []commandSuggestion {
	input = strings.TrimLeft(input, " ")
	if input == "" {
		out := make([]commandSuggestion, len(commandCompletions))
		copy(out, commandCompletions)
		return out
	}
	var out []commandSuggestion
	for _, c := range commandCompletions {
		if strings.HasPrefix(c.Complete, input) {
			out = append(out, c)
		}
	}
	return out
}

const maxCommandPopupRows = 8

func renderCommandPopup(m Model, width int) string {
	sugg := commandSuggestionsFor(m.commandInput.Value())
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	if len(sugg) == 0 {
		return paneInactive.Width(innerW).Render(searchHint.Render("(no matching commands)"))
	}

	truncated := sugg
	overflow := 0
	if len(sugg) > maxCommandPopupRows {
		truncated = sugg[:maxCommandPopupRows]
		overflow = len(sugg) - maxCommandPopupRows
	}

	var b strings.Builder
	textStyle := lipgloss.NewStyle().Foreground(colorText)
	helpStyle := lipgloss.NewStyle().Foreground(colorMuted)
	for i, s := range truncated {
		prefix := "  "
		style := textStyle
		if i == m.commandSuggIdx {
			prefix = "› "
			style = itemSelected
		}
		line := style.Render(prefix+s.Complete) + "  " + helpStyle.Render(s.Help)
		b.WriteString(line)
		if i < len(truncated)-1 || overflow > 0 {
			b.WriteString("\n")
		}
	}
	if overflow > 0 {
		b.WriteString(searchHint.Render(fmt.Sprintf("  … +%d more", overflow)))
	}
	return paneInactive.Width(innerW).Render(b.String())
}

// updateCommand handles keystrokes while the user is typing in the ':' prompt.
func (m Model) updateCommand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.focus = m.commandPrev
		if m.focus == focusCommand {
			m.focus = focusFeeds
		}
		m.commandInput.Blur()
		m.commandInput.SetValue("")
		m.commandSuggIdx = 0
		m.historyPos = -1
		m.historyStash = ""
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		line := strings.TrimSpace(m.commandInput.Value())
		m.commandInput.Blur()
		m.commandInput.SetValue("")
		m.commandSuggIdx = 0
		m.historyPos = -1
		m.historyStash = ""
		m.focus = m.commandPrev
		if m.focus == focusCommand {
			m.focus = focusFeeds
		}
		if line == "" {
			return m, nil
		}
		m.pushHistory(line)
		return dispatchCommand(m, line)
	case msg.String() == "ctrl+p":
		m.historyPrev()
		return m, nil
	case msg.String() == "ctrl+n":
		m.historyNext()
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.commandSuggIdx > 0 {
			m.commandSuggIdx--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		sugg := commandSuggestionsFor(m.commandInput.Value())
		limit := len(sugg)
		if limit > maxCommandPopupRows {
			limit = maxCommandPopupRows
		}
		if m.commandSuggIdx < limit-1 {
			m.commandSuggIdx++
		}
		return m, nil
	case msg.String() == "tab":
		sugg := commandSuggestionsFor(m.commandInput.Value())
		if len(sugg) > 0 && m.commandSuggIdx >= 0 && m.commandSuggIdx < len(sugg) {
			m.commandInput.SetValue(sugg[m.commandSuggIdx].Complete)
			m.commandInput.CursorEnd()
			m.commandSuggIdx = 0
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	m.commandSuggIdx = 0
	return m, cmd
}

// dispatchCommand parses a command line and mutates the model. The leading ':'
// is NOT part of line. Unknown commands set m.err and return.
func dispatchCommand(m Model, line string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return m, nil
	}
	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "q", "quit":
		return m, tea.Quit

	case "sync", "refresh":
		if m.fetching {
			return m, nil
		}
		m.fetching = true
		m.status = "fetching…"
		return m, tea.Batch(fetchAllCmd(m.fetcher), m.spin.Tick)

	case "sort":
		if len(args) == 0 {
			m.err = fmt.Errorf(":sort needs date|title")
			return m, nil
		}
		switch args[0] {
		case "date", "title":
			m.sortField = args[0]
			applySort(m.articles, m.sortField, m.sortReverse)
			m.status = "sort: " + m.sortField
			return m, nil
		default:
			m.err = fmt.Errorf("unknown sort field %q", args[0])
			return m, nil
		}

	case "sortreverse":
		m.sortReverse = !m.sortReverse
		applySort(m.articles, m.sortField, m.sortReverse)
		m.status = "sort reversed"
		return m, nil

	case "filter":
		if len(args) == 0 {
			m.err = fmt.Errorf(":filter needs all|unread|starred")
			return m, nil
		}
		switch args[0] {
		case "all":
			m.filter = filterAll
		case "unread":
			m.filter = filterUnread
		case "starred":
			m.filter = filterStarred
		default:
			m.err = fmt.Errorf("unknown filter %q", args[0])
			return m, nil
		}
		m.selArt = 0
		if len(m.feeds) > 0 {
			return m, m.loadCurrentCmd()
		}
		return m, nil

	case "read":
		if len(args) == 0 {
			m.err = fmt.Errorf(":read needs a query")
			return m, nil
		}
		return m, batchApplyCmd(m.db, strings.Join(args, " "), "read")

	case "unread":
		if len(args) == 0 {
			m.err = fmt.Errorf(":unread needs a query")
			return m, nil
		}
		return m, batchApplyCmd(m.db, strings.Join(args, " "), "unread")

	case "star":
		if len(args) == 0 {
			return m.toggleStarOnCurrent()
		}
		return m, batchApplyCmd(m.db, strings.Join(args, " "), "star")

	case "unstar":
		if len(args) == 0 {
			m.err = fmt.Errorf(":unstar needs a query")
			return m, nil
		}
		return m, batchApplyCmd(m.db, strings.Join(args, " "), "unstar")

	case "import":
		if len(args) == 0 {
			m.err = fmt.Errorf(":import needs a path")
			return m, nil
		}
		path := expandPath(strings.Join(args, " "))
		n, err := importOPMLFile(m.db, path)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.status = fmt.Sprintf("imported %d feeds", n)
		return m, tea.Batch(loadFeedsCmd(m.db), fetchAllCmd(m.fetcher), m.spin.Tick)

	case "export":
		if len(args) == 0 {
			m.err = fmt.Errorf(":export needs a path")
			return m, nil
		}
		path := expandPath(strings.Join(args, " "))
		n, err := exportOPMLFile(m.db, path)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.status = fmt.Sprintf("exported %d feeds to %s", n, path)
		return m, nil

	case "zen":
		m.zenMode = !m.zenMode
		return m, nil

	case "help":
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return m, nil

	case "settings":
		m.focus = focusSettings
		m.settingsMode = smList
		m.settingsSel = 0
		return m, nil

	case "search":
		m.searchPrev = focusFeeds
		m.focus = focusSearch
		m.searchInput.SetValue("")
		m.searchInput.Focus()
		m.searchSel = 0
		return m, loadSearchCmd(m.db)
	}

	m.err = fmt.Errorf("unknown command %q", cmd)
	return m, nil
}

// batchApplyCmd runs a query against the full article set and applies the
// given action to every match. action ∈ {"read","unread","star","unstar"}.
// The heavy work (load + filter + UPDATE) happens in a goroutine so the
// main loop stays responsive.
func batchApplyCmd(d *db.DB, queryStr, action string) tea.Cmd {
	return func() tea.Msg {
		atoms, err := ParseQuery(queryStr)
		if err != nil {
			return errMsg{fmt.Errorf("batch query: %w", err)}
		}
		all, err := d.ListAllArticles(5000)
		if err != nil {
			return errMsg{err}
		}
		var ids []int64
		for _, a := range all {
			it := db.SearchItem{
				Title:       a.Title,
				FeedName:    a.FeedName,
				Description: a.Description,
				PublishedAt: a.PublishedAt,
				ReadAt:      a.ReadAt,
				StarredAt:   a.StarredAt,
			}
			if EvalQuery(atoms, it) {
				ids = append(ids, a.ID)
			}
		}
		switch action {
		case "read":
			err = d.BulkMarkRead(ids)
		case "unread":
			err = d.BulkMarkUnread(ids)
		case "star":
			err = d.BulkSetStarred(ids, true)
		case "unstar":
			err = d.BulkSetStarred(ids, false)
		default:
			return errMsg{fmt.Errorf("batch: unknown action %q", action)}
		}
		if err != nil {
			return errMsg{err}
		}
		return batchAppliedMsg{action: action, count: len(ids)}
	}
}

// pushHistory prepends a command to history, de-duping against the last
// entry and capping size. Called only on successful submission.
func (m *Model) pushHistory(line string) {
	if len(m.commandHistory) > 0 && m.commandHistory[0] == line {
		return
	}
	const maxHistory = 50
	m.commandHistory = append([]string{line}, m.commandHistory...)
	if len(m.commandHistory) > maxHistory {
		m.commandHistory = m.commandHistory[:maxHistory]
	}
}

// historyPrev walks backwards through history (older commands). Stashes
// the current unsent input on the first step so Ctrl+N can restore it.
func (m *Model) historyPrev() {
	if len(m.commandHistory) == 0 {
		return
	}
	if m.historyPos == -1 {
		m.historyStash = m.commandInput.Value()
	}
	if m.historyPos < len(m.commandHistory)-1 {
		m.historyPos++
	}
	m.commandInput.SetValue(m.commandHistory[m.historyPos])
	m.commandInput.CursorEnd()
	m.commandSuggIdx = 0
}

// historyNext walks forward (newer commands). Past the end of history,
// restores the stashed unsent input.
func (m *Model) historyNext() {
	if m.historyPos == -1 {
		return
	}
	m.historyPos--
	if m.historyPos < 0 {
		m.historyPos = -1
		m.commandInput.SetValue(m.historyStash)
		m.commandInput.CursorEnd()
		m.commandSuggIdx = 0
		return
	}
	m.commandInput.SetValue(m.commandHistory[m.historyPos])
	m.commandInput.CursorEnd()
	m.commandSuggIdx = 0
}

// expandPath resolves a leading ~ to the user's home directory.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~/") || p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

func importOPMLFile(d *db.DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	entries, err := feed.ParseOPML(f)
	if err != nil {
		return 0, err
	}
	var added int
	for _, e := range entries {
		if _, err := d.UpsertFeed(e.Name, e.URL); err != nil {
			return added, fmt.Errorf("upsert %q: %w", e.Name, err)
		}
		added++
	}
	return added, nil
}

func exportOPMLFile(d *db.DB, path string) (int, error) {
	feeds, err := d.ListFeeds()
	if err != nil {
		return 0, err
	}
	entries := make([]feed.OPMLEntry, 0, len(feeds))
	for _, f := range feeds {
		entries = append(entries, feed.OPMLEntry{Name: f.Name, URL: f.URL})
	}
	f, err := os.Create(path)
	if err != nil {
		return 0, fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	if err := feed.WriteOPML(f, "rdr export", entries); err != nil {
		return 0, err
	}
	return len(entries), nil
}

// applySort sorts articles in place using stable sort. field ∈ {"date", "title"}.
// Default direction: date=DESC, title=ASC. reverse flips both.
func applySort(articles []db.Article, field string, reverse bool) {
	sort.SliceStable(articles, func(i, j int) bool {
		switch field {
		case "title":
			if reverse {
				return articles[i].Title > articles[j].Title
			}
			return articles[i].Title < articles[j].Title
		default: // date
			a, b := articles[i].PublishedAt, articles[j].PublishedAt
			if reverse {
				return a.Before(b)
			}
			return a.After(b)
		}
	})
}

