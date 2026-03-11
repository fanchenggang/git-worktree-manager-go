package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Worktree represents a git worktree
type Worktree struct {
	Branch string
	Path   string
	Clean  bool
	IsMain bool
}

// MergeHistory represents a merge operation record
type MergeHistory struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Pushed    bool   `json:"pushed"`
	Timestamp string `json:"timestamp"`
}

// Model represents the application state
type Model struct {
	Projects        []string
	ConfigDir       string
	SelectedIndex   int
	FocusedPanel    string // "projects" or "worktrees"
	Worktrees       []Worktree
	ShowAddDialog   bool
	ShowMergeDialog bool
	ShowPushPrompt  bool
	NewProjectPath  string
	MergeSource     string
	MergeTarget     string
	Message         string
	Err             error
	ShowFilePicker  bool
	FilePicker      filepicker.Model
	PathInput       textinput.Model
	MergeDialogStep int            // 0=select source, 1=select target
	MergeDialogIdx  int            // current selection index in merge dialog
	History         []MergeHistory // merge history
	HistoryIdx      int            // selected history index
	ShowHistory     bool           // show history panel
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	home, _ := os.UserHomeDir()
	fp.CurrentDirectory = home
	m.FilePicker = fp

	// Initialize path input
	ti := textinput.New()
	ti.Placeholder = "/path/to/git/repo"
	ti.Focus()
	m.PathInput = ti

	return nil
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle path input when add dialog is shown
	if m.ShowAddDialog {
		m.PathInput, _ = m.PathInput.Update(msg)
		m.NewProjectPath = m.PathInput.Value()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		newModel, cmd := m.handleKeyMsg(msg)
		m = newModel.(Model)
		return m, cmd
	case tea.WindowSizeMsg:
		return m, nil
	}

	// Handle file picker updates when shown (after handleKeyMsg may have set it)
	if m.ShowFilePicker {
		// Handle key events for dialog control
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" || keyMsg.String() == "q" {
				m.ShowFilePicker = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.FilePicker, cmd = m.FilePicker.Update(msg)
		if m.FilePicker.Path != "" {
			// File/directory was selected
			m.NewProjectPath = m.FilePicker.Path
			m.PathInput.SetValue(m.FilePicker.Path)
			m.ShowFilePicker = false
			m.FilePicker.Path = "" // Reset for next time
		}
		return m, cmd
	}

	return m, nil
}

