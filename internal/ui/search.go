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
	searchTitle        lipgloss.Style
	searchHint         lipgloss.Style
	searchCount        lipgloss.Style
	searchFeedTag      lipgloss.Style
	searchPreviewTitle lipgloss.Style
	searchPreviewMeta  lipgloss.Style
	searchPreviewBody  lipgloss.Style
	searchPreviewURL   lipgloss.Style
)

func init() {
	rebuildSearchStyles()
	registerStyleRebuild(rebuildSearchStyles)
}

func rebuildSearchStyles() {
	searchTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true).
		Padding(0, 0, 1, 0)

	searchHint = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG).
		Italic(true)

	searchCount = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	searchFeedTag = lipgloss.NewStyle().
		Foreground(colorGreen).
		Background(colorBG)

	searchPreviewTitle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true)

	searchPreviewMeta = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	searchPreviewBody = lipgloss.NewStyle().Foreground(colorText).Background(colorBG)

	searchPreviewURL = lipgloss.NewStyle().Foreground(colorTeal).Background(colorBG).Underline(true)
}

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
// so Update and View agree on what is visible. Layout overhead = 8 rows:
// title (2) + input box (2: input + border bottom) + hint (1) + blank (1) +
// blank-before-count (1) + count (1).
func searchListRows(m Model) int {
	helpH := lipgloss.Height(m.helpView())
	innerH := m.height - 1 - helpH - 2
	if innerH < 8 {
		innerH = 8
	}
	rows := innerH - 8
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
	if strings.TrimSpace(m.searchInput.Value()) == "" {
		return m, nil
	}
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
	m.reader.SetContent(buildReaderContent(art, item.FeedName, "", m.reader.Width-4, m.showImages, m.tr))
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
	//   input box (2 rows: input + border bottom)
	//   hint row (1 row: syntax cheatsheet or parser error)
	//   blank (1 row)
	//   list area (listRows, always padded)
	//   blank (1 row)
	//   count row (1 row)
	// Total: listRows + 8. Pane border adds 2 more.
	innerH := height - 2
	if innerH < 8 {
		innerH = 8
	}
	listRows := innerH - 8
	if listRows < 1 {
		listRows = 1
	}

	// Set Width on the textinput before View(). When Width is zero, bubbles
	// renders a zero-width visible window and typed characters scroll
	// horizontally out of view — that's the "input not visible" bug.
	inputBoxW := width - 4
	if inputBoxW < 12 {
		inputBoxW = 12
	}
	inputW := inputBoxW - lipgloss.Width(m.searchInput.Prompt) - 2
	if inputW < 8 {
		inputW = 8
	}
	m.searchInput.Width = inputW

	inputBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(colorAccent).
		Background(colorBG).
		Padding(0, 1).
		Width(inputBoxW).
		Render(m.searchInput.View())

	var hintLine string
	if m.searchErr != nil {
		hintLine = errStyle.Render(truncate("! "+m.searchErr.Error(), width-4))
	} else {
		hintLine = searchHint.Render(truncate(m.tr.Search.SyntaxHint, width-4))
	}

	var b strings.Builder
	b.WriteString(searchTitle.Render(m.tr.Search.Title))
	b.WriteString("\n")
	b.WriteString(inputBox)
	b.WriteString("\n")
	b.WriteString(hintLine)
	b.WriteString("\n\n")

	// Geometry of one result row inside the pane (width = pane content width):
	//   prefix (2) + title (titleW) + gap (2) + feedTag (≤12) = leftCol slot
	//   leftCol slot must fit inside Width(width-16) without wrapping, else
	//   long titles get re-flowed onto a 2nd line and the whole list area
	//   overflows listRows — pushing the pane title and input out of view.
	// → leftCol slot ≤ width-16  ⇒  titleW ≤ width-32.
	titleW := width - 32
	if titleW < 5 {
		titleW = 5
	}

	emptyTitle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	emptyHint := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG).Italic(true)

	switch {
	case len(m.searchAll) == 0:
		body := lipgloss.JoinVertical(lipgloss.Center,
			emptyTitle.Render(m.tr.Search.NoArticles),
			emptyHint.Render(m.tr.Search.NoArticlesHint),
		)
		b.WriteString(lipgloss.Place(width-4, listRows, lipgloss.Center, lipgloss.Center, body, lipgloss.WithWhitespaceBackground(colorBG)))
		b.WriteString("\n")
	case len(m.searchMatches) == 0:
		body := lipgloss.JoinVertical(lipgloss.Center,
			emptyTitle.Render(m.tr.Search.NoMatches),
			emptyHint.Render(m.tr.Search.NoMatchesHint),
		)
		b.WriteString(lipgloss.Place(width-4, listRows, lipgloss.Center, lipgloss.Center, body, lipgloss.WithWhitespaceBackground(colorBG)))
		b.WriteString("\n")
	default:
		start := m.searchScroll
		if start < 0 {
			start = 0
		}
		end := start + listRows
		if end > len(m.searchMatches) {
			end = len(m.searchMatches)
		}
		hasQuery := strings.TrimSpace(m.searchInput.Value()) != ""
		rendered := 0
		for i := start; i < end; i++ {
			item := m.searchAll[m.searchMatches[i]]
			prefix := "  "
			style := unreadStyle
			if item.ReadAt != nil {
				style = readStyle
			}
			if hasQuery && i == m.searchSel {
				prefix = "› "
				style = itemSelected
			}
			feedTag := searchFeedTag.Render(truncate(item.FeedName, 12))
			title := style.Render(prefix + truncate(item.Title, titleW))
			when := timeAgoStyle.Render(timeAgo(item.PublishedAt, m.tr))
			leftCol := lipgloss.NewStyle().Width(width - 16).MaxWidth(width - 16).Inline(true).Render(title + "  " + feedTag)
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
	b.WriteString(searchCount.Render(fmt.Sprintf(m.tr.Search.ResultsFmt, len(m.searchMatches), len(m.searchAll))))

	c := fillBackground(b.String(), width-4)
	return paneActive.Width(width - 2).Height(innerH).Render(c)
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
	b.WriteString(searchTitle.Render(m.tr.Search.PreviewTitle))
	b.WriteString("\n")

	hasQuery := strings.TrimSpace(m.searchInput.Value()) != ""
	hasSelection := hasQuery && len(m.searchMatches) > 0 && m.searchSel >= 0 && m.searchSel < len(m.searchMatches)
	if !hasSelection {
		b.WriteString(searchHint.Render(m.tr.Search.NoSelection))
		b.WriteString(strings.Repeat("\n", innerH-3))
		c := fillBackground(b.String(), width-4)
		return paneInactive.Width(width - 2).Height(innerH).Render(c)
	}

	item := m.searchAll[m.searchMatches[m.searchSel]]
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}

	b.WriteString(searchPreviewTitle.Render(truncate(item.Title, innerW)))
	b.WriteString("\n")
	meta := []string{searchFeedTag.Render(item.FeedName)}
	if ago := timeAgo(item.PublishedAt, m.tr); ago != "" {
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
		body = searchHint.Render(m.tr.Search.NoPreviewHint)
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

	c := fillBackground(b.String(), width-4)
	return paneInactive.Width(width - 2).Height(innerH).Render(c)
}

