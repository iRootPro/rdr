package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpSectionTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Padding(0, 0, 0, 0)

	helpKey = lipgloss.NewStyle().
		Foreground(colorTeal).
		Bold(true)

	helpDesc = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpScreenTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)
)

// updateHelp handles keystrokes while the full-screen help overlay is
// open. Anything except Back / Help is swallowed so stray keys don't
// leak into the previously-focused pane.
func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Help):
		m.focus = m.helpPrev
		return m, nil
	case key.Matches(msg, m.keys.Quit) && msg.String() == "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// renderHelpScreen paints the full-screen help overlay using the sections
// returned by fullHelpFor for the focus that was active when the user
// pressed '?'. Read-only — no scrolling for MVP.
func renderHelpScreen(m Model, width, height int) string {
	innerH := height - 2
	if innerH < 6 {
		innerH = 6
	}
	innerW := width - 4
	if innerW < 30 {
		innerW = 30
	}

	sections := fullHelpFor(m.helpPrev)

	var b strings.Builder
	b.WriteString(helpScreenTitle.Render(fmt.Sprintf("rdr · help · %s", focusLabel(m.helpPrev))))
	b.WriteString("\n")

	// Fixed key column width so descriptions line up across sections.
	const keyCol = 22
	for si, sec := range sections {
		b.WriteString(helpSectionTitle.Render(sec.Title))
		b.WriteString("\n")
		for _, e := range sec.Entries {
			keys := padRight(e.Keys, keyCol)
			line := "  " + helpKey.Render(keys) + helpDesc.Render(e.Desc)
			b.WriteString(line)
			b.WriteString("\n")
		}
		if si < len(sections)-1 {
			b.WriteString("\n")
		}
	}

	// Footer hint — fall-through to bottom of the pane.
	b.WriteString("\n")
	b.WriteString(searchHint.Render("esc close · ? toggle"))

	return paneActive.Width(width - 2).Height(innerH).Render(b.String())
}

// padRight right-pads s with spaces so its visible width is n. Used for
// the fixed key column in the help screen. Truncates with "…" if the
// input is already wider than n.
func padRight(s string, n int) string {
	w := lipgloss.Width(s)
	if w >= n {
		if n <= 1 {
			return "…"
		}
		// Too wide — hard-truncate with ellipsis.
		runes := []rune(s)
		if len(runes) <= n-1 {
			return s
		}
		return string(runes[:n-1]) + "…"
	}
	return s + strings.Repeat(" ", n-w)
}
