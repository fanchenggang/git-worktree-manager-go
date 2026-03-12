# Git Worktree Manager

A powerful Terminal User Interface (TUI) for managing Git worktrees, built with Go and Bubble Tea.

## Features

*   **Project Management**:
    *   Add and manage multiple Git repositories.
    *   **Quick Navigation**: Use numeric shortcuts (`1` - `9`) to instantly switch between projects.
    *   **Auto-refresh**: Automatically loads worktrees upon project selection.

*   **Worktree & Branch Management**:
    *   **Smart Sorting**: Branches are automatically sorted by recent commit activity (newest first).
    *   **Visual Status**: Clear indicators for clean/dirty states and main branches.
    *   **Async Loading**: Non-blocking UI with loading feedback for heavy git operations.

*   **Merge Wizard**:
    *   Step-by-step wizard for merging branches.
    *   **Intelligent Sorting**:
        *   **Source Step**: `develop` branches are sorted to the bottom (less likely to merge *from* develop).
        *   **Target Step**: `develop` branches are sorted to the top and auto-selected (most likely to merge *into* develop).
    *   **Preview**: View incoming commits before confirming the merge.
    *   **Push Prompt**: Option to push changes to remote immediately after a successful merge.

*   **Merge History & Replay**:
    *   Track all merge operations performed within the tool.
    *   **Replay**: Re-run past merges with a single keystroke.
    *   **Execution Logs**: View detailed git output logs for every replay operation.

## Installation

```bash
go install github.com/yourusername/git-worktree-manager@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/git-worktree-manager.git
cd git-worktree-manager
go build .
```

## Usage

Run the application:

```bash
git-worktree-manager
# or if built locally
./git-worktree-manager
```

### Keybindings

**Global Navigation**
- `Tab` / `Shift+Tab`: Switch focus between Projects and Worktrees panels.
- `h` / `l` / `Left` / `Right`: Alternative panel navigation.
- `j` / `k` / `Up` / `Down`: Navigate lists.
- `q` / `Ctrl+C`: Quit.

**Projects Panel**
- `n`: Add a new project (enter path).
- `d`: Delete selected project.
- `1` - `9`: Quick jump to project by index.

**Worktrees Panel**
- `r`: Refresh worktrees.
- `m`: Open Merge Wizard.
- `H`: View Merge History.

**Merge Wizard**
- `Enter`: Select / Confirm.
- `Esc`: Go back / Cancel.

## Configuration

Configuration is stored in `~/.git-worktree-manager/config.json`.

## License

MIT
