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

	for i, a := range articles {
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

		title := titleStyle.Render(prefix + truncate(a.Title, titleW))
		when := timeAgoStyle.Render(timeAgo(a.PublishedAt))

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(width-12).Render(title),
			when,
		)
		b.WriteString(line)
		b.WriteString("\n")
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