var (
	// Colors
	backgroundColor = lipgloss.Color("#1e1e2e")
	primaryColor    = lipgloss.Color("#89b4fa")
	secondaryColor  = lipgloss.Color("#a6e3a1")
	accentColor     = lipgloss.Color("#f38ba8")
	textColor       = lipgloss.Color("#cdd6f4")
	mutedColor      = lipgloss.Color("#6c7086")

	// Styles
	headerStyle = lipgloss.NewStyle().
			Background(primaryColor).
			Foreground(backgroundColor).
			Bold(true).
			Padding(0, 1)

	panelStyle = lipgloss.NewStyle().
			Background(backgroundColor).
			Foreground(textColor).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)

	listItemStyle = lipgloss.NewStyle().
			Foreground(textColor)

	selectedItemStyle = lipgloss.NewStyle().
				Background(primaryColor).
				Foreground(backgroundColor).
				Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	statusCleanStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	statusDirtyStyle = lipgloss.NewStyle().
				Foreground(accentColor)

	dialogBaseStyle = lipgloss.NewStyle().
			Background(backgroundColor).
			Border(lipgloss.RoundedBorder()).
			Padding(2)
)

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.ShowFilePicker || m.ShowAddDialog || m.ShowMergeDialog || m.ShowPushPrompt {
			m.closeAllDialogs()
		} else {
			return m, tea.Quit
		}
	case "esc":
		if m.ShowFilePicker || m.ShowAddDialog || m.ShowMergeDialog || m.ShowPushPrompt {
			m.closeAllDialogs()
		}
	case "j", "down":
		if m.ShowAddDialog {
			// Don't navigate in dialog
		} else if m.ShowMergeDialog {
			branches := m.getAllBranches()
			// Filter out source branch if in step 1
			if m.MergeDialogStep == 1 {
				var filtered []string
				for _, b := range branches {
					if b != m.MergeSource {
						filtered = append(filtered, b)
					}
				}
				branches = filtered
			}
			if len(branches) > 0 {
				m.MergeDialogIdx = (m.MergeDialogIdx + 1) % len(branches)
			}
		} else if m.ShowHistory && len(m.History) > 0 {
			m.HistoryIdx = (m.HistoryIdx + 1) % len(m.History)
		} else if m.FocusedPanel == "projects" {
			if m.SelectedIndex < len(m.Projects)-1 {
				m.SelectedIndex++
				m.loadWorktrees()
			}
		} else if m.FocusedPanel == "worktrees" && len(m.Worktrees) > 0 {
			m.SelectedIndex = (m.SelectedIndex + 1) % len(m.Projects)
		}
	case "k", "up":
		if m.ShowAddDialog || m.ShowPushPrompt {
			// Don't navigate in dialog
		} else if m.ShowMergeDialog {
			branches := m.getAllBranches()
			// Filter out source branch if in step 1
			if m.MergeDialogStep == 1 {
				var filtered []string
				for _, b := range branches {
					if b != m.MergeSource {
						filtered = append(filtered, b)
					}
				}
				branches = filtered
			}
			if len(branches) > 0 {
				m.MergeDialogIdx--
				if m.MergeDialogIdx < 0 {
					m.MergeDialogIdx = len(branches) - 1
				}
			}
		} else if m.ShowHistory && len(m.History) > 0 {
			m.HistoryIdx--
			if m.HistoryIdx < 0 {
				m.HistoryIdx = len(m.History) - 1
			}
		} else if m.FocusedPanel == "projects" {
			if m.SelectedIndex > 0 {
				m.SelectedIndex--
				m.loadWorktrees()
			}
		}
	case "left":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.FocusedPanel = "projects"
		}
	case "right", "l":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt && len(m.Projects) > 0 {
			m.FocusedPanel = "worktrees"
		}
	case "enter":
		if m.ShowAddDialog && !m.ShowFilePicker {
			m.addProject()
		} else if m.ShowMergeDialog {
			m.executeMerge()
		} else if m.ShowPushPrompt {
			m.handlePushResponse()
		} else if m.ShowHistory && len(m.History) > 0 {
			m.replayMerge()
		}
	case "n":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.ShowAddDialog = true
			m.NewProjectPath = ""
			m.PathInput.SetValue("")
			m.PathInput.Focus()
		}
	case "d":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt && m.FocusedPanel == "projects" && len(m.Projects) > 0 {
			m.deleteProject()
		}
	case "m":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt && len(m.Projects) > 0 {
			m.ShowMergeDialog = true
			m.MergeSource = ""
			m.MergeTarget = ""
			m.MergeDialogStep = 0
			m.MergeDialogIdx = 0
			m.Message = "Select source branch"
		}
	case "r":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.loadWorktrees()
			m.Message = "Refreshed"
		}
	case "h":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.ShowHistory = !m.ShowHistory
			m.HistoryIdx = 0
		}
	case "y":
		if m.ShowPushPrompt {
			m.MergeSource = "yes"
			m.handlePushResponse()
		}
	case "N":
		if m.ShowPushPrompt {
			m.MergeSource = "no"
			m.handlePushResponse()
		}
	}

	// Handle typing in dialogs
	if m.ShowAddDialog {
		// Handle newProjectPath input - handled by key events
	}

	return m, nil
}

func (m *Model) addProject() {
	path := strings.TrimSpace(m.NewProjectPath)
	if path == "" {
		m.Message = "Please enter a path"
		return
	}

	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = strings.Replace(path, "~", home, 1)
	}

	// Validate it's a git repository
	if !isGitRepo(path) {
		m.Message = "Not a valid git repository"
		return
	}

	// Check if already exists
	for _, p := range m.Projects {
		if p == path {
			m.Message = "Project already exists"
			return
		}
	}

	m.Projects = append(m.Projects, path)
	err := saveProjects(m.ConfigDir, m.Projects)
	if err != nil {
		m.Message = "Failed to save: " + err.Error()
		return
	}

	m.ShowAddDialog = false
	m.NewProjectPath = ""
	m.SelectedIndex = len(m.Projects) - 1
	m.loadWorktrees()
	m.Message = "Project added"
}

func (m *Model) deleteProject() {
	if m.SelectedIndex < 0 || m.SelectedIndex >= len(m.Projects) {
		return
	}

	m.Projects = append(m.Projects[:m.SelectedIndex], m.Projects[m.SelectedIndex+1:]...)
	err := saveProjects(m.ConfigDir, m.Projects)
	if err != nil {
		m.Message = "Failed to save: " + err.Error()
		return
	}

	if m.SelectedIndex >= len(m.Projects) && len(m.Projects) > 0 {
		m.SelectedIndex = len(m.Projects) - 1
	}
	m.Worktrees = []Worktree{}
	m.Message = "Project removed"
}

