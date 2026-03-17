package letterpress

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCmd(Dependencies{
		RunTUI: func() error {
			t.Fatal("expected help to short-circuit before launching the TUI")
			return nil
		},
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected help to succeed, got %v", err)
	}

	output := out.String()
	for _, fragment := range []string{
		"letterpress",
		"terminal-first letter and card composer",
		"tui",
	} {
		if !strings.Contains(strings.ToLower(output), fragment) {
			t.Fatalf("expected help output to contain %q, got %q", fragment, output)
		}
	}
}

func TestRootCommandRunsTUIByDefault(t *testing.T) {
	called := false

	cmd := NewRootCmd(Dependencies{
		RunTUI: func() error {
			called = true
			return nil
		},
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected root command to run, got %v", err)
	}

	if !called {
		t.Fatal("expected root command to launch the TUI stub")
	}
}
