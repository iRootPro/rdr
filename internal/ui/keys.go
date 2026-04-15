package ui

import "github.com/charmbracelet/bubbles/key"

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
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Up:         key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
		Down:       key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
		Left:       key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "back")),
		Right:      key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "forward")),
		Tab:        key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
		Top:        key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
		PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("^u", "page up")),
		PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("^d", "page down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		RefreshOne: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh current")),
		RefreshAll: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh all")),
		OpenURL:     key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
		FullArticle: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "full article")),
		Settings:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

		NextUnread:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next unread")),
		Zen:           key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "zen")),
		Command:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
		Star:          key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "toggle star")),
		FilterAll:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all articles")),
		FilterUnread:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unread only")),
		FilterStarred: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "starred only")),
		NextArticle:   key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "next article")),
		PrevArticle:   key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "prev article")),
		LinkPicker:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "links")),
		ToggleRead:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "toggle read")),
		MarkAllRead:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "mark all read")),
		YankURL:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "yank URL")),
		YankMarkdown:  key.NewBinding(key.WithKeys("Y"), key.WithHelp("Y", "yank [title](url)")),
		ToggleFold:    key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "collapse category")),
	}
}

// ShortHelp / FullHelp from bubbles keymap interface are no longer used;
// we route through shortHelpFor / fullHelpFor below. The methods stay
// as thin wrappers so anything still touching the interface keeps
// compiling, but they're not invoked from rendering.
func (k keyMap) ShortHelp() []key.Binding {
	return shortHelpFor(focusFeeds, k)
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
// every view.
func shortHelpFor(f focus, k keyMap) []key.Binding {
	switch f {
	case focusFeeds:
		return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.ToggleFold, k.FilterUnread, k.FilterStarred, k.Search, k.Command, k.Help, k.Quit}
	case focusArticles:
		return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.ToggleRead, k.Star, k.NextUnread, k.FilterUnread, k.FilterStarred, k.Search, k.Help, k.Quit}
	case focusReader:
		return []key.Binding{k.Up, k.Down, k.NextArticle, k.PrevArticle, k.FullArticle, k.LinkPicker, k.OpenURL, k.YankURL, k.Star, k.Back, k.Help}
	case focusSettings:
		return []key.Binding{k.Up, k.Down, k.Enter, k.Back, k.Help, k.Quit}
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
func fullHelpFor(f focus) []helpSection {
	global := helpSection{
		Title: "Global",
		Entries: []helpEntry{
			{"/", "open search picker"},
			{":", "open command mode"},
			{"R", "sync all feeds"},
			{"s", "open settings"},
			{"z", "toggle zen mode"},
			{"?", "toggle this help"},
			{"q", "quit"},
		},
	}

	switch f {
	case focusFeeds, focusArticles:
		nav := helpSection{
			Title: "Navigation",
			Entries: []helpEntry{
				{"j / k", "down / up"},
				{"tab", "switch pane"},
				{"enter", "open selection"},
				{"g / G", "top / bottom"},
				{"^u / ^d", "page up / down"},
				{"n", "next unread"},
				{"space", "collapse category (feeds pane)"},
			},
		}
		article := helpSection{
			Title: "Article ops",
			Entries: []helpEntry{
				{"x / X", "toggle / mark all read"},
				{"m", "toggle star"},
				{"y / Y", "yank URL / [title](url)"},
				{"o", "open in browser"},
			},
		}
		filters := helpSection{
			Title: "Filters",
			Entries: []helpEntry{
				{"a", "show all articles"},
				{"u", "show unread only"},
				{"S", "show starred only"},
			},
		}
		return []helpSection{nav, article, filters, global}

	case focusReader:
		nav := helpSection{
			Title: "Reader",
			Entries: []helpEntry{
				{"j / k", "scroll line"},
				{"space", "page down"},
				{"J / K", "next / prev article"},
				{"g / G", "top / bottom"},
				{"esc", "back to articles"},
			},
		}
		article := helpSection{
			Title: "Article ops",
			Entries: []helpEntry{
				{"f", "load full article"},
				{"L", "link picker"},
				{"o", "open in browser"},
				{"y / Y", "yank URL / [title](url)"},
				{"x", "toggle read"},
				{"m", "toggle star"},
				{":images", "toggle inline images"},
			},
		}
		return []helpSection{nav, article, global}

	case focusSettings:
		return []helpSection{
			{
				Title: "Feed settings",
				Entries: []helpEntry{
					{"j / k", "down / up"},
					{"a", "add feed"},
					{"d", "delete feed"},
					{"e", "rename feed"},
					{"esc", "close"},
				},
			},
			global,
		}

	case focusSearch:
		return []helpSection{
			{
				Title: "Search picker",
				Entries: []helpEntry{
					{"↑ / ↓", "navigate results"},
					{"enter", "open selection"},
					{"esc", "close"},
				},
			},
			{
				Title: "Query syntax",
				Entries: []helpEntry{
					{"word", "match title or feed name"},
					{"title:rust", "field match"},
					{"feed:habr", "feed name"},
					{"unread", "only unread"},
					{"starred", "only starred"},
					{"today", "published today"},
					{"newer:1w", "newer than 1 week"},
					{"~title:ad", "negate any atom"},
				},
			},
			global,
		}

	case focusCommand:
		return []helpSection{
			{
				Title: "Command mode",
				Entries: []helpEntry{
					{"↑ / ↓", "navigate suggestions"},
					{"tab", "complete highlighted"},
					{"^p / ^n", "history prev / next"},
					{"enter", "execute"},
					{"esc", "cancel"},
				},
			},
			{
				Title: "Common commands",
				Entries: []helpEntry{
					{":sync", "refresh all feeds"},
					{":sort date|title", "change sort"},
					{":filter unread|starred", "set filter"},
					{":read <query>", "mark matches read"},
					{":star <query>", "star matches"},
					{":copy url <query>", "copy URLs to clipboard"},
					{":import / :export <path>", "OPML in/out"},
					{":collapseall / :expandall", "toggle all categories"},
				},
			},
		}

	case focusLinks:
		return []helpSection{
			{
				Title: "Link picker",
				Entries: []helpEntry{
					{"j / k", "navigate"},
					{"g / G", "top / bottom"},
					{"enter", "open in browser"},
					{"esc", "close"},
				},
			},
			global,
		}

	case focusHelp:
		return []helpSection{
			{
				Title: "Help",
				Entries: []helpEntry{
					{"esc / ?", "close"},
				},
			},
		}
	}
	return []helpSection{global}
}

// focusLabel is the human-readable name of a focus, used in the help
// screen title and other places where we want to print the mode name.
func focusLabel(f focus) string {
	switch f {
	case focusFeeds:
		return "Feeds"
	case focusArticles:
		return "Articles"
	case focusReader:
		return "Reader"
	case focusSettings:
		return "Settings"
	case focusSearch:
		return "Search"
	case focusCommand:
		return "Command"
	case focusLinks:
		return "Links"
	case focusHelp:
		return "Help"
	}
	return ""
}
