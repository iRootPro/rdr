package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/i18n"
)

// CatalogEntry is one feed in the built-in discover catalog.
type CatalogEntry struct {
	Name     string
	URL      string
	Category string
}

// catalog is the built-in feed directory, grouped by category.
var catalog = []CatalogEntry{
	// Tech News
	{Name: "Hacker News", URL: "https://hnrss.org/frontpage", Category: "Tech News"},
	{Name: "Lobsters", URL: "https://lobste.rs/rss", Category: "Tech News"},
	{Name: "TechCrunch", URL: "https://techcrunch.com/feed/", Category: "Tech News"},
	{Name: "The Verge", URL: "https://www.theverge.com/rss/index.xml", Category: "Tech News"},
	{Name: "Ars Technica", URL: "https://feeds.arstechnica.com/arstechnica/index", Category: "Tech News"},

	// Programming
	{Name: "Go Blog", URL: "https://go.dev/blog/feed.atom", Category: "Programming"},
	{Name: "Rust Blog", URL: "https://blog.rust-lang.org/feed.xml", Category: "Programming"},
	{Name: "Dev.to", URL: "https://dev.to/feed", Category: "Programming"},
	{Name: "JavaScript Weekly", URL: "https://javascriptweekly.com/rss", Category: "Programming"},
	{Name: "CSS-Tricks", URL: "https://css-tricks.com/feed/", Category: "Programming"},

	// AI / ML
	{Name: "OpenAI Blog", URL: "https://openai.com/blog/rss.xml", Category: "AI / ML"},
	{Name: "Anthropic", URL: "https://www.anthropic.com/rss.xml", Category: "AI / ML"},
	{Name: "Hugging Face Blog", URL: "https://huggingface.co/blog/feed.xml", Category: "AI / ML"},
	{Name: "MIT AI News", URL: "https://news.mit.edu/topic/mitartificial-intelligence2-rss.xml", Category: "AI / ML"},

	// Security
	{Name: "Krebs on Security", URL: "https://krebsonsecurity.com/feed/", Category: "Security"},
	{Name: "The Hacker News", URL: "https://feeds.feedburner.com/TheHackersNews", Category: "Security"},
	{Name: "Schneier on Security", URL: "https://www.schneier.com/feed/", Category: "Security"},

	// Linux / Open Source
	{Name: "LWN.net", URL: "https://lwn.net/headlines/rss", Category: "Linux / Open Source"},
	{Name: "Phoronix", URL: "https://www.phoronix.com/rss.php", Category: "Linux / Open Source"},
	{Name: "OMG! Ubuntu", URL: "https://www.omgubuntu.co.uk/feed", Category: "Linux / Open Source"},
	{Name: "It's FOSS", URL: "https://itsfoss.com/feed/", Category: "Linux / Open Source"},

	// Science
	{Name: "Nature News", URL: "https://www.nature.com/nature.rss", Category: "Science"},
	{Name: "Quanta Magazine", URL: "https://www.quantamagazine.org/feed/", Category: "Science"},
	{Name: "New Scientist", URL: "https://www.newscientist.com/feed/home/", Category: "Science"},

	// Health & Fitness
	{Name: "Examine.com", URL: "https://examine.com/blog/feed/", Category: "Health & Fitness"},
	{Name: "Nerd Fitness", URL: "https://www.nerdfitness.com/blog/feed/", Category: "Health & Fitness"},
	{Name: "Huberman Lab", URL: "https://www.hubermanlab.com/rss", Category: "Health & Fitness"},
	{Name: "Precision Nutrition", URL: "https://www.precisionnutrition.com/feed", Category: "Health & Fitness"},

	// RU Tech
	{Name: "Habr", URL: "https://habr.com/ru/rss/articles/?fl=ru", Category: "RU Tech"},
	{Name: "Opennet", URL: "https://www.opennet.ru/opennews/opennews_all_utf.rss", Category: "RU Tech"},
	{Name: "3DNews", URL: "https://3dnews.ru/breaking/rss/", Category: "RU Tech"},

	// Design
	{Name: "Smashing Magazine", URL: "https://www.smashingmagazine.com/feed/", Category: "Design"},
	{Name: "A List Apart", URL: "https://alistapart.com/main/feed/", Category: "Design"},
}

// catalogCategories returns ordered unique categories.
func catalogCategories() []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range catalog {
		if !seen[e.Category] {
			seen[e.Category] = true
			out = append(out, e.Category)
		}
	}
	return out
}

// catalogByCategory returns entries for a given category.
func catalogByCategory(cat string) []CatalogEntry {
	var out []CatalogEntry
	for _, e := range catalog {
		if e.Category == cat {
			out = append(out, e)
		}
	}
	return out
}

// catalogRow is a pre-built row for the catalog display. Items with
// EntryIdx >= 0 are selectable feed entries; -1 marks category headers
// and blank separators.
type catalogRow struct {
	Line     string // rendered line
	EntryIdx int    // index into flat catalog, or -1 for non-selectable
}

