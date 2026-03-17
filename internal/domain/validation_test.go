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

func TestProjectValidateRejectsInvalidASCIIOptions(t *testing.T) {
	project := Project{
		Version:  CurrentSchemaVersion,
		Template: "classic-letter-a4",
		Page: ProjectPage{
			Size:        PageSizeA4,
			Orientation: OrientationPortrait,
		},
		Options: ProjectOptions{
			RenderMode: "vectorized",
			ASCII: ASCIIOptions{
				Density:    -1,
				Threshold:  1.2,
				Contrast:   -0.5,
				EdgeWeight: 2,
			},
		},
	}

	err := project.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	for _, fragment := range []string{
		"options.render_mode: must be compose or ascii",
		"options.ascii.density: must be greater than or equal to 0",
		"options.ascii.threshold: must be between 0 and 1",
		"options.ascii.contrast: must be greater than or equal to 0",
		"options.ascii.edge_weight: must be between 0 and 1",
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected %q in validation error, got %q", fragment, err.Error())
		}
	}
}

func TestProjectValidateAcceptsASCIIOptions(t *testing.T) {
	project := Project{
		Version:  CurrentSchemaVersion,
		Template: "classic-letter-a4",
		Page: ProjectPage{
			Size:        PageSizeA4,
			Orientation: OrientationPortrait,
		},
		Options: ProjectOptions{
			RenderMode: RenderModeASCII,
			ASCII: ASCIIOptions{
				Charset:    "@# ",
				Density:    96,
				Threshold:  0.42,
				Contrast:   1.25,
				Invert:     true,
				EdgeWeight: 0.35,
			},
		},
	}

	if err := project.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
