package ascii

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	_ "golang.org/x/image/webp"
)

const (
	DefaultCharset         = "@%#*+=-:. "
	DefaultDensity         = 80
	DefaultContrast        = 1.0
	DefaultCellAspectRatio = 0.5
)

type Cell struct {
	Glyph     rune
	Luminance float64
	Edge      float64
	Signal    float64
}

type Art struct {
	Width   int
	Height  int
	Charset string
	Lines   []string
	Cells   [][]Cell
}

func (a Art) String() string {
	return strings.Join(a.Lines, "\n")
}

type options struct {
	charset         []rune
	density         int
	threshold       float64
	contrast        float64
	invert          bool
	edgeWeight      float64
	cellAspectRatio float64
}

func RenderPath(path string, config domain.ASCIIOptions) (Art, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return Art{}, fmt.Errorf("open image %q: %w", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return Art{}, fmt.Errorf("decode image %q: %w", path, err)
	}

	return RenderImage(img, config)
}

func RenderImage(img image.Image, config domain.ASCIIOptions) (Art, error) {
	if img == nil {
		return Art{}, fmt.Errorf("image is required")
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return Art{}, fmt.Errorf("image has invalid dimensions %dx%d", bounds.Dx(), bounds.Dy())
	}

	opts := normalize(config)
	columns, rows := gridDimensions(bounds.Dx(), bounds.Dy(), opts.density, opts.cellAspectRatio)
	luminance := sampleLuminance(img, opts.contrast, columns, rows)
	edges := edgeStrengths(luminance)

	cells := make([][]Cell, rows)
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		cells[y] = make([]Cell, columns)
		var builder strings.Builder

		for x := 0; x < columns; x++ {
			signal := mixSignal(luminance[y][x], edges[y][x], opts.edgeWeight)
			if opts.invert {
				signal = 1 - signal
			}
			if opts.threshold > 0 {
				if signal >= opts.threshold {
					signal = 1
				} else {
					signal = 0
				}
			}

			glyph := selectGlyph(opts.charset, signal)
			cells[y][x] = Cell{
				Glyph:     glyph,
				Luminance: luminance[y][x],
				Edge:      edges[y][x],
				Signal:    clamp01(signal),
			}
			builder.WriteRune(glyph)
		}

		lines[y] = builder.String()
	}

	return Art{
		Width:   columns,
		Height:  rows,
		Charset: string(opts.charset),
		Lines:   lines,
		Cells:   cells,
	}, nil
}

func normalize(config domain.ASCIIOptions) options {
	charsetValue := config.Charset
	if strings.TrimSpace(charsetValue) == "" {
		charsetValue = DefaultCharset
	}

	charset := []rune(charsetValue)
	if len(charset) == 0 {
		charset = []rune(DefaultCharset)
	}

	density := config.Density
	if density <= 0 {
		density = DefaultDensity
	}

	contrast := config.Contrast
	if contrast <= 0 {
		contrast = DefaultContrast
	}

	return options{
		charset:         charset,
		density:         density,
		threshold:       clamp01(config.Threshold),
		contrast:        contrast,
		invert:          config.Invert,
		edgeWeight:      clamp01(config.EdgeWeight),
		cellAspectRatio: DefaultCellAspectRatio,
	}
}

func gridDimensions(width, height, density int, aspect float64) (int, int) {
	columns := density
	if columns < 1 {
		columns = 1
	}
	if aspect <= 0 {
		aspect = DefaultCellAspectRatio
	}

	rows := int(math.Round((float64(height) / float64(width)) * float64(columns) * aspect))
	if rows < 1 {
		rows = 1
	}

	return columns, rows
}

func sampleLuminance(img image.Image, contrast float64, columns, rows int) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	pixels := make([][]float64, height)
	for y := 0; y < height; y++ {
		row := make([]float64, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			luminance := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535.0
			row[x] = applyContrast(luminance, contrast)
		}
		pixels[y] = row
	}

	integral := integralImage(pixels)
	sampled := make([][]float64, rows)
	for y := 0; y < rows; y++ {
		sampled[y] = make([]float64, columns)
		for x := 0; x < columns; x++ {
			x0, x1 := sampleBounds(x, columns, width)
			y0, y1 := sampleBounds(y, rows, height)
			sampled[y][x] = regionAverage(integral, x0, y0, x1, y1)
		}
	}

	return sampled
}

func integralImage(pixels [][]float64) [][]float64 {
	height := len(pixels)
	width := len(pixels[0])

	integral := make([][]float64, height+1)
	for y := range integral {
		integral[y] = make([]float64, width+1)
	}

	for y := 0; y < height; y++ {
		rowSum := 0.0
		for x := 0; x < width; x++ {
			rowSum += pixels[y][x]
			integral[y+1][x+1] = integral[y][x+1] + rowSum
		}
	}

	return integral
}

func sampleBounds(index, total, size int) (int, int) {
	start := int(math.Floor(float64(index) * float64(size) / float64(total)))
	end := int(math.Floor(float64(index+1) * float64(size) / float64(total)))
	if end <= start {
		end = start + 1
	}
	if end > size {
		end = size
	}
	if start >= size {
		start = size - 1
	}
	if start < 0 {
		start = 0
	}
	return start, end
}

func regionAverage(integral [][]float64, x0, y0, x1, y1 int) float64 {
	area := float64((x1 - x0) * (y1 - y0))
	if area <= 0 {
		return 0
	}
	sum := integral[y1][x1] - integral[y0][x1] - integral[y1][x0] + integral[y0][x0]
	return sum / area
}

func edgeStrengths(luminance [][]float64) [][]float64 {
	height := len(luminance)
	width := len(luminance[0])

	edges := make([][]float64, height)
	for y := 0; y < height; y++ {
		edges[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			left := luminance[y][maxInt(0, x-1)]
			right := luminance[y][minInt(width-1, x+1)]
			up := luminance[maxInt(0, y-1)][x]
			down := luminance[minInt(height-1, y+1)][x]

			edges[y][x] = clamp01((math.Abs(right-left) + math.Abs(down-up)) / 2)
		}
	}

	return edges
}

func mixSignal(luminance, edge, edgeWeight float64) float64 {
	return clamp01((1-edgeWeight)*luminance + edgeWeight*(1-edge))
}

func selectGlyph(charset []rune, signal float64) rune {
	if len(charset) == 0 {
		charset = []rune(DefaultCharset)
	}
	index := int(math.Round(clamp01(signal) * float64(len(charset)-1)))
	return charset[index]
}

func applyContrast(value, contrast float64) float64 {
	return clamp01((value-0.5)*contrast + 0.5)
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
