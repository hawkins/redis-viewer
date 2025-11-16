// Package styles contains all lipgloss styles for the TUI
package styles

import "github.com/charmbracelet/lipgloss"

// Pane and list styles
var (
	ListViewStyle = lipgloss.NewStyle().
			PaddingRight(1).
			MarginRight(1).
			Border(lipgloss.RoundedBorder(), false, true, false, false)

	ListFocusedBorderColor   = lipgloss.Color("#FF5F87")
	ListUnfocusedBorderColor = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})

	FocusIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	LoadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

// Status bar styles
var (
	StatusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"})

	StatusStyle = StatusBarStyle.Copy().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1).
			MarginRight(1)

	StatusDangerStyle = StatusStyle.Copy().
				Background(lipgloss.Color("#FF0000"))

	EncodingStyle = StatusNugget.Copy().
			Background(lipgloss.Color("#A550DF")).
			Align(lipgloss.Right)

	WrapIndicatorStyle = StatusNugget.Copy().
				Background(lipgloss.Color("#50FA7B"))

	StatusText = StatusBarStyle.Copy()

	DatetimeStyle = StatusNugget.Copy().
			Background(lipgloss.Color("#6124DF"))
)

// Help and dialog styles
var (
	HelpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5F87"))

	HelpDialogStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Background(lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#353533"}).
			Padding(1, 2)
)

// Stats view styles
var (
	StatsTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5F87")).
			MarginBottom(1)

	StatsSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#A550DF")).
				MarginTop(1).
				MarginBottom(1)

	StatsLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}).
			Width(25)

	StatsValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#343433", Dark: "#C1C6B2"}).
			Bold(true)

	StatsHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#6124DF")).
				Width(15)

	StatsRowStyle = lipgloss.NewStyle().Width(15)

	StatsFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})

	StatsContentStyle = lipgloss.NewStyle().
				Padding(2, 4)

	StatsLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF5F87")).
				Bold(true)

	StatsErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)
)