func (m *Model) loadWorktrees() {
	if m.SelectedIndex < 0 || m.SelectedIndex >= len(m.Projects) {
		m.Worktrees = []Worktree{}
		return
	}

	repoPath := m.Projects[m.SelectedIndex]
	worktrees, err := getWorktrees(repoPath)
	if err != nil {
		m.Worktrees = []Worktree{}
		m.Message = "Error loading worktrees: " + err.Error()
		return
	}
	m.Worktrees = worktrees
}

func (m *Model) executeMerge() {
	branches := m.getAllBranches()
	if len(branches) == 0 {
		m.Message = "No branches found"
		return
	}

	// Ensure index is valid
	if m.MergeDialogIdx < 0 || m.MergeDialogIdx >= len(branches) {
		m.MergeDialogIdx = 0
	}

	selectedBranch := branches[m.MergeDialogIdx]

	if m.MergeDialogStep == 0 {
		// First step: select source branch
		m.MergeSource = selectedBranch
		m.MergeDialogStep = 1
		m.MergeDialogIdx = 0
		m.Message = "Select target branch"
	} else {
		// Second step: select target branch and execute merge
		m.MergeTarget = selectedBranch
		repoPath := m.Projects[m.SelectedIndex]
		err := gitMerge(repoPath, m.MergeSource, m.MergeTarget)
		if err != nil {
			m.Message = "Merge failed: " + err.Error()
			m.ShowMergeDialog = false
			return
		}
		m.ShowMergeDialog = false
		m.ShowPushPrompt = true
		m.Message = "Merge successful! Push to remote?"
	}
}

func (m *Model) handlePushResponse() {
	pushed := false
	if m.MergeSource == "yes" {
		// Push target branch
		repoPath := m.Projects[m.SelectedIndex]
		err := gitPush(repoPath, m.MergeTarget)
		if err != nil {
			m.Message = "Push failed: " + err.Error()
		} else {
			m.Message = "Pushed successfully"
			pushed = true
		}
	} else {
		m.Message = "Merge completed (not pushed)"
	}

	// Save to history
	history := MergeHistory{
		ID:        fmt.Sprintf("%d", time.Now().Unix()),
		Project:   m.Projects[m.SelectedIndex],
		Source:    m.MergeSource,
		Target:    m.MergeTarget,
		Pushed:    pushed,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
	}
	err := AddHistory(m.ConfigDir, history)
	if err == nil {
		m.History, _ = LoadHistory(m.ConfigDir)
	}

	m.ShowPushPrompt = false
	m.MergeSource = ""
	m.MergeTarget = ""
}

func (m *Model) replayMerge() {
	if m.HistoryIdx < 0 || m.HistoryIdx >= len(m.History) {
		return
	}

	entry := m.History[m.HistoryIdx]
	m.ShowHistory = false

	// Execute merge
	err := gitMerge(entry.Project, entry.Source, entry.Target)
	if err != nil {
		m.Message = "Replay merge failed: " + err.Error()
		return
	}

	// If originally pushed, push again
	if entry.Pushed {
		err := gitPush(entry.Project, entry.Target)
		if err != nil {
			m.Message = "Replay push failed: " + err.Error()
		} else {
			m.Message = "Replay: merged and pushed successfully"
		}
	} else {
		m.Message = "Replay: merged successfully (not pushed)"
	}

	m.ShowHistory = false
}

func (m *Model) getAllBranches() []string {
	if m.SelectedIndex < 0 || m.SelectedIndex >= len(m.Projects) {
		return []string{}
	}

	repoPath := m.Projects[m.SelectedIndex]
	branches, err := getBranches(repoPath)
	if err != nil {
		return []string{}
	}
	return branches
}

