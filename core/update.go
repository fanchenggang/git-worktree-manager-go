package core

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		// Update list sizes
		// Use proportional sizing
		// Header ~3 lines, Status ~2 lines
		listHeight := msg.Height - 6
		if listHeight < 5 {
			listHeight = 5
		}

		m.ProjectList.SetWidth(msg.Width / 3)
		m.ProjectList.SetHeight(listHeight)

		m.WorktreeList.SetWidth(msg.Width*2/3 - 4)
		m.WorktreeList.SetHeight(listHeight)

		m.HistoryList.SetWidth(msg.Width - 4)
		m.HistoryList.SetHeight(listHeight)

		m.MergeList.SetWidth(msg.Width / 2)
		m.MergeList.SetHeight(listHeight / 2)

		return m, nil

	case tea.KeyMsg:
		// Global Keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			// Only quit if no dialogs are open and not typing in filter
			if !m.ShowAddDialog && !m.ShowMergeDialog && !m.ShowPushPrompt && !m.ShowFilePicker && !m.ShowHistory && !m.ShowReplayLog {
				if m.ProjectList.FilterState() != list.Filtering && m.WorktreeList.FilterState() != list.Filtering {
					return m, tea.Quit
				}
			}
		}

		if m.ShowReplayLog {
			switch msg.String() {
			case "esc", "enter":
				m.ShowReplayLog = false
				return m, nil
			}
			return m, nil
		}

		// Dialog Handling
		if m.ShowFilePicker {
			var cmd tea.Cmd
			m.FilePicker, cmd = m.FilePicker.Update(msg)
			if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
				m.NewProjectPath = path
				m.PathInput.SetValue(path)
				m.ShowFilePicker = false
			}
			return m, cmd
		}

		if m.ShowAddDialog {
			switch msg.String() {
			case "enter":
				return m.addProject()
			case "esc":
				m.ShowAddDialog = false
				return m, nil
			}
			var cmd tea.Cmd
			m.PathInput, cmd = m.PathInput.Update(msg)
			m.NewProjectPath = m.PathInput.Value()
			return m, cmd
		}

		if m.ShowMergeDialog {
			switch msg.String() {
			case "enter":
				if m.MergeDialogStep == 2 {
					// Preview Step -> Confirm
					return m.executeMerge()
				}

				// Select item from list
				i := m.MergeList.SelectedItem()
				if i != nil {
					branch := i.(WorktreeItem).Branch
					if m.MergeDialogStep == 0 {
						// Source Selected
						m.MergeSource = branch
						m.MergeDialogStep = 1
						m = m.updateMergeList(m.MergeSource) // Filter out source

						// If target was already set (default), maybe auto-select it?
						// For now, let user confirm target.
					} else if m.MergeDialogStep == 1 {
						// Target Selected
						m.MergeTarget = branch
						m.MergeDialogStep = 2

						// Load Preview
						if i := m.ProjectList.SelectedItem(); i != nil {
							path := i.(ProjectItem).Path
							m.Loading = true
							m.MergeCommits = []string{}
							return m, tea.Batch(m.Spinner.Tick, getLogCmd(path, m.MergeTarget, m.MergeSource))
						}
					}
				}
				return m, nil
			case "esc":
				if m.MergeDialogStep > 0 {
					// Go back
					m.MergeDialogStep--
					if m.MergeDialogStep == 0 {
						m = m.updateMergeList("")
					} else if m.MergeDialogStep == 1 {
						m = m.updateMergeList(m.MergeSource)
					}
					return m, nil
				}
				m.ShowMergeDialog = false
				return m, nil
			}

			if m.MergeDialogStep < 2 {
				var cmd tea.Cmd
				m.MergeList, cmd = m.MergeList.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		if m.ShowPushPrompt {
			switch msg.String() {
			case "enter":
				return m.handlePushResponse(true)
			case "esc", "n", "N":
				return m.handlePushResponse(false)
			}
			return m, nil
		}

		if m.ShowHistory {
			switch msg.String() {
			case "esc", "h":
				m.ShowHistory = false
				return m, nil
			case "enter":
				// Replay
				if i := m.HistoryList.SelectedItem(); i != nil {
					h := i.(HistoryItem).MergeHistory
					m.Loading = true
					m.Replaying = true
					m.ReplayPush = h.Pushed
					m.ReplayTarget = h.Target
					m.ReplayProject = h.Project
					m.ReplayLog = fmt.Sprintf("Starting replay: %s -> %s\n\n[1/3] Fetching...\n", h.Source, h.Target)
					m.ShowHistory = false
					return m, tea.Batch(m.Spinner.Tick, replayFetchCmd(h.Project))
				}
			}
			var cmd tea.Cmd
			m.HistoryList, cmd = m.HistoryList.Update(msg)
			return m, cmd
		}

		// Panel Navigation (only if not filtering)
		if m.ProjectList.FilterState() != list.Filtering && m.WorktreeList.FilterState() != list.Filtering {
			switch msg.String() {
			case "1", "2", "3", "4", "5", "6", "7", "8", "9":
				if m.FocusedPanel == "projects" {
					idx := int(msg.String()[0] - '1')
					if idx < len(m.ProjectList.Items()) {
						m.ProjectList.Select(idx)
						// Load worktrees for the selected project
						if i := m.ProjectList.SelectedItem(); i != nil {
							path := i.(ProjectItem).Path
							m.Loading = true
							m.LoadingMessage = "Loading worktrees..."
							return m, tea.Batch(m.Spinner.Tick, getWorktreesCmd(path))
						}
					}
				}
			case "tab", "right", "l":
				if m.FocusedPanel == "projects" {
					m.FocusedPanel = "worktrees"
				}
			case "shift+tab", "left", "h":
				if m.FocusedPanel == "worktrees" {
					m.FocusedPanel = "projects"
				}
			case "n":
				m.ShowAddDialog = true
				m.PathInput.SetValue("")
				m.PathInput.Focus()
				return m, nil
			case "d":
				if m.FocusedPanel == "projects" {
					return m.deleteProject()
				}
			case "m":
				if len(m.Projects) > 0 {
					m.ShowMergeDialog = true
					m.MergeDialogStep = 0
					m.MergeSource = ""
					m.MergeTarget = ""

					// Default Target: If on a worktree item, use it as default target
					if m.FocusedPanel == "worktrees" {
						if i := m.WorktreeList.SelectedItem(); i != nil {
							wt := i.(WorktreeItem)
							if !wt.IsMain {
								m.MergeTarget = wt.Branch
							}
						}
					}

					// Load branches into merge list
					m = m.updateMergeList("")
				}
				return m, nil
			case "r":
				// Refresh current project's worktrees
				if i := m.ProjectList.SelectedItem(); i != nil {
					path := i.(ProjectItem).Path
					m.Loading = true
					m.LoadingMessage = "Refreshing worktrees..."
					return m, tea.Batch(m.Spinner.Tick, getWorktreesCmd(path))
				}
			case "H":
				m.ShowHistory = true
				return m, nil
			case "?":
				m.Help.ShowAll = !m.Help.ShowAll
				return m, nil
			}
		}

	// Async Git Messages
	case WorktreesLoadedMsg:
		m.Loading = false
		m.LoadingMessage = ""
		if msg.Err != nil {
			m.Message = "Error: " + msg.Err.Error()
			m.WorktreeList.SetItems([]list.Item{})
		} else {
			items := make([]list.Item, len(msg.Worktrees))
			for i, wt := range msg.Worktrees {
				items[i] = WorktreeItem{wt}
			}
			m.WorktreeList.SetItems(items)
			m.Message = "Refreshed"

			// If merge dialog is open, update merge list
			if m.ShowMergeDialog {
				if m.MergeDialogStep == 0 {
					m = m.updateMergeList("")
				} else if m.MergeDialogStep == 1 {
					m = m.updateMergeList(m.MergeSource)
				}
			}
		}

	case LogLoadedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Message = "Error loading commits: " + msg.Err.Error()
			m.MergeCommits = []string{}
		} else {
			m.MergeCommits = msg.Commits
		}

	case MergeCompletedMsg:
		m.Loading = false
		if msg.Err != nil {
			m.Message = "Merge Failed: " + msg.Err.Error()
			m.ShowMergeDialog = false
		} else {
			m.ShowMergeDialog = false
			m.ShowPushPrompt = true
			m.Message = "Merge Successful! Push?"
		}

	case PushCompletedMsg:
		m.Loading = false
		m.Pushing = false
		if msg.Err != nil {
			m.Message = "Push Failed: " + msg.Err.Error()
		} else {
			m.Message = "Pushed Successfully"
			// Add to history
			if i := m.ProjectList.SelectedItem(); i != nil {
				path := i.(ProjectItem).Path
				history := MergeHistory{
					ID:        time.Now().String(),
					Project:   path,
					Source:    m.MergeSource,
					Target:    m.MergeTarget,
					Pushed:    true,
					Timestamp: time.Now().Format("2006-01-02 15:04:05"),
				}
				AddHistory(m.ConfigDir, history)
				// Refresh history list
				m.History = append([]MergeHistory{history}, m.History...)
				hItems := make([]list.Item, len(m.History))
				for i, h := range m.History {
					hItems[i] = HistoryItem{h}
				}
				m.HistoryList.SetItems(hItems)
			}
		}
		m.ShowPushPrompt = false

	case ReplayFetchMsg:
		m.ReplayLog += msg.Output
		if msg.Err != nil {
			m.ReplayLog += "\nFetch failed: " + msg.Err.Error()
			m.Loading = false
			m.Replaying = false
			m.ShowReplayLog = true // Show the log on failure
			m.Message = "Replay Failed (Fetch)"
		} else {
			m.ReplayLog += "\n[2/3] Merging " + m.MergeSource + " into " + m.ReplayTarget + "...\n"
			return m, replayMergeOnlyCmd(m.ReplayProject, m.MergeSource, m.ReplayTarget)
		}

	case ReplayMergeMsg:
		m.ReplayLog += msg.Output
		if msg.Err != nil {
			m.ReplayLog += "\nMerge failed: " + msg.Err.Error()
			m.Loading = false
			m.Replaying = false
			m.ShowReplayLog = true // Show the log on failure
			m.Message = "Replay Failed (Merge)"
		} else {
			if m.ReplayPush {
				m.ReplayLog += "\n[3/3] Pushing...\n"
				return m, replayPushOnlyCmd(m.ReplayProject, m.ReplayTarget)
			} else {
				m.ReplayLog += "\nDone (Skipped Push).\n"
				m.Loading = false
				m.Replaying = false
				m.ShowReplayLog = true // Show log on success too? Or maybe just status.
				// Let's show it on success too so user sees the output.
				m.Message = "Replay Successful"
			}
		}

	case ReplayPushMsg:
		m.ReplayLog += msg.Output
		m.Loading = false
		m.Replaying = false
		m.ShowReplayLog = true
		if msg.Err != nil {
			m.ReplayLog += "\nPush failed: " + msg.Err.Error()
			m.Message = "Replay Failed (Push)"
		} else {
			m.ReplayLog += "\nDone.\n"
			m.Message = "Replay Successful"
		}

	case ReplayCompletedMsg:
		// Fallback for any old logic, just in case
		m.Loading = false
		m.Replaying = false
		m.ShowReplayLog = true
		if msg.Err != nil {
			m.Message = "Replay Failed: " + msg.Err.Error() + "\n\nOutput:\n" + msg.Output
		} else {
			m.Message = "Replay Successful\n\nOutput:\n" + msg.Output
		}

	case spinner.TickMsg:
		if m.Loading {
			var cmd tea.Cmd
			m.Spinner, cmd = m.Spinner.Update(msg)
			return m, cmd
		}
	}

	// Update Focused List
	if m.FocusedPanel == "projects" {
		prevItem := m.ProjectList.SelectedItem()
		m.ProjectList, cmd = m.ProjectList.Update(msg)
		cmds = append(cmds, cmd)

		// If selection changed, load worktrees
		currItem := m.ProjectList.SelectedItem()
		if currItem != nil && (prevItem == nil || prevItem.FilterValue() != currItem.FilterValue()) {
			path := currItem.(ProjectItem).Path
			m.Loading = true
			m.LoadingMessage = "Loading worktrees..."
			cmds = append(cmds, tea.Batch(m.Spinner.Tick, getWorktreesCmd(path)))
		}
	} else {
		m.WorktreeList, cmd = m.WorktreeList.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// Helpers

func (m Model) addProject() (Model, tea.Cmd) {
	path := strings.TrimSpace(m.NewProjectPath)
	if path == "" {
		m.Message = "Please enter a path"
		return m, nil
	}

	// Expand home directory
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		path = strings.Replace(path, "~", home, 1)
	}

	// Validate
	if !isGitRepo(path) {
		m.Message = "Not a valid git repository"
		return m, nil
	}

	// Check duplicates
	for _, p := range m.Projects {
		if p == path {
			m.Message = "Project already exists"
			return m, nil
		}
	}

	m.Projects = append(m.Projects, path)
	saveProjects(m.ConfigDir, m.Projects)

	m.ProjectList.InsertItem(len(m.Projects)-1, ProjectItem{Path: path, Index: len(m.Projects) - 1})
	m.ShowAddDialog = false
	m.NewProjectPath = ""
	m.Message = "Project added"

	// Select the new project
	m.ProjectList.Select(len(m.Projects) - 1)

	return m, nil
}