// buildCatalogRows pre-renders all catalog rows with styles applied.
func buildCatalogRows(m Model) []catalogRow {
	catStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)
	checkOn := lipgloss.NewStyle().Foreground(colorGreen).Background(colorBG)
	checkOff := lipgloss.NewStyle().Foreground(colorBorder).Background(colorBG)

	subscribed := map[string]bool{}
	for _, f := range m.feeds {
		subscribed[f.URL] = true
	}

	var rows []catalogRow
	idx := 0
	for ci, cat := range catalogCategories() {
		if ci > 0 {
			rows = append(rows, catalogRow{Line: "", EntryIdx: -1})
		}
		rows = append(rows, catalogRow{
			Line:     catStyle.Render("  " + cat),
			EntryIdx: -1,
		})
		for _, entry := range catalogByCategory(cat) {
			prefix := "    "
			style := nameStyle
			if idx == m.catalogSel {
				prefix = "  › "
				style = itemSelected
			}
			check := checkOff.Render("○")
			if subscribed[entry.URL] {
				check = checkOn.Render("●")
			}
			icon := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG).
				Render(feedIcon(entry.URL, entry.Name))
			rows = append(rows, catalogRow{
				Line:     prefix + check + " " + icon + " " + style.Render(entry.Name),
				EntryIdx: idx,
			})
			idx++
		}
	}
	return rows
}

// renderCatalog draws the feed discover/catalog overlay with scroll.
func renderCatalog(m Model, width, height int) string {
	tr := m.tr
	hintStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG).Italic(true)
	accentStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG)

	allRows := buildCatalogRows(m)

	// Find which row index contains the selected entry.
	selRow := 0
	for i, r := range allRows {
		if r.EntryIdx == m.catalogSel {
			selRow = i
			break
		}
	}

	var b strings.Builder

	// Welcome header for onboarding.
	headerRows := 0
	if m.catalogOnboarding {
		b.WriteString("\n")
		b.WriteString(accentStyle.Render("  " + tr.Catalog.Welcome1))
		b.WriteString("\n\n")
		b.WriteString(textStyle.Render("  " + tr.Catalog.Welcome2))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("  " + tr.Catalog.Welcome3))
		b.WriteString("\n\n")
		headerRows = 7
	} else {
		b.WriteString(hintStyle.Render(tr.Catalog.Subtitle))
		b.WriteString("\n\n")
		headerRows = 2
	}

	// Visible area for the scrollable list.
	visRows := height - headerRows - 2 // -2 for hint at bottom
	if visRows < 5 {
		visRows = 5
	}

	// Scroll window around selRow.
	start := selRow - visRows/2
	if start < 0 {
		start = 0
	}
	end := start + visRows
	if end > len(allRows) {
		end = len(allRows)
		start = end - visRows
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		b.WriteString(allRows[i].Line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render(tr.Catalog.Hint))

	title := "\U000f046b " + tr.Catalog.Title
	return framePaneWithTitle(b.String(), title, true, width, height)
}

// catalogFlatIndex returns the CatalogEntry at the given flat index
// (across all categories).
func catalogFlatIndex(idx int) *CatalogEntry {
	i := 0
	for _, cat := range catalogCategories() {
		for _, entry := range catalogByCategory(cat) {
			if i == idx {
				e := entry
				_ = cat
				return &e
			}
			i++
		}
	}
	return nil
}

// catalogLen returns the total number of entries in the catalog.
func catalogLen() int {
	return len(catalog)
}

// seedSmartFolders creates the default smart folders on first launch,
// using localized names from the current language.
func seedSmartFolders(database *db.DB, tr *i18n.Strings) {
	defaults := []struct {
		Name  string
		Query string
	}{
		{tr.Catalog.FolderInbox, "unread"},
		{tr.Catalog.FolderToday, "today"},
		{tr.Catalog.FolderThisWeek, "newer:1w unread"},
		{tr.Catalog.FolderStarred, "starred"},
	}
	for _, d := range defaults {
		_, _ = database.InsertSmartFolder(d.Name, d.Query)
	}
}

// updateCatalog handles keystrokes in the catalog overlay.
func (m Model) updateCatalog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := catalogLen()
	switch {
	case key.Matches(msg, m.keys.Quit):
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// q — treat as close catalog
		m.focus = focusFeeds
		if len(m.feeds) > 0 {
			tick := m.startBusy(m.tr.Status.Fetching)
			return m, tea.Batch(loadFeedsCmd(m.db), fetchAllCmd(m.fetcher), tick)
		}
		return m, nil
	case key.Matches(msg, m.keys.Back):
		m.focus = focusFeeds
		if len(m.feeds) > 0 {
			tick := m.startBusy(m.tr.Status.Fetching)
			return m, tea.Batch(loadFeedsCmd(m.db), fetchAllCmd(m.fetcher), tick)
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.catalogSel < total-1 {
			m.catalogSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.catalogSel > 0 {
			m.catalogSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.catalogSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if total > 0 {
			m.catalogSel = total - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		entry := catalogFlatIndex(m.catalogSel)
		if entry == nil {
			return m, nil
		}
		// Check if already subscribed.
		for _, f := range m.feeds {
			if f.URL == entry.URL {
				return m, m.showToast(m.tr.Catalog.Added + ": " + entry.Name)
			}
		}
		_, err := m.db.UpsertFeed(entry.Name, entry.URL, entry.Category)
		if err != nil {
			m.err = err
			return m, nil
		}
		return m, tea.Batch(
			loadFeedsCmd(m.db),
			m.showToast("+ "+entry.Name),
		)
	}
	return m, nil
}
