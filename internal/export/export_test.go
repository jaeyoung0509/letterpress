package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/schema"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

func TestComposeAndWriteExportsSampleLetterToPDFAndPNG(t *testing.T) {
	resolved := sampleResolvedLetter(t)
	tmpDir := t.TempDir()

	pdfOut, err := ComposeAndWrite(resolved, Options{
		Format:      domain.ExportFormatPDF,
		Out:         filepath.Join(tmpDir, "classic-letter"),
		Decorations: true,
	})
	if err != nil {
		t.Fatalf("ComposeAndWrite(pdf) error = %v", err)
	}
	assertFileExists(t, pdfOut)

	pngOut, err := ComposeAndWrite(resolved, Options{
		Format:      domain.ExportFormatPNG,
		Out:         filepath.Join(tmpDir, "classic-letter.png"),
		Decorations: true,
	})
	if err != nil {
		t.Fatalf("ComposeAndWrite(png) error = %v", err)
	}
	assertFileExists(t, pngOut)
}

func TestNormalizeOutputPathRejectsMismatchedExtension(t *testing.T) {
	if _, err := normalizeOutputPath("out/card.pdf", domain.ExportFormatPNG); err == nil {
		t.Fatal("expected mismatched extension to fail")
	}
}

func TestWriteDocumentRejectsUnsupportedFormat(t *testing.T) {
	resolved := sampleResolvedLetter(t)
	document, err := ComposeDocument(resolved, true)
	if err != nil {
		t.Fatalf("ComposeDocument() error = %v", err)
	}

	if _, err := WriteDocument(document, Options{
		Format: domain.ExportFormatSVG,
		Out:    filepath.Join(t.TempDir(), "card.svg"),
	}); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func sampleResolvedLetter(t *testing.T) templatepkg.ResolvedTemplate {
	t.Helper()

	templatePath := filepath.Join("..", "..", "templates", "letter", "classic-letter-a4.yaml")
	projectPath := filepath.Join("..", "..", "templates", "samples", "classic-letter-project.yaml")

	tmpl, err := schema.LoadTemplateFile(templatePath)
	if err != nil {
		t.Fatalf("LoadTemplateFile() error = %v", err)
	}
	project, err := schema.LoadProjectFile(projectPath)
	if err != nil {
		t.Fatalf("LoadProjectFile() error = %v", err)
	}

	resolved, err := templatepkg.Resolve(tmpl, project)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	return resolved
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file %q to exist: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected file %q to be non-empty", path)
	}
}
