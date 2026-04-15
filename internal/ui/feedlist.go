package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderFeedList draws the unified feeds pane: smart folders at the top with
// an icon prefix, then a subtle separator, then regular feeds with unread
// counters. Selection highlights the currently active row.
func renderFeedList(entries []feedEntry, selected int, active bool, width, height int) string {
	var b strings.Builder
	b.WriteString(paneTitle.Render("Feeds"))
	b.WriteString("\n")

	if len(entries) == 0 {
		b.WriteString(readStyle.Render("(no feeds)"))
		return framePane(b.String(), active, width, height)
	}

	// Inner text area after pane border (2) + padding (2) = width - 4.
	inner := width - 4
	if inner < 1 {
		inner = 1
	}

	// Reserve a fixed column for the counter based on both folder match
	// counts and feed unread counts. +1 gap between name and counter.
	counterW := maxEntryCounterWidth(entries)
	counterCol := 0
	if counterW > 0 {
		counterCol = counterW + 1
	}
	nameCellW := inner - counterCol
	if nameCellW < 1 {
		nameCellW = 1
	}

	// Find the first feed entry index so we can insert a visual separator
	// between folders and feeds. -1 means no feeds (don't render separator)
	// or no folders (also skip).
	firstFeedIdx := -1
	hasFolder := false
	for i, e := range entries {
		if e.Kind == entryFolder {
			hasFolder = true
		}
		if e.Kind == entryFeed && firstFeedIdx < 0 {
			firstFeedIdx = i
		}
	}
	showSeparator := hasFolder && firstFeedIdx > 0

	rowsBudget := listVisibleRows(height)
	// Reserve a row for the separator when we might draw one so the
	// content height stays constant across folder↔feed navigation.
	itemBudget := rowsBudget
	if showSeparator {
		itemBudget--
	}
	if itemBudget < 1 {
		itemBudget = 1
	}

	start, end := visibleWindow(len(entries), selected, itemBudget)
	rowsUsed := 0
	for i := start; i < end; i++ {
		e := entries[i]

		if showSeparator && i == firstFeedIdx {
			sep := lipgloss.NewStyle().
				Foreground(colorBorder).
				Render(strings.Repeat("─", nameCellW+counterCol))
			b.WriteString(sep)
			b.WriteString("\n")
			rowsUsed++
		}

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

		var icon string
		iconCells := 0
		switch e.Kind {
		case entryFolder:
			icon = lipgloss.NewStyle().Foreground(colorTeal).Render("◉ ")
			iconCells = 2
		case entryCategory:
			marker := "▼ "
			if e.Collapsed {
				marker = "▶ "
			}
			icon = lipgloss.NewStyle().Foreground(colorMuted).Bold(true).Render(marker)
			iconCells = 2
			// Category headers use a different name style so they read
			// like section headers rather than list items.
			if i != selected {
				nameStyle = lipgloss.NewStyle().Foreground(colorMuted).Bold(true)
			}
		case entryFeed:
			// Feeds live under a category — indent slightly for hierarchy.
			icon = "  "
			iconCells = 2
			if e.HasError {
				icon = errStyle.Render("● ")
			}
		}

		// prefix always 2 visible cells.
		nameBudget := nameCellW - 2 - iconCells
		if nameBudget < 1 {
			nameBudget = 1
		}
		name := nameStyle.Render(prefix + icon + truncate(e.Name, nameBudget))

		counter := ""
		if e.UnreadCount > 0 {
			counter = counterStyle.Render(fmt.Sprintf("%d", e.UnreadCount))
		}

		nameCellStyle := lipgloss.NewStyle().Width(nameCellW)
		counterCellStyle := lipgloss.NewStyle().Width(counterCol).Align(lipgloss.Right)
		if i == selected && active {
			nameCellStyle = nameCellStyle.Background(colorAltBG)
			counterCellStyle = counterCellStyle.Background(colorAltBG)
		}
		nameCell := nameCellStyle.Render(name)
		counterCell := counterCellStyle.Render(counter)

		line := lipgloss.JoinHorizontal(lipgloss.Top, nameCell, counterCell)
		b.WriteString(line)
		b.WriteString("\n")
		rowsUsed++
	}

	// Pad with blank lines to keep the content height stable regardless
	// of how many items / separators rendered. Prevents layout jumps
	// around the folder/feed boundary.
	for rowsUsed < rowsBudget {
		b.WriteString("\n")
		rowsUsed++
	}

	return framePane(b.String(), active, width, height)
}

func maxEntryCounterWidth(entries []feedEntry) int {
	w := 0
	for _, e := range entries {
		if e.UnreadCount <= 0 {
			continue
		}
		d := len(fmt.Sprintf("%d", e.UnreadCount))
		if d > w {
			w = d
		}
	}
	return w
}

func listVisibleRows(paneHeight int) int {
	// Pane border (2) + title row (1) + padding-below-title (1) = 4 rows overhead.
	n := paneHeight - 4
	if n < 1 {
		return 1
	}
	return n
}

func visibleWindow(total, selected, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}
	start = selected - maxVisible/2
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
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
