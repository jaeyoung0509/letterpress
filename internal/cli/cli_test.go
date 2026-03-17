package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplatesListCommand(t *testing.T) {
	dir := t.TempDir()
	writeSampleTemplate(t, filepath.Join(dir, "letter", "classic-letter.yaml"), "classic-letter")
	writeSampleTemplate(t, filepath.Join(dir, "card", "modern-card.yaml"), "modern-card")

	cmd := NewTemplatesCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"list", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("templates list failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected output lines: %q", lines)
	}
	if !strings.HasPrefix(lines[0], "classic-letter") || !strings.HasPrefix(lines[1], "modern-card") {
		t.Fatalf("templates not listed in expected order: %v", lines)
	}
}

func TestValidateCommand(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.yaml")
	projectPath := filepath.Join(dir, "project.yaml")

	writeSampleTemplate(t, templatePath, "classic-letter")
	writeSampleProject(t, projectPath, "classic-letter")

	cmd := NewValidateCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"--template", templatePath, "--project", projectPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("validate command failed: %v", err)
	}

	lines := output.String()
	for _, fragment := range []string{
		"template classic-letter: valid",
		"project classic-letter: valid",
		"template and project resolve successfully",
	} {
		if !strings.Contains(lines, fragment) {
			t.Fatalf("expected output to contain %q, got %q", fragment, lines)
		}
	}
}

func writeSampleTemplate(t *testing.T, path, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for template: %v", err)
	}
	content := fmt.Sprintf(`version: 1
id: %s
page:
  supported_sizes: [A4]
  default_orientation: portrait
layout:
  margin_mm: 12
slots:
  - id: title
    type: text
    x_mm: 10
    y_mm: 20
    w_mm: 160
    h_mm: 30
`, id)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
}

func writeSampleProject(t *testing.T, path, templateID string) {
	t.Helper()
	content := fmt.Sprintf(`version: 1
template: %s
page:
  size: A4
  orientation: portrait
content:
  title: Hello
`, templateID)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
}
