package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iRootPro/rdr/internal/i18n"
)

type keyMap struct {
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Tab        key.Binding
	Top        key.Binding
	Bottom     key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Enter      key.Binding
	Back       key.Binding
	RefreshOne key.Binding
	RefreshAll key.Binding
	OpenURL     key.Binding
	FullArticle key.Binding
	Settings    key.Binding
	Search      key.Binding
	Help        key.Binding

	NextUnread    key.Binding
	Zen           key.Binding
	Command       key.Binding
	Star          key.Binding
	Bookmark      key.Binding
	FilterAll     key.Binding
	FilterUnread  key.Binding
	FilterStarred key.Binding
	NextArticle   key.Binding
	PrevArticle   key.Binding
	LinkPicker    key.Binding
	ToggleRead    key.Binding
	MarkAllRead   key.Binding
	YankURL       key.Binding
	YankMarkdown  key.Binding
	ToggleFold    key.Binding
	AddLibrary    key.Binding
	DeleteLibrary key.Binding
}

func defaultKeys(tr *i18n.Strings) keyMap {
	k := tr.Keys
	// Russian layout equivalents (QWERTY → ЙЦУКЕН):
	// q→й w→ц e→у r→к t→е y→н u→г i→ш o→щ p→з
	// a→ф s→ы d→в f→а g→п h→р j→о k→л l→д
	// z→я x→ч c→с v→м b→и n→т m→ь
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "й", "ctrl+c"), key.WithHelp("q", k.Quit)),
		Up:         key.NewBinding(key.WithKeys("k", "л", "up"), key.WithHelp("k", k.Up)),
		Down:       key.NewBinding(key.WithKeys("j", "о", "down"), key.WithHelp("j", k.Down)),
		Left:       key.NewBinding(key.WithKeys("h", "р", "left"), key.WithHelp("h", k.Back)),
		Right:      key.NewBinding(key.WithKeys("l", "д", "right"), key.WithHelp("l", k.Forward)),
		Tab:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", k.SwitchPane)),
		Top:        key.NewBinding(key.WithKeys("g", "п", "home"), key.WithHelp("g", k.Top)),
		Bottom:     key.NewBinding(key.WithKeys("G", "П", "end"), key.WithHelp("G", k.Bottom)),
		PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("^u", k.PageUp)),
		PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("^d", k.PageDown)),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", k.Open)),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", k.Esc)),
		RefreshOne: key.NewBinding(key.WithKeys("r", "к"), key.WithHelp("r", k.RefreshOne)),
		RefreshAll: key.NewBinding(key.WithKeys("R", "К"), key.WithHelp("R", k.RefreshAll)),
		OpenURL:     key.NewBinding(key.WithKeys("o", "щ"), key.WithHelp("o", k.OpenBrowser)),
		FullArticle: key.NewBinding(key.WithKeys("f", "а"), key.WithHelp("f", k.FullArticle)),
		Settings:    key.NewBinding(key.WithKeys("s", "ы"), key.WithHelp("s", k.Settings)),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", k.Search)),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", k.Help)),

		NextUnread:    key.NewBinding(key.WithKeys("n", "т"), key.WithHelp("n", k.NextUnread)),
		Zen:           key.NewBinding(key.WithKeys("z", "я"), key.WithHelp("z", k.Zen)),
		Command:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", k.Command)),
		Star:          key.NewBinding(key.WithKeys("m", "ь"), key.WithHelp("m", k.ToggleStar)),
		Bookmark:      key.NewBinding(key.WithKeys("b", "и"), key.WithHelp("b", k.ToggleBookmark)),
		FilterAll:     key.NewBinding(key.WithKeys("a", "ф"), key.WithHelp("a", k.FilterAll)),
		FilterUnread:  key.NewBinding(key.WithKeys("u", "г"), key.WithHelp("u", k.FilterUnread)),
		FilterStarred: key.NewBinding(key.WithKeys("S", "Ы"), key.WithHelp("S", k.FilterStarred)),
		NextArticle:   key.NewBinding(key.WithKeys("J", "О"), key.WithHelp("J", k.NextArticle)),
		PrevArticle:   key.NewBinding(key.WithKeys("K", "Л"), key.WithHelp("K", k.PrevArticle)),
		LinkPicker:    key.NewBinding(key.WithKeys("L", "Д"), key.WithHelp("L", k.LinkPicker)),
		ToggleRead:    key.NewBinding(key.WithKeys("x", "ч"), key.WithHelp("x", k.ToggleRead)),
		MarkAllRead:   key.NewBinding(key.WithKeys("X", "Ч"), key.WithHelp("X", k.MarkAllRead)),
		YankURL:       key.NewBinding(key.WithKeys("y", "н"), key.WithHelp("y", k.YankURL)),
		YankMarkdown:  key.NewBinding(key.WithKeys("Y", "Н"), key.WithHelp("Y", k.YankMarkdown)),
		ToggleFold:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", k.ToggleFold)),
		AddLibrary:    key.NewBinding(key.WithKeys("B", "И"), key.WithHelp("B", k.AddLibrary)),
		DeleteLibrary: key.NewBinding(key.WithKeys("D", "В"), key.WithHelp("D", k.DeleteLibrary)),
	}
}

