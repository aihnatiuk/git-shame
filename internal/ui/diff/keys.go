package diff

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the diff view.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
	Quit         key.Binding
}

// DefaultKeyMap returns the default keybindings for the diff view.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g/Home", "first line"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G/End", "last line"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close diff"),
		),
	}
}
