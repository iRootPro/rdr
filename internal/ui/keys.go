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

	FilterAll     key.Binding
	FilterUnread  key.Binding
	FilterStarred key.Binding
	NextUnread    key.Binding
	Zen           key.Binding
	Command       key.Binding
	Star          key.Binding
	NextArticle   key.Binding
	PrevArticle   key.Binding
	LinkPicker    key.Binding
	ToggleRead    key.Binding
	MarkAllRead   key.Binding
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

		FilterAll:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "all")),
		FilterUnread:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "unread")),
		FilterStarred: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "starred")),
		NextUnread:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next unread")),
		Zen:           key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "zen")),
		Command:       key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
		Star:          key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "toggle star")),
		NextArticle:   key.NewBinding(key.WithKeys("J"), key.WithHelp("J", "next article")),
		PrevArticle:   key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "prev article")),
		LinkPicker:    key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "links")),
		ToggleRead:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "toggle read")),
		MarkAllRead:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "mark all read")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Back, k.Search, k.Command, k.NextUnread, k.Star, k.Zen, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.PageUp, k.PageDown},
		{k.Left, k.Right, k.Tab, k.Enter, k.Back},
		{k.NextArticle, k.PrevArticle, k.RefreshOne, k.RefreshAll, k.OpenURL, k.FullArticle, k.Star},
		{k.FilterAll, k.FilterUnread, k.FilterStarred, k.NextUnread, k.Zen},
		{k.Search, k.Command, k.Help, k.Settings, k.Quit},
	}
}
