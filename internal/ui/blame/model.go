package blame

import (
	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// LoadState tracks the async loading lifecycle of the blame view.
type LoadState int

const (
	LoadStateIdle    LoadState = iota
	LoadStateLoading           // git blame subprocess running
	LoadStateLoaded            // lines are ready
	LoadStateError             // subprocess failed
)

// HistoryEntry represents one frame in the blame navigation history stack.
type HistoryEntry struct {
	File       string
	Revision   string
	CursorLine int
}

// OpenDiffMsg is sent to the parent App when the user presses Enter.
type OpenDiffMsg struct {
	CommitHash string
}

// maxHistory is the maximum number of entries kept in the history stack.
const maxHistory = 50

// Model is the Bubble Tea model for the blame view.
// All fields are value types to satisfy Bubble Tea's immutability contract.
type Model struct {
	// Git data
	lines   []git.BlameLine
	loadErr error
	state   LoadState

	// Navigation
	cursor        int // current line index (0-based)
	offset        int // first visible line index (viewport top)
	hScroll       int // horizontal scroll offset in visible columns
	history       []HistoryEntry
	pendingCursor int // cursor position to restore after a goBack reload

	// Context
	repoRoot string
	relFile  string // path relative to repo root
	file     string // display path (original user input)
	revision string // commit/ref; empty = HEAD

	// Layout (updated on WindowSizeMsg)
	width  int
	height int
	bodyH  int // height - 2 (title bar + status bar)

	// Components
	columns []Column
	spinner spinner.Model
	keys    KeyMap
	styles  styles.BlameStyles
}

// New creates a new blame Model ready to load.
func New(repoRoot, relFile, displayFile, revision string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		state:    LoadStateLoading,
		repoRoot: repoRoot,
		relFile:  relFile,
		file:     displayFile,
		revision: revision,
		columns:  defaultColumns(),
		spinner:  sp,
		keys:     DefaultKeyMap(),
		styles:   styles.Default(),
	}
}

// Init starts the async git blame load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		git.RunBlameCmd(m.repoRoot, m.relFile, m.revision),
		m.spinner.Tick,
	)
}

// Update handles messages for the blame view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.bodyH = max(msg.Height-2, 1)
		m.columns = RecalcWidths(m.columns, m.lines, m.width)
		m.ensureCursorVisible()
		return m, nil

	case git.BlameResult:
		if msg.Err != nil {
			m.state = LoadStateError
			m.loadErr = msg.Err
			return m, nil
		}
		m.state = LoadStateLoaded
		m.lines = msg.Lines
		m.columns = RecalcWidths(m.columns, m.lines, m.width)
		// Restore cursor if we navigated back.
		if m.pendingCursor > 0 {
			m.cursor = m.pendingCursor
			m.pendingCursor = 0
			if m.cursor >= len(m.lines) {
				m.cursor = len(m.lines) - 1
			}
			m.ensureCursorVisible()
		}
		return m, nil

	case spinner.TickMsg:
		if m.state == LoadStateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.state != LoadStateLoaded {
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
			return m, nil
		}
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes key events when the view is fully loaded.
func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.HalfPageDown):
		m.moveCursor(m.bodyH / 2)
	case key.Matches(msg, m.keys.HalfPageUp):
		m.moveCursor(-(m.bodyH / 2))
	case key.Matches(msg, m.keys.GoToTop):
		m.cursor = 0
		m.offset = 0
	case key.Matches(msg, m.keys.GoToBottom):
		m.cursor = len(m.lines) - 1
		m.ensureCursorVisible()
	case key.Matches(msg, m.keys.ScrollRight):
		m.hScroll += 4
	case key.Matches(msg, m.keys.ScrollLeft):
		m.hScroll -= 4
		if m.hScroll < 0 {
			m.hScroll = 0
		}
	case key.Matches(msg, m.keys.Parent):
		return m.navigateToParent()
	case key.Matches(msg, m.keys.Back):
		return m.goBack()
	case key.Matches(msg, m.keys.OpenDiff):
		if len(m.lines) > 0 {
			hash := m.lines[m.cursor].CommitHash
			return m, func() tea.Msg { return OpenDiffMsg{CommitHash: hash} }
		}
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

// View renders the blame view as a string.
func (m Model) View() string {
	title := RenderTitleBar(m.file, m.revision, m.width, m.styles)
	body := RenderBody(&m)
	status := RenderStatusBar(m.cursor, len(m.lines), m.width, m.styles)
	return title + "\n" + body + "\n" + status
}

// WithSize updates the terminal dimensions. Called by App on WindowSizeMsg.
func (m Model) WithSize(w, h int) Model {
	m.width = w
	m.height = h
	m.bodyH = h - 2
	if m.bodyH < 1 {
		m.bodyH = 1
	}
	m.columns = RecalcWidths(m.columns, m.lines, m.width)
	m.ensureCursorVisible()
	return m
}

// moveCursor moves the cursor by delta lines and keeps it in bounds.
func (m *Model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.lines) {
		m.cursor = len(m.lines) - 1
	}
	m.ensureCursorVisible()
}

// ensureCursorVisible adjusts the viewport offset so the cursor is visible.
func (m *Model) ensureCursorVisible() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.bodyH {
		m.offset = m.cursor - m.bodyH + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// navigateToParent re-blames the file at the parent of the current line's commit.
func (m Model) navigateToParent() (Model, tea.Cmd) {
	if len(m.lines) == 0 {
		return m, nil
	}
	line := m.lines[m.cursor]

	// Push current state onto the history stack.
	entry := HistoryEntry{
		File:       m.file,
		Revision:   m.revision,
		CursorLine: m.cursor,
	}
	history := append([]HistoryEntry{}, m.history...) // copy
	if len(history) >= maxHistory {
		history = history[1:]
	}
	m.history = append(history, entry)

	parentRev := line.CommitHash + "^"
	m.revision = parentRev
	m.state = LoadStateLoading
	m.lines = nil
	m.cursor = 0
	m.offset = 0

	return m, tea.Batch(
		git.RunBlameCmd(m.repoRoot, m.relFile, parentRev),
		m.spinner.Tick,
	)
}

// goBack pops the history stack and reloads the previous blame state.
func (m Model) goBack() (Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	entry := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]

	m.file = entry.File
	m.revision = entry.Revision
	m.pendingCursor = entry.CursorLine
	m.state = LoadStateLoading
	m.lines = nil
	m.cursor = 0
	m.offset = 0

	return m, tea.Batch(
		git.RunBlameCmd(m.repoRoot, m.relFile, m.revision),
		m.spinner.Tick,
	)
}
