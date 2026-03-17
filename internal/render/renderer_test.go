package render

import (
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

func TestComposeBuildsCanvasDocument(t *testing.T) {
	renderer := NewCanvasRenderer()
	resolved := sampleResolvedTemplate()

	document, err := renderer.Compose(resolved)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if document == nil {
		t.Fatal("Compose() returned a nil document")
	}
	if document.Canvas() == nil {
		t.Fatal("expected a canvas surface")
	}
	if document.Context() == nil {
		t.Fatal("expected a canvas context")
	}
	if got := document.TemplateID(); got != resolved.TemplateID {
		t.Fatalf("document.TemplateID() = %q, want %q", got, resolved.TemplateID)
	}
	if got := len(document.Nodes()); got != len(resolved.Slots) {
		t.Fatalf("len(document.Nodes()) = %d, want %d", got, len(resolved.Slots))
	}
}

func TestComposeUsesISOPageDimensions(t *testing.T) {
	renderer := NewCanvasRenderer()
	resolved := sampleResolvedTemplate()
	resolved.ProjectPage = domain.ProjectPage{
		Size:        domain.PageSizeA5,
		Orientation: domain.OrientationLandscape,
	}

	document, err := renderer.Compose(resolved)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	page := document.Page()
	if page.Dimensions.WidthMM != domain.Millimetres(210) {
		t.Fatalf("page width = %v, want 210mm", page.Dimensions.WidthMM)
	}
	if page.Dimensions.HeightMM != domain.Millimetres(148) {
		t.Fatalf("page height = %v, want 148mm", page.Dimensions.HeightMM)
	}
}

func TestComposeConvertsTopLeftCoordinatesToCanvasSpace(t *testing.T) {
	renderer := NewCanvasRenderer()
	resolved := sampleResolvedTemplate()

	document, err := renderer.Compose(resolved)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	nodes := document.Nodes()
	title := nodes[0]
	if title.LayoutRect.XMM != domain.Millimetres(20) {
		t.Fatalf("layout x = %v, want 20mm", title.LayoutRect.XMM)
	}
	if title.CanvasRect.YMM != domain.Millimetres(247) {
		t.Fatalf("canvas y = %v, want 247mm", title.CanvasRect.YMM)
	}
}

func TestComposeMapsResolvedSlotPayloads(t *testing.T) {
	renderer := NewCanvasRenderer()
	resolved := sampleResolvedTemplate()

	document, err := renderer.Compose(resolved)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	nodes := document.Nodes()
	textNode := nodes[0]
	imageNode := nodes[1]
	decorationNode := nodes[2]

	if textNode.Text != "Hello" {
		t.Fatalf("text node content = %q, want %q", textNode.Text, "Hello")
	}
	if textNode.Style == nil || textNode.Style.Font != "SerifDisplay" {
		t.Fatalf("text node style was not copied correctly")
	}
	if imageNode.ImagePath != "images/hero.png" {
		t.Fatalf("image path = %q, want %q", imageNode.ImagePath, "images/hero.png")
	}
	if decorationNode.Decoration == nil || decorationNode.Decoration.Path != "assets/ribbon.svg" {
		t.Fatalf("decoration asset was not copied correctly")
	}
}

func TestComposeRejectsInvalidPageSizing(t *testing.T) {
	renderer := NewCanvasRenderer()
	resolved := sampleResolvedTemplate()
	resolved.ProjectPage.Size = "A0"

	if _, err := renderer.Compose(resolved); err == nil {
		t.Fatal("expected Compose() to reject an unsupported page size")
	}
}

func sampleResolvedTemplate() templatepkg.ResolvedTemplate {
	titleStyle := domain.Style{
		Font:   "SerifDisplay",
		SizePt: 22,
	}
	decoration := domain.Asset{
		ID:   "ribbon",
		Kind: domain.AssetKindDecoration,
		Path: "assets/ribbon.svg",
	}

	return templatepkg.ResolvedTemplate{
		TemplateID: "classic-letter-a4",
		ProjectPage: domain.ProjectPage{
			Size:        domain.PageSizeA4,
			Orientation: domain.OrientationPortrait,
		},
		Slots: []templatepkg.ResolvedSlot{
			{
				Slot: domain.Slot{
					ID:   "title",
					Type: domain.SlotTypeText,
					XMM:  20,
					YMM:  20,
					WMM:  170,
					HMM:  30,
				},
				Style: &titleStyle,
				Text:  "Hello",
			},
			{
				Slot: domain.Slot{
					ID:   "photo",
					Type: domain.SlotTypeImage,
					XMM:  20,
					YMM:  70,
					WMM:  90,
					HMM:  90,
				},
				ImagePath: "images/hero.png",
			},
			{
				Slot: domain.Slot{
					ID:   "ribbon",
					Type: domain.SlotTypeDecoration,
					XMM:  15,
					YMM:  15,
					WMM:  25,
					HMM:  10,
				},
				Decoration: &decoration,
			},
		},
	}
}
