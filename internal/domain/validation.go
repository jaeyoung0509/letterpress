package domain

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, err.Error())
	}
	return strings.Join(parts, "; ")
}

func (errs ValidationErrors) add(field, message string) ValidationErrors {
	return append(errs, ValidationError{
		Field:   field,
		Message: message,
	})
}

func (errs ValidationErrors) err() error {
	if len(errs) == 0 {
		return nil
	}
	return errs
}

func (p Project) Validate() error {
	var errs ValidationErrors

	if p.Version != CurrentSchemaVersion {
		errs = errs.add("version", fmt.Sprintf("must be %d", CurrentSchemaVersion))
	}
	if strings.TrimSpace(p.Template) == "" {
		errs = errs.add("template", "is required")
	}
	if !p.Page.Size.valid() {
		errs = errs.add("page.size", "must be one of A3, A4, A5, or A6")
	}
	if !p.Page.Orientation.valid() {
		errs = errs.add("page.orientation", "must be portrait or landscape")
	}
	if p.Options.RenderMode != "" && !p.Options.RenderMode.valid() {
		errs = errs.add("options.render_mode", "must be compose or ascii")
	}
	if p.Options.ASCII.Density < 0 {
		errs = errs.add("options.ascii.density", "must be greater than or equal to 0")
	}
	if p.Options.ASCII.Threshold < 0 || p.Options.ASCII.Threshold > 1 {
		errs = errs.add("options.ascii.threshold", "must be between 0 and 1")
	}
	if p.Options.ASCII.Contrast < 0 {
		errs = errs.add("options.ascii.contrast", "must be greater than or equal to 0")
	}
	if p.Options.ASCII.EdgeWeight < 0 || p.Options.ASCII.EdgeWeight > 1 {
		errs = errs.add("options.ascii.edge_weight", "must be between 0 and 1")
	}
	if p.Export.Format != "" && !p.Export.Format.valid() {
		errs = errs.add("export.format", "must be one of pdf, png, or svg")
	}

	for i, image := range p.Images {
		if strings.TrimSpace(image.Slot) == "" {
			errs = errs.add(fmt.Sprintf("images[%d].slot", i), "is required")
		}
		if strings.TrimSpace(image.Path) == "" {
			errs = errs.add(fmt.Sprintf("images[%d].path", i), "is required")
		}
	}

	return errs.err()
}

