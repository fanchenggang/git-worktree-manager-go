package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"gitwt/core"
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

	// Load history
	history, err := core.LoadHistory(configDir)
	if err != nil {
		history = []core.MergeHistory{}
	}

	m := core.NewModel(configDir, projects, history)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		println("Error running program:", err.Error())
		os.Exit(1)
	}
}
