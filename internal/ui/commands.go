package ui

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/rlog"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/i18n"
)

// osc52Copy writes an OSC 52 escape sequence to stderr so the terminal
// copies `text` into the system clipboard. Works over SSH and in tmux
// (when set-clipboard is on). Stderr is used instead of stdout so the
// write doesn't race bubbletea's altscreen redraws.
func osc52Copy(text string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// Use BEL (\x07) terminator — widest compatibility across terminals.
	fmt.Fprintf(os.Stderr, "\x1b]52;c;%s\x07", encoded)
}

const historyFileName = "history"
const collapsedCatsFileName = "collapsed_categories"
const maxHistory = 50

// readCollapsedCats loads the persisted set of collapsed category names.
// Missing file → empty map. Empty string is allowed as a valid key for
// the "Other" bucket.
func readCollapsedCats(home string) map[string]bool {
	out := make(map[string]bool)
	if home == "" {
		return out
	}
	f, err := os.Open(filepath.Join(home, collapsedCatsFileName))
	if err != nil {
		return out
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		name := strings.TrimRight(scanner.Text(), "\r\n")
		out[name] = true
	}
	return out
}

// writeCollapsedCats serialises the set, one category per line. Called
// after every toggle. Errors are swallowed (QoL feature).
func writeCollapsedCats(home string, cats map[string]bool) {
	if home == "" {
		return
	}
	var names []string
	for k, v := range cats {
		if v {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	data := strings.Join(names, "\n")
	if data != "" {
		data += "\n"
	}
	_ = os.WriteFile(filepath.Join(home, collapsedCatsFileName), []byte(data), 0o644)
}

// readHistoryFile loads a previously saved command history. The file format
// is one command per line, oldest first (append-only feel like ~/.bash_history).
// Missing file yields a nil slice.
func readHistoryFile(home string) []string {
	if home == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(home, historyFileName))
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	// File is oldest-first; Model stores newest-first.
	out := make([]string, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		out = append(out, lines[i])
	}
	if len(out) > maxHistory {
		out = out[:maxHistory]
	}
	return out
}

// writeHistoryFile serializes history (newest-first in memory) to disk
// in chronological (oldest-first) order. Failure is silent — history is
// a QoL feature and we never want to block command execution on it.
func writeHistoryFile(home string, history []string) {
	if home == "" {
		return
	}
	var b strings.Builder
	for i := len(history) - 1; i >= 0; i-- {
		b.WriteString(history[i])
		b.WriteByte('\n')
	}
	_ = os.WriteFile(filepath.Join(home, historyFileName), []byte(b.String()), 0o644)
}

type commandSuggestion struct {
	Complete string
	Help     string
}

func buildCommandCompletions(tr *i18n.Strings) []commandSuggestion {
	c := tr.Command
	return []commandSuggestion{
		{"sync", c.HelpSync},
		{"refresh", c.HelpRefresh},
		{"sort date", c.HelpSortDate},
		{"sort title", c.HelpSortTitle},
		{"sortreverse", c.HelpSortReverse},
		{"filter all", c.HelpFilterAll},
		{"filter unread", c.HelpFilterUnread},
		{"filter starred", c.HelpFilterStarred},
		{"star", c.HelpStar},
		{"read", c.HelpRead},
		{"unread", c.HelpUnread},
		{"unstar", c.HelpUnstar},
		{"copy url", c.HelpCopyURL},
		{"copy md", c.HelpCopyMD},
		{"import", c.HelpImport},
		{"export", c.HelpExport},
		{"images", c.HelpImages},
		{"retention", c.HelpRetention},
		{"collapseall", c.HelpCollapseAll},
		{"expandall", c.HelpExpandAll},
		{"zen", c.HelpZen},
		{"help", c.HelpHelp},
		{"discover", c.HelpDiscover},
		{"bookmark", c.HelpBookmark},
		{"unbookmark", c.HelpUnbookmark},
		{"log", "Application log"},
		{"settings", c.HelpSettings},
		{"search", c.HelpSearch},
		{"quit", c.HelpQuit},
		{"q", c.HelpQuitAlias},
	}
}

func commandSuggestionsFor(input string, tr *i18n.Strings) []commandSuggestion {
	all := buildCommandCompletions(tr)
	input = strings.TrimLeft(input, " ")
	if input == "" {
		return all
	}
	var out []commandSuggestion
	for _, c := range all {
		if strings.HasPrefix(c.Complete, input) {
			out = append(out, c)
		}
	}
	return out
}

const maxCommandPopupRows = 8

func renderCommandPopup(m Model, width int) string {
	sugg := commandSuggestionsFor(m.commandInput.Value(), m.tr)
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	if len(sugg) == 0 {
		return paneInactive.Width(innerW).Render(searchHint.Render(m.tr.Command.NoMatching))
	}

	truncated := sugg
	overflow := 0
	if len(sugg) > maxCommandPopupRows {
		truncated = sugg[:maxCommandPopupRows]
		overflow = len(sugg) - maxCommandPopupRows
	}

	var b strings.Builder
	textStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
	helpStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
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
		b.WriteString(searchHint.Render(fmt.Sprintf(m.tr.Command.MoreFmt, overflow)))
	}
	c := fillBackground(b.String(), innerW-2)
	return paneInactive.Width(innerW).Render(c)
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
		sugg := commandSuggestionsFor(m.commandInput.Value(), m.tr)
		limit := len(sugg)
		if limit > maxCommandPopupRows {
			limit = maxCommandPopupRows
		}
		if m.commandSuggIdx < limit-1 {
			m.commandSuggIdx++
		}
		return m, nil
	case msg.String() == "tab":
		sugg := commandSuggestionsFor(m.commandInput.Value(), m.tr)
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

	tr := m.tr
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}

	switch cmd {
	case "q", "quit":
		return m, tea.Quit

	case "sync", "refresh":
		if m.fetching {
			return m, nil
		}
		m.fetching = true
		m.status = tr.Status.Fetching
		return m, tea.Batch(fetchAllCmd(m.fetcher), m.spin.Tick)

	case "sort":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.SortNeedsArg)
			return m, nil
		}
		switch args[0] {
		case "date", "title":
			m.sortField = args[0]
			applySort(m.articles, m.sortField, m.sortReverse)
			if m.db != nil {
				_ = m.db.SetSortField(m.sortField)
			}
			m.status = fmt.Sprintf(tr.Status.SortFmt, m.sortField)
			return m, nil
		default:
			m.err = fmt.Errorf(tr.Errors.UnknownSortFmt, args[0])
			return m, nil
		}

	case "sortreverse":
		m.sortReverse = !m.sortReverse
		applySort(m.articles, m.sortField, m.sortReverse)
		if m.db != nil {
			_ = m.db.SetSortReverse(m.sortReverse)
		}
		m.status = tr.Status.SortReversed
		return m, nil

	case "filter":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.FilterNeedsArg)
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
			m.err = fmt.Errorf(tr.Errors.UnknownFilterFmt, args[0])
			return m, nil
		}
		m.selArt = 0
		if len(m.feeds) > 0 {
			return m, m.loadCurrentCmd()
		}
		return m, nil

	case "read":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.ReadNeedsQuery)
			return m, nil
		}
		tick := m.startBusy(tr.Status.MarkingRead)
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "read"), tick)

	case "unread":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.UnreadNeedsQuery)
			return m, nil
		}
		tick := m.startBusy(tr.Status.MarkingUnread)
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "unread"), tick)

	case "star":
		if len(args) == 0 {
			return m.toggleStarOnCurrent()
		}
		tick := m.startBusy(tr.Status.Starring)
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "star"), tick)

	case "unstar":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.UnstarNeedsQuery)
			return m, nil
		}
		tick := m.startBusy(tr.Status.Unstarring)
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "unstar"), tick)

	case "bookmark":
		if len(args) == 0 {
			return m.toggleBookmarkOnCurrent()
		}
		tick := m.startBusy("bookmark…")
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "bookmark"), tick)

	case "unbookmark":
		if len(args) == 0 {
			return m, nil
		}
		tick := m.startBusy("unbookmark…")
		return m, tea.Batch(batchApplyCmd(m.db, strings.Join(args, " "), "unbookmark"), tick)

	case "copy":
		if len(args) < 2 {
			m.err = fmt.Errorf("%s", tr.Errors.CopyNeedsArg)
			return m, nil
		}
		format := args[0]
		if format != "url" && format != "md" {
			m.err = fmt.Errorf(tr.Errors.UnknownCopyFmt, format)
			return m, nil
		}
		query := strings.Join(args[1:], " ")
		tick := m.startBusy(tr.Status.Copying)
		return m, tea.Batch(batchCopyCmd(m.db, format, query, tr), tick)

	case "import":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.ImportNeedsPath)
			return m, nil
		}
		path := expandPath(strings.Join(args, " "))
		n, err := importOPMLFile(m.db, path)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.status = fmt.Sprintf(tr.Status.ImportedFmt, n)
		return m, tea.Batch(loadFeedsCmd(m.db), fetchAllCmd(m.fetcher), m.spin.Tick)

	case "export":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.ExportNeedsPath)
			return m, nil
		}
		path := expandPath(strings.Join(args, " "))
		n, err := exportOPMLFile(m.db, path)
		if err != nil {
			m.err = err
			return m, nil
		}
		m.status = fmt.Sprintf(tr.Status.ExportedFmt, n, path)
		return m, nil

	case "zen":
		m.zenMode = !m.zenMode
		return m, nil

	case "collapseall":
		if m.collapsedCats == nil {
			m.collapsedCats = map[string]bool{}
		}
		// Collect category keys from current feeds and mark them all.
		for _, f := range m.feeds {
			m.collapsedCats[f.Category] = true
		}
		writeCollapsedCats(m.home, m.collapsedCats)
		m.status = tr.Status.CategoriesClosed
		return m, nil

	case "expandall":
		m.collapsedCats = map[string]bool{}
		writeCollapsedCats(m.home, m.collapsedCats)
		m.status = tr.Status.CategoriesOpened
		return m, nil

	case "images":
		m.showImages = !m.showImages
		if m.db != nil {
			_ = m.db.SetShowImages(m.showImages)
		}
		if m.showImages {
			m.status = tr.Status.ImagesOn
		} else {
			m.status = tr.Status.ImagesOff
		}
		// If reading an article right now, re-render in place.
		if m.focus == focusReader && m.readerArt != nil {
			feedName := readerFeedName(m.feeds, m.readerArt.FeedID)
			feedURL := readerFeedURL(m.feeds, m.readerArt.FeedID)
			m.reader.SetContent(buildReaderContent(*m.readerArt, feedName, feedURL, m.reader.Width-4, m.showImages, m.tr, m.readerImgURLs, m.readerImgPlacements))
			// Toggling on: kick off inline-image fetch. Toggling off:
			// drop any placements so they don't render on subsequent
			// frames.
			if m.showImages {
				if cmd := m.maybePrepareImagesCmd(); cmd != nil {
					return m, cmd
				}
			} else {
				deletePlacements(m.readerImgPlacements)
				m.readerImgURLs = nil
				m.readerImgPlacements = nil
			}
		}
		return m, nil

	case "retention":
		if len(args) == 0 {
			m.err = fmt.Errorf("%s", tr.Errors.RetentionNeedsArg)
			return m, nil
		}
		arg := strings.ToLower(args[0])
		var days int
		switch arg {
		case "off", "unlimited", "infinite", "0":
			days = 0
		default:
			n, err := strconv.Atoi(arg)
			if err != nil || n < 0 {
				m.err = fmt.Errorf(tr.Errors.RetentionInvalidFmt, args[0])
				return m, nil
			}
			days = n
		}
		if m.db != nil {
			if err := m.db.SetReadRetentionDays(days); err != nil {
				m.err = err
				return m, nil
			}
		}
		if days == 0 {
			m.status = tr.Status.RetentionUnlimited
		} else {
			m.status = fmt.Sprintf(tr.Status.RetentionSetFmt, days)
		}
		return m, nil

	case "help":
		m.helpPrev = m.focus
		m.focus = focusHelp
		return m, nil
	case "discover", "catalog":
		m.focus = focusCatalog
		m.catalogSel = 0
		return m, nil
	case "log":
		logPath := rlog.LogPath()
		data, err := os.ReadFile(logPath)
		if err != nil {
			m.logContent = "No log file: " + logPath
		} else {
			m.logContent = string(data)
		}
		m.focus = focusLog
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

	m.err = fmt.Errorf(tr.Errors.UnknownCommandFmt, cmd)
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
		case "bookmark":
			_, err = d.BulkSetBookmarked(ids, true)
		case "unbookmark":
			_, err = d.BulkSetBookmarked(ids, false)
		default:
			return errMsg{fmt.Errorf("batch: unknown action %q", action)}
		}
		if err != nil {
			return errMsg{err}
		}
		return batchAppliedMsg{action: action, count: len(ids)}
	}
}

