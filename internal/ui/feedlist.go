package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/iRootPro/rdr/internal/i18n"
)

type feedListInfo struct {
	Status     string
	AIProvider string
	Unread     int
	Feeds      int
}

// renderFeedList draws the unified feeds pane: smart folders at the top with
// an icon prefix, then a subtle separator, then regular feeds with unread
// counters. Selection highlights the currently active row.
func renderFeedList(entries []feedEntry, selected int, active bool, width, height int, tr *i18n.Strings, info feedListInfo) string {
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

	// Visual breaks: subtle separators sit between sections (Library →
	// smart folders → categories/feeds) so the user can tell them apart
	// at a glance. Sections are derived from entry Kind: entryLibrary,
	// entryFolder, then everything else (categories + feeds).
	sectionOf := func(k entryKind) int {
		switch k {
		case entryLibrary:
			return 0
		case entryFolder:
			return 1
		default:
			return 2
		}
	}
	separatorBefore := make(map[int]bool, 2)
	for i := 1; i < len(entries); i++ {
		if sectionOf(entries[i].Kind) != sectionOf(entries[i-1].Kind) {
			separatorBefore[i] = true
		}
	}

	rowsBudget := listVisibleRows(height)
	infoBlock := renderFeedInfoBlock(info, inner, tr)
	infoRows := 0
	if infoBlock != "" && rowsBudget >= 18 {
		infoRows = lipgloss.Height(infoBlock)
	}
	listRowsBudget := rowsBudget - infoRows
	if listRowsBudget < 1 {
		listRowsBudget = 1
		infoRows = 0
		infoBlock = ""
	}
	itemBudget := listRowsBudget - len(separatorBefore)
	if itemBudget < 1 {
		itemBudget = 1
	}

	start, end := visibleWindow(len(entries), selected, itemBudget)
	rowsUsed := 0
	for i := start; i < end; i++ {
		e := entries[i]

		if separatorBefore[i] {
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
		case entryLibrary:
			icon = lipgloss.NewStyle().Foreground(colorAccent).Background(rowBG).Render("\U000f02ba ")
			iconCells = 2
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
			icon = lipgloss.NewStyle().Foreground(colorMuted).Background(rowBG).Render("  " + fi + " ")
			iconCells = 4
			if e.HasError {
				icon = lipgloss.NewStyle().Foreground(colorRed).Background(rowBG).Render("  " + fi + " ")
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
	for rowsUsed < listRowsBudget {
		b.WriteString("\n")
		rowsUsed++
	}
	if infoRows > 0 {
		b.WriteString(infoBlock)
	}

	return framePaneWithTitle(b.String(), title, active, width, height)
}

func renderFeedInfoBlock(info feedListInfo, width int, tr *i18n.Strings) string {
	if width < 18 {
		return ""
	}
	provider := strings.TrimSpace(info.AIProvider)
	if provider == "" {
		provider = "openai"
	}
	status := strings.TrimSpace(info.Status)
	if status == "" {
		status = tr.Status.Ready
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG)
	valueStyle := lipgloss.NewStyle().Foreground(colorText).Background(colorBG).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(colorBorder).Background(colorBG)
	lineStyle := lipgloss.NewStyle().Width(width).MaxWidth(width).Background(colorBG)

	rows := []string{
		sepStyle.Render(strings.Repeat("─", width)),
		feedInfoLine(tr.Feeds.InfoUnread, fmt.Sprintf("%d", info.Unread), width, labelStyle, valueStyle, lineStyle),
		feedInfoLine(tr.Feeds.InfoFeeds, fmt.Sprintf("%d", info.Feeds), width, labelStyle, valueStyle, lineStyle),
		feedInfoLine(tr.Feeds.InfoAI, provider, width, labelStyle, valueStyle, lineStyle),
		feedInfoLine("", status, width, labelStyle, lipgloss.NewStyle().Foreground(colorMuted).Background(colorBG), lineStyle),
	}
	return strings.Join(rows, "\n")
}

func feedInfoLine(label, value string, width int, labelStyle, valueStyle, lineStyle lipgloss.Style) string {
	if label == "" {
		return lineStyle.Render("  " + valueStyle.Render(truncate(value, width-2)))
	}
	labelW := 8
	if width < 26 {
		labelW = 6
	}
	valueW := width - labelW - 3
	if valueW < 1 {
		valueW = 1
	}
	labelCell := lipgloss.NewStyle().Width(labelW).MaxWidth(labelW).Background(colorBG).Render(labelStyle.Render(label))
	valueCell := lipgloss.NewStyle().Width(valueW).MaxWidth(valueW).Background(colorBG).Render(valueStyle.Render(truncate(value, valueW)))
	return lineStyle.Render("  " + labelCell + " " + valueCell)
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
	// framePaneWithTitle adds top + bottom border rows outside height.
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
//
// Uses lipgloss Border for correct content clipping, then replaces the
// top border line with a custom title bar.
func framePaneWithTitle(content, title string, active bool, width, height int) string {
	borderColor := colorBorder
	if active {
		borderColor = colorAccent
	}
	bs := lipgloss.NewStyle().Foreground(borderColor).Background(colorBG)
	ts := lipgloss.NewStyle().Foreground(colorAccent).Background(colorBG).Bold(true)
	border := bs.Render("│")
	space := lipgloss.NewStyle().Background(colorBG).Render(" ")

	// Content area: width is the lipgloss-Width (includes padding).
	// We use 1-cell padding on each side, so text area = width - 2.
	contentW := width - 2
	if contentW < 1 {
		contentW = 1
	}

	// Use lipgloss to clip content to exact dimensions (no border).
	clipped := lipgloss.NewStyle().
		Width(contentW).
		MaxWidth(contentW).
		Height(height).
		MaxHeight(height).
		Render(content)

	// ── top border ──
	innerDash := width // dashes between ╭ and ╮
	var top string
	if title != "" {
		titleStr := " " + title + " "
		titleCells := lipgloss.Width(titleStr)
		dashesAfter := innerDash - 1 - titleCells
		if dashesAfter < 0 {
			dashesAfter = 0
		}
		top = bs.Render("╭─") + ts.Render(titleStr) +
			bs.Render(strings.Repeat("─", dashesAfter)+"╮")
	} else {
		top = bs.Render("╭" + strings.Repeat("─", innerDash) + "╮")
	}

	// ── bottom border ──
	bottom := bs.Render("╰" + strings.Repeat("─", innerDash) + "╯")

	// ── content rows ──
	lines := strings.Split(clipped, "\n")
	rows := make([]string, len(lines))
	for i, line := range lines {
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
	return ansi.Truncate(s, max-1, "") + "…"
}
