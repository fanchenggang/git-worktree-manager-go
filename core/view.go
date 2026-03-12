package core

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	// If window is too small, show a message
	if m.Width < 60 || m.Height < 10 {
		return "Terminal too small"
	}

	// 1. Header
	header := HeaderStyle.Width(m.Width - 4).Render("Git Worktree Manager")

	// 2. Main Content
	var mainContent string

	if m.ShowHistory {
		// Show History List
		mainContent = PanelStyle.Render(m.HistoryList.View())
	} else {
		// Projects Panel
		pStyle := PanelStyle
		if m.FocusedPanel == "projects" {
			pStyle = FocusedPanelStyle
		}
		projectsView := pStyle.Render(m.ProjectList.View())

		// Worktrees Panel
		wStyle := PanelStyle
		if m.FocusedPanel == "worktrees" {
			wStyle = FocusedPanelStyle
		}

		// If loading, show spinner in worktree panel
		var worktreesView string
		if m.Loading && m.FocusedPanel == "projects" {
			// Center the spinner
			msg := m.LoadingMessage
			if msg == "" {
				msg = "Loading..."
			}

			// If replaying, show log
			if m.Replaying {
				worktreesView = wStyle.Width(m.WorktreeList.Width()).Height(m.WorktreeList.Height()).Render(
					lipgloss.NewStyle().
						Width(m.WorktreeList.Width()).
						Height(m.WorktreeList.Height()).
						Padding(1).
						Render(m.ReplayLog + "\n" + m.Spinner.View()),
				)
			} else {
				worktreesView = wStyle.Width(m.WorktreeList.Width()).Height(m.WorktreeList.Height()).Render(
					lipgloss.Place(
						m.WorktreeList.Width(),
						m.WorktreeList.Height(),
						lipgloss.Center,
						lipgloss.Center,
						m.Spinner.View()+" "+msg,
					),
				)
			}
		} else {
			worktreesView = wStyle.Render(m.WorktreeList.View())
		}

		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			projectsView,
			worktreesView,
		)
	}

	// 3. Status / Message
	status := ""
	if m.Message != "" {
		if m.Err != nil {
			status = ErrorStyle.Render(m.Message)
		} else {
			status = DescStyle.Render(m.Message)
		}
	}

	// 4. Help
	m.Help.Width = m.Width
	helpView := HelpStyle.Render(m.Help.View(m.Keys))

	// Combine Layout
	ui := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		mainContent,
		status,
		helpView,
	)

	// 5. Overlays (Dialogs)

	if m.ShowAddDialog {
		dialog := DialogStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			DialogTitleStyle.Render("Add Project"),
			"Enter path:",
			m.PathInput.View(),
			HelpStyle.Render("Enter to confirm • Esc to cancel"),
		))
		return placeOverlay(m.Width, m.Height, ui, dialog)
	}

	if m.ShowFilePicker {
		dialog := DialogStyle.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			DialogTitleStyle.Render("Select Directory"),
			m.FilePicker.View(),
		))
		return placeOverlay(m.Width, m.Height, ui, dialog)
	}

	if m.ShowMergeDialog {
		// Calculate dynamic width
		dialogWidth := m.Width - 10
		if dialogWidth > 80 {
			dialogWidth = 80
		} else if dialogWidth < 40 {
			dialogWidth = 40
		}

		// Stepper Layout
		step0 := InactiveStepStyle
		step1 := InactiveStepStyle
		step2 := InactiveStepStyle

		if m.MergeDialogStep == 0 {
			step0 = ActiveStepStyle
		} else if m.MergeDialogStep == 1 {
			step1 = ActiveStepStyle
		} else {
			step2 = ActiveStepStyle
		}

		stepper := lipgloss.JoinHorizontal(
			lipgloss.Left,
			step0.Render("1. Source"),
			StepSeparatorStyle.Render(),
			step1.Render("2. Target"),
			StepSeparatorStyle.Render(),
			step2.Render("3. Confirm"),
		)

		// Ensure stepper fits
		stepper = lipgloss.NewStyle().Width(dialogWidth - 4).Align(lipgloss.Center).Render(stepper)

		var content string
		var helpText string

		// Content
		if m.MergeDialogStep == 2 {
			// Preview Step
			content = "Merging " + SelectedTitleStyle.Render(m.MergeSource) +
				" into " + SelectedTitleStyle.Render(m.MergeTarget) + "\n\n"

			if m.Loading {
				content += m.Spinner.View() + " Loading commits..."
			} else if len(m.MergeCommits) > 0 {
				content += "Commits to be merged:\n"
				for _, c := range m.MergeCommits {
					// Truncate long lines
					if len(c) > dialogWidth-6 {
						c = c[:dialogWidth-9] + "..."
					}
					content += PreviewCommitStyle.Render("• "+c) + "\n"
				}
			} else {
				content += "No commits to merge (or already up to date)."
			}
			helpText = "Enter to Merge • Esc to Cancel"
		} else {
			// Selection Steps
			title := "Select Source Branch"
			if m.MergeDialogStep == 1 {
				title = "Select Target Branch"
			}
			m.MergeList.Title = title
			m.MergeList.SetWidth(dialogWidth - 6)

			// Dynamic height: Try to fill ~50% of screen height, but at least 10 items (20 lines + chrome)
			// Each item is 2 lines (title + desc)
			// Chrome (pagination/help) is ~2 lines
			// So for 5 items we need 12 lines
			listHeight := m.Height / 2
			if listHeight < 14 {
				listHeight = 14 // Minimum for 5 items + pagination
			}
			if listHeight > m.Height-10 {
				listHeight = m.Height - 10 // Cap at screen height - dialog chrome
			}

			m.MergeList.SetHeight(listHeight)
			content = m.MergeList.View()
			helpText = "Enter to Select • Esc to Cancel"
		}

		dialog := DialogStyle.Width(dialogWidth).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			DialogTitleStyle.Render("Merge Wizard"),
			stepper,
			"\n",
			content,
			HelpStyle.Render(helpText),
		))
		return placeOverlay(m.Width, m.Height, ui, dialog)
	}

	if m.ShowReplayLog {
		dialog := DialogStyle.Width(m.Width - 10).Render(lipgloss.JoinVertical(
			lipgloss.Left,
			DialogTitleStyle.Render("Replay Log"),
			lipgloss.NewStyle().
				Width(m.Width-14).
				Height(m.Height-10).
				Render(m.ReplayLog),
			HelpStyle.Render("Esc/Enter to Close"),
		))
		return placeOverlay(m.Width, m.Height, ui, dialog)
	}

	if m.ShowPushPrompt {
		var content string
		var helpText string

		if m.Pushing {
			content = m.Spinner.View() + " Pushing to remote..."
			helpText = "Please wait..."
		} else {
			content = "Merge was successful."
			helpText = HelpStyle.Render("Enter to Push • Esc to Skip")
		}

		dialog := DialogStyle.Width(40).Render(lipgloss.JoinVertical(
			lipgloss.Center,
			DialogTitleStyle.Render("Push to Remote?"),
			content,
			"",
			helpText,
		))
		return placeOverlay(m.Width, m.Height, ui, dialog)
	}

	return AppStyle.Render(ui)
}

func placeOverlay(width, height int, base, overlay string) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		overlay,
	)
}
