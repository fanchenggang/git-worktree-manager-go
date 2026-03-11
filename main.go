package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"gitwt/core"
	"os"
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
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error running program:", err.Error())
		os.Exit(1)
	}
}
