package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

func renderFeedList(feeds []db.Feed, errors map[int64]error, selected int, active bool, width, height int) string {
	var b strings.Builder
	b.WriteString(paneTitle.Render("Feeds"))
	b.WriteString("\n")

	if len(feeds) == 0 {
		b.WriteString(readStyle.Render("(no feeds)"))
		return framePane(b.String(), active, width, height)
	}

	nameW := width - 10
	if nameW < 1 {
		nameW = 1
	}

	for i, f := range feeds {
		prefix := "  "
		nameStyle := lipgloss.NewStyle()
		if i == selected {
			prefix = "› "
			if active {
				nameStyle = itemSelected
			} else {
				nameStyle = itemSelectedInactive
			}
		}

		errMark := ""
		if _, ok := errors[f.ID]; ok {
			errMark = errStyle.Render("● ")
		}

		name := nameStyle.Render(prefix + errMark + truncate(f.Name, nameW))
		counter := ""
		if f.UnreadCount > 0 {
			counter = counterStyle.Render(fmt.Sprintf("%d", f.UnreadCount))
		}

		line := lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(width-4).Render(name),
			counter,
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return framePane(b.String(), active, width, height)
}

func framePane(content string, active bool, width, height int) string {
	style := paneInactive
	if active {
		style = paneActive
	}
	return style.Width(width).Height(height).Render(content)
}

func truncate(s string, max int) string {
	if max <= 1 {
		return "…"
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
