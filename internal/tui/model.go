package tui

import (
	"path/filepath"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/doganarif/grove/internal/git"
	"github.com/doganarif/grove/internal/store"
)

type mode int

const (
	modeList mode = iota
	modeAdd
	modeDelete
	modePrune
	modeColor
	modeNote
	modeHelp
	modeFilter
)

type item struct {
	git.Worktree
	git.WorktreeInfo
	Meta      store.Metadata
	Name      string
	Age       string
	IsCurrent bool
}

type pruneItem struct {
	item   item
	reason string // "gone", "merged", "inactive"
	selected bool
}

type Model struct {
	mode       mode
	width      int
	height     int
	showDetail bool

	items     []item
	cursor    int
	repoRoot  string
	repoName  string
	isBare    bool
	baseBranch string
	store     *store.Store

	// Filter
	filterInput textinput.Model
	filterText  string

	// Sort
	sortCol int
	sortAsc bool

	// Add modal
	addInput   textinput.Model
	branches   []string
	addMatches []string
	addIdx     int

	// Delete
	delBranch bool

	// Prune
	pruneItems  []pruneItem
	pruneCursor int

	// Color picker
	colorRow int // 0=color, 1=icon
	colorIdx int
	iconIdx  int

	// Note editor
	noteInput textarea.Model

	// Output
	Selected string

	// State
	loading   bool
	statusMsg string
	err       error
}

// Messages

type worktreesLoadedMsg struct {
	items []item
	err   error
}

type branchesLoadedMsg struct {
	branches []string
}

type actionDoneMsg struct {
	msg string
	err error
}

func New() (Model, error) {
	root, err := git.RepoRoot()
	if err != nil {
		return Model{}, err
	}

	st, err := store.New(root)
	if err != nil {
		return Model{}, err
	}

	fi := textinput.New()
	fi.Placeholder = "filter..."
	fi.CharLimit = 50

	ai := textinput.New()
	ai.Placeholder = "branch name"
	ai.CharLimit = 100

	ni := textarea.New()
	ni.Placeholder = "Write a note..."
	ni.CharLimit = 500
	ni.SetHeight(4)
	ni.SetWidth(50)

	return Model{
		repoRoot:    root,
		repoName:    filepath.Base(root),
		isBare:      git.IsBareRepo(),
		baseBranch:  git.BaseBranch(),
		store:       st,
		loading:     true,
		filterInput: fi,
		addInput:    ai,
		noteInput:   ni,
		sortAsc:     true,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return loadWorktrees(m.repoRoot, m.store)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case worktreesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.items = msg.items
		if m.cursor >= len(m.items) {
			m.cursor = max(0, len(m.items)-1)
		}
		return m, nil

	case branchesLoadedMsg:
		m.branches = msg.branches
		m.filterBranches()
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.statusMsg = "error: " + msg.err.Error()
		} else {
			m.statusMsg = msg.msg
		}
		return m, loadWorktrees(m.repoRoot, m.store)

	case tea.KeyMsg:
		// Global quit
		if m.mode == modeList && msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeFilter:
			return m.updateFilter(msg)
		case modeAdd:
			return m.updateAdd(msg)
		case modeDelete:
			return m.updateDelete(msg)
		case modePrune:
			return m.updatePrune(msg)
		case modeColor:
			return m.updateColor(msg)
		case modeNote:
			return m.updateNote(msg)
		case modeHelp:
			m.mode = modeList
			return m, nil
		}
	}

	return m, nil
}

func (m Model) updateList(msg tea.KeyMsg) (Model, tea.Cmd) {
	vis := m.visibleItems()
	n := len(vis)

	switch msg.String() {
	case "j", "down":
		if m.cursor < n-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if n > 0 {
			m.cursor = n - 1
		}
	case "l", "tab", "right":
		m.showDetail = !m.showDetail
	case "h", "left":
		m.showDetail = false
	case "enter":
		if n > 0 {
			m.Selected = vis[m.cursor].Path
			return m, tea.Quit
		}
	case "a":
		m.mode = modeAdd
		m.addInput.Reset()
		m.addInput.Focus()
		m.addIdx = 0
		return m, loadBranches()
	case "d":
		if n > 0 && !vis[m.cursor].IsMain {
			m.mode = modeDelete
			m.delBranch = false
		}
	case "D":
		if n > 0 && !vis[m.cursor].IsMain {
			m.mode = modeDelete
			m.delBranch = true
		}
	case "n":
		if n > 0 {
			m.mode = modeNote
			meta := vis[m.cursor].Meta
			m.noteInput.Reset()
			m.noteInput.SetValue(meta.Note)
			m.noteInput.Focus()
		}
	case "c":
		if n > 0 {
			m.mode = modeColor
			m.colorRow = 0
			meta := vis[m.cursor].Meta
			m.colorIdx = colorIndex(meta.Color)
			m.iconIdx = iconIndex(meta.Icon)
		}
	case "p":
		m.buildPruneList()
		if len(m.pruneItems) > 0 {
			m.mode = modePrune
			m.pruneCursor = 0
		} else {
			m.statusMsg = "nothing to prune"
		}
	case "/":
		m.mode = modeFilter
		m.filterInput.Reset()
		m.filterInput.Focus()
	case "r":
		m.loading = true
		return m, loadWorktrees(m.repoRoot, m.store)
	case "s":
		m.sortCol = (m.sortCol + 1) % 4
	case "?":
		m.mode = modeHelp
	}

	return m, nil
}

