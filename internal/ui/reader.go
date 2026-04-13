package ui

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/kitty"
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

var reImageRef = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

type imageRef struct {
	alt string
	url string
}

// extractImages returns the markdown split into text chunks around image
// refs, plus the image refs themselves in document order.
// len(chunks) == len(images) + 1; chunks[i] precedes images[i].
func extractImages(md string) (chunks []string, images []imageRef) {
	last := 0
	for _, m := range reImageRef.FindAllStringSubmatchIndex(md, -1) {
		chunks = append(chunks, md[last:m[0]])
		images = append(images, imageRef{alt: md[m[2]:m[3]], url: md[m[4]:m[5]]})
		last = m[1]
	}
	chunks = append(chunks, md[last:])
	return chunks, images
}

// imageURLs returns just the URLs of every image ref in md.
func imageURLs(md string) []string {
	_, images := extractImages(md)
	out := make([]string, len(images))
	for i, im := range images {
		out[i] = im.url
	}
	return out
}

func imageID(url string) uint32 {
	sum := sha256.Sum256([]byte(url))
	return binary.BigEndian.Uint32(sum[:4])
}

func imageCells(data []byte, termWidth int) (cols, rows int) {
	maxCols := termWidth - 4
	if maxCols < 10 {
		maxCols = 10
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width == 0 {
		return maxCols, 10
	}
	cols = maxCols
	rows = cols * cfg.Height / cfg.Width / 2
	if rows < 1 {
		rows = 1
	}
	if rows > 30 {
		rows = 30
	}
	return cols, rows
}

func renderWithKittyImages(md string, width int, imageCache string) string {
	chunks, images := extractImages(md)

	var out strings.Builder
	for i, chunk := range chunks {
		if chunk != "" {
			if rendered, err := renderMarkdown(chunk, width); err == nil {
				out.WriteString(rendered)
			} else {
				out.WriteString(chunk)
			}
		}
		if i < len(images) {
			spec := images[i]
			path := filepath.Join(imageCache, feed.ImageFileName(spec.url))
			data, err := os.ReadFile(path)
			if err != nil {
				out.WriteString("\n")
				out.WriteString(readerHint.Render("[📷 " + spec.alt + "]"))
				out.WriteString("\n")
				continue
			}
			id := imageID(spec.url)
			cols, rows := imageCells(data, width)
			out.WriteString(kitty.Transmit(id, data))
			out.WriteString(kitty.Placement(id, cols, rows))
			out.WriteString("\n")
			out.WriteString(kitty.PlaceholderBlock(id, cols, rows))
			out.WriteString("\n")
		}
	}
	return out.String()
}

func buildReaderContent(a db.Article, feedName string, width int, kittyOn bool, imageCache string) string {
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

	if a.CachedBody != "" {
		if kittyOn && imageCache != "" {
			b.WriteString(renderWithKittyImages(a.CachedBody, width, imageCache))
		} else if rendered, err := renderMarkdown(a.CachedBody, width); err == nil {
			b.WriteString(rendered)
		} else {
			b.WriteString(readerBody.Render(wrap(stripHTML(a.CachedBody), width)))
		}
	} else {
		body := stripHTML(a.Content)
		if body == "" {
			body = stripHTML(a.Description)
		}
		if body == "" {
			body = "(no content)"
		}
		b.WriteString(readerBody.Render(wrap(body, width)))
		b.WriteString("\n\n")
		b.WriteString(readerHint.Render("[f] load full article"))
	}
	return b.String()
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
