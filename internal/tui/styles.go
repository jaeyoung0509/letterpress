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

	subCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("239")).
			Padding(1, 1)

	subCardFocusedStyle = subCardStyle.Copy().
				BorderForeground(lipgloss.Color("111"))

	templateCardStyle = subCardStyle.Copy().
				Width(28)

	templateCardSelectedStyle = templateCardStyle.Copy().
					BorderForeground(lipgloss.Color("78"))

	templateCardFocusedStyle = templateCardSelectedStyle.Copy().
					BorderForeground(lipgloss.Color("117")).
					Bold(true)

	previewFrameStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("244")).
				Padding(1, 2)

	pagePreviewStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("247")).
				Padding(0, 1)

	eyebrowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("151")).
			Bold(true)

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("111")).
			Padding(1, 2)

	focusedBlockStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderForeground(lipgloss.Color("111")).
				PaddingLeft(1)

	shellCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	previewPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("239")).
				Padding(0, 1)

	promptPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)

	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("249"))

	pickerItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	pickerActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("60")).
				Bold(true)
)
