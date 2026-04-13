package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

var (
	readerTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	readerMeta = lipgloss.NewStyle().
			Foreground(colorMuted)

	readerSource = lipgloss.NewStyle().Foreground(colorGreen)
	readerURL    = lipgloss.NewStyle().Foreground(colorTeal).Underline(true)
	readerBody   = lipgloss.NewStyle().Foreground(colorText)

	readerHint = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)
)

func buildReaderContent(a db.Article, feedName string, width int) string {
	var b strings.Builder

	b.WriteString(readerTitle.Render(a.Title))
	b.WriteString("\n")

	metaParts := []string{readerSource.Render(feedName)}
	if ago := timeAgo(a.PublishedAt); ago != "" {
		metaParts = append(metaParts, readerMeta.Render(ago))
	}
	if a.URL != "" {
		metaParts = append(metaParts, readerURL.Render(a.URL))
	}
	b.WriteString(strings.Join(metaParts, readerMeta.Render(" · ")))
	b.WriteString("\n")

	if width < 1 {
		width = 1
	}
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n\n")

	body := stripHTML(a.Content)
	if body == "" {
		body = stripHTML(a.Description)
	}
	if body == "" {
		body = "(no content)"
	}
	b.WriteString(readerBody.Render(wrap(body, width)))
	b.WriteString("\n\n")

	b.WriteString(readerHint.Render("[f] load full article (Phase 2)"))
	return b.String()
}

var (
	reTag     = regexp.MustCompile(`<[^>]+>`)
	reEntity  = regexp.MustCompile(`&[a-zA-Z#0-9]+;`)
	reSpaces  = regexp.MustCompile(`[ \t]+`)
	reNewline = regexp.MustCompile(`\n{3,}`)
)

// stripHTML removes tags and collapses whitespace — Phase 1 MVP.
// Phase 2 replaces this with html-to-markdown + glamour.
func stripHTML(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "</p>", "\n\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = reTag.ReplaceAllString(s, "")
	s = reEntity.ReplaceAllStringFunc(s, decodeEntity)
	s = reSpaces.ReplaceAllString(s, " ")
	s = reNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func decodeEntity(e string) string {
	switch e {
	case "&amp;":
		return "&"
	case "&lt;":
		return "<"
	case "&gt;":
		return ">"
	case "&quot;":
		return `"`
	case "&#39;", "&apos;":
		return "'"
	case "&nbsp;":
		return " "
	case "&mdash;":
		return "—"
	case "&ndash;":
		return "–"
	case "&hellip;":
		return "…"
	}
	return e
}

func wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	for i, para := range strings.Split(s, "\n") {
		if i > 0 {
			out.WriteByte('\n')
		}
		if para == "" {
			continue
		}
		words := strings.Fields(para)
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
				continue
			}
			if len(line)+1+len(w) > width {
				out.WriteString(line)
				out.WriteByte('\n')
				line = w
				continue
			}
			line += " " + w
		}
		if line != "" {
			out.WriteString(line)
		}
	}
	return out.String()
}

func readerFeedName(feeds []db.Feed, feedID int64) string {
	for _, f := range feeds {
		if f.ID == feedID {
			return f.Name
		}
	}
	return ""
}
