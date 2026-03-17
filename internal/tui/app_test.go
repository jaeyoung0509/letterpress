package tui

import (
	"strings"
	"testing"
)

func TestViewIncludesBootstrapWorkflow(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		"letterpress",
		"Bootstrap workflow",
		"Template selection",
		"Press q to quit.",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected view to contain %q, got %q", fragment, view)
		}
	}
}
