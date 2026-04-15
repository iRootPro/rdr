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

	titleW := width - 14
	if titleW < 1 {
		titleW = 1
	}

	// Detect cross-feed view: if any row carries a FeedName the loader is
	// a folder/all-articles source, so reserve space for a feed tag.
	crossFeed := false
	for _, a := range articles {
		if a.FeedName != "" {
			crossFeed = true
			break
		}
	}
	feedTagW := 0
	if crossFeed {
		feedTagW = 12 // room for "[feedname] "
	}

	now := time.Now()
	groupStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

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
		lastBucket = dateBucket(articles[start-1].PublishedAt, now)
	}
	rowsUsed := 0
	for i := start; i < end; i++ {
		if rowsUsed >= rowsBudget {
			break
		}
		bucket := dateBucket(articles[i].PublishedAt, now)
		if bucket != lastBucket {
			if rowsUsed+1 >= rowsBudget {
				break
			}
			b.WriteString(groupStyle.Render("── " + bucket + " ──"))
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
				titleStyle = itemSelected
			} else {
				titleStyle = itemSelectedInactive
			}
		}
		star := "  "
		if a.StarredAt != nil {
			star = lipgloss.NewStyle().Foreground(colorYellow).Render("★ ")
		}

		titleBudget := titleW - 2 - feedTagW
		if titleBudget < 1 {
			titleBudget = 1
		}
		titleText := titleStyle.Render(prefix+star) + titleStyle.Render(truncate(a.Title, titleBudget))
		if crossFeed && a.FeedName != "" {
			tag := lipgloss.NewStyle().
				Foreground(colorGreen).
				Render("  " + truncate(a.FeedName, 10))
			titleText += tag
		}
		title := titleText
		when := timeAgoStyle.Render(timeAgo(a.PublishedAt))
		// Prepend a small reading-time hint when the body is cached.
		if a.CachedBody != "" {
			if mins := readingMinutes(a.CachedBody); mins > 0 {
				hint := lipgloss.NewStyle().
					Foreground(colorMuted).
					Render(fmt.Sprintf("%dm·", mins))
				when = hint + when
			}
		}

		titleCellStyle := lipgloss.NewStyle().Width(width - 12)
		whenCellStyle := lipgloss.NewStyle()
		if i == selected && active {
			titleCellStyle = titleCellStyle.Background(colorAltBG)
			whenCellStyle = whenCellStyle.Background(colorAltBG)
		}
		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			titleCellStyle.Render(title),
			whenCellStyle.Render(when),
		)
		b.WriteString(line)
		b.WriteString("\n")
		rowsUsed++
	}

	// Pad to the full budget so the pane's content height is constant.
	for rowsUsed < rowsBudget {
		b.WriteString("\n")
		rowsUsed++
	}

	return framePane(b.String(), active, width, height)
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
func dateBucket(t, now time.Time) string {
	if t.IsZero() {
		return "Older"
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case !t.Before(today):
		return "Today"
	case !t.Before(today.AddDate(0, 0, -1)):
		return "Yesterday"
	case !t.Before(today.AddDate(0, 0, -7)):
		return "This week"
	case !t.Before(today.AddDate(0, 0, -30)):
		return "This month"
	default:
		return "Older"
	}
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
