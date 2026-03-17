package export

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-fonts/latin-modern/lmmono10regular"
	asciipkg "github.com/jaeyoung0509/letterpress/internal/ascii"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/jaeyoung0509/letterpress/internal/render"
	"github.com/tdewolff/canvas"
	pdfrenderer "github.com/tdewolff/canvas/renderers/pdf"
)

const (
	asciiPointsPerMillimetre = 72.0 / 25.4
	defaultASCIIMarginMM     = 12.0
	defaultASCIILineHeight   = 1.0
	asciiMeasurementFontPt   = 12.0
)

type ASCIIFormat string

const (
	ASCIIFormatTXT ASCIIFormat = "txt"
	ASCIIFormatPDF ASCIIFormat = "pdf"
)

type ASCIIExportOptions struct {
	Format     ASCIIFormat
	Out        string
	MarginMM   float64
	FontSizePt float64
	LineHeight float64
}

type asciiLayout struct {
	marginMM   float64
	lineHeight float64
	face       *canvas.FontFace
	lineStepMM float64
}

var (
	monoFamilyOnce sync.Once
	monoFamily     *canvas.FontFamily
	monoFamilyErr  error
)

func ComposeAndWriteASCII(page domain.ProjectPage, imagePath string, asciiOptions domain.ASCIIOptions, exportOptions ASCIIExportOptions) (string, error) {
	composition, err := render.ComposeASCII(page, imagePath, asciiOptions)
	if err != nil {
		return "", err
	}
	return WriteASCII(composition, exportOptions)
}

func WriteASCII(composition *render.ASCIIComposition, options ASCIIExportOptions) (string, error) {
	if composition == nil {
		return "", fmt.Errorf("ascii composition is not initialized")
	}
	if composition.Art.Width <= 0 || composition.Art.Height <= 0 || len(composition.Art.Lines) == 0 {
		return "", fmt.Errorf("ascii composition has no generated text-art")
	}

	out, err := normalizeASCIIOutputPath(options.Out, options.Format)
	if err != nil {
		return "", err
	}
	if err := ensureOutputDir(out); err != nil {
		return "", err
	}

	switch options.Format {
	case ASCIIFormatTXT:
		if err := writeASCIITXT(composition, out); err != nil {
			return "", err
		}
	case ASCIIFormatPDF:
		if err := writeASCIIPDF(composition, out, options); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported ascii export format %q", options.Format)
	}

	return out, nil
}

func normalizeASCIIOutputPath(out string, format ASCIIFormat) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		return "", fmt.Errorf("output path is required")
	}
	if !format.valid() {
		return "", fmt.Errorf("unsupported ascii export format %q", format)
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

func writeASCIITXT(composition *render.ASCIIComposition, out string) error {
	if err := os.WriteFile(out, []byte(composition.Art.String()), 0o644); err != nil {
		return fmt.Errorf("write ascii txt %q: %w", out, err)
	}
	return nil
}

func writeASCIIPDF(composition *render.ASCIIComposition, out string, options ASCIIExportOptions) error {
	layout, err := resolveASCIILayout(composition, options)
	if err != nil {
		return err
	}

	file, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("create ascii pdf %q: %w", out, err)
	}
	defer file.Close()

	page := composition.Page
	surface := canvas.New(float64(page.Dimensions.WidthMM), float64(page.Dimensions.HeightMM))
	ctx := canvas.NewContext(surface)

	topY := float64(page.Dimensions.HeightMM) - layout.marginMM - layout.face.Metrics().Ascent
	for index, line := range composition.Art.Lines {
		y := topY - float64(index)*layout.lineStepMM
		ctx.DrawText(layout.marginMM, y, canvas.NewTextLine(layout.face, line, canvas.Left))
	}

	pdf := pdfrenderer.New(
		file,
		float64(page.Dimensions.WidthMM),
		float64(page.Dimensions.HeightMM),
		nil,
	)
	surface.RenderTo(pdf)
	if err := pdf.Close(); err != nil {
		return fmt.Errorf("finalize ascii pdf %q: %w", out, err)
	}

	return nil
}

