package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/i18n"
)

// articlePreviewRows is the fixed height of the inline preview block
// that appears directly below the selected article while the Articles
// pane is focused. Kept small so it doesn't dominate the list.
const articlePreviewRows = 3

func renderArticleList(articles []db.Article, selected int, active bool, width, height int, tr *i18n.Strings, showPreview bool, visualLo, visualHi int) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	title := " " + tr.Feeds.ArticlesPaneTitle
	var b strings.Builder

	if len(articles) == 0 {
		b.WriteString(readStyle.Render(tr.Feeds.NoArticles))
		return framePaneWithTitle(b.String(), title, active, width, height)
	}

	// Reserve fixed right-side columns for source chip and time + optional
	// reading hint. Keeping source/time out of the title cell makes rows
	// scan like a table instead of letting metadata float inside headlines.
	const whenCellW = 14
	const maxSourceCellW = 13
	const columnSepW = 2

	// Detect cross-feed view: if any row carries a FeedName the loader is
	// a folder/all-articles source, so reserve a fixed source column.
	crossFeed := false
	for _, a := range articles {
		if a.FeedName != "" {
			crossFeed = true
			break
		}
	}

	// Inner text area of the pane = width - 2 (padding inside border).
	innerW := width - 2
	sourceCellW := 0
	if crossFeed && innerW >= 54 {
		sourceCellW = maxSourceCellW
	}
	sepCellW := 0
	if sourceCellW > 0 && innerW >= 70 {
		sepCellW = columnSepW
	}
	titleCellW := innerW - whenCellW - sourceCellW - sepCellW
	if titleCellW < 1 {
		titleCellW = 1
	}

	// Max length for truncating the title text: subtract the prefix
	// "▌ "/"› "/"  " (2 cells) and marker "● "/"★ " (2 cells).
	titleTextBudget := titleCellW - 4
	if titleTextBudget < 1 {
		titleTextBudget = 1
	}

	now := time.Now()
	bucketCounts := make(map[string]int)
	for _, a := range articles {
		bucketCounts[dateBucket(a.PublishedAt, now, tr)]++
	}
	groupStyle := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)

	rowsBudget := listVisibleRows(height)
	// Worst case the visible window spans up to 4 date-group boundaries
	// (Today/Yesterday/This week/This month/Older). Reserve a handful of
	// rows up front so headers fit; unused rows are padded at the end.
	itemBudget := rowsBudget - 2
	if itemBudget < 1 {
		itemBudget = 1
	}

	start, end := visibleWindow(len(articles), selected, itemBudget)
	var lastBucket string
	// Prime lastBucket with the row ABOVE the window so the first visible
	// row still gets a header if it's the start of a bucket. When start==0
	// we force-emit the header for the first bucket.
	if start > 0 {
		lastBucket = dateBucket(articles[start-1].PublishedAt, now, tr)
	}
	// Line index within the assembled body string where the selected
	// row lands. Used after the loop to overlay the floating preview
	// box "on top" of rows below the cursor without reflowing the list.
	selY := -1
	rowsUsed := 0
	for i := start; i < end; i++ {
		if rowsUsed >= rowsBudget {
			break
		}
		bucket := dateBucket(articles[i].PublishedAt, now, tr)
		if bucket != lastBucket {
			if rowsUsed+1 >= rowsBudget {
				break
			}
			label := strings.ToUpper(bucket)
			if n := bucketCounts[bucket]; n > 0 {
				label = fmt.Sprintf("%s · %d", label, n)
			}
			b.WriteString(groupStyle.Render(" " + label))
			b.WriteString("\n")
			rowsUsed++
			lastBucket = bucket
		}
		a := articles[i]
		titleStyle := unreadStyle
		if a.ReadAt != nil {
			titleStyle = readStyle
		}
		prefix := "  "
		if i == selected {
			prefix = "› "
			if active {
				prefix = "▌ "
				titleStyle = itemSelected
			} else {
				titleStyle = itemSelectedInactive
			}
		}
		rowBG := colorBG
		inVisual := active && visualLo >= 0 && visualHi >= 0 && i >= visualLo && i <= visualHi
		if (i == selected && active) || inVisual {
			rowBG = colorAltBG
		}
		rowTitleStyle := titleStyle.Background(rowBG)
		whenFG := colorMuted
		if i == selected && active {
			whenFG = colorOrange
		}
		rowWhenStyle := lipgloss.NewStyle().Foreground(whenFG).Background(rowBG)

		prefixStyle := lipgloss.NewStyle().Background(rowBG)
		if i == selected && active {
			prefixStyle = prefixStyle.Foreground(colorAccent).Bold(true)
		}
		prefixRendered := prefixStyle.Render(prefix)
		var marker string
		switch {
		case a.StarredAt != nil:
			marker = lipgloss.NewStyle().Foreground(colorYellow).Background(rowBG).Render("★ ")
		case a.BookmarkedAt != nil:
			marker = lipgloss.NewStyle().Foreground(colorSecondary).Background(rowBG).Render("◆ ")
		case a.ReadAt == nil:
			// Unread state is already conveyed by bold title text. Avoid a
			// repeated dot on every unread row: in unread-heavy views it turns
			// into visual noise. Star/bookmark still use explicit markers.
			marker = lipgloss.NewStyle().Background(rowBG).Render("  ")
		default:
			marker = lipgloss.NewStyle().Background(rowBG).Render("  ")
		}
		title := prefixRendered + marker + rowTitleStyle.Render(truncate(a.Title, titleTextBudget))
		whenText := timeAgo(a.PublishedAt, tr)
		// Prepend a small reading-time hint when the body is cached. Build
		// the timestamp as plain text first so long localized labels (e.g.
		// "13 мин назад") are truncated before styling and never wrap into
		// a second visual row.
		if a.CachedBody != "" {
			if mins := readingMinutes(a.CachedBody); mins > 0 {
				whenText = fmt.Sprintf("%dm·%s", mins, whenText)
			}
		}
		when := rowWhenStyle.Render(truncate(whenText, whenCellW))

		titleCellStyle := lipgloss.NewStyle().Width(titleCellW).MaxWidth(titleCellW).Background(rowBG)
		whenCellStyle := lipgloss.NewStyle().Width(whenCellW).MaxWidth(whenCellW).Align(lipgloss.Right).Background(rowBG)
		cells := []string{titleCellStyle.Render(title)}
		if sepCellW > 0 {
			sep := lipgloss.NewStyle().Width(sepCellW).MaxWidth(sepCellW).Foreground(colorBorder).Background(rowBG).Render("│ ")
			cells = append(cells, sep)
		}
		if sourceCellW > 0 {
			sourceCellStyle := lipgloss.NewStyle().Width(sourceCellW).MaxWidth(sourceCellW).Background(rowBG)
			cells = append(cells, sourceCellStyle.Render(sourceChip(a.FeedName, sourceCellW, i == selected && active, rowBG)))
		}
		cells = append(cells, whenCellStyle.Render(when))
		line := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		if i == selected {
			// Record line index before appending: position the preview
			// overlay right below this row.
			selY = strings.Count(b.String(), "\n")
		}
		b.WriteString(line)
		b.WriteString("\n")
		rowsUsed++
	}

	// Pad to the full budget so the pane's content height is constant.
	for rowsUsed < rowsBudget {
		b.WriteString("\n")
		rowsUsed++
	}

	body := b.String()
	// Floating preview overlay: a bordered box drawn over the rows
	// immediately below the selected article (or above it if the cursor
	// is too close to the bottom). The underlying list is rendered at
	// full height; overlay replaces lines non-destructively so moving
	// the cursor restores previously-covered rows on next render.
	if active && showPreview && selY >= 0 && selected >= 0 && selected < len(articles) {
		box := buildArticlePreviewBox(articles[selected], width-2, articlePreviewRows, tr)
		body = overlayPreviewBox(body, box, selY, rowsBudget)
	}

	return framePaneWithTitle(body, title, active, width, height)
}

