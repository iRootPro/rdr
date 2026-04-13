package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

var (
	settingsTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)

	settingsKeyHint = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	settingsURL = lipgloss.NewStyle().
			Foreground(colorTeal)
)

func renderSettings(feeds []db.Feed, selected int, mode settingsMode, input string, width, height int) string {
	var b strings.Builder
	b.WriteString(settingsTitle.Render("Settings · Feeds"))
	b.WriteString("\n\n")

	switch mode {
	case smAddName:
		b.WriteString("New feed name:\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to continue · esc to cancel"))
	case smAddURL:
		b.WriteString("New feed URL:\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to save · esc to cancel"))
	case smRename:
		b.WriteString("Rename feed:\n\n")
		b.WriteString(input)
		b.WriteString("\n\n")
		b.WriteString(settingsKeyHint.Render("enter to save · esc to cancel"))
	default: // smList
		if len(feeds) == 0 {
			b.WriteString(readStyle.Render("(no feeds) — press a to add"))
		} else {
			for i, f := range feeds {
				prefix := "  "
				nameStyle := lipgloss.NewStyle().Foreground(colorText)
				if i == selected {
					prefix = "› "
					nameStyle = itemSelected
				}
				line := fmt.Sprintf("%s%s  %s",
					prefix,
					nameStyle.Render(f.Name),
					settingsURL.Render(f.URL),
				)
				b.WriteString(line)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(settingsKeyHint.Render("a add · d delete · e rename · esc close"))
	}

	return paneActive.Width(width - 2).Height(height - 2).Render(b.String())
}
