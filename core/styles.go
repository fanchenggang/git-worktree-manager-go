package core

import "github.com/charmbracelet/lipgloss"

// Theme colors (Catppuccin Macchiato inspired)
var (
	Base     = lipgloss.Color("#24273a")
	Mantle   = lipgloss.Color("#1e2030")
	Text     = lipgloss.Color("#cad3f5")
	Subtext  = lipgloss.Color("#a5adcb")
	Overlay  = lipgloss.Color("#5b6078")
	Surface  = lipgloss.Color("#363a4f")
	Blue     = lipgloss.Color("#8aadf4")
	Lavender = lipgloss.Color("#b7bdf8")
	Green    = lipgloss.Color("#a6da95")
	Red      = lipgloss.Color("#ed8796")
	Peach    = lipgloss.Color("#f5a97f")
	Yellow   = lipgloss.Color("#eed49f")
	Mauve    = lipgloss.Color("#c6a0f6")
)

// Common Styles
var (
	// App-wide
	AppStyle = lipgloss.NewStyle().Margin(1, 2)

	// Headers
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Mantle).
			Background(Mauve).
			Bold(true).
			Padding(0, 1).
			MarginBottom(1)

	// Panels
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Overlay).
			Padding(0, 1)

	FocusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Lavender).
				Padding(0, 1)

	// List Items
	TitleStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	DescStyle = lipgloss.NewStyle().
			Foreground(Subtext)

	SelectedTitleStyle = lipgloss.NewStyle().
				Foreground(Mauve).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(Mauve).
				PaddingLeft(1)

	SelectedDescStyle = lipgloss.NewStyle().
				Foreground(Lavender).
				PaddingLeft(1)

	// Status Indicators
	CleanStyle = lipgloss.NewStyle().
			Foreground(Green).
			SetString("✔")

	DirtyStyle = lipgloss.NewStyle().
			Foreground(Red).
			SetString("✗")

	MainBranchStyle = lipgloss.NewStyle().
			Foreground(Yellow).
			Bold(true).
			SetString("★")

	// Dialogs
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Lavender).
			Padding(1, 2)

	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(Blue).
				Bold(true).
				MarginBottom(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Subtext).
			MarginTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// Stepper
	ActiveStepStyle = lipgloss.NewStyle().
			Foreground(Lavender).
			Bold(true)

	InactiveStepStyle = lipgloss.NewStyle().
				Foreground(Overlay)

	StepSeparatorStyle = lipgloss.NewStyle().
				Foreground(Subtext).
				Padding(0, 1).
				SetString(">")

	// Preview
	PreviewCommitStyle = lipgloss.NewStyle().
				Foreground(Text).
				PaddingLeft(1)
)