func sourceChip(name string, cellW int, selected bool, rowBG lipgloss.Color) string {
	name = compactSourceName(strings.TrimSpace(name))
	if name == "" || cellW < 3 {
		return ""
	}
	labelW := cellW - 1 // one leading gap before the source label
	if labelW < 1 {
		labelW = 1
	}
	label := truncate(name, labelW)
	style := lipgloss.NewStyle().Foreground(colorTeal).Background(rowBG).Bold(true)
	if selected {
		style = style.Foreground(colorAccent)
	}
	return lipgloss.NewStyle().Background(rowBG).Render(" ") + style.Render(label)
}

func compactSourceName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "hacker news"):
		return "HN"
	case strings.Contains(lower, "ixbt"):
		return "Ixbt"
	case strings.Contains(lower, "habr"):
		return "Habr"
	}
	return name
}

// buildArticlePreviewBox renders a bordered floating-style popup with
// up to bodyRows lines of the article's richest available text. outerW
// is the total pane inner width including border; the box fits within
// it. Returns a lipgloss-rendered string already framed in a rounded
// border so callers can overlay it line-by-line onto the list output.
func buildArticlePreviewBox(a db.Article, outerW, bodyRows int, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}

	// Priority: CachedBody (full article fetched via 'f') → existing
	// articlePreviewText helper (Content → Description fallback).
	var source string
	if a.CachedBody != "" {
		source = sanitizeArticleMarkdown(a.CachedBody, false)
	} else {
		source = articlePreviewText(a)
	}
	plain := strings.TrimSpace(stripHTML(source))

	// Inner width inside the border + padding. outerW already accounts
	// for pane padding; the box itself adds 1 border cell + 1 padding
	// cell on each side, so content is outerW - 4.
	innerW := outerW - 4
	if innerW < 10 {
		innerW = 10
	}

	mutedStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	var content string
	if plain == "" {
		content = mutedStyle.Italic(true).Render(tr.Feeds.NoArticlePreview)
	} else {
		wrapped := wrap(plain, innerW)
		lines := strings.Split(wrapped, "\n")
		if len(lines) > bodyRows {
			lines = lines[:bodyRows]
			last := strings.TrimRight(lines[bodyRows-1], " ")
			if lipgloss.Width(last) > 0 {
				lines[bodyRows-1] = last + "…"
			}
		}
		for i, l := range lines {
			lines[i] = mutedStyle.Render(l)
		}
		content = strings.Join(lines, "\n")
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(innerW).
		MaxWidth(outerW)
	return boxStyle.Render(content)
}

