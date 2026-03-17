package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func TestRenderCommandExportsPDF(t *testing.T) {
	dir := t.TempDir()
	templatePath := filepath.Join(dir, "template.yaml")
	projectPath := filepath.Join(dir, "project.yaml")
	outPath := filepath.Join(dir, "output", "card.pdf")

	writeSampleTemplate(t, templatePath, "classic-letter")
	writeSampleProject(t, projectPath, "classic-letter")

	cmd := NewRenderCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"--template", templatePath,
		"--project", projectPath,
		"--out", outPath,
		"--format", "pdf",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("render command failed: %v", err)
	}

	if !strings.Contains(output.String(), "rendered classic-letter") {
		t.Fatalf("unexpected render output: %q", output.String())
	}
	if info, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	} else if info.Size() == 0 {
		t.Fatalf("expected output file to be non-empty")
	}
}

func TestResolveRenderTargetUsesProjectDefaults(t *testing.T) {
	project := domain.Project{
		Export: domain.ExportOptions{
			Format: domain.ExportFormatPNG,
			Out:    "outputs/card.png",
		},
	}

	format, out, err := resolveRenderTarget(project, "", "")
	if err != nil {
		t.Fatalf("resolveRenderTarget() error = %v", err)
	}
	if format != domain.ExportFormatPNG {
		t.Fatalf("format = %q, want png", format)
	}
	if out != "outputs/card.png" {
		t.Fatalf("out = %q, want outputs/card.png", out)
	}
}

func TestResolveRenderTargetRejectsMismatchedPath(t *testing.T) {
	project := domain.Project{}
	if _, _, err := resolveRenderTarget(project, "png", "out/card.pdf"); err == nil {
		t.Fatal("expected mismatched output path to fail")
	}
}
