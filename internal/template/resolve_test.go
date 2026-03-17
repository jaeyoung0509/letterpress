package template

import (
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func TestResolveTemplateProducesExpectedSlotData(t *testing.T) {
	template := domain.Template{
		ID: "classic-letter",
		Page: domain.TemplatePage{
			SupportedSizes:     []domain.PageSize{domain.PageSizeA4},
			DefaultOrientation: domain.OrientationPortrait,
		},
		Layout: domain.Layout{MarginMM: 12},
		Styles: map[string]domain.Style{
			"title": {Font: "SerifDisplay", SizePt: 24},
			"body":  {Font: "SansBody", SizePt: 12},
		},
		Slots: []domain.Slot{
			{ID: "title", Type: domain.SlotTypeText, Style: "title", XMM: 20, YMM: 20, WMM: 170, HMM: 30, Required: true},
			{ID: "body", Type: domain.SlotTypeText, Style: "body", XMM: 20, YMM: 60, WMM: 170, HMM: 150},
			{ID: "photo", Type: domain.SlotTypeImage, XMM: 20, YMM: 200, WMM: 80, HMM: 60, Required: true},
			{ID: "ornament", Type: domain.SlotTypeDecoration},
		},
		Assets: []domain.Asset{
			{ID: "ornament", Kind: domain.AssetKindDecoration, Path: "assets/floral.png"},
		},
	}

	project := domain.Project{
		Version:  domain.CurrentSchemaVersion,
		Template: template.ID,
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA4,
			Orientation: domain.OrientationPortrait,
		},
		Content: domain.Content{
			Title: "Happy Birthday",
			Body:  "Wishing you a lovely day of letters and tea.",
		},
		Images: []domain.ImageBinding{
			{Slot: "photo", Path: "images/hero.png"},
		},
	}

	resolved, err := Resolve(template, project)
	if err != nil {
		t.Fatalf("Resolve returned an error: %v", err)
	}
	if resolved.TemplateID != template.ID {
		t.Fatalf("resolved TemplateID = %s, want %s", resolved.TemplateID, template.ID)
	}
	if resolved.ProjectPage != project.Page {
		t.Fatalf("resolved ProjectPage = %+v, want %+v", resolved.ProjectPage, project.Page)
	}

	slots := mapSlots(resolved.Slots)
	if slots["title"].Text != project.Content.Title {
		t.Fatalf("title slot text = %q, want %q", slots["title"].Text, project.Content.Title)
	}
	if slots["title"].Style == nil || slots["title"].Style.Font != template.Styles["title"].Font {
		t.Fatalf("title slot style not wired correctly")
	}
	if slots["photo"].ImagePath != "images/hero.png" {
		t.Fatalf("photo slot path = %q, want %q", slots["photo"].ImagePath, "images/hero.png")
	}
	if slots["ornament"].Decoration == nil || slots["ornament"].Decoration.Path != "assets/floral.png" {
		t.Fatalf("ornament decoration not resolved")
	}
}

func TestResolveTemplateMissingRequiredImage(t *testing.T) {
	template := domain.Template{
		ID: "hero-card",
		Page: domain.TemplatePage{
			SupportedSizes: []domain.PageSize{domain.PageSizeA4},
		},
		Slots: []domain.Slot{
			{ID: "hero", Type: domain.SlotTypeImage, Required: true},
		},
	}

	project := domain.Project{
		Version:  domain.CurrentSchemaVersion,
		Template: template.ID,
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA4,
			Orientation: domain.OrientationPortrait,
		},
	}

	if _, err := Resolve(template, project); err == nil {
		t.Fatal("expected Resolve to fail when an image slot is required but missing")
	} else if !strings.Contains(err.Error(), "hero") {
		t.Fatalf("error message = %q, want it to mention the slot ID", err)
	}
}

func TestResolveTemplateSupportsCommonTextSlotAliases(t *testing.T) {
	template := domain.Template{
		ID: "note-card",
		Page: domain.TemplatePage{
			SupportedSizes: []domain.PageSize{domain.PageSizeA6},
		},
		Slots: []domain.Slot{
			{ID: "greeting", Type: domain.SlotTypeText, Required: true},
			{ID: "note", Type: domain.SlotTypeText, Required: true},
			{ID: "signoff", Type: domain.SlotTypeText, Required: true},
		},
	}

	project := domain.Project{
		Version:  domain.CurrentSchemaVersion,
		Template: template.ID,
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA6,
			Orientation: domain.OrientationLandscape,
		},
		Content: domain.Content{
			Title:     "For You",
			Body:      "A short note that should fill the card body slot.",
			Signature: "Warmly, Alex",
		},
	}

	resolved, err := Resolve(template, project)
	if err != nil {
		t.Fatalf("Resolve returned an error: %v", err)
	}

	slots := mapSlots(resolved.Slots)
	if slots["greeting"].Text != project.Content.Title {
		t.Fatalf("greeting slot text = %q, want %q", slots["greeting"].Text, project.Content.Title)
	}
	if slots["note"].Text != project.Content.Body {
		t.Fatalf("note slot text = %q, want %q", slots["note"].Text, project.Content.Body)
	}
	if slots["signoff"].Text != project.Content.Signature {
		t.Fatalf("signoff slot text = %q, want %q", slots["signoff"].Text, project.Content.Signature)
	}
}

func TestResolveTemplateRejectsSlotsOutsideResolvedPage(t *testing.T) {
	template := domain.Template{
		ID: "bad-card",
		Page: domain.TemplatePage{
			SupportedSizes:     []domain.PageSize{domain.PageSizeA6},
			DefaultOrientation: domain.OrientationLandscape,
		},
		Slots: []domain.Slot{
			{ID: "note", Type: domain.SlotTypeText, XMM: 10, YMM: 100, WMM: 120, HMM: 20},
		},
	}

	project := domain.Project{
		Version:  domain.CurrentSchemaVersion,
		Template: template.ID,
		Page: domain.ProjectPage{
			Size:        domain.PageSizeA6,
			Orientation: domain.OrientationLandscape,
		},
		Content: domain.Content{
			Body: "text",
		},
	}

	if _, err := Resolve(template, project); err == nil {
		t.Fatal("expected Resolve to reject out-of-bounds slots")
	} else if !strings.Contains(err.Error(), "page bounds") {
		t.Fatalf("expected bounds error, got %q", err)
	}
}

func mapSlots(slots []ResolvedSlot) map[string]ResolvedSlot {
	result := make(map[string]ResolvedSlot, len(slots))
	for _, slot := range slots {
		result[slot.Slot.ID] = slot
	}
	return result
}
