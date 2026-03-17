package tui

import (
	"fmt"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

type CompositionState struct {
	Project              domain.Project
	DecorationSelections map[string]bool
	BodyImportPath       string
}

func newCompositionState() CompositionState {
	return CompositionState{
		Project: domain.Project{
			Version:  domain.CurrentSchemaVersion,
			Template: "classic-letter-a4",
			Page: domain.ProjectPage{
				Size:        domain.PageSizeA4,
				Orientation: domain.OrientationPortrait,
			},
			Content: domain.Content{},
			Images:  nil,
			Options: domain.ProjectOptions{
				Decorations: false,
			},
			Export: domain.ExportOptions{
				Format: domain.ExportFormatPDF,
				Out:    "output/letterpress",
			},
		},
		DecorationSelections: map[string]bool{},
	}
}

func (c CompositionState) Summary() string {
	project := c.Project
	return fmt.Sprintf("template=%s size=%s orientation=%s", project.Template, project.Page.Size, project.Page.Orientation)
}

func (c CompositionState) DecorationEnabled(id string) bool {
	if c.DecorationSelections == nil {
		return false
	}
	return c.DecorationSelections[id]
}

func (c CompositionState) DecorationCount() int {
	return len(c.DecorationSelections)
}

func (c *CompositionState) EnableAllDecorations(assets []domain.Asset) {
	if c.DecorationSelections == nil {
		c.DecorationSelections = map[string]bool{}
	}
	for _, asset := range assets {
		c.DecorationSelections[asset.ID] = true
	}
	c.Project.Options.Decorations = len(c.DecorationSelections) > 0
}

func (c *CompositionState) DisableAllDecorations() {
	c.DecorationSelections = map[string]bool{}
	c.Project.Options.Decorations = false
}

func (c *CompositionState) SetDecorationEnabled(id string, enabled bool) {
	if c.DecorationSelections == nil {
		c.DecorationSelections = map[string]bool{}
	}
	if enabled {
		c.DecorationSelections[id] = true
	} else {
		delete(c.DecorationSelections, id)
	}
	c.Project.Options.Decorations = len(c.DecorationSelections) > 0
}

func (c *CompositionState) SetImage(slotID, path string) {
	for i, binding := range c.Project.Images {
		if binding.Slot == slotID {
			c.Project.Images[i].Path = path
			return
		}
	}

	c.Project.Images = append(c.Project.Images, domain.ImageBinding{
		Slot: slotID,
		Path: path,
	})
}

func (c *CompositionState) RemoveImage(slotID string) {
	filtered := c.Project.Images[:0]
	for _, binding := range c.Project.Images {
		if binding.Slot != slotID {
			filtered = append(filtered, binding)
		}
	}
	c.Project.Images = filtered
}

func (c CompositionState) ImagePath(slotID string) string {
	for _, binding := range c.Project.Images {
		if binding.Slot == slotID {
			return binding.Path
		}
	}
	return ""
}
