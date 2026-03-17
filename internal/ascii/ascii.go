package ascii

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jaeyoung0509/letterpress/internal/domain"
	_ "golang.org/x/image/webp"
)

const (
	DefaultToneCharset     = "@%#*+=-:. "
	DefaultDensity         = 80
	DefaultContrast        = 1.0
	DefaultGamma           = 1.0
	DefaultCellAspectRatio = 0.5
	DefaultEdgeThreshold   = 0.28
	DefaultFillText        = "LETTERPRESS"
)

type Cell struct {
	Glyph     rune
	Luminance float64
	Edge      float64
	Signal    float64
	Occupied  bool
}

type Art struct {
	Mode     domain.ASCIIMode
	Width    int
	Height   int
	Charset  string
	FillText string
	Lines    []string
	Cells    [][]Cell
}

func (a Art) String() string {
	return strings.Join(a.Lines, "\n")
}

type Segment struct {
	X1     float64
	Y1     float64
	X2     float64
	Y2     float64
	Weight float64
}

type options struct {
	mode            domain.ASCIIMode
	toneCharset     []rune
	fillText        []rune
	density         int
	threshold       float64
	contrast        float64
	gamma           float64
	invert          bool
	edgeWeight      float64
	edgeThreshold   float64
	dither          domain.DitherMode
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
	if requiresFillText(opts.mode) && len(opts.fillText) == 0 {
		return Art{}, fmt.Errorf("fill text is required for %s mode", opts.mode)
	}

	columns, rows := gridDimensions(bounds.Dx(), bounds.Dy(), opts.density, opts.cellAspectRatio)
	pixels := normalizedLuminance(img, opts.contrast, opts.gamma)
	luminance := sampleGrid(pixels, columns, rows)
	edges := sobelEdges(luminance)
	occupancy := occupancyMap(luminance, opts)
	toneSignals := toneSignalGrid(luminance, edges, opts)

	if opts.dither == domain.DitherModeFloyd && (opts.mode == domain.ASCIIModeTone || opts.mode == domain.ASCIIModeHybrid) {
		toneSignals = ditherGrid(toneSignals, len(opts.toneCharset))
	}

	fillCursor := 0
	cells := make([][]Cell, rows)
	lines := make([]string, rows)
	for y := 0; y < rows; y++ {
		cells[y] = make([]Cell, columns)
		var builder strings.Builder

		for x := 0; x < columns; x++ {
			lum := luminance[y][x]
			edge := edges[y][x]
			occupied := occupancy[y][x]
			signal := toneSignals[y][x]

			glyph := renderGlyph(opts, signal, edge, occupied, &fillCursor)
			cells[y][x] = Cell{
				Glyph:     glyph,
				Luminance: lum,
				Edge:      edge,
				Signal:    signal,
				Occupied:  occupied,
			}
			builder.WriteRune(glyph)
		}

		lines[y] = builder.String()
	}

	return Art{
		Mode:     opts.mode,
		Width:    columns,
		Height:   rows,
		Charset:  string(opts.toneCharset),
		FillText: string(opts.fillText),
		Lines:    lines,
		Cells:    cells,
	}, nil
}

func normalize(config domain.ASCIIOptions) options {
	mode := config.EffectiveMode()

	toneCharsetValue := config.EffectiveToneCharset()
	if strings.TrimSpace(toneCharsetValue) == "" {
		toneCharsetValue = DefaultToneCharset
	}
	toneCharset := []rune(toneCharsetValue)
	if len(toneCharset) == 0 {
		toneCharset = []rune(DefaultToneCharset)
	}

	fillText := []rune(normalizeFillText(config.FillText))

	density := config.Density
	if density <= 0 {
		density = DefaultDensity
	}

	contrast := config.Contrast
	if contrast <= 0 {
		contrast = DefaultContrast
	}

	gamma := config.Gamma
	if gamma <= 0 {
		gamma = DefaultGamma
	}

	cellAspect := config.CellAspect
	if cellAspect <= 0 {
		cellAspect = DefaultCellAspectRatio
	}

	edgeThreshold := clamp01(config.EdgeThreshold)
	if edgeThreshold == 0 {
		edgeThreshold = DefaultEdgeThreshold
	}

	dither := config.Dither
	if dither == "" {
		dither = domain.DitherModeOff
	}

	return options{
		mode:            mode,
		toneCharset:     toneCharset,
		fillText:        fillText,
		density:         density,
		threshold:       clamp01(config.Threshold),
		contrast:        contrast,
		gamma:           gamma,
		invert:          config.Invert,
		edgeWeight:      clamp01(config.EdgeWeight),
		edgeThreshold:   edgeThreshold,
		dither:          dither,
		cellAspectRatio: cellAspect,
	}
}