func (m Model) updateFilter(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filterText = m.filterInput.Value()
		m.mode = modeList
		m.cursor = 0
		return m, nil
	case "esc":
		m.filterText = ""
		m.mode = modeList
		m.cursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.cursor = 0
	return m, cmd
}

func (m Model) updateAdd(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "enter":
		branch := strings.TrimSpace(m.addInput.Value())
		if branch == "" {
			return m, nil
		}
		m.mode = modeList
		m.loading = true
		return m, addWorktreeCmd(m.repoRoot, branch, m.baseBranch)
	case "up":
		if m.addIdx > 0 {
			m.addIdx--
		}
		return m, nil
	case "down":
		if m.addIdx < len(m.addMatches)-1 {
			m.addIdx++
		}
		return m, nil
	case "tab":
		if len(m.addMatches) > 0 && m.addIdx < len(m.addMatches) {
			m.addInput.SetValue(m.addMatches[m.addIdx])
			m.addInput.CursorEnd()
			m.filterBranches()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	m.filterBranches()
	return m, cmd
}

func (m Model) updateDelete(msg tea.KeyMsg) (Model, tea.Cmd) {
	vis := m.visibleItems()
	if len(vis) == 0 {
		m.mode = modeList
		return m, nil
	}
	it := vis[m.cursor]

	switch msg.String() {
	case "y":
		m.mode = modeList
		m.loading = true
		return m, deleteWorktreeCmd(it.Path, it.Branch, m.delBranch)
	case "n", "esc":
		m.mode = modeList
	}
	return m, nil
}

func (m Model) updatePrune(msg tea.KeyMsg) (Model, tea.Cmd) {
	n := len(m.pruneItems)
	switch msg.String() {
	case "j", "down":
		if m.pruneCursor < n-1 {
			m.pruneCursor++
		}
	case "k", "up":
		if m.pruneCursor > 0 {
			m.pruneCursor--
		}
	case " ":
		if n > 0 {
			m.pruneItems[m.pruneCursor].selected = !m.pruneItems[m.pruneCursor].selected
		}
	case "a":
		allSelected := true
		for _, pi := range m.pruneItems {
			if !pi.selected {
				allSelected = false
				break
			}
		}
		for i := range m.pruneItems {
			m.pruneItems[i].selected = !allSelected
		}
	case "enter":
		var paths []string
		var branches []string
		for _, pi := range m.pruneItems {
			if pi.selected {
				paths = append(paths, pi.item.Path)
				if pi.item.Branch != "" {
					branches = append(branches, pi.item.Branch)
				}
			}
		}
		if len(paths) > 0 {
			m.mode = modeList
			m.loading = true
			return m, pruneCmd(paths, branches)
		}
	case "esc":
		m.mode = modeList
	}
	return m, nil
}

func (m Model) updateColor(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.colorRow = 1 - m.colorRow
	case "left", "h":
		if m.colorRow == 0 && m.colorIdx > 0 {
			m.colorIdx--
		} else if m.colorRow == 1 && m.iconIdx > 0 {
			m.iconIdx--
		}
	case "right", "l":
		if m.colorRow == 0 && m.colorIdx < len(colorPalette)-1 {
			m.colorIdx++
		} else if m.colorRow == 1 && m.iconIdx < len(iconPalette)-1 {
			m.iconIdx++
		}
	case "enter":
		vis := m.visibleItems()
		if len(vis) > 0 {
			it := vis[m.cursor]
			meta := it.Meta
			meta.Color = colorPalette[m.colorIdx].Name
			icon := iconPalette[m.iconIdx]
			if icon == "—" {
				icon = ""
			}
			meta.Icon = icon
			m.store.Set(it.Name, meta)
			// Update item in-place
			for i := range m.items {
				if m.items[i].Name == it.Name {
					m.items[i].Meta = meta
				}
			}
		}
		m.mode = modeList
	case "esc":
		m.mode = modeList
	}
	return m, nil
}

