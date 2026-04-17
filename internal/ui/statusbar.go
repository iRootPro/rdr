package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// segment is one chunk of a powerline status bar.
type segment struct {
	Icon string
	Text string
	FG   lipgloss.Color
	BG   lipgloss.Color
	Bold bool
}

// powerlineSep is the right-pointing triangle from the Powerline glyph set.
const powerlineSep = "\ue0b0"

// renderPowerline builds a powerline-style status bar from left-to-right
// segments. Each segment renders as " icon text " with its own fg/bg, then a
// separator glyph colored fg=currentBG, bg=nextBG.
func renderPowerline(segs []segment, width int) string {
	if len(segs) == 0 {
		return lipgloss.NewStyle().Background(colorBG).Width(width).Render("")
	}

	var b strings.Builder
	for i, s := range segs {
		style := lipgloss.NewStyle().Foreground(s.FG).Background(s.BG)
		if s.Bold {
			style = style.Bold(true)
		}
		text := " "
		if s.Icon != "" {
			text += s.Icon + " "
		}
		text += s.Text + " "
		b.WriteString(style.Render(text))

		// Separator: fg = current bg, bg = next bg (or theme bg).
		nextBG := colorBG
		if i+1 < len(segs) {
			nextBG = segs[i+1].BG
		}
		sepStyle := lipgloss.NewStyle().
			Foreground(s.BG).
			Background(nextBG)
		b.WriteString(sepStyle.Render(powerlineSep))
	}

	// Fill remaining width with theme background.
	rendered := b.String()
	if w := lipgloss.Width(rendered); w < width {
		rendered += lipgloss.NewStyle().Background(colorBG).
			Render(strings.Repeat(" ", width-w))
	}
	return rendered
}

// appSegment returns the branded "rdr" segment used in every status bar.
func appSegment() segment {
	return segment{
		Icon: "\U000f046b", // 󰑫
		Text: "rdr",
		FG:   colorBG,
		BG:   colorAccent,
		Bold: true,
	}
}

// statusSegments builds the segment list for the main two-pane view.
func statusSegments(status, filterLabel string, sortField string, sortReverse bool, zenMode bool) []segment {
	segs := []segment{
		appSegment(),
		{Text: status, FG: colorText, BG: colorAltBG},
		{Text: filterLabel, FG: colorText, BG: colorBorder},
	}
	if sortField != "date" || sortReverse {
		dir := "↓"
		if sortReverse {
			dir = "↑"
		}
		segs = append(segs, segment{
			Text: sortField + dir,
			FG:   colorText,
			BG:   colorBorder,
		})
	}
	if zenMode {
		segs = append(segs, segment{
			Text: "zen",
			FG:   colorBG,
			BG:   colorMuted,
		})
	}
	return segs
}

// readerSegments builds the segment list for the full-screen reader view.
// scrollPct is 0-100 (percentage through article), -1 if unknown.
func readerSegments(feedName, articleTitle string, maxTitleW int, scrollPct int) []segment {
	if len(articleTitle) > maxTitleW && maxTitleW > 3 {
		articleTitle = articleTitle[:maxTitleW-1] + "…"
	}
	segs := []segment{
		appSegment(),
		{Text: feedName, FG: colorBG, BG: colorGreen, Bold: true},
		{Text: articleTitle, FG: colorText, BG: colorAltBG},
	}
	if scrollPct >= 0 {
		var label string
		switch {
		case scrollPct <= 0:
			label = "TOP"
		case scrollPct >= 100:
			label = "BOT"
		default:
			label = fmt.Sprintf("%d%%", scrollPct)
		}
		segs = append(segs, segment{Text: label, FG: colorBG, BG: colorMuted, Bold: true})
	}
	return segs
}
