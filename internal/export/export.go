package export

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/render"
	templatepkg "github.com/jaeyoung0509/letterpress/internal/template"
	"github.com/tdewolff/canvas"
	pdfrenderer "github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

type Options struct {
	Format      domain.ExportFormat
	Out         string
	Decorations bool
}

func ComposeDocument(resolved templatepkg.ResolvedTemplate, decorations bool) (*render.Document, error) {
	renderer := render.NewCanvasRenderer()
	document, err := renderer.Compose(resolved)
	if err != nil {
		return nil, fmt.Errorf("compose render document: %w", err)
	}

	if err := render.RenderTextSlots(document); err != nil {
		return nil, fmt.Errorf("render text slots: %w", err)
	}

	for _, node := range document.Nodes() {
		switch node.SlotType {
		case domain.SlotTypeImage:
			if err := render.RenderImageSlot(document.Context(), node, node.Fit); err != nil {
				return nil, fmt.Errorf("render image slot %s: %w", node.SlotID, err)
			}
		case domain.SlotTypeDecoration:
			if err := render.RenderDecorationFrame(document.Context(), node, decorations); err != nil {
				return nil, fmt.Errorf("render decoration slot %s: %w", node.SlotID, err)
			}
		}
	}

	return document, nil
}

func WriteDocument(document *render.Document, options Options) (string, error) {
	if document == nil || document.Canvas() == nil {
		return "", fmt.Errorf("document is not initialized")
	}

	out, err := normalizeOutputPath(options.Out, options.Format)
	if err != nil {
		return "", err
	}
	if err := ensureOutputDir(out); err != nil {
		return "", err
	}

	switch options.Format {
	case domain.ExportFormatPDF:
		if err := writePDF(document, out); err != nil {
			return "", err
		}
	case domain.ExportFormatPNG:
		if err := writePNG(document, out, canvas.DPI(300)); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported export format %q", options.Format)
	}

	return out, nil
}

func ComposeAndWrite(resolved templatepkg.ResolvedTemplate, options Options) (string, error) {
	document, err := ComposeDocument(resolved, options.Decorations)
	if err != nil {
		return "", err
	}
	return WriteDocument(document, options)
}

func normalizeOutputPath(out string, format domain.ExportFormat) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return "", fmt.Errorf("output path is required")
	}

	expectedExt := "." + string(format)
	ext := strings.ToLower(filepath.Ext(out))
	if ext == "" {
		return out + expectedExt, nil
	}
	if ext != expectedExt {
		return "", fmt.Errorf("output path %q must use %s for %s export", out, expectedExt, format)
	}
	return out, nil
}

func ensureOutputDir(out string) error {
	dir := filepath.Dir(out)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", dir, err)
	}
	return nil
}

func writePDF(document *render.Document, out string) error {
	file, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create pdf %q: %w", out, err)
	}
	defer file.Close()

	page := document.Page()
	pdf := pdfrenderer.New(
		file,
		float64(page.Dimensions.WidthMM),
		float64(page.Dimensions.HeightMM),
		nil,
	)
	document.Canvas().RenderTo(pdf)
	if err := pdf.Close(); err != nil {
		return fmt.Errorf("finalize pdf %q: %w", out, err)
	}

	return nil
}

func writePNG(document *render.Document, out string, resolution canvas.Resolution) error {
	file, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create png %q: %w", out, err)
	}
	defer file.Close()

	page := document.Page()
	raster := rasterizer.New(
		float64(page.Dimensions.WidthMM),
		float64(page.Dimensions.HeightMM),
		resolution,
		canvas.DefaultColorSpace,
	)
	defer raster.Close()

	document.Canvas().RenderTo(raster)
	if err := png.Encode(file, raster.Image); err != nil {
		return fmt.Errorf("encode png %q: %w", out, err)
	}

	return nil
}
