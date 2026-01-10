package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#F59E0B") // Yellow/Amber
	secondaryColor = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	dangerColor    = lipgloss.Color("#EF4444") // Red
	infoColor      = lipgloss.Color("#3B82F6") // Blue
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	bgColor        = lipgloss.Color("#1F2937") // Dark blue-gray

	// Title style
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	// Summary box styles
	SummaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			MarginBottom(1)

	SummaryLabelStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	SummaryValueStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(mutedColor)

	TableRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	TableSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(primaryColor)

	// Status styles
	PaidStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	UnpaidStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	OverdueStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	// Help style
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	// Loading style
	LoadingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Italic(true)

	// Non-deductible style
	NonDeductibleStyle = lipgloss.NewStyle().
				Foreground(dangerColor)
)
