package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// articleLink is a single link extracted from an article body.
type articleLink struct {
	Text string
	URL  string
}

var (
	// Inline markdown link syntax: [text](url). Any kind of URL is accepted,
	// we de-dupe and filter later.
	reMarkdownLink = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)[^)]*\)`)

	// Bare http(s) URLs that aren't inside markdown link brackets. Trailing
	// punctuation (commas, periods, closing parens) is trimmed below.
	reBareURL = regexp.MustCompile(`https?://[^\s<>"']+`)
)

// extractLinks pulls every unique link out of an article markdown body. It
// deliberately skips images (![alt](url)) by stripping them first so CDN
// junk doesn't flood the picker. Bare URLs that look like images are also
// filtered — the user opens images with the article's own 'o' binding.
func extractLinks(md string) []articleLink {
	// Remove image syntax first so it doesn't get picked up as a link.
	md = reMarkdownImage.ReplaceAllString(md, "")

	var out []articleLink
	seen := make(map[string]bool)

	// Markdown-style links.
	for _, m := range reMarkdownLink.FindAllStringSubmatch(md, -1) {
		text := strings.TrimSpace(m[1])
		url := strings.TrimSpace(m[2])
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		out = append(out, articleLink{Text: text, URL: url})
	}

	// Bare URLs (not already captured as markdown links).
	for _, raw := range reBareURL.FindAllString(md, -1) {
		// Trim stray trailing punctuation that often follows a URL in prose.
		url := strings.TrimRight(raw, ".,;:)\"'")
		if url == "" || seen[url] {
			continue
		}
		// Skip what's obviously an image CDN URL.
		if looksLikeImageURL(url) {
			continue
		}
		seen[url] = true
		out = append(out, articleLink{Text: url, URL: url})
	}
	return out
}

func looksLikeImageURL(url string) bool {
	lower := strings.ToLower(url)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// updateLinks handles keystrokes while the link picker is open.
func (m Model) updateLinks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.focus = focusReader
		m.links = nil
		return m, nil
	case key.Matches(msg, m.keys.Quit) && msg.String() == "ctrl+c":
		return m, tea.Quit
	case key.Matches(msg, m.keys.Down):
		if m.linksSel < len(m.links)-1 {
			m.linksSel++
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.linksSel > 0 {
			m.linksSel--
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.linksSel = 0
		return m, nil
	case key.Matches(msg, m.keys.Bottom):
		if len(m.links) > 0 {
			m.linksSel = len(m.links) - 1
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if len(m.links) == 0 || m.linksSel < 0 || m.linksSel >= len(m.links) {
			return m, nil
		}
		url := m.links[m.linksSel].URL
		if err := openInBrowser(url); err != nil {
			m.err = err
		} else {
			m.status = "opened: " + truncate(url, 40)
		}
		m.focus = focusReader
		m.links = nil
		return m, nil
	}
	return m, nil
}

// openLinkPickerOnCurrent populates m.links from the currently open article
// and switches focus to the link picker. No-op if no article is loaded or
// the article has no links.
func (m Model) openLinkPickerOnCurrent() (tea.Model, tea.Cmd) {
	if m.readerArt == nil {
		return m, nil
	}
	source := m.readerArt.CachedBody
	if source == "" {
		source = m.readerArt.Content
	}
	if source == "" {
		source = m.readerArt.Description
	}
	links := extractLinks(source)
	if len(links) == 0 {
		m.status = "no links in article"
		return m, nil
	}
	m.links = links
	m.linksSel = 0
	m.focus = focusLinks
	return m, nil
}

var (
	linkPickerTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 0, 1, 0)

	linkPickerText = lipgloss.NewStyle().Foreground(colorText)
	linkPickerURL  = lipgloss.NewStyle().Foreground(colorTeal).Underline(true)
)

func renderLinkPicker(m Model, width, height int) string {
	innerW := width - 4
	if innerW < 20 {
		innerW = 20
	}
	innerH := height - 2
	if innerH < 6 {
		innerH = 6
	}

	var b strings.Builder
	b.WriteString(linkPickerTitle.Render(fmt.Sprintf("Links · %d", len(m.links))))
	b.WriteString("\n")

	if len(m.links) == 0 {
		b.WriteString(readStyle.Render("(no links)"))
		return paneActive.Width(width - 2).Height(innerH).Render(b.String())
	}

	// Reserve space for two blank lines at bottom + help hint (3 rows).
	listRows := innerH - 3 - 2 // minus title(1)+padBelow(1)+hint(1)
	if listRows < 1 {
		listRows = 1
	}

	// Simple scroll: keep selection within [scroll, scroll+listRows).
	scroll := 0
	if m.linksSel >= listRows {
		scroll = m.linksSel - listRows + 1
	}
	end := scroll + listRows
	if end > len(m.links) {
		end = len(m.links)
	}

	// Text column ~ 40% of inner width, URL takes the rest.
	textW := innerW / 3
	if textW < 10 {
		textW = 10
	}
	urlW := innerW - textW - 4 // 4 for prefix "› " and spacing
	if urlW < 10 {
		urlW = 10
	}

	for i := scroll; i < end; i++ {
		l := m.links[i]
		prefix := "  "
		textStyle := linkPickerText
		if i == m.linksSel {
			prefix = "› "
			textStyle = itemSelected
		}
		text := textStyle.Render(prefix + truncate(l.Text, textW))
		url := linkPickerURL.Render(truncate(l.URL, urlW))
		textCell := lipgloss.NewStyle().Width(textW + 2).Render(text)
		line := lipgloss.JoinHorizontal(lipgloss.Top, textCell, url)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Pad to fixed list height so hint stays at bottom.
	for i := end - scroll; i < listRows; i++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(searchHint.Render("↑↓ navigate · enter open · esc back"))

	return paneActive.Width(width - 2).Height(innerH).Render(b.String())
}
