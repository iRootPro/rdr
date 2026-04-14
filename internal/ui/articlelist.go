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

	start, end := visibleWindow(len(articles), selected, listVisibleRows(height))
	for i := start; i < end; i++ {
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

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(width-12).Render(title),
			when,
		)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return framePane(b.String(), active, width, height)
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
