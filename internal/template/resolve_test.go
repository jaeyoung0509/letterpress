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
			{ID: "photo", Type: domain.SlotTypeImage, XMM: 20, YMM: 220, WMM: 80, HMM: 80, Required: true},
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

func mapSlots(slots []ResolvedSlot) map[string]ResolvedSlot {
	result := make(map[string]ResolvedSlot, len(slots))
	for _, slot := range slots {
		result[slot.Slot.ID] = slot
	}
	return result
}
