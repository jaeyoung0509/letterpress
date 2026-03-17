package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Model is the minimal Bubble Tea shell for the initial bootstrap issue.
type Model struct {
	width  int
	height int
}

// NewModel creates the initial TUI model.
func NewModel() Model {
	return Model{}
}

// Run launches the Bubble Tea program.
func Run() error {
	program := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	lines := []string{
		"letterpress",
		"",
		"Terminal-first letter and card composer",
		"",
		"Bootstrap workflow",
		"1. Template selection",
		"2. Page size and orientation",
		"3. Content composition",
		"4. Review and export",
		"",
		"Press q to quit.",
	}

	if m.width > 0 || m.height > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Viewport: %dx%d", m.width, m.height))
	}

	return strings.Join(lines, "\n")
}
