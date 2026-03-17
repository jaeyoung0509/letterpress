package render

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/tdewolff/canvas"
	_ "golang.org/x/image/webp"
)

// RenderImageSlot draws the image bound to the node into its canvas rect using the provided fit mode.
func RenderImageSlot(ctx *canvas.Context, node Node, fit domain.FitMode) error {
	if node.ImagePath == "" {
		return nil
	}

	if node.CanvasRect.WMM <= 0 || node.CanvasRect.HMM <= 0 {
		return fmt.Errorf("slot %q has invalid dimensions %v×%v", node.SlotID, node.CanvasRect.WMM, node.CanvasRect.HMM)
	}

	img, err := loadImage(node.ImagePath)
	if err != nil {
		return err
	}

	resolution, drawWidth, drawHeight, err := fitImageToSlot(node.SlotID, img.Bounds(), float64(node.CanvasRect.WMM), float64(node.CanvasRect.HMM), fit)
	if err != nil {
		return err
	}

	x := float64(node.CanvasRect.XMM) + (float64(node.CanvasRect.WMM)-drawWidth)/2
	y := float64(node.CanvasRect.YMM) + (float64(node.CanvasRect.HMM)-drawHeight)/2

	ctx.DrawImage(x, y, img, resolution)
	return nil
}

func fitImageToSlot(slotID string, bounds image.Rectangle, slotWidthMM, slotHeightMM float64, fit domain.FitMode) (canvas.Resolution, float64, float64, error) {
	if slotWidthMM <= 0 || slotHeightMM <= 0 {
		return 0, 0, 0, fmt.Errorf("slot %q has non-positive dimensions %vx%v", slotID, slotWidthMM, slotHeightMM)
	}
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0, 0, 0, fmt.Errorf("slot %q source image has zero dimensions", slotID)
	}

	pixelWidth := float64(bounds.Dx())
	pixelHeight := float64(bounds.Dy())

	ratioX := pixelWidth / slotWidthMM
	ratioY := pixelHeight / slotHeightMM

	var scale float64
	switch fit {
	case domain.FitModeCover:
		scale = math.Min(ratioX, ratioY)
	default:
		scale = math.Max(ratioX, ratioY)
	}
	if scale <= 0 {
		scale = 1
	}

	resolution := canvas.DPMM(scale)
	drawWidth := pixelWidth / scale
	drawHeight := pixelHeight / scale

	return resolution, drawWidth, drawHeight, nil
}

func loadImage(path string) (image.Image, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open image %q: %w", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("decode image %q: %w", path, err)
	}
	return img, nil
}
