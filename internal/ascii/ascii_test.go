package ascii

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaeyoung0509/letterpress/internal/domain"
)

func TestRenderPathMatchesFixtureOutput(t *testing.T) {
	path := writeFixturePNG(t, twoToneImage())
	expectedBytes, err := os.ReadFile(filepath.Join("testdata", "two-tone.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	art, err := RenderPath(path, domain.ASCIIOptions{
		ToneCharset: "@ ",
		Density:     4,
	})
	if err != nil {
		t.Fatalf("RenderPath() error = %v", err)
	}

	expected := strings.TrimRight(string(expectedBytes), "\n")
	if art.String() != expected {
		t.Fatalf("art.String() = %q, want %q", art.String(), expected)
	}
	if art.Width != 4 || art.Height != 2 {
		t.Fatalf("art dimensions = %dx%d, want 4x2", art.Width, art.Height)
	}
}

func TestRenderImageAppliesThresholdAndInvert(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{R: 48, G: 48, B: 48, A: 255})
	img.Set(1, 0, color.NRGBA{R: 208, G: 208, B: 208, A: 255})

	art, err := RenderImage(img, domain.ASCIIOptions{
		ToneCharset: "@ ",
		Density:     2,
		Threshold:   0.5,
	})
	if err != nil {
		t.Fatalf("RenderImage() error = %v", err)
	}
	if art.String() != "@ " {
		t.Fatalf("art.String() = %q, want %q", art.String(), "@ ")
	}

	inverted, err := RenderImage(img, domain.ASCIIOptions{
		ToneCharset: "@ ",
		Density:     2,
		Threshold:   0.5,
		Invert:      true,
	})
	if err != nil {
		t.Fatalf("RenderImage() with invert error = %v", err)
	}
	if inverted.String() != " @" {
		t.Fatalf("inverted art.String() = %q, want %q", inverted.String(), " @")
	}
}

func TestRenderImageUsesEdgeWeightAndKeepsOutputDeterministic(t *testing.T) {
	img := checkerImage()

	first, err := RenderImage(img, domain.ASCIIOptions{
		Mode:        domain.ASCIIModeOutline,
		ToneCharset: "@# ",
		Density:     6,
		EdgeWeight:  0.75,
	})
	if err != nil {
		t.Fatalf("first RenderImage() error = %v", err)
	}

	second, err := RenderImage(img, domain.ASCIIOptions{
		Mode:        domain.ASCIIModeOutline,
		ToneCharset: "@# ",
		Density:     6,
		EdgeWeight:  0.75,
	})
	if err != nil {
		t.Fatalf("second RenderImage() error = %v", err)
	}

	if first.String() != second.String() {
		t.Fatalf("deterministic output mismatch: %q vs %q", first.String(), second.String())
	}
	if first.Width != 6 || first.Height < 1 {
		t.Fatalf("unexpected dimensions %dx%d", first.Width, first.Height)
	}
}

func TestRenderImageSeparatesFillTextFromToneCharset(t *testing.T) {
	img := twoToneImage()

	art, err := RenderImage(img, domain.ASCIIOptions{
		Mode:        domain.ASCIIModeFill,
		ToneCharset: "@ ",
		FillText:    "HI",
		Density:     4,
	})
	if err != nil {
		t.Fatalf("RenderImage() error = %v", err)
	}

	if art.FillText != "HI" {
		t.Fatalf("art.FillText = %q, want %q", art.FillText, "HI")
	}
	if art.Charset != "@ " {
		t.Fatalf("art.Charset = %q, want %q", art.Charset, "@ ")
	}
	if !strings.Contains(art.String(), "H") || !strings.Contains(art.String(), "I") {
		t.Fatalf("expected fill text glyphs in art, got %q", art.String())
	}
}

func TestRenderImageRequiresFillTextForVectorMode(t *testing.T) {
	_, err := RenderImage(twoToneImage(), domain.ASCIIOptions{
		Mode:    domain.ASCIIModeVector,
		Density: 4,
	})
	if err == nil {
		t.Fatal("expected missing fill text error")
	}
}

func TestExtractContourSegmentsReturnsRankedSegments(t *testing.T) {
	art, err := RenderImage(checkerImage(), domain.ASCIIOptions{
		Mode:        domain.ASCIIModeOutline,
		ToneCharset: "@# ",
		Density:     8,
		EdgeWeight:  1.0,
	})
	if err != nil {
		t.Fatalf("RenderImage() error = %v", err)
	}

	segments := ExtractContourSegments(art, 0.2)
	if len(segments) == 0 {
		t.Fatal("expected contour segments")
	}
	if segments[0].Weight <= 0 {
		t.Fatalf("segment weight = %f", segments[0].Weight)
	}
}

func writeFixturePNG(t *testing.T, img image.Image) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.png")
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

func twoToneImage() image.Image {
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
	return img
}

func checkerImage() image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 6; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.NRGBA{R: 24, G: 24, B: 24, A: 255})
				continue
			}
			img.Set(x, y, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
		}
	}
	return img
}