func (t Template) Validate() error {
	var errs ValidationErrors

	if t.Version != CurrentSchemaVersion {
		errs = errs.add("version", fmt.Sprintf("must be %d", CurrentSchemaVersion))
	}
	if strings.TrimSpace(t.ID) == "" {
		errs = errs.add("id", "is required")
	}
	if len(t.Page.SupportedSizes) == 0 {
		errs = errs.add("page.supported_sizes", "must include at least one page size")
	}
	for i, size := range t.Page.SupportedSizes {
		if !size.valid() {
			errs = errs.add(fmt.Sprintf("page.supported_sizes[%d]", i), "must be one of A3, A4, A5, or A6")
		}
	}
	if !t.Page.DefaultOrientation.valid() {
		errs = errs.add("page.default_orientation", "must be portrait or landscape")
	}
	if t.Layout.MarginMM < 0 {
		errs = errs.add("layout.margin_mm", "must be greater than or equal to 0")
	}
	if len(t.Slots) == 0 {
		errs = errs.add("slots", "must include at least one slot")
	}

	seenSlots := map[string]struct{}{}
	for i, slot := range t.Slots {
		fieldPrefix := fmt.Sprintf("slots[%d]", i)
		if strings.TrimSpace(slot.ID) == "" {
			errs = errs.add(fieldPrefix+".id", "is required")
		} else {
			if _, exists := seenSlots[slot.ID]; exists {
				errs = errs.add(fieldPrefix+".id", "must be unique")
			}
			seenSlots[slot.ID] = struct{}{}
		}
		if !slot.Type.valid() {
			errs = errs.add(fieldPrefix+".type", "must be one of text, image, or decoration")
		}
		if slot.WMM <= 0 {
			errs = errs.add(fieldPrefix+".w_mm", "must be greater than 0")
		}
		if slot.HMM <= 0 {
			errs = errs.add(fieldPrefix+".h_mm", "must be greater than 0")
		}
		if slot.Fit != "" && !slot.Fit.valid() {
			errs = errs.add(fieldPrefix+".fit", "must be contain or cover")
		}
		if slot.Style != "" {
			if _, ok := t.Styles[slot.Style]; !ok {
				errs = errs.add(fieldPrefix+".style", "must reference a defined style")
			}
		}
	}

	for name, style := range t.Styles {
		fieldPrefix := fmt.Sprintf("styles.%s", name)
		if strings.TrimSpace(style.Font) == "" {
			errs = errs.add(fieldPrefix+".font", "is required")
		}
		if style.SizePt <= 0 {
			errs = errs.add(fieldPrefix+".size_pt", "must be greater than 0")
		}
		if style.LineHeight < 0 {
			errs = errs.add(fieldPrefix+".line_height", "must be greater than or equal to 0")
		}
		if style.Align != "" && !style.Align.valid() {
			errs = errs.add(fieldPrefix+".align", "must be left, center, or right")
		}
		if style.MaxLines < 0 {
			errs = errs.add(fieldPrefix+".max_lines", "must be greater than or equal to 0")
		}
		if style.Overflow != "" && !style.Overflow.valid() {
			errs = errs.add(fieldPrefix+".overflow", "must be clip, truncate, or error")
		}
	}

	seenAssets := map[string]struct{}{}
	for i, asset := range t.Assets {
		fieldPrefix := fmt.Sprintf("assets[%d]", i)
		if strings.TrimSpace(asset.ID) == "" {
			errs = errs.add(fieldPrefix+".id", "is required")
		} else {
			if _, exists := seenAssets[asset.ID]; exists {
				errs = errs.add(fieldPrefix+".id", "must be unique")
			}
			seenAssets[asset.ID] = struct{}{}
		}
		if !asset.Kind.valid() {
			errs = errs.add(fieldPrefix+".kind", "must be decoration, image, or font")
		}
		if strings.TrimSpace(asset.Path) == "" {
			errs = errs.add(fieldPrefix+".path", "is required")
		}
	}

	return errs.err()
}

func (s PageSize) valid() bool {
	switch s {
	case PageSizeA3, PageSizeA4, PageSizeA5, PageSizeA6:
		return true
	default:
		return false
	}
}

func (o Orientation) valid() bool {
	switch o {
	case OrientationPortrait, OrientationLandscape:
		return true
	default:
		return false
	}
}

func (f ExportFormat) valid() bool {
	switch f {
	case ExportFormatPDF, ExportFormatPNG, ExportFormatSVG:
		return true
	default:
		return false
	}
}

func (m RenderMode) valid() bool {
	switch m {
	case RenderModeCompose, RenderModeASCII:
		return true
	default:
		return false
	}
}

func (t SlotType) valid() bool {
	switch t {
	case SlotTypeText, SlotTypeImage, SlotTypeDecoration:
		return true
	default:
		return false
	}
}

func (a TextAlign) valid() bool {
	switch a {
	case TextAlignLeft, TextAlignCenter, TextAlignRight:
		return true
	default:
		return false
	}
}

func (o OverflowMode) valid() bool {
	switch o {
	case OverflowModeClip, OverflowModeTruncate, OverflowModeError:
		return true
	default:
		return false
	}
}

func (f FitMode) valid() bool {
	switch f {
	case FitModeContain, FitModeCover:
		return true
	default:
		return false
	}
}

func (a AssetKind) valid() bool {
	switch a {
	case AssetKindDecoration, AssetKindImage, AssetKindFont:
		return true
	default:
		return false
	}
}
