package render

import (
	"fmt"
	"strings"

	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
)

type ASCIIComposition struct {
	ImagePath string
	Page      Page
	Options   domain.ASCIIOptions
	Art       asciipkg.Art
}

func ComposeASCII(page domain.ProjectPage, imagePath string, options domain.ASCIIOptions) (*ASCIIComposition, error) {
	if strings.TrimSpace(imagePath) == "" {
		return nil, fmt.Errorf("image path is required")
	}

	dimensions, err := domain.ISOPage(page.Size, page.Orientation)
	if err != nil {
		return nil, fmt.Errorf("resolve ascii page dimensions: %w", err)
	}

	art, err := asciipkg.RenderPath(imagePath, options)
	if err != nil {
		return nil, fmt.Errorf("render ascii art: %w", err)
	}

	return &ASCIIComposition{
		ImagePath: imagePath,
		Page: Page{
			Size:        page.Size,
			Orientation: page.Orientation,
			Dimensions:  dimensions,
		},
		Options: options,
		Art:     art,
	}, nil
}
