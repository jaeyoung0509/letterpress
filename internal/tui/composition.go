package tui

import (
	"fmt"
	"github.com/jaeyoung0509/letterpress/internal/domain"
)

type CompositionState struct {
	Project domain.Project
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
				Decorations: true,
			},
			Export: domain.ExportOptions{
				Format: domain.ExportFormatPDF,
			},
		},
	}
}

func (c CompositionState) Summary() string {
	project := c.Project
	return fmt.Sprintf("template=%s size=%s orientation=%s", project.Template, project.Page.Size, project.Page.Orientation)
}
