package domain

import (
	"strings"
	"testing"
)

func TestProjectValidateReportsMissingFields(t *testing.T) {
	project := Project{}

	err := project.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	for _, fragment := range []string{
		"version: must be 1",
		"template: is required",
		"page.size",
		"page.orientation",
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected %q in validation error, got %q", fragment, err.Error())
		}
	}
}

func TestTemplateValidateReportsStyleReferenceErrors(t *testing.T) {
	template := Template{
		Version: CurrentSchemaVersion,
		ID:      "classic-letter",
		Page: TemplatePage{
			SupportedSizes:     []PageSize{PageSizeA4},
			DefaultOrientation: OrientationPortrait,
		},
		Slots: []Slot{
			{
				ID:    "body",
				Type:  SlotTypeText,
				WMM:   100,
				HMM:   80,
				Style: "missing-style",
			},
		},
	}

	err := template.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "slots[0].style: must reference a defined style") {
		t.Fatalf("expected missing style validation error, got %q", err.Error())
	}
}
