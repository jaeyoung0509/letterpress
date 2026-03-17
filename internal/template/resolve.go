package template

import (
	"fmt"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

// ResolvedTemplate represents a template merged with a project instance.
type ResolvedTemplate struct {
	TemplateID   string
	Layout       domain.Layout
	TemplatePage domain.TemplatePage
	ProjectPage  domain.ProjectPage
	Slots        []ResolvedSlot
	Styles       map[string]domain.Style
	Assets       map[string]domain.Asset
}

// ResolvedSlot carries template slot metadata together with resolved content bindings.
type ResolvedSlot struct {
	Slot       domain.Slot
	Style      *domain.Style
	Text       string
	ImagePath  string
	Decoration *domain.Asset
}

// Resolve materializes a template and project pairing into concrete slot data for rendering.
func Resolve(template domain.Template, project domain.Project) (ResolvedTemplate, error) {
	pageSize, err := resolvePageSize(template, project)
	if err != nil {
		return ResolvedTemplate{}, err
	}

	orientation, err := resolveOrientation(template, project)
	if err != nil {
		return ResolvedTemplate{}, err
	}

	assets := make(map[string]domain.Asset, len(template.Assets))
	for _, asset := range template.Assets {
		assets[asset.ID] = asset
	}

	slots := make([]ResolvedSlot, 0, len(template.Slots))
	for _, slot := range template.Slots {
		resolved, err := resolveSlot(slot, template.Styles, assets, project.Content, project.Images)
		if err != nil {
			return ResolvedTemplate{}, err
		}
		slots = append(slots, resolved)
	}

	resolved := ResolvedTemplate{
		TemplateID:   template.ID,
		Layout:       template.Layout,
		TemplatePage: template.Page,
		ProjectPage: domain.ProjectPage{
			Size:        pageSize,
			Orientation: orientation,
		},
		Slots:  slots,
		Styles: template.Styles,
		Assets: assets,
	}

	return resolved, nil
}

func resolveSlot(slot domain.Slot, styles map[string]domain.Style, assets map[string]domain.Asset, content domain.Content, images []domain.ImageBinding) (ResolvedSlot, error) {
	resolved := ResolvedSlot{Slot: slot}

	if slot.Style != "" {
		if style, ok := styles[slot.Style]; ok {
			copy := style
			resolved.Style = &copy
		}
	}

	switch slot.Type {
	case domain.SlotTypeText:
		resolved.Text = resolveTextContent(content, slot.ID)
		if slot.Required && resolved.Text == "" {
			return ResolvedSlot{}, fmt.Errorf("required text slot %s is empty", slot.ID)
		}
	case domain.SlotTypeImage:
		path, ok := findImageBinding(images, slot.ID)
		if slot.Required && !ok {
			return ResolvedSlot{}, fmt.Errorf("required image slot %s is missing a binding", slot.ID)
		}
		resolved.ImagePath = path
	case domain.SlotTypeDecoration:
		asset, ok := assets[slot.ID]
		if slot.Required && !ok {
			return ResolvedSlot{}, fmt.Errorf("required decoration slot %s lacks an asset", slot.ID)
		}
		if ok {
			copy := asset
			resolved.Decoration = &copy
		}
	default:
		return ResolvedSlot{}, fmt.Errorf("unsupported slot type %s", slot.Type)
	}

	return resolved, nil
}

func resolvePageSize(template domain.Template, project domain.Project) (domain.PageSize, error) {
	if len(template.Page.SupportedSizes) == 0 {
		return "", fmt.Errorf("template %s defines no supported page sizes", template.ID)
	}

	size := project.Page.Size
	if size == "" {
		size = template.Page.SupportedSizes[0]
	}

	if !containsSize(template.Page.SupportedSizes, size) {
		return "", fmt.Errorf("template %s does not support page size %s", template.ID, size)
	}

	return size, nil
}

func resolveOrientation(template domain.Template, project domain.Project) (domain.Orientation, error) {
	orientation := project.Page.Orientation
	if orientation == "" {
		orientation = template.Page.DefaultOrientation
	}
	if orientation == "" {
		orientation = domain.OrientationPortrait
	}

	if !isValidOrientation(orientation) {
		return "", fmt.Errorf("invalid orientation %s", orientation)
	}

	return orientation, nil
}

func containsSize(sizes []domain.PageSize, size domain.PageSize) bool {
	for _, candidate := range sizes {
		if candidate == size {
			return true
		}
	}
	return false
}

func isValidOrientation(value domain.Orientation) bool {
	return value == domain.OrientationPortrait || value == domain.OrientationLandscape
}

func resolveTextContent(content domain.Content, slotID string) string {
	switch slotID {
	case "title":
		return content.Title
	case "body":
		return content.Body
	case "signature":
		return content.Signature
	default:
		return ""
	}
}

func findImageBinding(bindings []domain.ImageBinding, slotID string) (string, bool) {
	for _, binding := range bindings {
		if binding.Slot == slotID {
			return binding.Path, true
		}
	}
	return "", false
}