// ruMap maps Russian ЙЦУКЕН keys to their QWERTY equivalents.
var ruMap = map[string]string{
	"й": "q", "ц": "w", "у": "e", "к": "r", "е": "t", "н": "y", "г": "u", "ш": "i", "щ": "o", "з": "p",
	"ф": "a", "ы": "s", "в": "d", "а": "f", "п": "g", "р": "h", "о": "j", "л": "k", "д": "l",
	"я": "z", "ч": "x", "с": "c", "м": "v", "и": "b", "т": "n", "ь": "m",
	"Й": "Q", "Ц": "W", "У": "E", "К": "R", "Е": "T", "Н": "Y", "Г": "U", "Ш": "I", "Щ": "O", "З": "P",
	"Ф": "A", "Ы": "S", "В": "D", "А": "F", "П": "G", "Р": "H", "О": "J", "Л": "K", "Д": "L",
	"Я": "Z", "Ч": "X", "С": "C", "М": "V", "И": "B", "Т": "N", "Ь": "M",
}

// keyIs checks if a key message matches one of the given keys, accounting
// for the Russian keyboard layout automatically.
func keyIs(msg tea.KeyMsg, keys ...string) bool {
	s := msg.String()
	for _, k := range keys {
		if s == k {
			return true
		}
	}
	// Check Russian equivalent.
	if en, ok := ruMap[s]; ok {
		for _, k := range keys {
			if en == k {
				return true
			}
		}
	}
	return false
}

// ShortHelp / FullHelp from bubbles keymap interface are no longer used;
// we route through shortHelpFor / fullHelpFor below. The methods stay
// as thin wrappers so anything still touching the interface keeps
// compiling, but they're not invoked from rendering.
func (k keyMap) ShortHelp() []key.Binding {
	return shortHelpFor(focusFeeds, k, false)
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Help, k.Quit}}
}

// helpEntry / helpSection describe a row in the full-screen help.
type helpEntry struct {
	Keys string // e.g. "j / k" or "space"
	Desc string // e.g. "scroll line"
}

type helpSection struct {
	Title   string
	Entries []helpEntry
}

