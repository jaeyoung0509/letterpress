package render

import (
	"fmt"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
	"github.com/tdewolff/canvas"
)

type Renderer interface {
	Compose(resolved templatepkg.ResolvedTemplate) (*Document, error)
}

type CanvasRenderer struct{}

func NewCanvasRenderer() Renderer {
	return CanvasRenderer{}
}

func (CanvasRenderer) Compose(resolved templatepkg.ResolvedTemplate) (*Document, error) {
	dimensions, err := domain.ISOPage(resolved.ProjectPage.Size, resolved.ProjectPage.Orientation)
	if err != nil {
		return nil, fmt.Errorf("resolve page dimensions: %w", err)
	}

	page := Page{
		Size:        resolved.ProjectPage.Size,
		Orientation: resolved.ProjectPage.Orientation,
		Dimensions:  dimensions,
	}

	surface := canvas.New(float64(dimensions.WidthMM), float64(dimensions.HeightMM))
	ctx := canvas.NewContext(surface)

	nodes := make([]Node, 0, len(resolved.Slots))
	for _, slot := range resolved.Slots {
		nodes = append(nodes, composeNode(page, slot))
	}

	return &Document{
		templateID: resolved.TemplateID,
		page:       page,
		nodes:      nodes,
		surface:    surface,
		ctx:        ctx,
	}, nil
}

func composeNode(page Page, resolved templatepkg.ResolvedSlot) Node {
	layoutRect := Rect{
		XMM: domain.Millimetres(resolved.Slot.XMM),
		YMM: domain.Millimetres(resolved.Slot.YMM),
		WMM: domain.Millimetres(resolved.Slot.WMM),
		HMM: domain.Millimetres(resolved.Slot.HMM),
	}

	return Node{
		SlotID:     resolved.Slot.ID,
		SlotType:   resolved.Slot.Type,
		LayoutRect: layoutRect,
		CanvasRect: page.LayoutToCanvasRect(layoutRect),
		Style:      cloneStyle(resolved.Style),
		Text:       resolved.Text,
		ImagePath:  resolved.ImagePath,
		Decoration: cloneAsset(resolved.Decoration),
		Required:   resolved.Slot.Required,
	}
}

func cloneStyle(style *domain.Style) *domain.Style {
	if style == nil {
		return nil
	}

	copy := *style
	return &copy
}

func cloneAsset(asset *domain.Asset) *domain.Asset {
	if asset == nil {
		return nil
	}

	copy := *asset
	return &copy
}
