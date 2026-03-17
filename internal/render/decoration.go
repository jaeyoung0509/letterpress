package render

import (
	"fmt"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/tdewolff/canvas"
)

// RenderDecorationFrame draws a simple stroke frame around the slot when the decoration is enabled.
func RenderDecorationFrame(ctx *canvas.Context, node Node, enabled bool) error {
	if !enabled || node.Decoration == nil {
		return nil
	}

	if err := validateDecorationAsset(node.Decoration); err != nil {
		return err
	}

	width := float64(node.CanvasRect.WMM)
	height := float64(node.CanvasRect.HMM)
	if width <= 0 || height <= 0 {
		return fmt.Errorf("slot %q has invalid bounds %vx%v", node.SlotID, width, height)
	}

	ctx.Push()
	defer ctx.Pop()

	ctx.SetFillColor(canvas.Transparent)
	ctx.SetStrokeColor(canvas.Hex("#d98880"))
	ctx.SetStrokeWidth(0.5)

	path := canvas.Rectangle(width, height)
	ctx.DrawPath(float64(node.CanvasRect.XMM), float64(node.CanvasRect.YMM), path)

	return nil
}

func validateDecorationAsset(asset *domain.Asset) error {
	if asset.Kind != domain.AssetKindDecoration {
		return fmt.Errorf("asset %s is not a decoration", asset.ID)
	}
	if asset.Path == "" {
		return fmt.Errorf("decoration asset %s has no path", asset.ID)
	}
	return nil
}
