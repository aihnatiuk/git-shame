package blame

import (
	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/highlight"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	RelFile    string // repo-relative path (may differ from File after renames)
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
	lines            []git.BlameLine
	highlightedLines []string // ANSI-escaped lines produced by Chroma; same length as lines
	loadErr          error
	state            LoadState

	// Navigation
	cursor           int // current line index (0-based)
	pendingCursor    int // cursor position to restore after a goBack reload
	vScrollOffset    int // first visible line index (viewport top)
	hScrollOffset    int // horizontal scroll offset in visible columns
	maxHScrollOffset int // upper bound for hScrollOffset
	history          []HistoryEntry
	statusMessage    string // transient message shown in the status bar; cleared on next key press

	// Context
	repoRoot string
	relFile  string // path relative to repo root
	file     string // display path (original user input)
	revision string // commit/ref; empty = HEAD

	// Layout (updated on WindowSizeMsg)
	terminalWidth  int
	terminalHeight int
	bodyWidth      int
	bodyHeight     int

	// Components
	columns []Column
	spinner spinner.Model
	keys    KeyMap
	styles  styles.BlameStyles
}

// New creates a new blame Model ready to load.
func New(repoRoot, relFile, displayFile, revision string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points

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

	case git.BlameResult:
		if msg.Err != nil {
			m.state = LoadStateError
			m.loadErr = msg.Err
			return m, nil
		}
		m.state = LoadStateLoaded
		m.lines = msg.Lines
		contents := make([]string, len(msg.Lines))
		for i, l := range msg.Lines {
			contents[i] = l.Content
		}
		m.highlightedLines = highlight.HighlightLines(m.relFile, contents)
		m.columns = RecalcWidths(m.columns, m.lines, m.bodyWidth)
		m.maxHScrollOffset = CalcMaxHScroll(m.columns, m.lines)
		// Restore cursor if we navigated back, otherwise clamp to new line count.
		if m.pendingCursor > 0 {
			m.cursor = m.pendingCursor
			m.pendingCursor = 0
		}
		if m.cursor >= len(m.lines) {
			m.cursor = max(len(m.lines)-1, 0)
		}
		m.adjustVerticalScrollOffset()
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
	m.statusMessage = ""
	switch {
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.HalfPageDown):
		m.moveCursor(m.bodyHeight / 2)
	case key.Matches(msg, m.keys.HalfPageUp):
		m.moveCursor(-(m.bodyHeight / 2))
	case key.Matches(msg, m.keys.GoToTop):
		m.cursor = 0
		m.vScrollOffset = 0
	case key.Matches(msg, m.keys.GoToBottom):
		m.cursor = len(m.lines) - 1
		m.adjustVerticalScrollOffset()
	case key.Matches(msg, m.keys.ScrollRight):
		m.hScrollOffset = min(m.hScrollOffset+4, m.maxHScrollOffset)
	case key.Matches(msg, m.keys.ScrollLeft):
		m.hScrollOffset = max(m.hScrollOffset-4, 0)
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
func (m Model) View() tea.View {
	title := RenderTitleBar(m.file, m.revision, m.bodyWidth, m.styles)
	body := RenderBody(&m)
	status := RenderStatusBar(&m)

	content := lipgloss.JoinVertical(lipgloss.Position(0), title, body, status)

	output := lipgloss.NewStyle().
		Width(m.terminalWidth).
		MaxWidth(m.terminalWidth).
		Height(m.terminalHeight).
		MaxHeight(m.terminalHeight).
		Render(content)

	view := tea.NewView(output)
	view.AltScreen = true

	return view
}

// WithSize updates the terminal dimensions. Called by App on WindowSizeMsg.
func (m Model) WithSize(w, h int) Model {
	titleHeight, statusHeight := 1, 1

	m.terminalWidth = w
	m.terminalHeight = h
	m.bodyWidth = w
	m.bodyHeight = max(m.terminalHeight-(titleHeight+statusHeight), 1)
	m.columns = RecalcWidths(m.columns, m.lines, m.bodyWidth)
	m.maxHScrollOffset = CalcMaxHScroll(m.columns, m.lines)
	m.hScrollOffset = min(m.hScrollOffset, m.maxHScrollOffset)
	m.adjustVerticalScrollOffset()

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
	m.adjustVerticalScrollOffset()
}

// adjustVerticalScrollOffset adjusts the viewport offset so that:
//  1. the cursor row is visible (scroll down / up as needed), and
//  2. no empty rows appear at the bottom when lines above could fill them
//     (e.g. after the terminal grows taller or a new blame loads).
func (m *Model) adjustVerticalScrollOffset() {
	if m.cursor < m.vScrollOffset {
		m.vScrollOffset = m.cursor
	}
	if m.cursor >= m.vScrollOffset+m.bodyHeight {
		m.vScrollOffset = m.cursor - m.bodyHeight + 1
	}

	// Try to fill the viewport with lines if we have fewer lines than the current offset allows.
	if len(m.lines) > 0 {
		if maxOffset := max(0, len(m.lines)-m.bodyHeight); m.vScrollOffset > maxOffset {
			m.vScrollOffset = maxOffset
		}
	}
	if m.vScrollOffset < 0 {
		m.vScrollOffset = 0
	}
}

// navigateToParent re-blames the file at the parent of the current line's commit.
// It uses the parsed Previous field to determine the parent commit and filename,
// which handles file renames correctly and avoids a git subprocess on root commits.
func (m Model) navigateToParent() (Model, tea.Cmd) {
	if len(m.lines) == 0 {
		return m, nil
	}
	line := m.lines[m.cursor]

	if line.Previous.Hash == "" {
		m.statusMessage = "No parent commit"
		return m, nil
	}

	// Push current state onto the history stack.
	entry := HistoryEntry{
		File:       m.file,
		RelFile:    m.relFile,
		Revision:   m.revision,
		CursorLine: m.cursor,
	}
	history := append([]HistoryEntry{}, m.history...)
	if len(history) >= maxHistory {
		history = history[1:]
	}
	m.history = append(history, entry)

	m.revision = line.Previous.Hash
	m.relFile = line.Previous.Filename
	m.state = LoadStateLoading

	return m, tea.Batch(
		git.RunBlameCmd(m.repoRoot, m.relFile, m.revision),
		m.spinner.Tick,
	)
}

// goBack pops the history stack and reloads the previous blame state.
func (m Model) goBack() (Model, tea.Cmd) {
	if len(m.history) == 0 {
		m.statusMessage = "Already at start of history"
		return m, nil
	}
	entry := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]

	m.file = entry.File
	m.relFile = entry.RelFile
	m.revision = entry.Revision
	m.pendingCursor = entry.CursorLine
	m.state = LoadStateLoading

	return m, tea.Batch(
		git.RunBlameCmd(m.repoRoot, m.relFile, m.revision),
		m.spinner.Tick,
	)
}
