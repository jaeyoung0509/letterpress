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
				Mode:          "posterized",
				Density:       -1,
				Threshold:     1.2,
				Contrast:      -0.5,
				Gamma:         -1,
				EdgeWeight:    2,
				EdgeThreshold: 2,
				Dither:        "blue-noise",
				CellAspect:    -0.2,
			},
		},
	}

	err := project.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	for _, fragment := range []string{
		"options.render_mode: must be compose or ascii",
		"options.ascii.mode: must be tone, outline, fill, hybrid, or vector",
		"options.ascii.density: must be greater than or equal to 0",
		"options.ascii.threshold: must be between 0 and 1",
		"options.ascii.contrast: must be greater than or equal to 0",
		"options.ascii.gamma: must be greater than or equal to 0",
		"options.ascii.edge_weight: must be between 0 and 1",
		"options.ascii.edge_threshold: must be between 0 and 1",
		"options.ascii.dither: must be off or floyd",
		"options.ascii.cell_aspect: must be greater than or equal to 0",
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
				Mode:          ASCIIModeHybrid,
				ToneCharset:   "@# ",
				FillText:      "Happy Birthday",
				Density:       96,
				Threshold:     0.42,
				Contrast:      1.25,
				Gamma:         1.1,
				Invert:        true,
				EdgeWeight:    0.35,
				EdgeThreshold: 0.4,
				Dither:        DitherModeFloyd,
				CellAspect:    0.5,
			},
		},
	}

	if err := project.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProjectValidateRequiresFillTextForFillModes(t *testing.T) {
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
				Mode: ASCIIModeVector,
			},
		},
	}

	err := project.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "options.ascii.fill_text: is required for fill, hybrid, and vector modes") {
		t.Fatalf("unexpected error: %v", err)
	}
}
