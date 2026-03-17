package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestViewShowsShellLayout(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		"letterpress",
		"Bubble Tea composition shell",
		"Steps:",
		"Template Selection",
		"Review & Export",
		"Available templates:",
		"Use j/k to cycle templates",
		"[Forward:",
		"[Back:",
		"[Quit:",
		"Composition in progress",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected view to contain %q, got %q", fragment, view)
		}
	}
}

func TestStateAdvancesAndRewinds(t *testing.T) {
	model := NewModel()
	initialView := model.View()
	initialSummary := model.composition.Summary()

	model.state = model.state.withNext()
	if model.state.Current != StepSize {
		t.Fatalf("expected step %q after forward, got %q", StepSize, model.state.Current)
	}

	interstitialView := model.View()
	if interstitialView == initialView {
		t.Fatalf("expected view to change after a forward step")
	}

	model.state = model.state.withPrev()
	if model.state.Current != StepTemplate {
		t.Fatalf("expected step %q after back, got %q", StepTemplate, model.state.Current)
	}

	if model.View() != initialView {
		t.Fatalf("expected view to return to the initial state after rewinding")
	}

	if model.composition.Summary() != initialSummary {
		t.Fatalf("composition summary changed after navigation: before=%q after=%q", initialSummary, model.composition.Summary())
	}
}

func TestTemplatePickerCycles(t *testing.T) {
	model := NewModel()
	if len(model.templates) < 2 {
		t.Skip("at least two templates required for cycling")
	}

	nextIdx := (model.templateIndex + 1) % len(model.templates)
	expected := model.templates[nextIdx].ID

	model = model.cycleTemplate(1)
	if model.composition.Project.Template != expected {
		t.Fatalf("expected template %q after cycling, got %q", expected, model.composition.Project.Template)
	}
}

func TestSizeSelectionAndOrientation(t *testing.T) {
	model := NewModel()
	model.state.Current = StepSize

	entry, ok := model.currentTemplateEntry()
	if !ok || len(entry.SupportedSizes) < 2 {
		t.Skip("template needs at least two supported sizes")
	}

	initial := model.composition.Project.Page.Size
	model = model.cycleSize(1)
	if model.composition.Project.Page.Size == initial {
		t.Fatalf("expected page size to change after cycling")
	}

	initialOrientation := model.composition.Project.Page.Orientation
	model = model.toggleOrientation()
	if model.composition.Project.Page.Orientation == initialOrientation {
		t.Fatalf("expected orientation to toggle")
	}
}

func TestContentEditingFields(t *testing.T) {
	model := NewModel()
	model.state.Current = StepContent
	model.contentField = FieldTitle

	model = model.appendContentRunes([]rune("Hello"))
	if model.composition.Project.Content.Title != "Hello" {
		t.Fatalf("expected title to update, got %q", model.composition.Project.Content.Title)
	}

	model = model.handleContentKey(tea.KeyMsg{Type: tea.KeyTab})
	if model.contentField != FieldBody {
		t.Fatalf("expected content field to cycle to Body, got %v", model.contentField)
	}

	model = model.appendContentRunes([]rune("Message"))
	if model.composition.Project.Content.Body != "Message" {
		t.Fatalf("expected body to update, got %q", model.composition.Project.Content.Body)
	}

	model = model.deleteContentRune()
	if model.composition.Project.Content.Body != "Messag" {
		t.Fatalf("expected body to trim last rune, got %q", model.composition.Project.Content.Body)
	}
}
