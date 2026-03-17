package export

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/render"
)

func TestComposeAndWriteASCIIExportsTXT(t *testing.T) {
	imagePath := writeASCIIImageFixture(t)
	tmpDir := t.TempDir()

	out, err := ComposeAndWriteASCII(domain.ProjectPage{
		Size:        domain.PageSizeA4,
		Orientation: domain.OrientationPortrait,
	}, imagePath, domain.ASCIIOptions{
		Charset: "@ ",
		Density: 4,
	}, ASCIIExportOptions{
		Format: ASCIIFormatTXT,
		Out:    filepath.Join(tmpDir, "ascii-output"),
	})
	if err != nil {
		t.Fatalf("ComposeAndWriteASCII() error = %v", err)
	}

	assertFileExists(t, out)

	actualBytes, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	expectedBytes, err := os.ReadFile(filepath.Join("testdata", "ascii-output.txt"))
	if err != nil {
		t.Fatalf("ReadFile(expected) error = %v", err)
	}

	expected := strings.TrimRight(string(expectedBytes), "\n")
	if string(actualBytes) != expected {
		t.Fatalf("txt output = %q, want %q", string(actualBytes), expected)
	}
}

func TestWriteASCIIExportsPDFForPortraitAndLandscape(t *testing.T) {
	for _, tc := range []struct {
		name string
		page domain.ProjectPage
	}{
		{
			name: "portrait",
			page: domain.ProjectPage{
				Size:        domain.PageSizeA4,
				Orientation: domain.OrientationPortrait,
			},
		},
		{
			name: "landscape",
			page: domain.ProjectPage{
				Size:        domain.PageSizeA5,
				Orientation: domain.OrientationLandscape,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := WriteASCII(sampleASCIIComposition(tc.page), ASCIIExportOptions{
				Format: ASCIIFormatPDF,
				Out:    filepath.Join(t.TempDir(), tc.name),
			})
			if err != nil {
				t.Fatalf("WriteASCII() error = %v", err)
			}

			assertFileExists(t, out)
		})
	}
}

func TestWriteASCIIPDFUsesGeneratedArtWithoutImagePath(t *testing.T) {
	composition := sampleASCIIComposition(domain.ProjectPage{
		Size:        domain.PageSizeA6,
		Orientation: domain.OrientationLandscape,
	})
	composition.ImagePath = ""

	out, err := WriteASCII(composition, ASCIIExportOptions{
		Format: ASCIIFormatPDF,
		Out:    filepath.Join(t.TempDir(), "generated-only.pdf"),
	})
	if err != nil {
		t.Fatalf("WriteASCII() error = %v", err)
	}

	assertFileExists(t, out)
}

func TestNormalizeASCIIOutputPathRejectsMismatchedExtension(t *testing.T) {
	if _, err := normalizeASCIIOutputPath("out/ascii.txt", ASCIIFormatPDF); err == nil {
		t.Fatal("expected mismatched extension to fail")
	}
}

func sampleASCIIComposition(page domain.ProjectPage) *render.ASCIIComposition {
	dimensions, _ := domain.ISOPage(page.Size, page.Orientation)

	return &render.ASCIIComposition{
		Page: render.Page{
			Size:        page.Size,
			Orientation: page.Orientation,
			Dimensions:  dimensions,
		},
		Options: domain.ASCIIOptions{
			Charset: "@ ",
			Density: 4,
		},
		Art: asciipkg.Art{
			Width:   4,
			Height:  2,
			Charset: "@ ",
			Lines: []string{
				"@@  ",
				"@@  ",
			},
		},
	}
}

func writeASCIIImageFixture(t *testing.T) string {
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

	path := filepath.Join(t.TempDir(), "ascii-fixture.png")
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
