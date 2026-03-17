package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/export"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

func TestViewShowsQuickStartScaffold(t *testing.T) {
	view := NewModel().View()

	for _, fragment := range []string{
		"letterpress",
		"Quick Start",
		"Edit Content",
		"Style & Decorations",
		"Review & Export",
		"Template",
		"Text file (.txt / .md)",
		"Output",
	} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(fragment)) {
			t.Fatalf("expected view to contain %q, got %q", fragment, view)
		}
	}
}

func TestFocusMovementAndStepProgression(t *testing.T) {
	model := NewModel()

	if model.state.Current != StepQuickStart {
		t.Fatalf("expected initial step %q, got %q", StepQuickStart, model.state.Current)
	}
	if model.focused != focusQuickTemplateList {
		t.Fatalf("expected initial focus %q, got %q", focusQuickTemplateList, model.focused)
	}

	model.focusNext()
	if model.focused != focusQuickPageSize {
		t.Fatalf("expected focus to move to %q, got %q", focusQuickPageSize, model.focused)
	}

	model.moveNextStep()
	if model.state.Current != StepEdit {
		t.Fatalf("expected step %q after advancing, got %q", StepEdit, model.state.Current)
	}
	if model.focused != focusEditTitle {
		t.Fatalf("expected edit step focus %q, got %q", focusEditTitle, model.focused)
	}

	model.moveNextStep()
	if model.state.Current != StepStyle {
		t.Fatalf("expected step %q after advancing, got %q", StepStyle, model.state.Current)
	}

	model.movePrevStep()
	if model.state.Current != StepEdit {
		t.Fatalf("expected step %q after rewinding, got %q", StepEdit, model.state.Current)
	}
}

func TestSuggestedOutputTracksTemplateAndFormat(t *testing.T) {
	model := NewModel()

	if got := model.composition.Project.Export.Out; got != "exports/modern-card-a6.pdf" {
		t.Fatalf("expected suggested output %q, got %q", "exports/modern-card-a6.pdf", got)
	}

	model = model.toggleExportFormat(StepQuickStart)
	if got := model.composition.Project.Export.Out; got != "exports/modern-card-a6.png" {
		t.Fatalf("expected suggested output to switch to png, got %q", got)
	}

	model.outputInput.SetValue("custom/output/final.png")
	model.updateOutputPathFromInput()
	model = model.selectTemplate(1)
	if got := model.composition.Project.Export.Out; got != "custom/output/final.png" {
		t.Fatalf("expected custom output path to survive template change, got %q", got)
	}
}

func TestBodyImportLoadsTextAndMarkdownIntoBodyOnly(t *testing.T) {
	for _, tt := range []struct {
		name     string
		filename string
		body     string
	}{
		{name: "txt", filename: "message.txt", body: "plain text body"},
		{name: "md", filename: "message.md", body: "# heading\n\nmarkdown body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			model := NewModel()
			model.composition.Project.Content.Title = "Existing Title"
			model.composition.Project.Content.Signature = "Existing Signature"

			path := writeTempFile(t, tt.filename, tt.body)
			model.bodyImportInput.SetValue(path)

			model = model.applyBodyImport()

			if got := model.composition.Project.Content.Body; got != tt.body {
				t.Fatalf("expected imported body %q, got %q", tt.body, got)
			}
			if got := model.composition.Project.Content.Title; got != "Existing Title" {
				t.Fatalf("title changed during import: %q", got)
			}
			if got := model.composition.Project.Content.Signature; got != "Existing Signature" {
				t.Fatalf("signature changed during import: %q", got)
			}
			if status := model.status(StepQuickStart); status.IsError {
				t.Fatalf("expected successful import, got error message %q", status.Text)
			}
		})
	}
}

func TestPrimaryImageSlotPriority(t *testing.T) {
	entry := TemplateEntry{
		ImageSlots: []domain.Slot{
			{ID: "cover", Type: domain.SlotTypeImage},
			{ID: "photo", Type: domain.SlotTypeImage},
			{ID: "detail", Type: domain.SlotTypeImage},
		},
	}

	slot, ok := entry.PrimaryImageSlot()
	if !ok {
		t.Fatal("expected a primary image slot")
	}
	if slot.ID != "photo" {
		t.Fatalf("expected priority slot %q, got %q", "photo", slot.ID)
	}
}

