package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/filepicker"
)

// Worktree represents a git worktree
type Worktree struct {
	Branch string
	Path   string
	Clean  bool
	IsMain bool
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
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	home, _ := os.UserHomeDir()
	fp.CurrentDirectory = home
	m.FilePicker = fp
	return nil
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle file picker updates when shown
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
			m.ShowFilePicker = false
			m.FilePicker.Path = "" // Reset for next time
		}
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m, nil
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
			// Navigate in merge dialog
		} else if m.FocusedPanel == "projects" {
			if m.SelectedIndex < len(m.Projects)-1 {
				m.SelectedIndex++
				m.loadWorktrees()
			}
		} else if m.FocusedPanel == "worktrees" && len(m.Worktrees) > 0 {
			m.SelectedIndex = (m.SelectedIndex + 1) % len(m.Projects)
		}
	case "k", "up":
		if m.ShowAddDialog || m.ShowMergeDialog || m.ShowPushPrompt {
			// Don't navigate in dialog
		} else if m.FocusedPanel == "projects" {
			if m.SelectedIndex > 0 {
				m.SelectedIndex--
				m.loadWorktrees()
			}
		}
	case "left", "h":
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
		}
	case "tab":
		if m.ShowAddDialog && !m.ShowFilePicker {
			m.ShowFilePicker = true
		}
	case "n":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.ShowAddDialog = true
			m.NewProjectPath = ""
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
			m.Message = "Select source branch"
		}
	case "r":
		if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt {
			m.loadWorktrees()
			m.Message = "Refreshed"
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
	if m.MergeSource == "" {
		// Show branch selection - first select source
		branches := m.getAllBranches()
		if len(branches) == 0 {
			m.Message = "No branches found"
			return
		}
		m.Message = "Select target branch"
		return
	}

	if m.MergeTarget == "" {
		m.Message = "Select target branch"
		return
	}

	// Execute merge
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

func (m *Model) handlePushResponse() {
	if m.MergeSource == "yes" {
		// Push target branch
		repoPath := m.Projects[m.SelectedIndex]
		err := gitPush(repoPath, m.MergeTarget)
		if err != nil {
			m.Message = "Push failed: " + err.Error()
		} else {
			m.Message = "Pushed successfully"
		}
	} else {
		m.Message = "Merge completed (not pushed)"
	}
	m.ShowPushPrompt = false
	m.MergeSource = ""
	m.MergeTarget = ""
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
	header := headerStyle.Width(60).Render(" Git Worktree Manager ")

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

	// Help text
	help := helpStyle.Render(" [n] Add project  [d] Delete  [m] Merge  [r] Refresh  [q] Quit ")

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
		msg,
	)

	if dialog != "" {
		mainContent = lipgloss.JoinVertical(
			lipgloss.Left,
			mainContent,
			dialog,
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

func (m Model) renderAddDialog() string {
	dialogStyle := dialogBaseStyle.BorderForeground(primaryColor).Width(50)

	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Add Project\n")
	content += "\nEnter project path:\n"
	content += lipgloss.NewStyle().Foreground(secondaryColor).Render("> " + m.NewProjectPath + "\n")
	content += "\n" + helpStyle.Render("Press Tab to browse, Enter to add, Esc to cancel")

	return dialogStyle.Render(content)
}

func (m Model) renderMergeDialog() string {
	dialogStyle := dialogBaseStyle.BorderForeground(primaryColor).Width(50)

	branches := m.getAllBranches()
	content := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("Merge Branch\n")
	content += "\nAvailable branches:\n"

	for i, b := range branches {
		prefix := "  "
		if i == 0 {
			prefix = "> "
			content += selectedItemStyle.Render(prefix+b) + "\n"
		} else {
			content += listItemStyle.Render(prefix+b) + "\n"
		}
	}

	content += "\n" + helpStyle.Render("Use j/k to select, Enter to confirm")

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
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		if line != "" && !strings.HasPrefix(line, "remotes/") {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

func gitMerge(repoPath, source, target string) error {
	// First checkout to target branch
	cmd := exec.Command("git", "checkout", target)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return err
	}

	// Then merge source
	cmd = exec.Command("git", "merge", source)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		// If merge failed, try to abort
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = repoPath
		abortCmd.Run()
		return err
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