// View implements tea.Model
func (m Model) View() string {
	// Header
	header := headerStyle.Width(80).Render(" Git Worktree Manager ")

	// Project list panel
	projectList := m.renderProjectList()

	// Worktree list panel
	worktreeList := m.renderWorktreeList()

	// Panels side by side
	panels := lipgloss.JoinHorizontal(
		lipgloss.Top,
		projectList,
		worktreeList,
	)

	// History panel (if enabled)
	if m.ShowHistory {
		historyPanel := m.renderHistoryPanel()
		panels = lipgloss.JoinHorizontal(
			lipgloss.Top,
			panels,
			historyPanel,
		)
	}

	// Help text
	helpText := " [n] Add  [d] Delete  [m] Merge  [r] Refresh  [h] History  [q] Quit "
	if m.ShowHistory {
		helpText = " [↑/↓] Select  [Enter] Replay  [h] Close  [q] Quit "
	}
	help := helpStyle.Render(helpText)

	// Message
	msg := ""
	if m.Message != "" {
		msg = lipgloss.NewStyle().Foreground(secondaryColor).Render(m.Message)
	}

	// Dialogs
	dialog := ""
	if m.ShowFilePicker {
		dialog = m.renderFilePicker()
	} else if m.ShowAddDialog {
		dialog = m.renderAddDialog()
	} else if m.ShowMergeDialog {
		dialog = m.renderMergeDialog()
	} else if m.ShowPushPrompt {
		dialog = m.renderPushPrompt()
	}

	// Combine all
	mainContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		panels,
		help,
	)

	// Show dialog as overlay in center
	if dialog != "" {
		overlay := lipgloss.Place(
			80, // terminal width
			30, // terminal height
			lipgloss.Center,
			lipgloss.Center,
			dialog,
		)
		return overlay
	}

	// Show message at bottom
	if msg != "" {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			mainContent,
			msg,
		)
	}

	return mainContent
}

func (m Model) renderProjectList() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Projects")

	var items []string
	for i, p := range m.Projects {
		name := filepath.Base(p)
		if i == m.SelectedIndex && m.FocusedPanel == "projects" {
			items = append(items, selectedItemStyle.Render(" "+name))
		} else {
			items = append(items, listItemStyle.Render(" "+name))
		}
	}

	if len(items) == 0 {
		items = append(items, listItemStyle.Render(" (no projects)"))
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		[]string{title}...,
	)
	content += "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)

	return panelStyle.Width(25).Height(15).Render(content)
}

func (m Model) renderWorktreeList() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Worktrees")

	if len(m.Projects) == 0 {
		content := title + "\n" + listItemStyle.Render(" (select a project first)")
		return panelStyle.Width(50).Height(15).Render(content)
	}

	var items []string
	for _, wt := range m.Worktrees {
		var status string
		if wt.Clean {
			status = statusCleanStyle.Render("clean")
		} else {
			status = statusDirtyStyle.Render("dirty")
		}
		if wt.IsMain {
			status += " " + lipgloss.NewStyle().Foreground(primaryColor).Render("[main]")
		}
		items = append(items, listItemStyle.Render(" "+wt.Branch+"  "+status))
	}

	if len(items) == 0 {
		items = append(items, listItemStyle.Render(" (no worktrees)"))
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		[]string{title}...,
	)
	content += "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)

	return panelStyle.Width(50).Height(15).Render(content)
}

func (m Model) renderHistoryPanel() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("History")

	if len(m.History) == 0 {
		content := title + "\n" + listItemStyle.Render(" (no history)")
		return panelStyle.Width(40).Height(15).Render(content)
	}

	var items []string
	for i, h := range m.History {
		projectName := filepath.Base(h.Project)
		pushStatus := "○"
		if h.Pushed {
			pushStatus = "✓"
		}
		line := fmt.Sprintf("%s %s → %s [%s]", projectName, h.Source, h.Target, pushStatus)
		if i == m.HistoryIdx {
			items = append(items, selectedItemStyle.Render(" "+line))
		} else {
			items = append(items, listItemStyle.Render(" "+line))
		}
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		[]string{title}...,
	)
	content += "\n" + lipgloss.JoinVertical(lipgloss.Left, items...)

	return panelStyle.Width(40).Height(15).Render(content)
}

func (m Model) renderAddDialog() string {
	dialogStyle := dialogBaseStyle.BorderForeground(primaryColor).Width(60)

	content := ">>> Add Project <<<\n\n"
	content += "Enter project path:\n"
	content += m.PathInput.View() + "\n"
	content += "\n" + helpStyle.Render("Enter to add, Esc to cancel")

	return dialogStyle.Render(content)
}

