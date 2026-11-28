package tui

import (
	"context"
	"strings"
	"time"

	"factory-cli/internal/gitboard"
	"factory-cli/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode controls which screen is active inside the TUI.
type ViewMode int

const (
	ModeList ViewMode = iota
	ModeDetail
	ModeSearch
	ModeHelp
)

// Options configures the interactive TUI program.
type Options struct {
	RepoDir         string
	PreferredRemote string
	PreferredBranch string
	RootDir         string
	Interval        time.Duration
	NoFetch         bool
	NoPull          bool
}

type (
	tickMsg struct{}

	refreshMsg struct {
		board       *model.Board
		activeRuns  []gitboard.ActiveRun
		diffs       []string
		fetchErr    string
		loadErr     string
		unchanged   bool
		localState  gitboard.LocalSyncState
		localDetail string
		syncedAt    time.Time
		seq         int
	}
)

// RefreshPayload contains the result of a background fetch + board load cycle.
type RefreshPayload struct {
	Board       *model.Board
	ActiveRuns  []gitboard.ActiveRun
	FetchErr    string
	Unchanged   bool
	LocalState  gitboard.LocalSyncState
	LocalDetail string
}

// FetchSnapshotFunc abstracts branch fetching and board loading for testability.
type FetchSnapshotFunc func(ctx context.Context, opts Options, candidate gitboard.BranchCandidate, prevSHA string, forceReload bool) (RefreshPayload, error)

// AppModel is the root Bubble Tea model for factory-cli.
type AppModel struct {
	Opts           Options
	Loader         FetchSnapshotFunc
	Candidates     []gitboard.BranchCandidate
	CandidateIdx   int
	Board          *model.Board
	Mode           ViewMode
	StageIdx       int
	SelectedIdx    int
	SelectedFile   string
	SelectedLabel  string
	MineOnly       bool
	ShowStop       bool
	SearchInput    textinput.Model
	DetailViewport viewport.Model
	DetailHeader   string
	SectionOffsets []int
	Refreshing     bool
	RefreshSeq     int
	LastFetchErr   string
	InitialErr     string
	LastDiff       []string
	LastDiffAt     time.Time
	LastSyncAt     time.Time
	LocalSyncState gitboard.LocalSyncState
	LocalSyncNote  string
	LocalSyncAt    time.Time
	ActiveRuns     []gitboard.ActiveRun
	Width          int
	Height         int
}

// NewAppModel constructs an AppModel ready for tea.NewProgram.
func NewAppModel(opts Options, candidates []gitboard.BranchCandidate, initialBoard *model.Board, initialRuns []gitboard.ActiveRun, loader FetchSnapshotFunc) AppModel {
	if loader == nil {
		loader = defaultFetchSnapshot
	}
	ti := textinput.New()
	ti.Placeholder = "Search ID, title, branch..."
	ti.Prompt = "/ "
	ti.CharLimit = 80

	vp := viewport.New(80, 20)

	m := AppModel{
		Opts:           opts,
		Loader:         loader,
		Candidates:     candidates,
		Board:          gitboard.ApplyActiveRuns(initialBoard, initialRuns),
		ActiveRuns:     initialRuns,
		SearchInput:    ti,
		DetailViewport: vp,
		Width:          100,
		Height:         30,
	}
	if initialBoard != nil {
		m.LastSyncAt = initialBoard.FetchedAt
	}
	m.clampSelection()
	return m
}

func defaultFetchSnapshot(ctx context.Context, opts Options, candidate gitboard.BranchCandidate, prevSHA string, forceReload bool) (RefreshPayload, error) {
	syncRes := gitboard.SyncBranch(ctx, opts.RepoDir, candidate, opts.PreferredRemote, opts.NoFetch, opts.NoPull)
	payload := RefreshPayload{
		LocalState:  syncRes.LocalState,
		LocalDetail: syncRes.LocalDetail,
	}
	if syncRes.FetchErr != nil {
		payload.FetchErr = syncRes.FetchErr.Error()
	}

	sha, err := gitboard.ResolveSHA(ctx, opts.RepoDir, candidate.Ref)
	if err != nil {
		return payload, err
	}
	if !forceReload && prevSHA != "" && sha == prevSHA {
		payload.Unchanged = true
		return payload, nil
	}
	now := time.Now()
	board, err := gitboard.LoadSnapshot(ctx, opts.RepoDir, candidate, now)
	if err != nil {
		return payload, err
	}
	runs := gitboard.DetectActiveRuns(board.GitHubSlug, opts.RootDir, now)
	payload.Board = gitboard.ApplyActiveRuns(board, runs)
	payload.ActiveRuns = runs
	return payload, nil
}

// Init kicks off initial async refresh (when enabled) and periodic refresh ticks.
func (m AppModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	if len(m.Candidates) > 0 && (m.Board == nil || !m.Opts.NoFetch) {
		m.Refreshing = true
		cmds = append(cmds, m.triggerRefresh(m.Board == nil))
	}
	if m.Opts.Interval > 0 {
		cmds = append(cmds, tickCmd(m.Opts.Interval))
	}
	return tea.Batch(cmds...)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m AppModel) triggerRefresh(force bool) tea.Cmd {
	if len(m.Candidates) == 0 {
		return nil
	}
	candidate := m.Candidates[m.CandidateIdx]
	prevSHA := ""
	prevBoard := m.Board
	if prevBoard != nil {
		prevSHA = prevBoard.CommitSHA
	}
	seq := m.RefreshSeq
	opts := m.Opts
	loader := m.Loader

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		payload, err := loader(ctx, opts, candidate, prevSHA, force)
		msg := refreshMsg{
			fetchErr:    payload.FetchErr,
			unchanged:   payload.Unchanged,
			localState:  payload.LocalState,
			localDetail: payload.LocalDetail,
			syncedAt:    time.Now(),
			seq:         seq,
		}
		if err != nil {
			msg.loadErr = err.Error()
			return msg
		}
		if payload.Unchanged && prevBoard != nil {
			msg.activeRuns = gitboard.DetectActiveRuns(prevBoard.GitHubSlug, opts.RootDir, msg.syncedAt)
			return msg
		}
		msg.board = payload.Board
		msg.activeRuns = payload.ActiveRuns
		if prevBoard != nil && payload.Board != nil {
			msg.diffs = model.DiffBoards(prevBoard, payload.Board)
		}
		return msg
	}
}

