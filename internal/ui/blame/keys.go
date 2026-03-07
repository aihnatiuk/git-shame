package blame

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the blame view.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
	Parent       key.Binding // navigate to parent commit
	Back         key.Binding // go back in history
	OpenDiff     key.Binding // open diff view
	Quit         key.Binding
	ScrollLeft   key.Binding
	ScrollRight  key.Binding
}

// DefaultKeyMap returns the default keybindings.
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
		Parent: key.NewBinding(
			key.WithKeys(","),
			key.WithHelp(",", "go to parent commit"),
		),
		Back: key.NewBinding(
			key.WithKeys("<"),
			key.WithHelp("<", "go back"),
		),
		OpenDiff: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open diff"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "scroll left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "scroll right"),
		),
	}
}
