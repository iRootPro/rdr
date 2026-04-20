package ui

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/i18n"
)

// readerMaxContentWidth caps the reader body so long articles stay
// readable on wide terminals. Contents wider than this are rendered at
// the cap and horizontally centered inside the available pane width.
const readerMaxContentWidth = 85

var (
	readerTitleLarge lipgloss.Style
	readerSource     lipgloss.Style
	readerMetaMuted  lipgloss.Style
	readerMetaURL    lipgloss.Style
	readerBody       lipgloss.Style
	readerEmptyBox   lipgloss.Style
	readerEmptyCTA   lipgloss.Style
)

func init() {
	rebuildReaderStyles()
	registerStyleRebuild(rebuildReaderStyles)
}

func rebuildReaderStyles() {
	readerTitleLarge = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true)

	readerSource = lipgloss.NewStyle().Foreground(colorGreen).Background(colorBG)

	readerMetaMuted = lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)

	readerMetaURL = lipgloss.NewStyle().Foreground(colorTeal).Background(colorBG)

	readerBody = lipgloss.NewStyle().Foreground(colorText).Background(colorBG)

	readerEmptyBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Background(colorBG).
		Padding(1, 3).
		Foreground(colorText)

	readerEmptyCTA = lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true)
}

// buildReaderContent is the entry point used by the model. It delegates
// to layoutReader and keeps the original signature so every call-site in
// model.go / search.go stays unchanged.
func buildReaderContent(a db.Article, feedName, feedURL string, width int, showImages bool, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	return layoutReader(a, feedName, feedURL, width, showImages, tr)
}

// layoutReader renders the full reader string and, when the available
// outer width exceeds readerMaxContentWidth, horizontally centres it by
// left-padding every line. Narrow terminals fall through untouched.
func layoutReader(a db.Article, feedName, feedURL string, outerWidth int, showImages bool, tr *i18n.Strings) string {
	contentW := outerWidth
	if contentW > readerMaxContentWidth {
		contentW = readerMaxContentWidth
	}
	if contentW < 20 {
		contentW = 20
	}

	body := renderReaderBody(a, feedName, feedURL, contentW, showImages, tr)

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
	return fillBackground(body, outerWidth)
}

// fillBackground ensures every line has the theme background across its
// full width. It prepends the ANSI bg-set sequence so that any mid-line
// resets (from glamour, lipgloss, etc.) still start on the right bg,
// and pads the line to the target width.
func fillBackground(content string, width int) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = paintLineBG(line, width)
	}
	return strings.Join(lines, "\n")
}

