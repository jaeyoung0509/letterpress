package tui

import "github.com/charmbracelet/lipgloss"

var (
	appFrameStyle = lipgloss.NewStyle().
			Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	summaryPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("241")).
				Padding(1, 2)

	sectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("117"))

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("223"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("84"))

	stepChipStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)

	activeStepChipStyle = stepChipStyle.Copy().
				BorderForeground(lipgloss.Color("111")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	buttonFocusedStyle = buttonStyle.Copy().
				BorderForeground(lipgloss.Color("111")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	buttonPrimaryStyle = buttonStyle.Copy().
				BorderForeground(lipgloss.Color("78")).
				Foreground(lipgloss.Color("84"))

	buttonPrimaryFocusedStyle = buttonFocusedStyle.Copy().
					BorderForeground(lipgloss.Color("78")).
					Foreground(lipgloss.Color("84"))

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("111")).
			Padding(1, 2)

	focusedBlockStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(lipgloss.Color("111")).
				PaddingLeft(1)
)
