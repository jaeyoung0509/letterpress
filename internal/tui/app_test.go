package tui

import (
	"strings"
	"testing"
)

func TestViewShowsShellLayout(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		"letterpress",
		"Bubble Tea composition shell",
		"Steps:",
		"Template Selection",
		"Review & Export",
		"[Forward:",
		"[Back:",
		"[Quit:",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected view to contain %q, got %q", fragment, view)
		}
	}
}

func TestStateAdvancesAndRewinds(t *testing.T) {
	model := NewModel()
	initialView := model.View()

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
}