func TestAllowedTypesForTarget(t *testing.T) {
	textTypes := allowedTypesForTarget(fileTargetBodyImport)
	if strings.Join(textTypes, ",") != ".txt,.md" {
		t.Fatalf("unexpected text file types: %v", textTypes)
	}

	imageTypes := allowedTypesForTarget(fileTargetPrimaryImage)
	if strings.Join(imageTypes, ",") != ".png,.jpg,.jpeg,.webp" {
		t.Fatalf("unexpected image file types: %v", imageTypes)
	}
}

func TestInvalidBodyImportPreservesExistingState(t *testing.T) {
	model := NewModel()
	model.composition.Project.Content.Title = "Safe Title"
	model.composition.Project.Content.Body = "Existing body"
	model.composition.Project.Content.Signature = "Safe Signature"
	model.bodyInput.SetValue("Existing body")
	model.bodyImportInput.SetValue(filepath.Join(t.TempDir(), "missing.txt"))

	model = model.applyBodyImport()

	if !model.status(StepQuickStart).IsError {
		t.Fatal("expected invalid import to surface an inline error")
	}
	if got := model.composition.Project.Content.Title; got != "Safe Title" {
		t.Fatalf("title changed after failed import: %q", got)
	}
	if got := model.composition.Project.Content.Body; got != "Existing body" {
		t.Fatalf("body changed after failed import: %q", got)
	}
	if got := model.composition.Project.Content.Signature; got != "Safe Signature" {
		t.Fatalf("signature changed after failed import: %q", got)
	}
}

func TestReviewScreenPreservesEarlierSelections(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.setFocused(focusReviewExport)
	model.composition.Project.Content.Title = "Hello"
	model.composition.Project.Content.Body = "Review body"
	model.composition.Project.Content.Signature = "From Letterpress"
	model.composition.Project.Export.Format = domain.ExportFormatPNG
	model.composition.Project.Export.Out = "output/card.png"
	model.outputInput.SetValue("output/card.png")
	model.composition.SetDecorationEnabled("corner-ornament", true)

	view := model.View()

	for _, fragment := range []string{
		"Hello",
		"Review body",
		"From Letterpress",
		"output/card.png",
		"PNG",
		"1 selected",
	} {
		if !strings.Contains(view, fragment) {
			t.Fatalf("expected review view to contain %q, got %q", fragment, view)
		}
	}
}

func TestExportCompositionCallsExistingActors(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.outputInput.SetValue("output/review")
	model.composition.Project.Export.Out = "output/review"
	model.composition.Project.Export.Format = domain.ExportFormatPDF
	model.composition.Project.Options.Decorations = true

	origResolve := resolveTemplate
	origSave := saveProject
	origCompose := composeAndWrite
	defer func() {
		resolveTemplate = origResolve
		saveProject = origSave
		composeAndWrite = origCompose
	}()

	resolveTemplate = func(t domain.Template, project domain.Project) (templatepkg.ResolvedTemplate, error) {
		return templatepkg.ResolvedTemplate{TemplateID: t.ID}, nil
	}

	savedProjectPath := ""
	saveProject = func(path string, project domain.Project) error {
		savedProjectPath = path
		return nil
	}

	var composed export.Options
	composeAndWrite = func(res templatepkg.ResolvedTemplate, opts export.Options) (string, error) {
		composed = opts
		return opts.Out + ".pdf", nil
	}

	model = model.exportComposition()

	if savedProjectPath == "" {
		t.Fatal("expected export flow to save the project")
	}
	if composed.Out != "output/review" {
		t.Fatalf("expected export out %q, got %q", "output/review", composed.Out)
	}
	if !composed.Decorations {
		t.Fatal("expected decorations flag to propagate")
	}
	if status := model.status(StepReview); status.IsError {
		t.Fatalf("expected export success, got message %q", status.Text)
	}
	if !strings.Contains(model.status(StepReview).Text, "Export saved to") {
		t.Fatalf("unexpected review message %q", model.status(StepReview).Text)
	}
}

func writeTempFile(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