// renderReaderBody builds the reader content at a fixed content width.
// All typography decisions live here so layoutReader can stay a thin
// centring wrapper.
func renderReaderBody(a db.Article, feedName, feedURL string, contentW int, showImages bool, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
	var b strings.Builder

	// Title block — a blank line above plus bold accent gives the title
	// more visual weight than the URL below.
	b.WriteByte('\n')
	b.WriteString(readerTitleLarge.Render(a.Title))
	b.WriteString("\n\n")

	// Meta: source + time + (optional) reading time, divided by
	// double-padded middots.
	metaParts := []string{readerSource.Render(feedIcon(feedURL, feedName) + " " + feedName)}
	if ago := timeAgo(a.PublishedAt, tr); ago != "" {
		metaParts = append(metaParts, readerMetaMuted.Render(ago))
	}
	if rt := readingTime(a, tr); rt != "" {
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

	// Body vs preview vs empty state.
	//
	// Precedence:
	//   1. cached_body — full article fetched via 'f'. Render via glamour.
	//   2. description / content — most RSS items ship with a readable
	//      summary. Render that inline + a subtle footer inviting the
	//      user to pull the full article.
	//   3. nothing useful — fall back to the big empty-state card
	//      (HN-style stubs where Content is just metadata).
	switch {
	case a.CachedBody != "":
		md := sanitizeArticleMarkdown(a.CachedBody, showImages)
		if rendered, err := renderMarkdown(md, contentW); err == nil {
			b.WriteString(rendered)
		} else {
			b.WriteString(readerBody.Render(wrap(stripHTML(md), contentW)))
		}
	case hasReadablePreview(a):
		preview := articlePreviewText(a)
		// Try glamour (description is often HTML-ish Markdown) and fall
		// back to stripHTML + manual wrap if that fails.
		if rendered, err := renderMarkdown(preview, contentW); err == nil {
			b.WriteString(rendered)
		} else {
			b.WriteString(readerBody.Render(wrap(stripHTML(preview), contentW)))
		}
		b.WriteString("\n\n")
		b.WriteString(readerMetaMuted.Render(strings.Repeat("─", contentW)))
		b.WriteString("\n")
		hint := readerEmptyCTA.Render(tr.Reader.PressF) +
			readerMetaMuted.Render(tr.Reader.LoadFullSuffix)
		b.WriteString(hint)
	default:
		b.WriteString(renderEmptyReader(a, contentW, tr))
	}
	return b.String()
}

// hasReadablePreview decides whether an article's description / content
// has enough text to show as the main reader body. We count non-HTML
// words; anything under 20 words is treated as metadata-only (HN stubs)
// and routed to the empty-state card instead.
func hasReadablePreview(a db.Article) bool {
	return len(strings.Fields(stripHTML(articlePreviewText(a)))) >= 20
}

// articlePreviewText picks the richest preview field available. Prefers
// Content (often longer on Atom feeds) then falls back to Description.
func articlePreviewText(a db.Article) string {
	if c := strings.TrimSpace(a.Content); c != "" {
		return c
	}
	return strings.TrimSpace(a.Description)
}

// renderEmptyReader draws a centred rounded-border card prompting the
// user to press [f] to pull the article body. Any fallback metadata
// (feed items with just "Points: N" stubs in Content) is shown dim and
// centred below the box.
func renderEmptyReader(a db.Article, contentW int, tr *i18n.Strings) string {
	boxW := 50
	if contentW < boxW+6 {
		boxW = contentW - 6
	}
	if boxW < 20 {
		boxW = 20
	}

	inside := strings.Join([]string{
		"",
		tr.Reader.EmptyHeadline,
		"",
		readerEmptyCTA.Render(tr.Reader.PressF) + tr.Reader.PressFSuffix,
		tr.Reader.PressFSuffix2,
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
func readingTime(a db.Article, tr *i18n.Strings) string {
	if tr == nil {
		tr = i18n.For(i18n.EN)
	}
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
	return fmt.Sprintf(tr.Reader.MinReadFmt, mins)
}

func renderMarkdown(md string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyleConfig()),
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

func glamourStyleConfig() ansi.StyleConfig {
	var cfg ansi.StyleConfig
	if glamourStyle == "light" {
		cfg = styles.LightStyleConfig
	} else {
		cfg = styles.DarkStyleConfig
	}
	bg := string(colorBG)
	cfg.Document.Margin = uintPtr(0)
	cfg.Document.StylePrimitive.BackgroundColor = &bg
	cfg.Paragraph.StylePrimitive.BackgroundColor = &bg
	cfg.BlockQuote.StylePrimitive.BackgroundColor = &bg
	cfg.List.StyleBlock.StylePrimitive.BackgroundColor = &bg
	cfg.Item.BackgroundColor = &bg
	cfg.Heading.StylePrimitive.BackgroundColor = &bg
	cfg.H1.StylePrimitive.BackgroundColor = &bg
	cfg.H2.StylePrimitive.BackgroundColor = &bg
	cfg.H3.StylePrimitive.BackgroundColor = &bg
	cfg.H4.StylePrimitive.BackgroundColor = &bg
	cfg.H5.StylePrimitive.BackgroundColor = &bg
	cfg.H6.StylePrimitive.BackgroundColor = &bg
	cfg.Emph.BackgroundColor = &bg
	cfg.Strong.BackgroundColor = &bg
	cfg.Link.BackgroundColor = &bg
	cfg.LinkText.BackgroundColor = &bg
	cfg.HorizontalRule.BackgroundColor = &bg
	cfg.Table.StyleBlock.StylePrimitive.BackgroundColor = &bg

	// Code: distinct foreground + alt background so inline `code` and fenced
	// blocks visibly stand out from body text on every theme.
	altBG := string(colorAltBG)
	accent := string(colorAccent)
	cfg.Code.StylePrimitive.BackgroundColor = &altBG
	cfg.Code.StylePrimitive.Color = &accent
	// Pad inline code with a space on each side so the background isn't flush
	// against neighbouring letters — gives the highlighted run some breathing room.
	cfg.Code.StylePrimitive.Prefix = " "
	cfg.Code.StylePrimitive.Suffix = " "

	indent := uint(4)
	cfg.CodeBlock.StyleBlock.Indent = &indent
	cfg.CodeBlock.StyleBlock.StylePrimitive.BackgroundColor = &altBG
	if cfg.CodeBlock.Chroma != nil {
		cfg.CodeBlock.Chroma.Background = ansi.StylePrimitive{
			BackgroundColor: &altBG,
		}
		overrideChromaBG(cfg.CodeBlock.Chroma, &altBG)
	}
	return cfg
}

// overrideChromaBG forces every syntax-token sub-style in the Chroma config to
// share the same BackgroundColor. Without this, glamour's default chroma styles
// keep their own backgrounds (or none), which renders as horizontal stripes
// across the code block: tokens on one bg, the spaces between them on another.
func overrideChromaBG(c *ansi.Chroma, bg *string) {
	v := reflect.ValueOf(c).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() != reflect.Struct {
			continue
		}
		bgField := f.FieldByName("BackgroundColor")
		if bgField.IsValid() && bgField.CanSet() {
			bgField.Set(reflect.ValueOf(bg))
		}
	}
}

func uintPtr(u uint) *uint { return &u }

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

func readerFeedURL(feeds []db.Feed, feedID int64) string {
	for _, f := range feeds {
		if f.ID == feedID {
			return f.URL
		}
	}
	return ""
}
