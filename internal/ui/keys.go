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
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
		Up:         key.NewBinding(key.WithKeys("k", "up")),
		Down:       key.NewBinding(key.WithKeys("j", "down")),
		Left:       key.NewBinding(key.WithKeys("h", "left")),
		Right:      key.NewBinding(key.WithKeys("l", "right")),
		Tab:        key.NewBinding(key.WithKeys("tab")),
		Top:        key.NewBinding(key.WithKeys("g", "home")),
		Bottom:     key.NewBinding(key.WithKeys("G", "end")),
		PageUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
		PageDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown")),
		Enter:      key.NewBinding(key.WithKeys("enter")),
		Back:       key.NewBinding(key.WithKeys("esc")),
		RefreshOne: key.NewBinding(key.WithKeys("r")),
		RefreshAll: key.NewBinding(key.WithKeys("R")),
	}
}
