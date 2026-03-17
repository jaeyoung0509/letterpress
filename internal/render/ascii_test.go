package render

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func TestComposeASCIIBuildsPageAwareComposition(t *testing.T) {
	imagePath := writeASCIIFixturePNG(t)

	composition, err := ComposeASCII(domain.ProjectPage{
		Size:        domain.PageSizeA5,
		Orientation: domain.OrientationLandscape,
	}, imagePath, domain.ASCIIOptions{
		Mode:        domain.ASCIIModeOutline,
		ToneCharset: "@ ",
		Density:     4,
	})
	if err != nil {
		t.Fatalf("ComposeASCII() error = %v", err)
	}

	if composition.Page.Dimensions.WidthMM != domain.Millimetres(210) {
		t.Fatalf("page width = %v, want 210mm", composition.Page.Dimensions.WidthMM)
	}
	if composition.Page.Dimensions.HeightMM != domain.Millimetres(148) {
		t.Fatalf("page height = %v, want 148mm", composition.Page.Dimensions.HeightMM)
	}
	if composition.Art.Width != 4 {
		t.Fatalf("art width = %d, want 4", composition.Art.Width)
	}
	if composition.ImagePath != imagePath {
		t.Fatalf("image path = %q, want %q", composition.ImagePath, imagePath)
	}
	if len(composition.Segments) == 0 {
		t.Fatal("expected contour segments")
	}
}

func TestComposeASCIIRejectsMissingImagePath(t *testing.T) {
	_, err := ComposeASCII(domain.ProjectPage{
		Size:        domain.PageSizeA4,
		Orientation: domain.OrientationPortrait,
	}, "", domain.ASCIIOptions{})
	if err == nil {
		t.Fatal("expected missing image path error")
	}
}

func writeASCIIFixturePNG(t *testing.T) string {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if x < 2 {
				img.Set(x, y, color.NRGBA{A: 255})
				continue
			}
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "ascii.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	return path
}
