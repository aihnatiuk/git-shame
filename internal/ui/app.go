package ui

import (
	"github.com/aihnatiuk/git-shame/internal/ui/blame"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewID identifies which view is currently active.
type ViewID int

const (
	ViewBlame ViewID = iota
	// ViewDiff is added in Phase 2.
)

// App is the root Bubble Tea model. It owns view switching and forwards
// WindowSizeMsg to all child models.
type App struct {
	activeView ViewID
	blameModel blame.Model
}

// NewApp constructs the root App model.
func NewApp(repoRoot, relFile, displayFile, revision string) App {
	return App{
		activeView: ViewBlame,
		blameModel: blame.New(repoRoot, relFile, displayFile, revision),
	}
}

// Init starts the initial data load.
func (a App) Init() tea.Cmd {
	return a.blameModel.Init()
}

// Update handles all top-level messages, delegating to the active child model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.blameModel = a.blameModel.WithSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// ctrl+c always quits regardless of active view.
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}

	case blame.OpenDiffMsg:
		// Phase 2: initialize and switch to diff view.
		// For now, no-op.
		_ = msg
		return a, nil
	}

	// Delegate to the active child model.
	switch a.activeView {
	case ViewBlame:
		newBlame, cmd := a.blameModel.Update(msg)
		a.blameModel = newBlame
		return a, cmd
	}

	return a, nil
}

// View renders the currently active view.
func (a App) View() string {
	switch a.activeView {
	case ViewBlame:
		return a.blameModel.View()
	}
	return ""
}