func resolveASCIILayout(composition *render.ASCIIComposition, options ASCIIExportOptions) (asciiLayout, error) {
	page := composition.Page
	marginMM := options.MarginMM
	if marginMM <= 0 {
		marginMM = defaultASCIIMarginMM
	}

	safeWidth := float64(page.Dimensions.WidthMM) - marginMM*2
	safeHeight := float64(page.Dimensions.HeightMM) - marginMM*2
	if safeWidth <= 0 || safeHeight <= 0 {
		return asciiLayout{}, fmt.Errorf("ascii export margin %.2fmm leaves no printable area", marginMM)
	}

	lineHeight := options.LineHeight
	if lineHeight <= 0 {
		lineHeight = defaultASCIILineHeight
	}

	fontSizePt := options.FontSizePt
	if fontSizePt <= 0 {
		auto, err := autoFitASCIIFontSize(composition.Art, safeWidth, safeHeight, lineHeight)
		if err != nil {
			return asciiLayout{}, err
		}
		fontSizePt = auto
	}

	face, err := monoFace(fontSizePt)
	if err != nil {
		return asciiLayout{}, err
	}

	lineStepMM := face.Metrics().LineHeight * lineHeight
	if sampleWidth := asciiLineWidth(face, composition.Art.Width); sampleWidth > safeWidth {
		return asciiLayout{}, fmt.Errorf("ascii text width %.2fmm exceeds printable width %.2fmm", sampleWidth, safeWidth)
	}
	if totalHeight := lineStepMM * float64(composition.Art.Height); totalHeight > safeHeight {
		return asciiLayout{}, fmt.Errorf("ascii text height %.2fmm exceeds printable height %.2fmm", totalHeight, safeHeight)
	}

	return asciiLayout{
		marginMM:   marginMM,
		lineHeight: lineHeight,
		face:       face,
		lineStepMM: lineStepMM,
	}, nil
}

func autoFitASCIIFontSize(art asciipkg.Art, safeWidth, safeHeight, lineHeight float64) (float64, error) {
	face, err := monoFace(asciiMeasurementFontPt)
	if err != nil {
		return 0, err
	}

	sampleWidth := asciiLineWidth(face, art.Width)
	if sampleWidth <= 0 {
		return 0, fmt.Errorf("ascii sample width is invalid")
	}

	lineStep := face.Metrics().LineHeight * lineHeight
	if lineStep <= 0 {
		return 0, fmt.Errorf("ascii line height is invalid")
	}

	widthScale := safeWidth / sampleWidth
	heightScale := safeHeight / (lineStep * float64(art.Height))
	scale := widthScale
	if heightScale < scale {
		scale = heightScale
	}
	if scale <= 0 {
		return 0, fmt.Errorf("ascii layout cannot fit within the selected page")
	}

	return asciiMeasurementFontPt * scale, nil
}

func asciiLineWidth(face *canvas.FontFace, columns int) float64 {
	sample := canvas.NewTextLine(face, strings.Repeat("M", columns), canvas.Left)
	return sample.Bounds().W()
}

func monoFace(sizePt float64) (*canvas.FontFace, error) {
	if err := ensureMonoFamily(); err != nil {
		return nil, err
	}

	return monoFamily.Face(
		pointsToMillimetres(sizePt),
		color.Black,
		canvas.FontRegular,
		canvas.FontNormal,
	), nil
}

func ensureMonoFamily() error {
	monoFamilyOnce.Do(func() {
		monoFamily = canvas.NewFontFamily("Letterpress Mono")
		monoFamily.MustLoadFont(lmmono10regular.TTF, 0, canvas.FontRegular)
	})

	return monoFamilyErr
}

func pointsToMillimetres(points float64) float64 {
	return points / asciiPointsPerMillimetre
}

func (f ASCIIFormat) valid() bool {
	switch f {
	case ASCIIFormatTXT, ASCIIFormatPDF:
		return true
	default:
		return false
	}
}