// overlayPreviewBox replaces rows in body with the lines of box so the
// box visually floats over the list. selY is the line index of the
// selected article row in body. Box is anchored below the row when
// possible; if the selected row is too close to the bottom of the
// visible area, the box jumps above it. body is expected to be padded
// to exactly rowsBudget content rows (plus the title row above).
func overlayPreviewBox(body, box string, selY, rowsBudget int) string {
	baseLines := strings.Split(body, "\n")
	boxLines := strings.Split(box, "\n")
	boxH := len(boxLines)
	if boxH == 0 || len(baseLines) == 0 {
		return body
	}

	// body starts with the pane title row. Rows 1..rowsBudget are the
	// data region where we may overlay. lastDataLine is inclusive.
	firstDataLine := 1
	lastDataLine := rowsBudget // inclusive index of last data row

	// Preferred position: directly below the selected row.
	startY := selY + 1
	// If the box would overflow past the bottom of the data region,
	// flip to above the selected row instead.
	if startY+boxH-1 > lastDataLine {
		startY = selY - boxH
		if startY < firstDataLine {
			// List shorter than the box — anchor to the top of the
			// data area and let the box cover whatever fits.
			startY = firstDataLine
		}
	}

	for i, bl := range boxLines {
		y := startY + i
		if y < firstDataLine || y > lastDataLine {
			continue
		}
		if y >= len(baseLines) {
			break
		}
		baseLines[y] = bl
	}
	return strings.Join(baseLines, "\n")
}

// readingMinutes returns the estimated read time in whole minutes for a
// chunk of article text. Returns 0 for stubs too short to be meaningful
// (<20 words) so the article list doesn't show "1m·" everywhere.
func readingMinutes(body string) int {
	text := stripHTML(body)
	words := len(strings.Fields(text))
	if words < 20 {
		return 0
	}
	mins := (words + 199) / 200
	if mins < 1 {
		mins = 1
	}
	return mins
}

// dateBucket labels an article by how recent it is, for the group headers
// inserted into the article list. now is injected so tests are deterministic.
func dateBucket(t, now time.Time, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	if t.IsZero() {
		return tr.Reader.BucketOlder
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return tr.Reader.BucketToday
	case !t.Before(today.AddDate(0, 0, -1)):
		return tr.Reader.BucketYesterday
	case !t.Before(today.AddDate(0, 0, -7)):
		return tr.Reader.BucketWeek
	case !t.Before(today.AddDate(0, 0, -30)):
		return tr.Reader.BucketMonth
	default:
		return tr.Reader.BucketOlder
	}
}

func timeAgo(t time.Time, tr *i18n.Strings) string {
	if t.IsZero() {
		return ""
	}
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf(tr.Reader.TimeAgoMinFmt, int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf(tr.Reader.TimeAgoHourFmt, int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf(tr.Reader.TimeAgoDayFmt, int(d.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}
