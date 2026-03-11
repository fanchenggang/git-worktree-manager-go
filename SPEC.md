# Git Worktree Manager TUI - Specification

## 1. Project Overview

- **Project Name**: gitwt (Git Worktree Manager)
- **Type**: Terminal UI Application (Go + Bubble Tea)
- **Core Functionality**: Manage Git worktrees with a keyboard-driven TUI interface
- **Target Users**: Developers who use git worktrees for feature development

## 2. UI/UX Specification

### Layout Structure

- **Single page layout** with multiple panels:
  - Header: App title and current project path
  - Main panel: Split into left (projects list) and right (worktrees list)
  - Footer: Help text and keyboard shortcuts

### Visual Design

- **Color Palette**:
  - Background: `#1e1e2e` (dark)
  - Primary: `#89b4fa` (blue)
  - Secondary: `#a6e3a1` (green)
  - Accent: `#f38ba8` (pink/red for warnings)
  - Text: `#cdd6f4` (light gray)
  - Muted: `#6c7086` (dimmed text)

- **Typography**:
  - Font: System default monospace
  - Header: Bold, larger size
  - Body: Regular size

- **Spacing**:
  - Padding: 1 cell between elements
  - List item height: 1 row

### Components

1. **Project List Panel** (Left, 30% width)
   - List of saved project paths
   - Selected item highlighted with primary color
   - "+" button to add new project
   - "x" button to remove selected project

2. **Worktree List Panel** (Right, 70% width)
   - Shows worktrees for selected project
   - Displays: branch name, worktree path, clean/dirty status
   - Current branch marked with indicator

3. **Merge Dialog** (Modal)
   - Source branch selector
   - Target branch selector
   - Commit message input (optional)
   - Confirm/Cancel buttons

4. **Message/Confirmation Dialogs**
   - For merge success with push prompt
   - For errors

### Keyboard Navigation

- `j/↓`: Move down in list
- `k/↑`: Move up in list
- `Enter`: Select/Open
- `n`: New project (add path)
- `d`: Delete selected project
- `m`: Open merge dialog
- `r`: Refresh worktree list
- `q/Esc`: Quit

## 3. Functionality Specification

### Core Features

1. **Project Management**
   - Add project path (via file picker or manual input)
   - Remove project from list
   - Persist project list to local JSON file (`~/.gitwt/projects.json`)
   - Validate path is a valid git repository

2. **Worktree Listing**
   - List all worktrees for selected project
   - Show branch name, path, and status (clean/dirty)
   - Auto-refresh on project selection

3. **Merge Operation**
   - Select source branch (from worktrees or main)
   - Select target branch
   - Execute `git merge`
   - Show success/failure message
   - Prompt: "Push to remote?" with Yes/No options

4. **Data Persistence**
   - Store projects in `~/.gitwt/projects.json`
   - Format: `{"projects": ["/path/to/repo1", "/path/to/repo2"]}`

### User Flows

1. **Add Project Flow**:
   - Press `n` → Input dialog appears → Enter path → Validate → Add to list

2. **Merge Flow**:
   - Select project → Press `m` → Select source → Select target → Execute → Success prompt → Optional push

### Error Handling

- Invalid git repository: Show error message
- Merge conflict: Display conflict message, don't auto-push
- Git command failure: Show error with details

## 4. Acceptance Criteria

- [ ] Application starts and displays project list
- [ ] Can add/remove project paths
- [ ] Selecting a project shows its worktrees
- [ ] Can execute merge between branches
- [ ] After successful merge, prompts to push
- [ ] Keyboard navigation works as specified
- [ ] Data persists between sessions

## 5. Technical Stack

- **Framework**: Bubble Tea (github.com/charmbracelet/bubbletea)
- **File Picker**: github.com/charmbracelet/bubbles/filepicker
- **Key handling**: bubbles/key
- **Storage**: Standard JSON encoding
