package render

import (
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/tdewolff/canvas"
)

func TestFitModeContainResolution(t *testing.T) {
	resolution, width, height, err := fitImageToSlot("photo", image.Rect(0, 0, 400, 200), 80, 60, domain.FitModeContain)
	if err != nil {
		t.Fatalf("fitImageToSlot() error = %v", err)
	}
	if float64(resolution) != 5 {
		t.Fatalf("resolution = %v, want 5", resolution)
	}
	if width != 80 {
		t.Fatalf("draw width = %v, want 80", width)
	}
	if height != 40 {
		t.Fatalf("draw height = %v, want 40", height)
	}
}

func TestFitModeCoverResolution(t *testing.T) {
	resolution, width, height, err := fitImageToSlot("photo", image.Rect(0, 0, 120, 60), 60, 30, domain.FitModeCover)
	if err != nil {
		t.Fatalf("fitImageToSlot() error = %v", err)
	}
	if float64(resolution) != 2 {
		t.Fatalf("resolution = %v, want 2", resolution)
	}
	if width != 60 {
		t.Fatalf("draw width = %v, want 60", width)
	}
	if height != 30 {
		t.Fatalf("draw height = %v, want 30", height)
	}
}

func TestRenderImageSlotMissingFile(t *testing.T) {
	ctx := canvas.NewContext(canvas.New(100, 100))
	node := Node{
		SlotID:     "photo",
		CanvasRect: Rect{WMM: 40, HMM: 40},
		ImagePath:  "no/such/path.png",
	}
	err := RenderImageSlot(ctx, node, domain.FitModeContain)
	if err == nil {
		t.Fatal("expected error for missing image file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestRenderImageSlotDraws(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hero.png")
	f, err := os.Create(file)
	if err != nil {
		t.Fatalf("create sample image: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode sample image: %v", err)
	}
	f.Close()

	ctx := canvas.NewContext(canvas.New(200, 200))
	node := Node{
		SlotID: "photo",
		CanvasRect: Rect{
			XMM: 10,
			YMM: 15,
			WMM: 80,
			HMM: 60,
		},
		ImagePath: file,
	}

	if err := RenderImageSlot(ctx, node, domain.FitModeContain); err != nil {
		t.Fatalf("RenderImageSlot() error = %v", err)
	}
}

func TestRenderDecorationFrameValidation(t *testing.T) {
	ctx := canvas.NewContext(canvas.New(150, 150))
	node := Node{
		SlotID:     "frame",
		CanvasRect: Rect{WMM: 60, HMM: 30},
		Decoration: &domain.Asset{
			ID:   "photo",
			Kind: domain.AssetKindImage,
			Path: "unused",
		},
	}

	if err := RenderDecorationFrame(ctx, node, true); err == nil {
		t.Fatal("expected error when decoration asset is not a decoration")
	}
}

func TestRenderDecorationFrameDisabled(t *testing.T) {
	ctx := canvas.NewContext(canvas.New(150, 150))
	if err := RenderDecorationFrame(ctx, Node{
		SlotID:     "frame",
		CanvasRect: Rect{WMM: 60, HMM: 30},
	}, false); err != nil {
		t.Fatalf("RenderDecorationFrame() error = %v", err)
	}
}

func TestRenderDecorationFrameDraws(t *testing.T) {
	ctx := canvas.NewContext(canvas.New(150, 150))
	node := Node{
		SlotID:     "frame",
		CanvasRect: Rect{XMM: 5, YMM: 5, WMM: 60, HMM: 30},
		Decoration: &domain.Asset{
			ID:   "corner-ornament",
			Kind: domain.AssetKindDecoration,
			Path: "assets/corner.svg",
		},
	}
	if err := RenderDecorationFrame(ctx, node, true); err != nil {
		t.Fatalf("RenderDecorationFrame() error = %v", err)
	}
}
