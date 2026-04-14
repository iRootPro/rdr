package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
)

// readerMaxContentWidth caps the reader body so long articles stay
// readable on wide terminals. Contents wider than this are rendered at
// the cap and horizontally centered inside the available pane width.
const readerMaxContentWidth = 85

var (
	readerTitleLarge = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	readerSource = lipgloss.NewStyle().Foreground(colorGreen)

	readerMetaMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	// URL on its own line — teal but without the loud underline that
	// previously competed with the title for visual weight.
	readerMetaURL = lipgloss.NewStyle().Foreground(colorTeal)

	readerBody = lipgloss.NewStyle().Foreground(colorText)

	readerEmptyBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 3).
			Foreground(colorText)

	readerEmptyCTA = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
)

// buildReaderContent is the entry point used by the model. It delegates
// to layoutReader and keeps the original signature so every call-site in
// model.go / search.go stays unchanged.
func buildReaderContent(a db.Article, feedName string, width int, showImages bool) string {
	return layoutReader(a, feedName, width, showImages)
}

// layoutReader renders the full reader string and, when the available
// outer width exceeds readerMaxContentWidth, horizontally centres it by
// left-padding every line. Narrow terminals fall through untouched.
func layoutReader(a db.Article, feedName string, outerWidth int, showImages bool) string {
	contentW := outerWidth
	if contentW > readerMaxContentWidth {
		contentW = readerMaxContentWidth
	}
	if contentW < 20 {
		contentW = 20
	}

	body := renderReaderBody(a, feedName, contentW, showImages)

	if outerWidth > contentW {
		indent := (outerWidth - contentW) / 2
		if indent < 0 {
			indent = 0
		}
		pad := strings.Repeat(" ", indent)
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		body = strings.Join(lines, "\n")
	}
	return body
}

// renderReaderBody builds the reader content at a fixed content width.
// All typography decisions live here so layoutReader can stay a thin
// centring wrapper.
func renderReaderBody(a db.Article, feedName string, contentW int, showImages bool) string {
	var b strings.Builder

	// Title block — a blank line above plus bold accent gives the title
	// more visual weight than the URL below.
	b.WriteByte('\n')
	b.WriteString(readerTitleLarge.Render(a.Title))
	b.WriteString("\n\n")

	// Meta: source + time + (optional) reading time, divided by
	// double-padded middots.
	metaParts := []string{readerSource.Render(feedName)}
	if ago := timeAgo(a.PublishedAt); ago != "" {
		metaParts = append(metaParts, readerMetaMuted.Render(ago))
	}
	if rt := readingTime(a); rt != "" {
		metaParts = append(metaParts, readerMetaMuted.Render(rt))
	}
	b.WriteString(strings.Join(metaParts, readerMetaMuted.Render("  ·  ")))
	b.WriteString("\n")

	// URL on its own line so it doesn't elbow the title; truncated if
	// longer than the content column.
	if a.URL != "" {
		url := a.URL
		if lipgloss.Width(url) > contentW {
			url = truncate(url, contentW-1)
		}
		b.WriteString(readerMetaURL.Render(url))
		b.WriteString("\n")
	}
	b.WriteByte('\n')

	// Content-width divider (not pane-width).
	b.WriteString(readerMetaMuted.Render(strings.Repeat("─", contentW)))
	b.WriteString("\n\n")

	// Body vs empty state.
	if a.CachedBody != "" {
		md := sanitizeArticleMarkdown(a.CachedBody, showImages)
		if rendered, err := renderMarkdown(md, contentW); err == nil {
			b.WriteString(rendered)
		} else {
			b.WriteString(readerBody.Render(wrap(stripHTML(md), contentW)))
		}
	} else {
		b.WriteString(renderEmptyReader(a, contentW))
	}
	return b.String()
}

// renderEmptyReader draws a centred rounded-border card prompting the
// user to press [f] to pull the article body. Any fallback metadata
// (feed items with just "Points: N" stubs in Content) is shown dim and
// centred below the box.
func renderEmptyReader(a db.Article, contentW int) string {
	boxW := 50
	if contentW < boxW+6 {
		boxW = contentW - 6
	}
	if boxW < 20 {
		boxW = 20
	}

	inside := strings.Join([]string{
		"",
		"This article has no full body loaded yet.",
		"",
		readerEmptyCTA.Render("Press [f]") + " to fetch & render it",
		"with the readability extractor.",
		"",
	}, "\n")
	box := readerEmptyBox.Width(boxW).Render(inside)

	// Centre box inside contentW. boxW excludes the 2-column border, so
	// the rendered box is boxW+2 wide.
	indent := (contentW - boxW - 2) / 2
	if indent < 0 {
		indent = 0
	}
	pad := strings.Repeat(" ", indent)
	boxLines := strings.Split(box, "\n")
	for i := range boxLines {
		boxLines[i] = pad + boxLines[i]
	}

	var b strings.Builder
	// Breathing room above the card.
	b.WriteString("\n\n")
	b.WriteString(strings.Join(boxLines, "\n"))
	b.WriteString("\n\n")

	// Fallback stub text (e.g. HN "Points: 42" lines) shown dim and
	// left-indented slightly so it visually hangs off the card.
	if stub := strings.TrimSpace(stripHTML(a.Content)); stub != "" {
		wrapped := wrap(stub, contentW-4)
		for _, line := range strings.Split(wrapped, "\n") {
			if line == "" {
				b.WriteByte('\n')
				continue
			}
			b.WriteString("  ")
			b.WriteString(readerMetaMuted.Render(line))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

var (
	// Markdown image syntax ![alt](url) — optionally with "title" after
	// the URL. We strip these wholesale for a cleaner reader view.
	reMarkdownImage = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)

	// A bare URL on its own line (optionally inside <angle> brackets), typical
	// of html-to-markdown output for standalone images that didn't become
	// proper image syntax. Matches http(s) URLs ending in common image
	// extensions so we don't strip legitimate article links.
	reBareImageURL = regexp.MustCompile(`(?m)^\s*<?https?://\S+\.(?:png|jpe?g|gif|webp|svg)>?\s*$`)

	// Collapse 3+ consecutive newlines down to a paragraph break.
	reExtraBlankLines = regexp.MustCompile(`\n{3,}`)
)

// sanitizeArticleMarkdown cleans up markdown for reader rendering. When
// showImages is false, all image syntax and bare image URLs are removed.
// This is a pragmatic fix for articles where every paragraph is padded
// with CDN image links that glamour renders as noisy text.
func sanitizeArticleMarkdown(md string, showImages bool) string {
	if showImages {
		return md
	}
	md = reMarkdownImage.ReplaceAllString(md, "")
	md = reBareImageURL.ReplaceAllString(md, "")
	md = reExtraBlankLines.ReplaceAllString(md, "\n\n")
	return strings.TrimSpace(md)
}

// readingTime returns a human label like "5 min read" estimated at 200
// words per minute. Prefers the cached body; falls back to content then
// description. Returns "" when there's nothing to count (e.g. HN stubs).
func readingTime(a db.Article) string {
	source := a.CachedBody
	if source == "" {
		source = a.Content
	}
	if source == "" {
		source = a.Description
	}
	if source == "" {
		return ""
	}
	text := stripHTML(source)
	words := len(strings.Fields(text))
	if words < 20 {
		// Too short to be a real article body (HN metadata stubs, empty
		// descriptions). Skip the label rather than show "1 min read"
		// for every feed-list preview.
		return ""
	}
	mins := (words + 199) / 200 // round up
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("%d min read", mins)
}

func renderMarkdown(md string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(out, "\n"), nil
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
