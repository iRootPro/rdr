package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

var (
	searchTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)

	searchHint = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	searchCount = lipgloss.NewStyle().
			Foreground(colorMuted)

	searchFeedTag = lipgloss.NewStyle().
			Foreground(colorGreen)

	searchPreviewTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	searchPreviewMeta = lipgloss.NewStyle().
				Foreground(colorMuted)

	searchPreviewBody = lipgloss.NewStyle().Foreground(colorText)

	searchPreviewURL = lipgloss.NewStyle().Foreground(colorTeal).Underline(true)
)

func recomputeMatches(m *Model) {
	q := strings.TrimSpace(m.searchInput.Value())
	m.searchMatches = m.searchMatches[:0]
	if q == "" {
		for i := range m.searchAll {
			m.searchMatches = append(m.searchMatches, i)
		}
		m.searchErr = nil
	} else {
		atoms, err := ParseQuery(q)
		if err != nil {
			// Keep last valid results visible; surface parse error in hint.
			m.searchErr = err
		} else {
			m.searchErr = nil
			for i, it := range m.searchAll {
				if EvalQuery(atoms, it) {
					m.searchMatches = append(m.searchMatches, i)
				}
			}
		}
	}
	// Query changed → results changed → reset viewport so user sees first hits.
	m.searchSel = 0
	m.searchScroll = 0
}

// searchListRows computes how many result rows fit in the search pane given
// the current terminal height. Mirrors the geometry used in renderSearchLeft
// so Update and View agree on what is visible.
func searchListRows(m Model) int {
	helpH := lipgloss.Height(m.helpView())
	innerH := m.height - 1 - helpH - 2
	if innerH < 6 {
		innerH = 6
	}
	rows := innerH - 6
	if rows < 1 {
		rows = 1
	}
	return rows
}

// clampSearchScroll adjusts searchScroll so the selected index stays visible
// without re-centering: items only shift once the cursor leaves the window.
func clampSearchScroll(m *Model) {
	total := len(m.searchMatches)
	if total == 0 {
		m.searchScroll = 0
		return
	}
	rows := searchListRows(*m)
	if total <= rows {
		m.searchScroll = 0
		return
	}
	if m.searchSel < m.searchScroll {
		m.searchScroll = m.searchSel
	}
	if m.searchSel >= m.searchScroll+rows {
		m.searchScroll = m.searchSel - rows + 1
	}
	if m.searchScroll < 0 {
		m.searchScroll = 0
	}
	maxScroll := total - rows
	if m.searchScroll > maxScroll {
		m.searchScroll = maxScroll
	}
}

func loadSearchCmd(d *db.DB) tea.Cmd {
	return func() tea.Msg {
		items, err := d.SearchArticles(2000)
		if err != nil {
			return errMsg{err}
		}
		return searchLoadedMsg{items: items}
	}
}

func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.focus = m.searchPrev
		if m.focus == focusSearch {
			m.focus = focusFeeds
		}
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.searchMatches = nil
		m.searchScroll = 0
		return m, nil
	case key.Matches(msg, m.keys.Quit) && msg.String() == "ctrl+c":
		return m, tea.Quit
	case key.Matches(msg, m.keys.Enter):
		return m.openSearchSelection()
	case key.Matches(msg, m.keys.Down), msg.String() == "ctrl+n":
		if m.searchSel < len(m.searchMatches)-1 {
			m.searchSel++
		}
		clampSearchScroll(&m)
		return m, nil
	case key.Matches(msg, m.keys.Up), msg.String() == "ctrl+p":
		if m.searchSel > 0 {
			m.searchSel--
		}
		clampSearchScroll(&m)
		return m, nil
	case msg.String() == "pgdown", msg.String() == "ctrl+d":
		m.searchSel += searchListRows(m)
		if m.searchSel >= len(m.searchMatches) {
			m.searchSel = len(m.searchMatches) - 1
		}
		if m.searchSel < 0 {
			m.searchSel = 0
		}
		clampSearchScroll(&m)
		return m, nil
	case msg.String() == "pgup", msg.String() == "ctrl+u":
		m.searchSel -= searchListRows(m)
		if m.searchSel < 0 {
			m.searchSel = 0
		}
		clampSearchScroll(&m)
		return m, nil
	case msg.String() == "home":
		m.searchSel = 0
		clampSearchScroll(&m)
		return m, nil
	case msg.String() == "end":
		if len(m.searchMatches) > 0 {
			m.searchSel = len(m.searchMatches) - 1
		}
		clampSearchScroll(&m)
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	recomputeMatches(&m)
	clampSearchScroll(&m)
	return m, cmd
}

func (m Model) openSearchSelection() (tea.Model, tea.Cmd) {
	if len(m.searchMatches) == 0 {
		return m, nil
	}
	if m.searchSel < 0 || m.searchSel >= len(m.searchMatches) {
		return m, nil
	}
	item := m.searchAll[m.searchMatches[m.searchSel]]

	art := db.Article{
		ID:          item.ArticleID,
		FeedID:      item.FeedID,
		Title:       item.Title,
		URL:         item.URL,
		Description: item.Description,
		CachedBody:  item.CachedBody,
		PublishedAt: item.PublishedAt,
		ReadAt:      item.ReadAt,
	}
	m.readerArt = &art
	m.focus = focusReader
	m.reader.Width = m.width - 4
	m.reader.Height = m.height - 2
	m.reader.SetContent(buildReaderContent(art, item.FeedName, m.reader.Width-4, m.showImages))
	m.reader.GotoTop()

	m.searchInput.Blur()
	m.searchInput.SetValue("")
	m.searchMatches = nil

	cmds := []tea.Cmd{}
	if art.ReadAt == nil {
		cmds = append(cmds, markReadCmd(m.db, art.ID))
	}
	return m, tea.Batch(cmds...)
}