// shortHelpFor returns the subset of bindings that actually do something
// in the given focus. Rendered as a single ShortHelpView row beneath
// every view. inLibrary surfaces context-sensitive bindings (currently
// just D for delete) so the short help line reflects what actually
// works under the cursor.
func shortHelpFor(f focus, k keyMap, inLibrary bool) []key.Binding {
	switch f {
	case focusFeeds:
		return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Search, k.Help, k.Quit}
	case focusArticles:
		base := []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.ToggleRead, k.Star}
		if inLibrary {
			base = append(base, k.DeleteLibrary)
		}
		return append(base, k.Help, k.Quit)
	case focusReader:
		base := []key.Binding{k.Up, k.Down, k.NextArticle, k.PrevArticle, k.FullArticle}
		if inLibrary {
			base = append(base, k.DeleteLibrary)
		}
		return append(base, k.Back, k.Help)
	case focusAddURL:
		return []key.Binding{k.Enter, k.Back}
	case focusSettings:
		return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Back, k.Help, k.Quit}
	case focusSearch:
		return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help}
	case focusCommand:
		return []key.Binding{k.Up, k.Down, k.Enter, k.Back}
	case focusLinks:
		return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help}
	case focusHelp:
		return []key.Binding{k.Back, k.Help}
	}
	return []key.Binding{k.Help, k.Quit}
}

