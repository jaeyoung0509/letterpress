package render

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/schema"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
)

func TestRenderTextSlotsDrawsSampleComposition(t *testing.T) {
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

	renderer := NewCanvasRenderer()
	document, err := renderer.Compose(resolved)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}

	if err := RenderTextSlots(document); err != nil {
		t.Fatalf("RenderTextSlots() error = %v", err)
	}
	if document.Canvas().Empty() {
		t.Fatal("expected the canvas to contain text drawing operations")
	}
}

func TestBuildTextBoxReturnsErrorForOverflowModeError(t *testing.T) {
	node := Node{
		SlotID:   "body",
		SlotType: domain.SlotTypeText,
		Text:     strings.Repeat("overflow ", 80),
		Style: &domain.Style{
			Font:       "SansBody",
			SizePt:     12,
			LineHeight: 1.2,
			Overflow:   domain.OverflowModeError,
		},
		CanvasRect: Rect{
			WMM: 30,
			HMM: 20,
		},
	}

	if _, err := buildTextBox(node); err == nil {
		t.Fatal("expected overflow mode error when text cannot fit")
	}
}

func TestBuildTextBoxTruncatesToFit(t *testing.T) {
	node := Node{
		SlotID:   "body",
		SlotType: domain.SlotTypeText,
		Text:     strings.Repeat("truncate ", 60),
		Style: &domain.Style{
			Font:       "SansBody",
			SizePt:     12,
			LineHeight: 1.2,
			Overflow:   domain.OverflowModeTruncate,
		},
		CanvasRect: Rect{
			WMM: 45,
			HMM: 18,
		},
	}

	text, err := buildTextBox(node)
	if err != nil {
		t.Fatalf("buildTextBox() error = %v", err)
	}
	if text == nil {
		t.Fatal("expected a text box")
	}
	if text.OverflowsX || text.OverflowsY {
		t.Fatal("expected truncated text to fit the slot")
	}
	if !strings.HasSuffix(text.Text, "...") {
		t.Fatalf("expected truncated text to end with ellipsis, got %q", text.Text)
	}
}

func TestEffectiveTextStyleAppliesOpinionatedDefaults(t *testing.T) {
	title := effectiveTextStyle(Node{SlotID: "title"})
	if title.Align != domain.TextAlignCenter {
		t.Fatalf("title align = %q, want center", title.Align)
	}
	if title.Font != "SerifDisplay" {
		t.Fatalf("title font = %q, want SerifDisplay", title.Font)
	}

	body := effectiveTextStyle(Node{SlotID: "body"})
	if body.Align != domain.TextAlignLeft {
		t.Fatalf("body align = %q, want left", body.Align)
	}
	if body.Font != "SansBody" {
		t.Fatalf("body font = %q, want SansBody", body.Font)
	}

	signature := effectiveTextStyle(Node{SlotID: "signature"})
	if signature.Align != domain.TextAlignRight {
		t.Fatalf("signature align = %q, want right", signature.Align)
	}
	if signature.Overflow != domain.OverflowModeTruncate {
		t.Fatalf("signature overflow = %q, want truncate", signature.Overflow)
	}
}
