package schema

import (
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func TestParseProjectValidYAML(t *testing.T) {
	project, err := ParseProject([]byte(`
version: 1
template: classic-letter-a4
page:
  size: A4
  orientation: portrait
content:
  title: For You
  body: Thank you for everything.
  signature: Jaeyoung
images:
  - slot: photo1
    path: ./assets/photo.jpg
export:
  format: pdf
`))
	if err != nil {
		t.Fatalf("expected project YAML to parse, got %v", err)
	}

	if project.Version != domain.CurrentSchemaVersion {
		t.Fatalf("expected schema version %d, got %d", domain.CurrentSchemaVersion, project.Version)
	}
	if project.Page.Size != domain.PageSizeA4 {
		t.Fatalf("expected page size A4, got %q", project.Page.Size)
	}
	if project.Export.Format != domain.ExportFormatPDF {
		t.Fatalf("expected export format pdf, got %q", project.Export.Format)
	}
}

func TestParseTemplateValidYAML(t *testing.T) {
	template, err := ParseTemplate([]byte(`
version: 1
id: classic-letter-a4
page:
  supported_sizes: [A4, A5]
  default_orientation: portrait
layout:
  margin_mm: 14
slots:
  - id: title
    type: text
    x_mm: 20
    y_mm: 20
    w_mm: 170
    h_mm: 20
    style: title
  - id: body
    type: text
    x_mm: 20
    y_mm: 48
    w_mm: 170
    h_mm: 180
    style: body
styles:
  title:
    font: SerifDisplay
    size_pt: 22
    align: center
  body:
    font: SansBody
    size_pt: 11
    line_height: 1.4
assets:
  - id: floral-corner
    kind: decoration
    path: ./assets/floral-corner.svg
`))
	if err != nil {
		t.Fatalf("expected template YAML to parse, got %v", err)
	}

	if template.ID != "classic-letter-a4" {
		t.Fatalf("expected template id classic-letter-a4, got %q", template.ID)
	}
	if len(template.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(template.Slots))
	}
}

func TestParseProjectRejectsUnknownFields(t *testing.T) {
	_, err := ParseProject([]byte(`
version: 1
template: classic-letter-a4
page:
  size: A4
  orientation: portrait
bogus: true
`))
	if err == nil {
		t.Fatal("expected strict YAML decode error")
	}

	if !strings.Contains(err.Error(), "decode project YAML") || !strings.Contains(err.Error(), "field bogus not found") {
		t.Fatalf("expected strict decode error, got %q", err.Error())
	}
}

func TestParseTemplateRejectsValidationErrors(t *testing.T) {
	_, err := ParseTemplate([]byte(`
version: 1
id: invalid-template
page:
  supported_sizes: [A4]
  default_orientation: portrait
slots:
  - id: body
    type: text
    x_mm: 10
    y_mm: 10
    w_mm: 100
    h_mm: 80
    style: body
styles:
  body:
    font: SansBody
    size_pt: 0
`))
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "validate template YAML") || !strings.Contains(err.Error(), "styles.body.size_pt") {
		t.Fatalf("expected validation error details, got %q", err.Error())
	}
}
