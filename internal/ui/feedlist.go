package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/i18n"
)

// renderFeedList draws the unified feeds pane: smart folders at the top with
// an icon prefix, then a subtle separator, then regular feeds with unread
// counters. Selection highlights the currently active row.
func renderFeedList(entries []feedEntry, selected int, active bool, width, height int, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	title := "\U000f046b " + tr.Feeds.PaneTitle // 󰑫
	var b strings.Builder

	if len(entries) == 0 {
		b.WriteString(readStyle.Render(tr.Feeds.NoFeeds))
		return framePaneWithTitle(b.String(), title, active, width, height)
	}

	// Inner text area = width - 2 (1-cell padding each side inside border).
	inner := width - 2
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

	// Visual break: a subtle separator sits between the smart-folders
	// section and the first category (or the first uncategorised feed
	// when there are no categories). That groups the pane into two
	// obvious sections without adding noise inside either.
	firstBreakIdx := -1
	hasFolder := false
	for i, e := range entries {
		if e.Kind == entryFolder {
			hasFolder = true
			continue
		}
		if firstBreakIdx < 0 {
			firstBreakIdx = i
			break
		}
	}
	showSeparator := hasFolder && firstBreakIdx > 0

	rowsBudget := listVisibleRows(height)
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

		if showSeparator && i == firstBreakIdx {
			sep := lipgloss.NewStyle().
				Foreground(colorBorder).
				Background(colorBG).
				Render(strings.Repeat("─", nameCellW+counterCol))
			b.WriteString(sep)
			b.WriteString("\n")
			rowsUsed++
		}

		rowBG := colorBG
		if i == selected && active {
			rowBG = colorAltBG
		}

		prefix := "  "
		nameStyle := lipgloss.NewStyle().Foreground(colorText).Background(rowBG)
		if i == selected {
			prefix = "› "
			if active {
				nameStyle = itemSelected.Background(rowBG)
			} else {
				nameStyle = itemSelectedInactive.Background(rowBG)
			}
		}

		var icon string
		iconCells := 0
		switch e.Kind {
		case entryFolder:
			icon = lipgloss.NewStyle().Foreground(colorTeal).Background(rowBG).Render("◉ ")
			iconCells = 2
		case entryCategory:
			marker := "▼ "
			if e.Collapsed {
				marker = "▶ "
			}
			icon = lipgloss.NewStyle().Foreground(colorAccent).Background(rowBG).Bold(true).Render(marker)
			iconCells = 2
			if i != selected {
				nameStyle = lipgloss.NewStyle().Foreground(colorAccent).Background(rowBG).Bold(true)
			}
		case entryFeed:
			fi := feedIcon(e.FeedURL, e.Name)
			icon = lipgloss.NewStyle().Foreground(colorMuted).Background(rowBG).Render("  "+fi+" ")
			iconCells = 4
			if e.HasError {
				icon = lipgloss.NewStyle().Foreground(colorRed).Background(rowBG).Render("  "+fi+" ")
				iconCells = 4
			}
		}

		nameBudget := nameCellW - 2 - iconCells
		if nameBudget < 1 {
			nameBudget = 1
		}
		prefixRendered := lipgloss.NewStyle().Background(rowBG).Render(prefix)
		nameText := nameStyle.Render(truncate(e.Name, nameBudget))
		name := prefixRendered + icon + nameText

		counter := ""
		if e.UnreadCount > 0 {
			counter = lipgloss.NewStyle().Foreground(colorGreen).Background(rowBG).Render(fmt.Sprintf("%d", e.UnreadCount))
		}

		nameCellStyle := lipgloss.NewStyle().Width(nameCellW).MaxWidth(nameCellW).Background(rowBG)
		counterCellStyle := lipgloss.NewStyle().Width(counterCol).MaxWidth(counterCol).Align(lipgloss.Right).Background(rowBG)
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

	return framePaneWithTitle(b.String(), title, active, width, height)
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
	// Title now lives in the border, so overhead = 0 (border is outside height).
	n := paneHeight
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

// framePaneWithTitle renders content inside a bordered pane with the title
// embedded in the top border line, lazygit-style:
//
//	╭─ 󰑫 Ленты ──────────╮
//	│ content               │
//	╰───────────────────────╯
// framePaneWithTitle draws a bordered pane. width/height are the CONTENT
// dimensions (same semantics as lipgloss Width before). The function adds
// border chars (+2 cols, +2 rows) and 1-cell padding on each side.
func framePaneWithTitle(content, title string, active bool, width, height int) string {
	borderColor := colorBorder
	if active {
		borderColor = colorAccent
	}
	bs := lipgloss.NewStyle().Foreground(borderColor).Background(colorBG)
	ts := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	pad := lipgloss.NewStyle().Background(colorBG)

	// innerW = width of the horizontal dash run between ╭ and ╮.
	// This equals the content width passed in (matches old lipgloss Width
	// semantics where Width = content area, border added on top).
	innerW := width
	if innerW < 4 {
		innerW = 4
	}

	// ── top border ──
	var top string
	if title != "" {
		titleStr := " " + title + " "
		titleCells := lipgloss.Width(titleStr)
		dashesAfter := innerW - 1 - titleCells
		if dashesAfter < 0 {
			dashesAfter = 0
		}
		top = bs.Render("╭─") + ts.Render(titleStr) +
			bs.Render(strings.Repeat("─", dashesAfter)+"╮")
	} else {
		top = bs.Render("╭" + strings.Repeat("─", innerW) + "╮")
	}

	// ── bottom border ──
	bottom := bs.Render("╰" + strings.Repeat("─", innerW) + "╯")

	// ── content lines ──
	// Each row: │ + 1 pad + content + 1 pad + │
	// So visible content per row = innerW - 2.
	contentW := innerW - 2
	if contentW < 1 {
		contentW = 1
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}

	border := bs.Render("│")
	space := pad.Render(" ")
	rows := make([]string, len(lines))
	for i, line := range lines {
		// Fill the line to contentW using paintLineBG which handles
		// ANSI resets properly and pads with theme background.
		filled := paintLineBG(line, contentW)
		rows[i] = border + space + filled + space + border
	}

	all := make([]string, 0, len(rows)+2)
	all = append(all, top)
	all = append(all, rows...)
	all = append(all, bottom)
	return strings.Join(all, "\n")
}

func truncate(s string, max int) string {
	if max <= 1 {
		return "…"
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	// Start from approximate position and shrink until it fits.
	hi := max - 1
	if hi > len(runes) {
		hi = len(runes)
	}
	for hi > 0 {
		candidate := string(runes[:hi]) + "…"
		if lipgloss.Width(candidate) <= max {
			return candidate
		}
		hi--
	}
	return "…"
}
