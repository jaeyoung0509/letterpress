package projectio

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func sampleProject() domain.Project {
	return domain.Project{
		Version:  domain.CurrentSchemaVersion,
		Template: "classic-letter-a4",
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA4,
			Orientation: domain.OrientationPortrait,
		},
		Content: domain.Content{
			Title:     "Hello",
			Body:      "Thanks for being amazing.",
			Signature: "Jane",
		},
		Images: []domain.ImageBinding{
			{Slot: "photo", Path: "assets/photo.jpg"},
		},
		Options: domain.ProjectOptions{
			Decorations: true,
		},
		Export: domain.ExportOptions{
			Format: domain.ExportFormatPDF,
			Out:    "out/card.pdf",
		},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "nested", "project.yaml")

	project := sampleProject()
	if err := Save(path, project); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !reflect.DeepEqual(project, loaded) {
		t.Fatalf("loaded project differs from saved project: got %+v want %+v", loaded, project)
	}
}

func TestSaveRejectsInvalidProject(t *testing.T) {
	err := Save(filepath.Join(t.TempDir(), "bad.yaml"), domain.Project{})
	if err == nil {
		t.Fatal("expected validation error from Save()")
	}
	var ve domain.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
	}
}

func TestLoadRejectsInvalidProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	data := []byte("version: 1\npage:\n  size: A4\n  orientation: portrait\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("Write invalid project: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected validation error from Load()")
	} else {
		var ve domain.ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T: %v", err, err)
		}
	}
}
