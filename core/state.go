package core

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// List Items

type ProjectItem struct {
	Path  string
	Index int
}

func (p ProjectItem) FilterValue() string { return p.Path }
func (p ProjectItem) Title() string       { return fmt.Sprintf("%d. %s", p.Index+1, filepath.Base(p.Path)) }
func (p ProjectItem) Description() string { return p.Path }

type WorktreeItem struct {
	Worktree
}

func (w WorktreeItem) FilterValue() string { return w.Branch }
func (w WorktreeItem) Title() string       { return w.Branch }
func (w WorktreeItem) Description() string {
	status := "Dirty"
	if w.Clean {
		status = "Clean"
	}
	if w.IsMain {
		status += " [Main]"
	}
	return status
}

// Worktree represents a git worktree (moved from model.go)
type Worktree struct {
	Branch string
	Path   string
	Clean  bool
	IsMain bool
}

// MergeHistory (moved from model.go)
type MergeHistory struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Pushed    bool   `json:"pushed"`
	Timestamp string `json:"timestamp"`
}

// KeyMap defines keybindings
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Enter   key.Binding
	Back    key.Binding
	Quit    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Add     key.Binding
	Delete  key.Binding
	Merge   key.Binding
	History key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Enter},
		{k.Add, k.Delete, k.Merge, k.Refresh, k.History},
		{k.Help, k.Quit},
	}
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "projects"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "worktrees"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Add: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "add project"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Merge: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "merge"),
		),
		History: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "history"),
		),
	}
}

// Model represents the application state
type Model struct {
	ConfigDir      string
	Projects       []string // Raw strings
	NewProjectPath string

	// Lists
	ProjectList  list.Model
	WorktreeList list.Model

	// Focused State
	FocusedPanel string // "projects" or "worktrees"

	// Dialogs & Modals
	ShowAddDialog   bool
	ShowMergeDialog bool
	ShowPushPrompt  bool
	ShowFilePicker  bool
	ShowHistory     bool

	// Inputs
	FilePicker filepicker.Model
	PathInput  textinput.Model

	// Merge State
	MergeSource     string
	MergeTarget     string
	MergeDialogStep int        // 0=select source, 1=select target, 2=preview
	MergeBranches   []string   // Cached branches for merge dialog
	MergeList       list.Model // List for selecting branches in merge dialog
	MergeCommits    []string   // Preview commits

	// History
	History     []MergeHistory
	HistoryList list.Model // List for history items

	// Feedback
	Message        string
	LoadingMessage string
	Err            error
	Spinner        spinner.Model
	Loading        bool
	Pushing        bool
	Replaying      bool
	ReplayLog      string
	ReplayPush     bool   // Track if we need to push after merge
	ReplayTarget   string // Track target for push
	ReplayProject  string // Track project for push
	ShowReplayLog  bool   // Whether to show the replay log overlay

	// Components
	Help help.Model
	Keys KeyMap

	// Window Size
	Width  int
	Height int
}

func (m Model) Init() tea.Cmd {
	var cmd tea.Cmd
	if len(m.Projects) > 0 {
		cmd = getWorktreesCmd(m.Projects[0])
		// We can't easily set m.Loading = true here because Init returns Cmd, not Model.
		// But the Spinner tick will start.
		// To show loading state immediately, we rely on the fact that m.Loading is false initially,
		// but we can't mutate m here.
		// However, the command will fire WorktreesLoadedMsg, which sets Loading=false.
		// So we might miss the "Loading" indicator for the initial load unless we set it in NewModel
		// or handle a "StartLoading" msg.
		// For simplicity, just firing the cmd is enough to load data.
	}

	return tea.Batch(
		m.Spinner.Tick,
		cmd,
	)
}

func NewModel(configDir string, projects []string, history []MergeHistory) Model {
	// Project List
	pItems := make([]list.Item, len(projects))
	for i, p := range projects {
		pItems[i] = ProjectItem{Path: p, Index: i}
	}

	pList := list.New(pItems, list.NewDefaultDelegate(), 0, 0)
	pList.Title = "Projects"
	pList.Styles.Title = TitleStyle
	pList.SetShowHelp(false)
	pList.SetShowStatusBar(false)
	pList.DisableQuitKeybindings()

	// Worktree List
	wtList := list.New([]list.Item{}, NewWorktreeDelegate(), 0, 0)
	wtList.Title = "Worktrees"
	wtList.Styles.Title = TitleStyle
	wtList.SetShowHelp(false)
	wtList.SetShowStatusBar(false)
	wtList.DisableQuitKeybindings()

	// Merge List (reused for source/target selection)
	mDelegate := list.NewDefaultDelegate()
	mDelegate.Styles.SelectedTitle = SelectedTitleStyle
	mDelegate.Styles.SelectedDesc = SelectedDescStyle
	mList := list.New([]list.Item{}, mDelegate, 0, 0)
	mList.SetShowHelp(false)
	mList.SetShowTitle(false)
	mList.SetShowStatusBar(false)
	mList.DisableQuitKeybindings()

	// History List
	hItems := make([]list.Item, len(history))
	for i, h := range history {
		// Reverse order (newest first)
		hItems[len(history)-1-i] = HistoryItem{h}
	}
	hList := list.New(hItems, list.NewDefaultDelegate(), 0, 0)
	hList.Title = "Merge History"
	hList.SetShowHelp(false)
	hList.DisableQuitKeybindings()

	// File Picker
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	home, _ := os.UserHomeDir()
	fp.CurrentDirectory = home

	// Text Input
	ti := textinput.New()
	ti.Placeholder = "/path/to/git/repo"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(Lavender)
	ti.TextStyle = lipgloss.NewStyle().Foreground(Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(Mauve)

	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(Mauve)

	// Set initial loading state if projects exist
	loading := false
	if len(projects) > 0 {
		loading = true
	}

	return Model{
		ConfigDir:    configDir,
		Projects:     projects,
		ProjectList:  pList,
		WorktreeList: wtList,
		MergeList:    mList,
		HistoryList:  hList,
		FocusedPanel: "projects",
		FilePicker:   fp,
		PathInput:    ti,
		Spinner:      s,
		Loading:      loading,
		Help:         help.New(),
		Keys:         DefaultKeyMap(),
		History:      history,
	}
}

// Custom Delegates

type WorktreeDelegate struct{}

func NewWorktreeDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	d.Styles.SelectedTitle = SelectedTitleStyle
	d.Styles.SelectedDesc = SelectedDescStyle
	return d
}

// History Item
type HistoryItem struct {
	MergeHistory
}

func (h HistoryItem) FilterValue() string { return h.Project }
func (h HistoryItem) Title() string {
	return fmt.Sprintf("%s: %s → %s", filepath.Base(h.Project), h.Source, h.Target)
}
func (h HistoryItem) Description() string {
	status := "Merged"
	if h.Pushed {
		status += " & Pushed"
	}
	return fmt.Sprintf("%s (%s)", status, h.Timestamp)
}
