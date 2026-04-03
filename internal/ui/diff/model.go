package diff

import (
	"github.com/aihnatiuk/git-shame/internal/git"
	"github.com/aihnatiuk/git-shame/internal/highlight"
	"github.com/aihnatiuk/git-shame/internal/ui/commitinfo"
	"github.com/aihnatiuk/git-shame/internal/ui/styles"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// CloseDiffMsg is sent to the parent App when the user quits the diff view.
type CloseDiffMsg struct{}

// LoadState tracks the async loading lifecycle of the diff view.
type LoadState int

const (
	LoadStateIdle    LoadState = iota
	LoadStateLoading           // git show subprocess running
	LoadStateLoaded            // data ready to display
	LoadStateError             // subprocess failed
)

// Model is the Bubble Tea model for the diff view.
type Model struct {
	commit           git.CommitInfo
	diff             git.FileDiff
	allDiffLines     []git.DiffLine // flattened: synthetic hunk headers + content lines
	highlightedLines []string       // parallel to allDiffLines, Chroma ANSI
	loadErr          error
	state            LoadState

	repoRoot string
	relFile  string
	hash     string

	terminalWidth  int
	terminalHeight int
	headerHeight   int // computed after ShowResult arrives
	bodyHeight     int

	vScrollOffset int

	spinner spinner.Model
	keys    KeyMap
	styles  styles.DiffStyles
}

// New creates a new diff Model for the given commit and file.
func New(repoRoot, relFile, hash string) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Points

	return Model{
		state:        LoadStateLoading,
		repoRoot:     repoRoot,
		relFile:      relFile,
		hash:         hash,
		headerHeight: 1, // spinner occupies one line until data arrives
		spinner:      sp,
		keys:         DefaultKeyMap(),
		styles:       styles.DefaultDiff(),
	}
}

// Init starts the async git show load.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		git.RunShowCmd(m.repoRoot, m.relFile, m.hash),
		m.spinner.Tick,
	)
}

// Update handles messages for the diff view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case git.ShowResult:
		if msg.Err != nil {
			m.state = LoadStateError
			m.loadErr = msg.Err
			return m, nil
		}
		m.state = LoadStateLoaded
		m.commit = msg.Commit
		m.diff = msg.Diff
		m.allDiffLines = flattenDiffLines(m.diff)
		m.highlightedLines = buildHighlightedLines(m.allDiffLines, m.relFile)
		m.headerHeight = computeHeaderHeight(m.commit, m.terminalWidth, m.styles)
		m.bodyHeight = computeBodyHeight(m.terminalHeight, m.headerHeight)
		m.adjustScrollOffset()
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
			return m, nil
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Down):
		m.vScrollOffset = min(m.vScrollOffset+1, max(0, len(m.allDiffLines)-m.bodyHeight))
	case key.Matches(msg, m.keys.Up):
		m.vScrollOffset = max(m.vScrollOffset-1, 0)
	case key.Matches(msg, m.keys.HalfPageDown):
		m.vScrollOffset = min(m.vScrollOffset+m.bodyHeight/2, max(0, len(m.allDiffLines)-m.bodyHeight))
	case key.Matches(msg, m.keys.HalfPageUp):
		m.vScrollOffset = max(m.vScrollOffset-m.bodyHeight/2, 0)
	case key.Matches(msg, m.keys.GoToTop):
		m.vScrollOffset = 0
	case key.Matches(msg, m.keys.GoToBottom):
		m.vScrollOffset = max(0, len(m.allDiffLines)-m.bodyHeight)
	case key.Matches(msg, m.keys.Quit):
		return m, func() tea.Msg { return CloseDiffMsg{} }
	}
	return m, nil
}

// View renders the diff view.
func (m Model) View() tea.View {
	title := RenderTitleBar(m.relFile, m.hash, m.terminalWidth, m.styles)
	header := RenderHeader(&m)
	body := RenderBody(&m)
	status := RenderStatusBar(&m)

	content := lipgloss.JoinVertical(lipgloss.Position(0), title, header, body, status)

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

// WithSize updates terminal dimensions and recomputes layout.
func (m Model) WithSize(w, h int) Model {
	m.terminalWidth = w
	m.terminalHeight = h
	if m.state == LoadStateLoaded {
		m.headerHeight = computeHeaderHeight(m.commit, w, m.styles)
	}
	m.bodyHeight = computeBodyHeight(h, m.headerHeight)
	m.adjustScrollOffset()
	return m
}

func (m *Model) adjustScrollOffset() {
	maxOff := max(0, len(m.allDiffLines)-m.bodyHeight)
	if m.vScrollOffset > maxOff {
		m.vScrollOffset = maxOff
	}
	if m.vScrollOffset < 0 {
		m.vScrollOffset = 0
	}
}

// flattenDiffLines converts the hunk structure into a flat list of DiffLine,
// injecting a synthetic DiffHunkHeader entry before each hunk's lines.
func flattenDiffLines(fd git.FileDiff) []git.DiffLine {
	var result []git.DiffLine
	for _, h := range fd.Hunks {
		result = append(result, git.DiffLine{
			Type:    git.DiffHunkHeader,
			Content: h.Header,
		})
		result = append(result, h.Lines...)
	}
	return result
}

// buildHighlightedLines applies Chroma highlighting to all content lines in
// the flattened diff, skipping hunk headers and no-newline markers.
func buildHighlightedLines(flat []git.DiffLine, relFile string) []string {
	var contents []string
	var contentIndices []int
	var overrides map[int]string

	for i, dl := range flat {
		if dl.Type == git.DiffHunkHeader || dl.Type == git.DiffNoNewline {
			continue
		}
		if dl.Type == git.DiffRemoved {
			if overrides == nil {
				overrides = make(map[int]string)
			}
			overrides[len(contents)] = styles.OldLineNumFG
		}
		contents = append(contents, dl.Content)
		contentIndices = append(contentIndices, i)
	}

	result := make([]string, len(flat))
	for i, dl := range flat {
		result[i] = dl.Content
	}

	if len(contents) == 0 {
		return result
	}

	highlighted := highlight.HighlightLinesWithFgOverride(relFile, contents, overrides)
	for j, idx := range contentIndices {
		result[idx] = highlighted[j]
	}
	return result
}

func computeHeaderHeight(info git.CommitInfo, width int, s styles.DiffStyles) int {
	rendered := commitinfo.Render(info, 5, width, s)
	return lipgloss.Height(rendered)
}

// computeBodyHeight returns the number of lines available for the diff body.
// Title bar and status bar each occupy one line.
func computeBodyHeight(termHeight, headerHeight int) int {
	return max(termHeight-2-headerHeight, 1)
}