// Update handles user input, window resizes, and background refresh messages.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Mode == ModeDetail {
			m.updateDetailLayout(false)
		} else {
			m.resizeViewport()
		}
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		if m.Opts.Interval > 0 {
			cmds = append(cmds, tickCmd(m.Opts.Interval))
		}
		if !m.Refreshing && len(m.Candidates) > 0 {
			m.Refreshing = true
			m.RefreshSeq++
			cmds = append(cmds, m.triggerRefresh(false))
		}
		return m, tea.Batch(cmds...)

	case refreshMsg:
		if msg.seq != m.RefreshSeq {
			return m, nil
		}
		m.Refreshing = false
		m.LastSyncAt = msg.syncedAt
		m.LastFetchErr = msg.fetchErr
		m.LocalSyncState = msg.localState
		switch msg.localState {
		case gitboard.LocalSyncUpToDate, gitboard.LocalSyncNone:
			m.LocalSyncNote = ""
		default:
			if msg.localDetail != "" {
				m.LocalSyncNote = msg.localDetail
				m.LocalSyncAt = msg.syncedAt
			}
		}
		if msg.loadErr != "" {
			if m.Board == nil {
				m.InitialErr = msg.loadErr
			} else if m.LastFetchErr == "" {
				m.LastFetchErr = msg.loadErr
			}
			return m, nil
		}
		m.InitialErr = ""
		m.ActiveRuns = msg.activeRuns
		if !msg.unchanged && msg.board != nil {
			m.Board = gitboard.ApplyActiveRuns(msg.board, m.ActiveRuns)
			if len(msg.diffs) > 0 {
				m.LastDiff = msg.diffs
				m.LastDiffAt = msg.syncedAt
			}
			m.restoreSelection()
			if m.Mode == ModeDetail {
				m.updateDetailLayout(false)
			}
		} else if m.Board != nil {
			m.Board = gitboard.ApplyActiveRuns(m.Board, m.ActiveRuns)
			if m.Mode == ModeDetail {
				m.updateDetailLayout(false)
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.Mode == ModeSearch {
		var cmd tea.Cmd
		m.SearchInput, cmd = m.SearchInput.Update(msg)
		m.clampSelection()
		return m, cmd
	}
	return m, nil
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if m.Mode == ModeSearch {
		switch key {
		case "esc":
			m.SearchInput.SetValue("")
			m.SearchInput.Blur()
			m.Mode = ModeList
			m.clampSelection()
			return m, nil
		case "enter":
			m.SearchInput.Blur()
			m.Mode = ModeList
			m.clampSelection()
			return m, nil
		default:
			var cmd tea.Cmd
			m.SearchInput, cmd = m.SearchInput.Update(msg)
			m.SelectedIdx = 0
			m.clampSelection()
			return m, cmd
		}
	}

	if m.Mode == ModeHelp {
		switch key {
		case "esc", "?", "q":
			m.Mode = ModeList
		}
		return m, nil
	}

	if m.Mode == ModeDetail {
		switch key {
		case "esc", "backspace", "q":
			m.Mode = ModeList
			return m, nil
		case "r":
			if !m.Refreshing {
				m.Refreshing = true
				m.RefreshSeq++
				return m, m.triggerRefresh(true)
			}
			return m, nil
		case "g":
			m.DetailViewport.GotoTop()
			return m, nil
		case "G":
			m.DetailViewport.GotoBottom()
			return m, nil
		case "n":
			m.jumpSection(1)
			return m, nil
		case "p":
			m.jumpSection(-1)
			return m, nil
		default:
			var cmd tea.Cmd
			m.DetailViewport, cmd = m.DetailViewport.Update(msg)
			return m, cmd
		}
	}

	// ModeList keybindings.
	switch key {
	case "q":
		return m, tea.Quit
	case "?":
		m.Mode = ModeHelp
		return m, nil
	case "esc":
		if m.SearchInput.Value() != "" {
			m.SearchInput.SetValue("")
			m.clampSelection()
		}
		return m, nil
	case "/":
		m.Mode = ModeSearch
		m.SearchInput.Focus()
		return m, textinput.Blink
	case "j", "down":
		items := m.filteredItems()
		if m.SelectedIdx+1 < len(items) {
			m.SelectedIdx++
			m.recordSelection(items)
		}
		return m, nil
	case "k", "up":
		items := m.filteredItems()
		if m.SelectedIdx > 0 {
			m.SelectedIdx--
			m.recordSelection(items)
		}
		return m, nil
	case "g":
		items := m.filteredItems()
		if len(items) > 0 {
			m.SelectedIdx = 0
			m.recordSelection(items)
		}
		return m, nil
	case "G":
		items := m.filteredItems()
		if len(items) > 0 {
			m.SelectedIdx = len(items) - 1
			m.recordSelection(items)
		}
		return m, nil
	case "h", "left", "shift+tab":
		tabs := m.activeStageTabs()
		m.StageIdx = (m.StageIdx - 1 + len(tabs)) % len(tabs)
		m.SelectedIdx = 0
		m.clampSelection()
		return m, nil
	case "l", "right", "tab":
		tabs := m.activeStageTabs()
		m.StageIdx = (m.StageIdx + 1) % len(tabs)
		m.SelectedIdx = 0
		m.clampSelection()
		return m, nil
	case "1", "2", "3", "4", "5", "6", "7":
		idx := int(key[0] - '1')
		tabs := m.activeStageTabs()
		if idx >= 0 && idx < len(tabs) {
			m.StageIdx = idx
			m.SelectedIdx = 0
			m.clampSelection()
		}
		return m, nil
	case "m":
		m.MineOnly = !m.MineOnly
		m.SelectedIdx = 0
		m.clampSelection()
		return m, nil
	case "s":
		m.ShowStop = !m.ShowStop
		tabs := m.activeStageTabs()
		if m.StageIdx >= len(tabs) {
			m.StageIdx = 0
		}
		m.clampSelection()
		return m, nil
	case "b":
		if len(m.Candidates) > 1 && !m.Refreshing {
			m.CandidateIdx = (m.CandidateIdx + 1) % len(m.Candidates)
			m.Refreshing = true
			m.RefreshSeq++
			return m, m.triggerRefresh(true)
		}
		return m, nil
	case "r":
		if !m.Refreshing && len(m.Candidates) > 0 {
			m.Refreshing = true
			m.RefreshSeq++
			return m, m.triggerRefresh(true)
		}
		return m, nil
	case "enter":
		if m.selectedItem() != nil {
			m.Mode = ModeDetail
			m.updateDetailLayout(true)
		}
		return m, nil
	}
	return m, nil
}

// View renders the active TUI screen.
func (m AppModel) View() string {
	width := maxInt(40, m.Width)
	height := maxInt(12, m.Height)

	header := m.renderHeader(width)
	footer := m.renderFooter(width)
	headerH := strings.Count(header, "\n") + 1
	footerH := strings.Count(footer, "\n") + 1
	bodyHeight := maxInt(3, height-headerH-footerH)

	var body string
	switch m.Mode {
	case ModeDetail:
		if m.DetailHeader != "" {
			body = m.DetailHeader + "\n" + m.DetailViewport.View()
		} else {
			body = m.DetailViewport.View()
		}
	case ModeHelp:
		body = m.renderHelpView(width, bodyHeight)
	default:
		body = m.renderListView(width, bodyHeight)
	}

	return header + "\n" + body + "\n" + footer
}

func (m AppModel) activeStageTabs() []string {
	tabs := []string{"", model.FolderNew, model.FolderReview, model.FolderReady, model.FolderBacklog, model.FolderDone}
	if m.ShowStop {
		tabs = append(tabs, model.FolderStop)
	}
	return tabs
}

func (m AppModel) currentStageTab() string {
	tabs := m.activeStageTabs()
	if m.StageIdx < 0 || m.StageIdx >= len(tabs) {
		return ""
	}
	return tabs[m.StageIdx]
}

func (m AppModel) filteredItems() []model.Item {
	if m.Board == nil {
		return nil
	}
	stage := m.currentStageTab()
	base := m.Board.ItemsInStage(stage, m.ShowStop)
	q := strings.ToLower(strings.TrimSpace(m.SearchInput.Value()))

	out := make([]model.Item, 0, len(base))
	for _, it := range base {
		if m.MineOnly && it.Turn != model.TurnYou {
			continue
		}
		if q != "" {
			hay := strings.ToLower(it.Label + " " + it.Title + " " + it.Slug + " " + it.Branch)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, it)
	}
	return out
}

func (m AppModel) selectedItem() *model.Item {
	items := m.filteredItems()
	if len(items) == 0 || m.SelectedIdx < 0 || m.SelectedIdx >= len(items) {
		return nil
	}
	it := items[m.SelectedIdx]
	return &it
}

func (m *AppModel) recordSelection(items []model.Item) {
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(items) {
		m.SelectedFile = items[m.SelectedIdx].File
		m.SelectedLabel = items[m.SelectedIdx].Label
	}
}

func (m *AppModel) clampSelection() {
	items := m.filteredItems()
	if len(items) == 0 {
		m.SelectedIdx = 0
		return
	}
	if m.SelectedIdx >= len(items) {
		m.SelectedIdx = len(items) - 1
	}
	if m.SelectedIdx < 0 {
		m.SelectedIdx = 0
	}
	m.recordSelection(items)
}

func (m *AppModel) restoreSelection() {
	items := m.filteredItems()
	if len(items) == 0 {
		m.SelectedIdx = 0
		return
	}
	if m.SelectedFile != "" {
		for i, it := range items {
			if it.File == m.SelectedFile {
				m.SelectedIdx = i
				m.recordSelection(items)
				return
			}
		}
	}
	if m.SelectedLabel != "" && m.SelectedLabel != "new" {
		for i, it := range items {
			if it.Label == m.SelectedLabel {
				m.SelectedIdx = i
				m.recordSelection(items)
				return
			}
		}
	}
	m.clampSelection()
}

func (m *AppModel) resizeViewport() {
	width := maxInt(40, m.Width)
	height := maxInt(12, m.Height)
	header := m.renderHeader(width)
	footer := m.renderFooter(width)
	headerH := strings.Count(header, "\n") + 1
	footerH := strings.Count(footer, "\n") + 1
	detailHeaderH := 0
	if m.DetailHeader != "" {
		detailHeaderH = strings.Count(m.DetailHeader, "\n") + 1
	}
	m.DetailViewport.Width = width
	m.DetailViewport.Height = maxInt(3, height-headerH-detailHeaderH-footerH)
}

func (m *AppModel) updateDetailLayout(resetTop bool) {
	width := maxInt(40, m.Width)
	it := m.selectedItem()
	if it == nil {
		m.DetailHeader = ""
		m.resizeViewport()
		m.DetailViewport.SetContent("Item is no longer present in the current filter.")
		m.SectionOffsets = nil
		return
	}
	m.DetailHeader = renderDetailHeader(m.Board, it, width)
	m.resizeViewport()
	prevOffset := m.DetailViewport.YOffset
	content, offsets := renderDetailThread(m.Board, it, width)
	m.DetailViewport.SetContent(content)
	m.SectionOffsets = offsets
	if resetTop {
		m.DetailViewport.GotoTop()
	} else {
		m.DetailViewport.SetYOffset(prevOffset)
	}
}

func (m *AppModel) jumpSection(dir int) {
	if len(m.SectionOffsets) == 0 {
		return
	}
	cur := m.DetailViewport.YOffset
	if dir > 0 {
		for _, off := range m.SectionOffsets {
			if off > cur {
				m.DetailViewport.SetYOffset(off)
				return
			}
		}
		return
	}
	for i := len(m.SectionOffsets) - 1; i >= 0; i-- {
		if m.SectionOffsets[i] < cur {
			m.DetailViewport.SetYOffset(m.SectionOffsets[i])
			return
		}
	}
}