func requiresFillText(mode domain.ASCIIMode) bool {
	switch mode {
	case domain.ASCIIModeFill, domain.ASCIIModeHybrid, domain.ASCIIModeVector:
		return true
	default:
		return false
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

func normalizedLuminance(img image.Image, contrast, gamma float64) [][]float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	pixels := make([][]float64, height)
	minLum := 1.0
	maxLum := 0.0
	for y := 0; y < height; y++ {
		row := make([]float64, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			luminance := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 65535.0
			row[x] = luminance
			if luminance < minLum {
				minLum = luminance
			}
			if luminance > maxLum {
				maxLum = luminance
			}
		}
		pixels[y] = row
	}

	span := maxLum - minLum
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := pixels[y][x]
			if span > 1e-6 {
				value = (value - minLum) / span
			}
			value = applyGamma(value, gamma)
			value = applyContrast(value, contrast)
			pixels[y][x] = clamp01(value)
		}
	}

	return pixels
}

func sampleGrid(pixels [][]float64, columns, rows int) [][]float64 {
	height := len(pixels)
	width := len(pixels[0])

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

func sobelEdges(luminance [][]float64) [][]float64 {
	height := len(luminance)
	width := len(luminance[0])

	edges := make([][]float64, height)
	maxMag := 0.0
	for y := 0; y < height; y++ {
		edges[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			gx := -at(luminance, x-1, y-1) + at(luminance, x+1, y-1) +
				-2*at(luminance, x-1, y) + 2*at(luminance, x+1, y) +
				-at(luminance, x-1, y+1) + at(luminance, x+1, y+1)
			gy := at(luminance, x-1, y-1) + 2*at(luminance, x, y-1) + at(luminance, x+1, y-1) +
				-at(luminance, x-1, y+1) - 2*at(luminance, x, y+1) - at(luminance, x+1, y+1)
			mag := math.Sqrt(gx*gx + gy*gy)
			edges[y][x] = mag
			if mag > maxMag {
				maxMag = mag
			}
		}
	}

	if maxMag <= 1e-6 {
		return edges
	}
	for y := range edges {
		for x := range edges[y] {
			edges[y][x] = clamp01(edges[y][x] / maxMag)
		}
	}
	return edges
}

func toneSignalGrid(luminance, edges [][]float64, opts options) [][]float64 {
	height := len(luminance)
	width := len(luminance[0])
	signals := make([][]float64, height)
	edgeWeight := effectiveEdgeWeight(opts)

	for y := 0; y < height; y++ {
		signals[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			brightness := effectiveBrightness(luminance[y][x], opts.invert)
			signals[y][x] = mixSignal(brightness, edges[y][x], edgeWeight)
		}
	}

	return signals
}

func effectiveEdgeWeight(opts options) float64 {
	if opts.edgeWeight > 0 {
		return opts.edgeWeight
	}
	switch opts.mode {
	case domain.ASCIIModeOutline:
		return 1.0
	case domain.ASCIIModeHybrid, domain.ASCIIModeVector:
		return 0.55
	default:
		return 0
	}
}

func occupancyMap(luminance [][]float64, opts options) [][]bool {
	height := len(luminance)
	width := len(luminance[0])
	mask := make([][]bool, height)

	for y := 0; y < height; y++ {
		mask[y] = make([]bool, width)
		for x := 0; x < width; x++ {
			darkness := 1 - effectiveBrightness(luminance[y][x], opts.invert)
			threshold := opts.threshold
			if threshold <= 0 {
				local := localAverageDarkness(luminance, x, y, opts.invert)
				threshold = clamp01(local*0.88 + 0.04)
			}
			mask[y][x] = darkness >= threshold
		}
	}

	return mask
}

func localAverageDarkness(luminance [][]float64, x, y int, invert bool) float64 {
	sum := 0.0
	count := 0.0
	for yy := maxInt(0, y-1); yy <= minInt(len(luminance)-1, y+1); yy++ {
		for xx := maxInt(0, x-1); xx <= minInt(len(luminance[0])-1, x+1); xx++ {
			sum += 1 - effectiveBrightness(luminance[yy][xx], invert)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func ditherGrid(signal [][]float64, levels int) [][]float64 {
	height := len(signal)
	width := len(signal[0])

	out := make([][]float64, height)
	for y := 0; y < height; y++ {
		out[y] = make([]float64, width)
		copy(out[y], signal[y])
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldValue := clamp01(out[y][x])
			newValue := quantizeSignal(oldValue, levels)
			err := oldValue - newValue
			out[y][x] = newValue

			diffuse(out, x+1, y, err*7.0/16.0)
			diffuse(out, x-1, y+1, err*3.0/16.0)
			diffuse(out, x, y+1, err*5.0/16.0)
			diffuse(out, x+1, y+1, err*1.0/16.0)
		}
	}

	return out
}

func diffuse(grid [][]float64, x, y int, delta float64) {
	if y < 0 || y >= len(grid) || x < 0 || x >= len(grid[0]) {
		return
	}
	grid[y][x] = clamp01(grid[y][x] + delta)
}

func quantizeSignal(value float64, levels int) float64 {
	if levels <= 1 {
		return clamp01(value)
	}
	index := int(math.Round(clamp01(value) * float64(levels-1)))
	return float64(index) / float64(levels-1)
}

func renderGlyph(opts options, signal, edge float64, occupied bool, fillCursor *int) rune {
	switch opts.mode {
	case domain.ASCIIModeOutline:
		if edge < opts.edgeThreshold {
			return ' '
		}
		return selectGlyph(opts.toneCharset, clamp01(1-edge))
	case domain.ASCIIModeFill:
		if occupied {
			return nextFillRune(opts.fillText, fillCursor)
		}
		return ' '
	case domain.ASCIIModeHybrid, domain.ASCIIModeVector:
		if edge >= opts.edgeThreshold {
			return selectGlyph(opts.toneCharset, clamp01(1-edge))
		}
		if occupied {
			return nextFillRune(opts.fillText, fillCursor)
		}
		return ' '
	case domain.ASCIIModeTone:
		fallthrough
	default:
		return selectGlyph(opts.toneCharset, signal)
	}
}

func nextFillRune(fill []rune, cursor *int) rune {
	if len(fill) == 0 {
		fill = []rune(DefaultFillText)
	}
	r := fill[*cursor%len(fill)]
	*cursor++
	return r
}

func ExtractContourSegments(art Art, edgeThreshold float64) []Segment {
	if art.Width == 0 || art.Height == 0 || len(art.Cells) == 0 {
		return nil
	}
	threshold := clamp01(edgeThreshold)
	if threshold == 0 {
		threshold = DefaultEdgeThreshold
	}

	visited := make([][]bool, art.Height)
	for y := range visited {
		visited[y] = make([]bool, art.Width)
	}

	segments := make([]Segment, 0, 32)
	for y := 0; y < art.Height; y++ {
		for x := 0; x < art.Width; x++ {
			if visited[y][x] || art.Cells[y][x].Edge < threshold {
				continue
			}
			component := traceComponent(art, visited, x, y, threshold)
			if len(component) == 0 {
				continue
			}
			segment := componentToSegment(component)
			if segment.Weight <= 0 {
				continue
			}
			segments = append(segments, segment)
		}
	}

	sort.Slice(segments, func(i, j int) bool {
		return segmentScore(segments[i]) > segmentScore(segments[j])
	})
	if len(segments) > 96 {
		segments = segments[:96]
	}
	return segments
}

type point struct {
	x      float64
	y      float64
	weight float64
}

func traceComponent(art Art, visited [][]bool, startX, startY int, threshold float64) []point {
	queue := [][2]int{{startX, startY}}
	visited[startY][startX] = true
	points := make([]point, 0, 12)

	for len(queue) > 0 {
		cell := queue[0]
		queue = queue[1:]

		x := cell[0]
		y := cell[1]
		edge := art.Cells[y][x].Edge
		points = append(points, point{
			x:      float64(x) + 0.5,
			y:      float64(y) + 0.5,
			weight: edge,
		})

		for ny := maxInt(0, y-1); ny <= minInt(art.Height-1, y+1); ny++ {
			for nx := maxInt(0, x-1); nx <= minInt(art.Width-1, x+1); nx++ {
				if visited[ny][nx] || art.Cells[ny][nx].Edge < threshold {
					continue
				}
				visited[ny][nx] = true
				queue = append(queue, [2]int{nx, ny})
			}
		}
	}

	return points
}

func componentToSegment(points []point) Segment {
	if len(points) == 0 {
		return Segment{}
	}
	if len(points) == 1 {
		p := points[0]
		return Segment{
			X1:     p.x - 0.45,
			Y1:     p.y,
			X2:     p.x + 0.45,
			Y2:     p.y,
			Weight: p.weight,
		}
	}

	cx := 0.0
	cy := 0.0
	weight := 0.0
	for _, p := range points {
		cx += p.x
		cy += p.y
		weight += p.weight
	}
	cx /= float64(len(points))
	cy /= float64(len(points))
	weight /= float64(len(points))

	xx, xy, yy := 0.0, 0.0, 0.0
	for _, p := range points {
		dx := p.x - cx
		dy := p.y - cy
		xx += dx * dx
		xy += dx * dy
		yy += dy * dy
	}
	theta := 0.5 * math.Atan2(2*xy, xx-yy)
	dirX := math.Cos(theta)
	dirY := math.Sin(theta)

	minProj := math.MaxFloat64
	maxProj := -math.MaxFloat64
	for _, p := range points {
		proj := (p.x-cx)*dirX + (p.y-cy)*dirY
		if proj < minProj {
			minProj = proj
		}
		if proj > maxProj {
			maxProj = proj
		}
	}

	if maxProj-minProj < 0.9 {
		maxProj = 0.45
		minProj = -0.45
	}

	return Segment{
		X1:     cx + dirX*minProj,
		Y1:     cy + dirY*minProj,
		X2:     cx + dirX*maxProj,
		Y2:     cy + dirY*maxProj,
		Weight: weight,
	}
}

func segmentScore(segment Segment) float64 {
	return segment.Weight * math.Hypot(segment.X2-segment.X1, segment.Y2-segment.Y1)
}

func mixSignal(brightness, edge, edgeWeight float64) float64 {
	return clamp01((1-edgeWeight)*brightness + edgeWeight*(1-edge))
}

func selectGlyph(charset []rune, signal float64) rune {
	if len(charset) == 0 {
		charset = []rune(DefaultToneCharset)
	}
	index := int(math.Round(clamp01(signal) * float64(len(charset)-1)))
	return charset[index]
}

func effectiveBrightness(value float64, invert bool) float64 {
	if invert {
		return 1 - value
	}
	return value
}

func applyGamma(value, gamma float64) float64 {
	if gamma <= 0 {
		return clamp01(value)
	}
	return clamp01(math.Pow(clamp01(value), 1.0/gamma))
}

func applyContrast(value, contrast float64) float64 {
	return clamp01((value-0.5)*contrast + 0.5)
}

func normalizeFillText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func at(grid [][]float64, x, y int) float64 {
	x = maxInt(0, minInt(len(grid[0])-1, x))
	y = maxInt(0, minInt(len(grid)-1, y))
	return grid[y][x]
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