func (m Model) deleteProject() (Model, tea.Cmd) {
	idx := m.ProjectList.Index()
	if idx < 0 || idx >= len(m.Projects) {
		return m, nil
	}

	// Remove from slice
	// Need to find the actual index in m.Projects based on the item
	// list.Index() returns visual index, but filtering might change it.
	// However, we disabled filtering for delete logic above, or assumed it matches.
	// To be safe, we should match by value.

	selectedItem := m.ProjectList.SelectedItem()
	if selectedItem == nil {
		return m, nil
	}
	selectedPath := selectedItem.(ProjectItem).Path

	newProjects := []string{}
	for _, p := range m.Projects {
		if p != selectedPath {
			newProjects = append(newProjects, p)
		}
	}
	m.Projects = newProjects

	saveProjects(m.ConfigDir, m.Projects)
	m.ProjectList.RemoveItem(idx)

	// Re-index remaining items
	items := m.ProjectList.Items()
	for i := range items {
		item := items[i].(ProjectItem)
		item.Index = i
		items[i] = item
	}
	m.ProjectList.SetItems(items)

	if len(m.Projects) == 0 {
		m.WorktreeList.SetItems([]list.Item{})
	}

	m.Message = "Project removed"
	return m, nil
}

func (m Model) updateMergeList(exclude string) Model {
	var items []list.Item
	for _, item := range m.WorktreeList.Items() {
		wt := item.(WorktreeItem)
		if wt.Branch != exclude {
			items = append(items, wt)
		}
	}

	// Sort items based on step
	// isTargetStep is true when we are selecting target (step 1)
	// isTargetStep is false when we are selecting source (step 0)
	isTargetStep := m.MergeDialogStep == 1

	isDevelop := func(name string) bool {
		return name == "develop" || strings.HasSuffix(name, "_develop") || strings.HasSuffix(name, "-develop")
	}

	sort.SliceStable(items, func(i, j int) bool {
		bi := items[i].(WorktreeItem).Branch
		bj := items[j].(WorktreeItem).Branch

		devI := isDevelop(bi)
		devJ := isDevelop(bj)

		if isTargetStep {
			// Target: Develop first
			if devI && !devJ {
				return true
			}
			if !devI && devJ {
				return false
			}
		} else {
			// Source: Develop last
			if devI && !devJ {
				return false
			}
			if !devI && devJ {
				return true
			}
		}
		// Keep original order (recent commits) for others
		return i < j
	})

	m.MergeList.SetItems(items)
	m.MergeList.Title = "Select Branch"

	// Auto-select develop for target
	if isTargetStep {
		for i, item := range items {
			if isDevelop(item.(WorktreeItem).Branch) {
				m.MergeList.Select(i)
				break
			}
		}
	} else {
		m.MergeList.Select(0)
	}

	return m
}