func (m Model) renderMergeDialog() string {
	dialogStyle := dialogBaseStyle.BorderForeground(primaryColor).Width(90)

	allBranches := m.getAllBranches()
	var availableBranches []string

	stepText := "Select source branch"
	if m.MergeDialogStep == 1 {
		stepText = "Select target branch"
		// Filter out the selected source branch
		for _, b := range allBranches {
			if b != m.MergeSource {
				availableBranches = append(availableBranches, b)
			}
		}
	} else {
		availableBranches = allBranches
	}

	// Validate index
	if len(availableBranches) > 0 {
		if m.MergeDialogIdx < 0 {
			m.MergeDialogIdx = 0
		}
		if m.MergeDialogIdx >= len(availableBranches) {
			m.MergeDialogIdx = len(availableBranches) - 1
		}
	} else {
		m.MergeDialogIdx = 0
	}

	// Left panel: show selected branch(es)
	leftContent := ""
	if m.MergeDialogStep == 0 {
		leftContent = "Source:\n\n  (select a branch)"
	} else {
		leftContent = "Source:\n  " + m.MergeSource
	}

	// Right panel: show available branches
	rightContent := ""
	if m.MergeDialogStep == 0 {
		rightContent = "Available:\n"
	} else {
		rightContent = "Select Target:\n"
	}

	for i, b := range availableBranches {
		prefix := "  "
		if i == m.MergeDialogIdx {
			prefix = "> "
			rightContent += selectedItemStyle.Render(prefix+b) + "\n"
		} else {
			rightContent += listItemStyle.Render(prefix+b) + "\n"
		}
	}

	// Build final content
	var content string
	content += ">>> Merge Branch <<<\n"
	content += "Step: " + stepText + "\n\n"

	// Side by side panels
	leftPanel := lipgloss.NewStyle().Width(30).Render(leftContent)
	rightPanel := lipgloss.NewStyle().Width(40).Render(rightContent)
	content += lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)
	content += "\n\n" + helpStyle.Render("↑/↓ to select, Enter to confirm")

	return dialogStyle.Render(content)
}

func (m Model) renderPushPrompt() string {
	dialogStyle := dialogBaseStyle.BorderForeground(secondaryColor).Width(50)

	content := lipgloss.NewStyle().Bold(true).Foreground(secondaryColor).Render("Merge Successful!\n")
	content += "\nPush to remote?\n"
	content += "\n" + selectedItemStyle.Render(" [y] Yes ")
	content += " " + listItemStyle.Render("[n] No")

	return dialogStyle.Render(content)
}

func (m Model) renderFilePicker() string {
	dialogStyle := dialogBaseStyle.BorderForeground(primaryColor).Width(60).Height(20)

	title := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Select Project Directory\n")
	content := title + m.FilePicker.View()

	return dialogStyle.Render(content)
}

// Helper functions

func (m *Model) closeAllDialogs() {
	m.ShowAddDialog = false
	m.ShowMergeDialog = false
	m.ShowPushPrompt = false
	m.ShowFilePicker = false
	m.Message = ""
	m.MergeDialogStep = 0
	m.MergeDialogIdx = 0
}

func isGitRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir() || info.Name() == ".git"
}

func getWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	lines := strings.Split(string(output), "\n")

	var current Worktree
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current.Branch != "" {
				worktrees = append(worktrees, current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			current = Worktree{Path: path}
		} else if strings.HasPrefix(line, "branch ") {
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			current.Branch = branch
			current.IsMain = (branch == "main" || branch == "master")
		} else if line == "clean" {
			current.Clean = true
		} else if line == "dirty" {
			current.Clean = false
		}
	}

	if current.Branch != "" {
		worktrees = append(worktrees, current)
	}

	// Check main worktree status
	mainPath := repoPath
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = mainPath
	output, _ = cmd.Output()
	if len(strings.TrimSpace(string(output))) == 0 {
		for i := range worktrees {
			if worktrees[i].IsMain {
				worktrees[i].Clean = true
			}
		}
	}

	return worktrees, nil
}

func getBranches(repoPath string) ([]string, error) {
	// Only get branches from worktrees (exclude main/master)
	worktrees, err := getWorktrees(repoPath)
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, wt := range worktrees {
		if wt.Branch != "" && !wt.IsMain {
			branches = append(branches, wt.Branch)
		}
	}

	return branches, nil
}

func gitMerge(repoPath, source, target string) error {
	// Find the worktree path for target branch
	worktrees, err := getWorktrees(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %v", err)
	}

	var targetPath string
	for _, wt := range worktrees {
		if wt.Branch == target {
			targetPath = wt.Path
			break
		}
	}

	// If target is main/master, use repoPath
	if targetPath == "" {
		targetPath = repoPath
	}

	// Merge source into target branch in the target's worktree
	cmd := exec.Command("git", "fetch", ".")
	cmd.Dir = targetPath
	cmd.Run() // ignore fetch errors

	cmd = exec.Command("git", "merge", source)
	cmd.Dir = targetPath
	if out, err := cmd.CombinedOutput(); err != nil {
		// If merge failed, try to abort
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = targetPath
		abortCmd.Run()
		return fmt.Errorf("merge failed: %s", string(out))
	}

	return nil
}

func gitPush(repoPath, branch string) error {
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
