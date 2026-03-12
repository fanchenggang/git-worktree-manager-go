package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages

type WorktreesLoadedMsg struct {
	Worktrees []Worktree
	Err       error
}

type BranchesLoadedMsg struct {
	Branches []string
	Err      error
}

type MergeCompletedMsg struct {
	Err error
}

type PushCompletedMsg struct {
	Err error
}

type ReplayCompletedMsg struct {
	Output string
	Err    error
}

type ReplayFetchMsg struct {
	Output string
	Err    error
}

type ReplayMergeMsg struct {
	Output string
	Err    error
}

type ReplayPushMsg struct {
	Output string
	Err    error
}

// LogLoadedMsg
type LogLoadedMsg struct {
	Commits []string
	Err     error
}

// Granular Replay Commands

func replayFetchCmd(project string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "fetch", ".")
		cmd.Dir = project
		out, err := cmd.CombinedOutput()
		return ReplayFetchMsg{Output: string(out), Err: err}
	}
}

func replayMergeOnlyCmd(project, source, target string) tea.Cmd {
	return func() tea.Msg {
		// Find worktree
		worktrees, err := getWorktrees(project)
		if err != nil {
			return ReplayMergeMsg{Err: err}
		}

		var targetPath string
		for _, wt := range worktrees {
			if wt.Branch == target {
				targetPath = wt.Path
				break
			}
		}
		if targetPath == "" {
			targetPath = project
		}

		cmd := exec.Command("git", "merge", source)
		cmd.Dir = targetPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			// Abort
			abort := exec.Command("git", "merge", "--abort")
			abort.Dir = targetPath
			abort.Run()
		}
		return ReplayMergeMsg{Output: string(out), Err: err}
	}
}

func replayPushOnlyCmd(project, target string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("git", "push", "origin", target)
		cmd.Dir = project
		out, err := cmd.CombinedOutput()
		return ReplayPushMsg{Output: string(out), Err: err}
	}
}

// Async Git Commands

func getWorktreesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		worktrees, err := getWorktrees(repoPath)
		return WorktreesLoadedMsg{Worktrees: worktrees, Err: err}
	}
}

func getBranchesCmd(repoPath string) tea.Cmd {
	return func() tea.Msg {
		branches, err := getBranches(repoPath)
		return BranchesLoadedMsg{Branches: branches, Err: err}
	}
}

func gitMergeCmd(repoPath, source, target string) tea.Cmd {
	return func() tea.Msg {
		_, err := gitMergeWithOutput(repoPath, source, target)
		return MergeCompletedMsg{Err: err}
	}
}

func gitPushCmd(repoPath, branch string) tea.Cmd {
	return func() tea.Msg {
		err := gitPush(repoPath, branch)
		return PushCompletedMsg{Err: err}
	}
}

func getLogCmd(repoPath, target, source string) tea.Cmd {
	return func() tea.Msg {
		commits, err := getLog(repoPath, target, source)
		return LogLoadedMsg{Commits: commits, Err: err}
	}
}

// Synchronous Helpers (keep these for now, but used by async wrappers)

func isGitRepo(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir() || info.Name() == ".git"
}

func getWorktrees(repoPath string) ([]Worktree, error) {
	// 1. Get worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	worktreeMap := make(map[string]Worktree)

	var current Worktree
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			if current.Branch != "" {
				worktreeMap[current.Branch] = current
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
		worktreeMap[current.Branch] = current
	}

	// 2. Get branches sorted by recent commit date
	// git for-each-ref --sort=-committerdate refs/heads/ --format='%(refname:short)'
	cmd = exec.Command("git", "for-each-ref", "--sort=-committerdate", "refs/heads/", "--format=%(refname:short)")
	cmd.Dir = repoPath
	output, err = cmd.Output()
	if err != nil {
		// Fallback to map iteration order (random) if sorting fails
		var res []Worktree
		for _, wt := range worktreeMap {
			res = append(res, wt)
		}
		return res, nil
	}

	sortedBranches := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []Worktree

	// Add worktrees in sorted order
	for _, branch := range sortedBranches {
		if wt, exists := worktreeMap[branch]; exists {
			result = append(result, wt)
		}
	}

	// Add any remaining worktrees (e.g. detached heads or weird states not in refs/heads)
	// Usually covered above, but just in case
	for _, wt := range worktreeMap {
		found := false
		for _, res := range result {
			if res.Branch == wt.Branch {
				found = true
				break
			}
		}
		if !found {
			result = append(result, wt)
		}
	}

	// Check main worktree status if needed (optimization: only check if dirty status is critical)
	mainPath := repoPath
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = mainPath
	output, _ = cmd.Output()
	if len(strings.TrimSpace(string(output))) == 0 {
		for i := range result {
			if result[i].IsMain {
				result[i].Clean = true
			}
		}
	}

	return result, nil
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

func gitMergeWithOutput(repoPath, source, target string) (string, error) {
	// Find the worktree path for target branch
	worktrees, err := getWorktrees(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get worktrees: %v", err)
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

	var output strings.Builder

	// Merge source into target branch in the target's worktree
	cmd := exec.Command("git", "fetch", ".")
	cmd.Dir = targetPath
	out, err := cmd.CombinedOutput()
	output.Write(out)
	if err != nil {
		// fetch errors might be ignorable, but let's log them
		output.WriteString(fmt.Sprintf("\nFetch warning: %v\n", err))
	}

	cmd = exec.Command("git", "merge", source)
	cmd.Dir = targetPath
	out, err = cmd.CombinedOutput()
	output.Write(out)

	if err != nil {
		// If merge failed, try to abort
		abortCmd := exec.Command("git", "merge", "--abort")
		abortCmd.Dir = targetPath
		abortCmd.Run()
		return output.String(), fmt.Errorf("merge failed: %s", string(out))
	}

	return output.String(), nil
}

func gitPush(repoPath, branch string) error {
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func getLog(repoPath, target, source string) ([]string, error) {
	// We want to see what source has that target doesn't (merging source INTO target)
	// git log target..source
	// But wait, we need to run this where? Ideally in the repo.
	cmd := exec.Command("git", "log", fmt.Sprintf("%s..%s", target, source), "--oneline", "-n", "10")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []string{}, nil
	}
	return lines, nil
}