func (m Model) executeMerge() (Model, tea.Cmd) {
	if i := m.ProjectList.SelectedItem(); i != nil {
		path := i.(ProjectItem).Path
		m.Loading = true
		return m, tea.Batch(m.Spinner.Tick, gitMergeCmd(path, m.MergeSource, m.MergeTarget))
	}
	return m, nil
}

func (m Model) handlePushResponse(push bool) (Model, tea.Cmd) {
	if push {
		if i := m.ProjectList.SelectedItem(); i != nil {
			path := i.(ProjectItem).Path
			m.Loading = true
			m.Pushing = true
			return m, tea.Batch(m.Spinner.Tick, gitPushCmd(path, m.MergeTarget))
		}
	} else {
		// Just save history if not pushing
		m.ShowPushPrompt = false
		if i := m.ProjectList.SelectedItem(); i != nil {
			path := i.(ProjectItem).Path
			history := MergeHistory{
				ID:        time.Now().String(),
				Project:   path,
				Source:    m.MergeSource,
				Target:    m.MergeTarget,
				Pushed:    false,
				Timestamp: time.Now().Format("2006-01-02 15:04:05"),
			}
			AddHistory(m.ConfigDir, history)
			// Refresh history list
			m.History = append([]MergeHistory{history}, m.History...)
			hItems := make([]list.Item, len(m.History))
			for i, h := range m.History {
				hItems[i] = HistoryItem{h}
			}
			m.HistoryList.SetItems(hItems)
		}
	}
	return m, nil
}