func renderSearch(m Model, width, height int) string {
	if width < 60 {
		return renderSearchLeft(m, width, height)
	}
	leftW := width / 2
	if leftW < 30 {
		leftW = 30
	}
	rightW := width - leftW - 2
	if rightW < 20 {
		rightW = 20
	}

	left := renderSearchLeft(m, leftW, height)
	right := renderSearchPreview(m, rightW, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func renderSearchLeft(m Model, width, height int) string {
	// Fixed inner layout (so nothing jumps as user navigates):
	//   title block (paneTitle → 2 rows: title + bottom padding)
	//   input row (1 row)
	//   blank (1 row)
	//   list area (listRows, always padded)
	//   blank (1 row)
	//   count row (1 row)
	// Total: listRows + 6. Pane border adds 2 more.
	innerH := height - 2
	if innerH < 6 {
		innerH = 6
	}
	listRows := innerH - 6
	if listRows < 1 {
		listRows = 1
	}

	var b strings.Builder
	b.WriteString(searchTitle.Render("Search"))
	b.WriteString("\n")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	titleW := width - 16
	if titleW < 1 {
		titleW = 1
	}

	switch {
	case len(m.searchAll) == 0:
		b.WriteString(readStyle.Render("(no articles)"))
		// Pad the remaining list rows so count stays at bottom.
		b.WriteString(strings.Repeat("\n", listRows-1))
	case len(m.searchMatches) == 0:
		b.WriteString(readStyle.Render("(no matches)"))
		b.WriteString(strings.Repeat("\n", listRows-1))
	default:
		start := m.searchScroll
		if start < 0 {
			start = 0
		}
		end := start + listRows
		if end > len(m.searchMatches) {
			end = len(m.searchMatches)
		}
		rendered := 0
		for i := start; i < end; i++ {
			item := m.searchAll[m.searchMatches[i]]
			prefix := "  "
			style := unreadStyle
			if item.ReadAt != nil {
				style = readStyle
			}
			if i == m.searchSel {
				prefix = "› "
				style = itemSelected
			}
			feedTag := searchFeedTag.Render(truncate(item.FeedName, 12))
			title := style.Render(prefix + truncate(item.Title, titleW))
			when := timeAgoStyle.Render(timeAgo(item.PublishedAt))
			leftCol := lipgloss.NewStyle().Width(width - 16).Render(title + "  " + feedTag)
			line := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, when)
			b.WriteString(line)
			b.WriteString("\n")
			rendered++
		}
		// Pad with blank rows so items remain at fixed screen positions.
		for i := rendered; i < listRows; i++ {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	if m.searchErr != nil {
		b.WriteString(errStyle.Render("! " + m.searchErr.Error()))
	} else {
		b.WriteString(searchCount.Render(fmt.Sprintf("%d/%d results", len(m.searchMatches), len(m.searchAll))))
	}

	return paneActive.Width(width - 2).Height(innerH).Render(b.String())
}

func renderSearchPreview(m Model, width, height int) string {
	// Fixed inner layout:
	//   title (paneTitle → 2 rows: title + bottom pad)
	//   item title (1 row)
	//   meta (1 row)
	//   divider (1 row)
	//   blank (1 row)
	//   body (bodyRows, always padded)
	// Total: bodyRows + 6. Border adds 2.
	innerH := height - 2
	if innerH < 6 {
		innerH = 6
	}
	bodyRows := innerH - 6
	if bodyRows < 1 {
		bodyRows = 1
	}

	var b strings.Builder
	b.WriteString(searchTitle.Render("Preview"))
	b.WriteString("\n")

	hasSelection := len(m.searchMatches) > 0 && m.searchSel >= 0 && m.searchSel < len(m.searchMatches)
	if !hasSelection {
		b.WriteString(searchHint.Render("(no selection)"))
		b.WriteString(strings.Repeat("\n", innerH-3))
		return paneInactive.Width(width - 2).Height(innerH).Render(b.String())
	}

	item := m.searchAll[m.searchMatches[m.searchSel]]
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}

	b.WriteString(searchPreviewTitle.Render(truncate(item.Title, innerW)))
	b.WriteString("\n")
	meta := []string{searchFeedTag.Render(item.FeedName)}
	if ago := timeAgo(item.PublishedAt); ago != "" {
		meta = append(meta, searchPreviewMeta.Render(ago))
	}
	if item.URL != "" {
		meta = append(meta, searchPreviewURL.Render(truncate(item.URL, innerW-30)))
	}
	b.WriteString(strings.Join(meta, searchPreviewMeta.Render(" · ")))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", innerW))
	b.WriteString("\n\n")

	body := ""
	switch {
	case item.CachedBody != "":
		if rendered, err := renderMarkdown(item.CachedBody, innerW); err == nil {
			body = rendered
		} else {
			body = searchPreviewBody.Render(wrap(stripHTML(item.CachedBody), innerW))
		}
	case item.Description != "":
		body = searchPreviewBody.Render(wrap(stripHTML(item.Description), innerW))
	default:
		body = searchHint.Render("(no preview — press enter, then f to load full)")
	}

	// Clip AND pad body to exact bodyRows so the pane never changes height.
	lines := strings.Split(body, "\n")
	if len(lines) > bodyRows {
		lines = lines[:bodyRows]
	}
	for len(lines) < bodyRows {
		lines = append(lines, "")
	}
	b.WriteString(strings.Join(lines, "\n"))

	return paneInactive.Width(width - 2).Height(innerH).Render(b.String())
}

