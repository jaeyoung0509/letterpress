package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Step string

const (
	StepTemplate Step = "Template Selection"
	StepSize     Step = "Paper Size & Orientation"
	StepContent  Step = "Content Composition"
	StepReview   Step = "Review & Export"
)

var stepOrder = []Step{
	StepTemplate,
	StepSize,
	StepContent,
	StepReview,
}

type RouteState struct {
	Title       string
	Description string
	Placeholder string
}

type State struct {
	Current Step
	Routes  map[Step]RouteState
}

func newState() State {
	return State{
		Current: StepTemplate,
		Routes: map[Step]RouteState{
			StepTemplate: {
				Title:       "Template Selection",
				Description: "Placeholder route for curated templates and layouts.",
				Placeholder: "Future work: list letter, card, and note templates.",
			},
			StepSize: {
				Title:       "Paper Size & Orientation",
				Description: "Frame the composition for A3–A6 in portrait or landscape.",
				Placeholder: "Future work: preview mm dimensions and safe margins.",
			},
			StepContent: {
				Title:       "Content Composition",
				Description: "Compose title, body, signature, and decorative slots.",
				Placeholder: "Future work: edit text slots and attach optional assets.",
			},
			StepReview: {
				Title:       "Review & Export",
				Description: "Finalize layout, toggle decorations, and export.",
				Placeholder: "Future work: show export targets (PDF/PNG) and notes.",
			},
		},
	}
}

func (s State) currentIndex() int {
	for i, step := range stepOrder {
		if step == s.Current {
			return i
		}
	}

	return 0
}

func (s State) withNext() State {
	idx := s.currentIndex()
	if idx+1 < len(stepOrder) {
		s.Current = stepOrder[idx+1]
	}

	return s
}

func (s State) withPrev() State {
	idx := s.currentIndex()
	if idx > 0 {
		s.Current = stepOrder[idx-1]
	}

	return s
}

type KeyMap struct {
	Forward string
	Back    string
	Quit    string
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Forward: "enter / right / down",
		Back:    "backspace / left / up",
		Quit:    "q / ctrl+c",
	}
}

type Model struct {
	state       State
	composition CompositionState
	keyMap      KeyMap
	width       int
	height      int
}

func NewModel() Model {
	return Model{
		state:       newState(),
		composition: newCompositionState(),
		keyMap:      DefaultKeyMap(),
	}
}

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
		case "enter", "right", "down", " ":
			m.state = m.state.withNext()
		case "left", "up", "backspace", "esc":
			m.state = m.state.withPrev()
		}
	}

	return m, nil
}

func (m Model) View() string {
	lines := []string{
		"letterpress",
		"Bubble Tea composition shell",
		"",
		m.renderSteps(),
		"",
		m.renderRoute(),
		"",
		m.renderCompositionSummary(),
		"",
		fmt.Sprintf("Viewport: %dx%d", m.width, m.height),
		"",
		m.renderKeyLegend(),
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderSteps() string {
	var steps []string
	steps = append(steps, "Steps:")
	for _, step := range stepOrder {
		prefix := "   "
		if step == m.state.Current {
			prefix = "→ "
		}
		steps = append(steps, fmt.Sprintf("%s%s", prefix, step))
	}

	return strings.Join(steps, "\n")
}

func (m Model) renderRoute() string {
	route, ok := m.state.Routes[m.state.Current]
	if !ok {
		return "Route placeholder unavailable."
	}

	return fmt.Sprintf("%s\n%s\n\n%s", route.Title, route.Description, route.Placeholder)
}

func (m Model) renderCompositionSummary() string {
	return fmt.Sprintf("Composition in progress (%s)", m.composition.Summary())
}

func (m Model) renderKeyLegend() string {
	return fmt.Sprintf("[Forward: %s]  [Back: %s]  [Quit: %s]", m.keyMap.Forward, m.keyMap.Back, m.keyMap.Quit)
}
