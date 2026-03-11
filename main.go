package main

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gitwt/core"
)

var (
	backgroundColor = lipgloss.Color("#1e1e2e")
	textColor       = lipgloss.Color("#cdd6f4")
	secondaryColor  = lipgloss.Color("#a6e3a1")
)

func main() {
	configDir, err := core.GetConfigDir()
	if err != nil {
		println("Error getting config dir:", err.Error())
		os.Exit(1)
	}

	projects, err := core.LoadProjects(configDir)
	if err != nil {
		projects = []string{}
	}

	// Initialize FilePicker
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	home, _ := os.UserHomeDir()
	fp.CurrentDirectory = home

	// Initialize PathInput with styling
	ti := textinput.New()
	ti.Placeholder = "/path/to/git/repo"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(secondaryColor).Background(backgroundColor)
	ti.TextStyle = lipgloss.NewStyle().Foreground(textColor).Background(backgroundColor)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Background(backgroundColor)

	// Load history
	history, _ := core.LoadHistory(configDir)

	m := core.Model{
		Projects:        projects,
		ConfigDir:       configDir,
		SelectedIndex:   0,
		FocusedPanel:    "projects",
		Worktrees:       []core.Worktree{},
		ShowAddDialog:   false,
		ShowMergeDialog: false,
		ShowPushPrompt:  false,
		NewProjectPath:  "",
		MergeSource:     "",
		MergeTarget:     "",
		Message:         "",
		Err:             nil,
		FilePicker:      fp,
		PathInput:       ti,
		MergeDialogStep: 0,
		MergeDialogIdx:  0,
		History:         history,
		HistoryIdx:      0,
		ShowHistory:     false,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error running program:", err.Error())
		os.Exit(1)
	}
}