// batchCopyCmd evaluates a query, collects matching articles, formats
// them, and writes to the system clipboard via OSC 52.
func batchCopyCmd(d *db.DB, format, query string, tr *i18n.Strings) tea.Cmd {
	return func() tea.Msg {
		atoms, err := ParseQuery(query)
		if err != nil {
			return errMsg{fmt.Errorf("copy query: %w", err)}
		}
		all, err := d.ListAllArticles(5000)
		if err != nil {
			return errMsg{err}
		}
		var lines []string
		for _, a := range all {
			it := db.SearchItem{
				Title:       a.Title,
				FeedName:    a.FeedName,
				Description: a.Description,
				PublishedAt: a.PublishedAt,
				ReadAt:      a.ReadAt,
				StarredAt:   a.StarredAt,
			}
			if !EvalQuery(atoms, it) {
				continue
			}
			if a.URL == "" {
				continue
			}
			switch format {
			case "url":
				lines = append(lines, a.URL)
			case "md":
				title := a.Title
				if title == "" {
					title = a.URL
				}
				lines = append(lines, fmt.Sprintf("- [%s](%s)", title, a.URL))
			}
		}
		if len(lines) == 0 {
			return errMsg{fmt.Errorf("%s", tr.Errors.NoMatches)}
		}
		osc52Copy(strings.Join(lines, "\n"))
		return copiedMsg{count: len(lines), format: format}
	}
}

// pushHistory prepends a command to history, de-duping against the last
// entry and capping size. Called only on successful submission. Persists
// to disk best-effort (errors are swallowed).
func (m *Model) pushHistory(line string) {
	if len(m.commandHistory) > 0 && m.commandHistory[0] == line {
		return
	}
	m.commandHistory = append([]string{line}, m.commandHistory...)
	if len(m.commandHistory) > maxHistory {
		m.commandHistory = m.commandHistory[:maxHistory]
	}
	writeHistoryFile(m.home, m.commandHistory)
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
		if _, err := d.UpsertFeed(e.Name, e.URL, "", e.Username, e.Password); err != nil {
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
		entries = append(entries, feed.OPMLEntry{Name: f.Name, URL: f.URL, Username: f.Username, Password: f.Password})
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

