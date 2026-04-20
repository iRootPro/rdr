package ui

import (
	"context"
	"net/url"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iRootPro/rdr/internal/db"
	"github.com/iRootPro/rdr/internal/feed"
	"github.com/iRootPro/rdr/internal/rlog"
)

// currentEntryIsLibrary reports whether the cursor is on the Library
// section. Used by the help renderer to decide whether to surface the
// D (delete) binding — it would be misleading to advertise it from any
// other context, since the handler no-ops there.
func (m Model) currentEntryIsLibrary() bool {
	e, ok := m.currentEntry()
	return ok && e.Kind == entryLibrary
}

// clipboardURLOrEmpty returns the clipboard contents iff they look like
// an http(s) URL, else "". Used to pre-fill the AddURL modal so a fresh
// "copy → press B" round-trip is one keystroke.
func clipboardURLOrEmpty() string {
	s, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	s = strings.TrimSpace(s)
	if !looksLikeURL(s) {
		return ""
	}
	return s
}

func looksLikeURL(s string) bool {
	if s == "" {
		return false
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Host != ""
}

// openAddURLModal switches focus into the AddURL modal, pre-fills the
// input from the clipboard when it looks like a URL, and remembers the
// previous focus so esc/save can restore it.
func (m Model) openAddURLModal() (tea.Model, tea.Cmd) {
	m.addURLPrev = m.focus
	m.focus = focusAddURL
	pre := clipboardURLOrEmpty()
	m.addURLInput.SetValue(pre)
	m.addURLInput.CursorEnd()
	m.addURLInput.Focus()
	return m, textinput.Blink
}

// updateAddURL handles key events while the AddURL modal is active.
// Enter validates and saves; esc cancels; everything else is forwarded
// to the textinput model.
func (m Model) updateAddURL(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.focus = m.addURLPrev
		m.addURLInput.Blur()
		m.addURLInput.SetValue("")
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		raw := strings.TrimSpace(m.addURLInput.Value())
		if !looksLikeURL(raw) {
			return m, m.showToast(m.tr.Library.InvalidURL)
		}
		m.focus = m.addURLPrev
		m.addURLInput.Blur()
		m.addURLInput.SetValue("")
		return m, saveLibraryURLCmd(m.db, m.libraryFeedID, raw)
	}
	var cmd tea.Cmd
	m.addURLInput, cmd = m.addURLInput.Update(msg)
	return m, cmd
}

// saveLibraryURLCmd inserts (or no-ops on duplicate) a URL into the
// Library feed. On success it returns librarySavedMsg with the new id;
// on duplicate it still returns the existing id so the caller can re-
// fetch metadata if it wants.
func saveLibraryURLCmd(d *db.DB, libraryFeedID int64, raw string) tea.Cmd {
	return func() tea.Msg {
		placeholder := raw
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			placeholder = u.Host
		}
		id, _, err := d.SaveLibraryURL(libraryFeedID, raw, placeholder)
		if err != nil {
			return errMsg{err}
		}
		return librarySavedMsg{articleID: id, url: raw}
	}
}

// deleteCurrentLibraryArticle removes the article under the cursor when
// the active entry is the Library section. Silent no-op anywhere else
// so the hotkey doesn't accidentally nuke an RSS article.
func (m Model) deleteCurrentLibraryArticle() (tea.Model, tea.Cmd) {
	if m.focus != focusArticles && m.focus != focusReader {
		return m, nil
	}
	e, ok := m.currentEntry()
	if !ok || e.Kind != entryLibrary {
		return m, nil
	}
	if len(m.articles) == 0 {
		return m, nil
	}
	idx := m.selArt
	if m.focus == focusReader && m.readerArt != nil {
		for i, a := range m.articles {
			if a.ID == m.readerArt.ID {
				idx = i
				break
			}
		}
	}
	if idx < 0 || idx >= len(m.articles) {
		return m, nil
	}
	id := m.articles[idx].ID
	// Drop from local slices first so the UI reflects the change before
	// the DB roundtrip completes. Reload happens via libraryDeletedMsg.
	m.articles = append(m.articles[:idx], m.articles[idx+1:]...)
	if m.selArt >= len(m.articles) && m.selArt > 0 {
		m.selArt = len(m.articles) - 1
	}
	if m.focus == focusReader {
		m.focus = focusArticles
		m.readerArt = nil
	}
	return m, deleteLibraryArticleCmd(m.db, id)
}

func deleteLibraryArticleCmd(d *db.DB, id int64) tea.Cmd {
	return func() tea.Msg {
		if err := d.DeleteArticle(id); err != nil {
			return errMsg{err}
		}
		return libraryDeletedMsg{articleID: id}
	}
}

// fetchLibraryArticleCmd runs the readability extractor in the
// background and writes the resolved title + Markdown body to the DB.
// Returns libraryFetchedMsg regardless of outcome — the handler decides
// whether to surface a success or failure toast.
func fetchLibraryArticleCmd(f *feed.Fetcher, d *db.DB, articleID int64, articleURL string) tea.Cmd {
	return func() tea.Msg {
		title, md, err := f.FetchFullWithTitle(context.Background(), articleURL)
		if err != nil {
			rlog.Logf("library", "fetch %s: %v", articleURL, err)
			return libraryFetchedMsg{articleID: articleID, err: err}
		}
		if err := d.UpdateLibraryFetched(articleID, title, md); err != nil {
			rlog.Logf("library", "store %s: %v", articleURL, err)
			return libraryFetchedMsg{articleID: articleID, err: err}
		}
		return libraryFetchedMsg{articleID: articleID, title: title}
	}
}

// renderAddURLModal draws a centered overlay with the URL input.
// width/height are the screen dims; the modal sizes itself relative.
func (m Model) renderAddURLModal(width, height int) string {
	tr := m.tr.Library
	modalW := width - 8
	if modalW > 80 {
		modalW = 80
	}
	if modalW < 30 {
		modalW = 30
	}
	m.addURLInput.Width = modalW - 4

	titleStyle := lipgloss.NewStyle().
		Foreground(colorAccent).
		Background(colorBG).
		Bold(true)
	hintStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Background(colorBG)
	promptStyle := lipgloss.NewStyle().
		Foreground(colorText).
		Background(colorBG)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(tr.AddURLTitle),
		"",
		promptStyle.Render(tr.AddURLPrompt),
		m.addURLInput.View(),
		"",
		hintStyle.Render(tr.AddURLHint),
	)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Background(colorBG).
		Padding(1, 2).
		Width(modalW).
		Render(body)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(colorBG),
	)
}