func (m Model) updateNote(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		return m, nil
	case "ctrl+s":
		vis := m.visibleItems()
		if len(vis) > 0 {
			it := vis[m.cursor]
			meta := it.Meta
			meta.Note = m.noteInput.Value()
			m.store.Set(it.Name, meta)
			for i := range m.items {
				if m.items[i].Name == it.Name {
					m.items[i].Meta = meta
				}
			}
		}
		m.mode = modeList
		return m, nil
	}

	var cmd tea.Cmd
	m.noteInput, cmd = m.noteInput.Update(msg)
	return m, cmd
}

// Commands

func loadWorktrees(root string, st *store.Store) tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees()
		if err != nil {
			return worktreesLoadedMsg{err: err}
		}

		infos := git.ParallelLoadInfo(wts)
		currentPath := git.CurrentWorktreePath()

		items := make([]item, len(wts))
		for i, wt := range wts {
			name := filepath.Base(wt.Path)
			items[i] = item{
				Worktree:     wt,
				WorktreeInfo: infos[i],
				Meta:         st.Get(name),
				Name:         name,
				Age:          git.TimeAgo(infos[i].LastTime),
				IsCurrent:    wt.Path == currentPath,
			}
		}

		return worktreesLoadedMsg{items: items}
	}
}

func loadBranches() tea.Cmd {
	return func() tea.Msg {
		branches, _ := git.ListBranches()
		return branchesLoadedMsg{branches: branches}
	}
}

func addWorktreeCmd(root, branch, base string) tea.Cmd {
	return func() tea.Msg {
		slug := strings.ReplaceAll(branch, "/", "-")
		path := filepath.Join(filepath.Dir(root), slug)
		if git.IsBareRepo() {
			path = filepath.Join(root, slug)
		}

		createBranch := !git.BranchExists(branch)
		err := git.AddWorktree(path, branch, base, createBranch)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{msg: "created worktree: " + branch}
	}
}

func deleteWorktreeCmd(path, branch string, deleteBranch bool) tea.Cmd {
	return func() tea.Msg {
		if err := git.RemoveWorktree(path, true); err != nil {
			return actionDoneMsg{err: err}
		}
		if deleteBranch && branch != "" {
			git.DeleteBranch(branch)
		}
		msg := "deleted worktree"
		if deleteBranch {
			msg += " + branch"
		}
		return actionDoneMsg{msg: msg}
	}
}

func pruneCmd(paths, branches []string) tea.Cmd {
	return func() tea.Msg {
		for _, p := range paths {
			git.RemoveWorktree(p, true)
		}
		for _, b := range branches {
			git.DeleteBranch(b)
		}
		return actionDoneMsg{msg: fmt.Sprintf("pruned %d worktree(s)", len(paths))}
	}
}

// Helpers

func (m *Model) filterBranches() {
	query := m.addInput.Value()
	if query == "" {
		m.addMatches = m.branches
		m.addIdx = 0
		return
	}

	matches := fuzzy.Find(query, m.branches)
	m.addMatches = make([]string, len(matches))
	for i, match := range matches {
		m.addMatches[i] = match.Str
	}
	if m.addIdx >= len(m.addMatches) {
		m.addIdx = 0
	}
}

func (m Model) visibleItems() []item {
	if m.filterText == "" {
		return m.items
	}
	var out []item
	lower := strings.ToLower(m.filterText)
	for _, it := range m.items {
		if strings.Contains(strings.ToLower(it.Name), lower) ||
			strings.Contains(strings.ToLower(it.Branch), lower) {
			out = append(out, it)
		}
	}
	return out
}

func (m *Model) buildPruneList() {
	m.pruneItems = nil
	for _, it := range m.items {
		if it.IsMain {
			continue
		}
		if it.Remote.Gone {
			m.pruneItems = append(m.pruneItems, pruneItem{item: it, reason: "gone"})
		} else if git.IsMerged(it.Branch, m.baseBranch) {
			m.pruneItems = append(m.pruneItems, pruneItem{item: it, reason: "merged"})
		}
	}
}

func colorIndex(name string) int {
	for i, c := range colorPalette {
		if c.Name == name {
			return i
		}
	}
	return len(colorPalette) - 1
}

func iconIndex(icon string) int {
	if icon == "" {
		return len(iconPalette) - 1
	}
	for i, ic := range iconPalette {
		if ic == icon {
			return i
		}
	}
	return len(iconPalette) - 1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