// fullHelpFor returns grouped sections shown on the full-screen help
// overlay. Each focus contributes its own context sections; a Global
// section always comes last.
func fullHelpFor(f focus, tr *i18n.Strings, inLibrary bool) []helpSection {
	h := tr.Help
	global := helpSection{
		Title: h.SectionGlobal,
		Entries: []helpEntry{
			{"/", h.DescGlobalSearch},
			{":", h.DescGlobalCommand},
			{"R", h.DescGlobalSync},
			{"s", h.DescGlobalSettings},
			{"z", h.DescGlobalZen},
			{"B", h.DescGlobalAddURL},
			{"?", h.DescGlobalHelp},
			{"q", h.DescGlobalQuit},
		},
	}

	switch f {
	case focusFeeds, focusArticles:
		nav := helpSection{
			Title: h.SectionNav,
			Entries: []helpEntry{
				{"j / k", h.DescDownUp},
				{"tab", h.DescSwitchPane},
				{"enter", h.DescOpenSelection},
				{"g / G", h.DescTopBottom},
				{"^u / ^d", h.DescPagePrefix},
				{"n", h.DescNextUnread},
				{"space", h.DescCollapseCat},
			},
		}
		articleEntries := []helpEntry{
			{"x / X", h.DescToggleMarkAll},
			{"m", h.DescToggleStar},
			{"p", h.DescTogglePreview},
			{"y / Y", h.DescYankURLMD},
			{"o", h.DescOpenBrowser},
		}
		if inLibrary {
			articleEntries = append(articleEntries, helpEntry{"D", h.DescDeleteLibrary})
		}
		article := helpSection{
			Title:   h.SectionArticle,
			Entries: articleEntries,
		}
		filters := helpSection{
			Title: h.SectionFilters,
			Entries: []helpEntry{
				{"a", h.DescShowAll},
				{"u", h.DescShowUnread},
				{"S", h.DescShowStarred},
			},
		}
		return []helpSection{nav, article, filters, global}

	case focusReader:
		nav := helpSection{
			Title: h.SectionReader,
			Entries: []helpEntry{
				{"j / k", h.DescScrollLine},
				{"space", h.DescPageDown},
				{"J / K", h.DescNextPrevArt},
				{"g / G", h.DescTopBottom},
				{"esc", h.DescBackToArticles},
			},
		}
		readerArticleEntries := []helpEntry{
			{"f", h.DescLoadFull},
			{"L", h.DescLinkPicker},
			{"o", h.DescOpenBrowser},
			{"y / Y", h.DescYankURLMD},
			{"x", h.DescToggleRead},
			{"m", h.DescToggleStar},
			{":images", h.DescToggleImages},
		}
		if inLibrary {
			readerArticleEntries = append(readerArticleEntries, helpEntry{"D", h.DescDeleteLibrary})
		}
		article := helpSection{
			Title:   h.SectionArticle,
			Entries: readerArticleEntries,
		}
		return []helpSection{nav, article, global}

	case focusSettings:
		return []helpSection{
			{
				Title: h.SectionFeedSet,
				Entries: []helpEntry{
					{"j / k", h.DescFeedsDownUp},
					{"tab", h.DescFeedsSwitchSec},
					{"a", h.DescFeedsAdd},
					{"d", h.DescFeedsDel},
					{"e", h.DescFeedsRename},
					{"c", h.DescFeedsCategory},
					{"p", h.DescFeedsCredentials},
					{"i", h.DescFeedsImport},
					{"E", h.DescFeedsExport},
					{"esc", h.DescFeedsClose},
				},
			},
			{
				Title: h.SectionFoldersSet,
				Entries: []helpEntry{
					{"j / k", h.DescFeedsDownUp},
					{"tab", h.DescFeedsSwitchSec},
					{"e", h.DescFoldersEdit},
					{"d", h.DescFoldersDel},
					{"esc", h.DescFeedsClose},
				},
			},
			{
				Title: h.SectionSmartFolderSet,
				Entries: []helpEntry{
					{"j / k", h.DescFeedsDownUp},
					{"tab", h.DescFeedsSwitchSec},
					{"a", h.DescSmartFolderAdd},
					{"d", h.DescSmartFolderDel},
					{"e", h.DescSmartFolderEdit},
					{"esc", h.DescFeedsClose},
				},
			},
			global,
		}

	case focusSearch:
		return []helpSection{
			{
				Title: h.SectionSearch,
				Entries: []helpEntry{
					{"↑ / ↓", h.DescSearchNav},
					{"enter", h.DescSearchOpen},
					{"esc", h.DescSearchClose},
				},
			},
			{
				Title: h.SectionQuerySyn,
				Entries: []helpEntry{
					{"word", h.DescSynWord},
					{"title:rust", h.DescSynTitle},
					{"feed:habr", h.DescSynFeed},
					{"unread", h.DescSynUnread},
					{"starred", h.DescSynStarred},
					{"today", h.DescSynToday},
					{"newer:1w", h.DescSynNewer},
					{"~title:ad", h.DescSynNegate},
				},
			},
			global,
		}

	case focusCommand:
		return []helpSection{
			{
				Title: h.SectionCommand,
				Entries: []helpEntry{
					{"↑ / ↓", h.DescCmdNavSugg},
					{"tab", h.DescCmdComplete},
					{"^p / ^n", h.DescCmdHistory},
					{"enter", h.DescCmdExecute},
					{"esc", h.DescCmdCancel},
				},
			},
			{
				Title: h.SectionCommon,
				Entries: []helpEntry{
					{":sync", h.DescCmdSync},
					{":sort date|title", h.DescCmdSort},
					{":filter unread|starred", h.DescCmdFilter},
					{":read <query>", h.DescCmdReadQ},
					{":star <query>", h.DescCmdStarQ},
					{":copy url <query>", h.DescCmdCopyUrlQ},
					{":import / :export <path>", h.DescCmdImportExport},
					{":collapseall / :expandall", h.DescCmdCollapseAll},
				},
			},
		}

	case focusLinks:
		return []helpSection{
			{
				Title: h.SectionLinks,
				Entries: []helpEntry{
					{"j / k", h.DescLinksNav},
					{"g / G", h.DescLinksTop},
					{"enter", h.DescLinksOpen},
					{"esc", h.DescLinksClose},
				},
			},
			global,
		}

	case focusHelp:
		return []helpSection{
			{
				Title: h.SectionHelp,
				Entries: []helpEntry{
					{"esc / ?", h.DescHelpClose},
				},
			},
		}
	}
	return []helpSection{global}
}

// focusLabel is the human-readable name of a focus, used in the help
// screen title and other places where we want to print the mode name.
func focusLabel(f focus, tr *i18n.Strings) string {
	fs := tr.Focus
	switch f {
	case focusFeeds:
		return fs.Feeds
	case focusArticles:
		return fs.Articles
	case focusReader:
		return fs.Reader
	case focusSettings:
		return fs.Settings
	case focusSearch:
		return fs.Search
	case focusCommand:
		return fs.Command
	case focusLinks:
		return fs.Links
	case focusHelp:
		return fs.Help
	case focusAddURL:
		return fs.AddURL
	}
	return ""
}
