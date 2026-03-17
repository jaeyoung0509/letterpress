package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/export"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
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

func TestReviewPathInputApplies(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.reviewPathInput = "exports/card"
	model.composition.Project.Export.Out = ""

	model = model.applyReviewPathInput()

	if model.composition.Project.Export.Out != "exports/card" {
		t.Fatalf("expected export path to update, got %q", model.composition.Project.Export.Out)
	}
	if model.reviewMessage == "" {
		t.Fatalf("expected review message after applying path")
	}
	if model.reviewPathInput != "" {
		t.Fatalf("expected review path buffer to clear, got %q", model.reviewPathInput)
	}
}

func TestReviewToggleFormatCycles(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.composition.Project.Export.Format = domain.ExportFormatPDF

	model = model.toggleExportFormat()
	if model.composition.Project.Export.Format != domain.ExportFormatPNG {
		t.Fatalf("expected format to toggle to PNG, got %s", model.composition.Project.Export.Format)
	}

	model = model.toggleExportFormat()
	if model.composition.Project.Export.Format != domain.ExportFormatPDF {
		t.Fatalf("expected format to toggle back to PDF, got %s", model.composition.Project.Export.Format)
	}
}

func TestReviewExportRequiresPath(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.composition.Project.Export.Out = ""

	model = model.exportComposition()
	if !strings.Contains(model.reviewMessage, "set an export path") {
		t.Fatalf("expected error message about export path, got %q", model.reviewMessage)
	}
	if !model.reviewError {
		t.Fatalf("expected review error to be true")
	}
}

func TestReviewExportCallsActors(t *testing.T) {
	model := NewModel()
	model.state.Current = StepReview
	model.composition.Project.Export.Out = "output/review"
	model.composition.DecorationSelections = map[string]bool{"ribbon": true}

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

	saved := ""
	saveProject = func(path string, project domain.Project) error {
		saved = path
		return nil
	}

	var composed export.Options
	composeAndWrite = func(res templatepkg.ResolvedTemplate, opts export.Options) (string, error) {
		composed = opts
		return opts.Out + ".done", nil
	}

	model = model.exportComposition()

	if saved == "" {
		t.Fatalf("expected project to be saved")
	}
	if !strings.Contains(model.reviewMessage, "export saved to") {
		t.Fatalf("unexpected review message: %q", model.reviewMessage)
	}
	if !composed.Decorations {
		t.Fatalf("expected decorations flag to propagate")
	}
	if composed.Out != "output/review" {
		t.Fatalf("expected export out to be output/review, got %s", composed.Out)
	}
}

func TestImageAssignmentFlow(t *testing.T) {
	model := NewModel()
	found := false
	for idx, entry := range model.templates {
		if len(entry.ImageSlots) > 0 {
			model = model.selectTemplate(idx)
			found = true
			break
		}
	}
	if !found {
		t.Skip("no templates with image slots available")
	}

	model.state.Current = StepImages
	model.imageInput = "./assets/custom.jpg"
	_, ok := model.currentImageSlot()
	if !ok {
		t.Skip("selected template has no image slots")
	}

	model = model.assignCurrentImage()
	slot, _ := model.currentImageSlot()
	if path := model.imagePathForSlot(slot.ID); path != "./assets/custom.jpg" {
		t.Fatalf("expected image assigned to %q, got %q", slot.ID, path)
	}
}

func TestDecorationToggleFlow(t *testing.T) {
	model := NewModel()
	found := false
	for idx, entry := range model.templates {
		if len(entry.DecorationAssets) > 0 {
			model = model.selectTemplate(idx)
			found = true
			break
		}
	}
	if !found {
		t.Skip("no templates with decoration assets available")
	}

	model.state.Current = StepDecorations
	asset, ok := model.currentDecorationAsset()
	if !ok {
		t.Skip("selected template has no decoration assets")
	}

	model = model.toggleCurrentDecoration()
	if !model.composition.DecorationSelections[asset.ID] {
		t.Fatalf("expected decoration %q to be enabled", asset.ID)
	}

	model = model.toggleCurrentDecoration()
	if model.composition.DecorationSelections[asset.ID] {
		t.Fatalf("expected decoration %q to be disabled after toggling twice", asset.ID)
	}
}
