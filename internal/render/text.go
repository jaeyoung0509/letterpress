package render

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"

	"github.com/go-fonts/latin-modern/lmroman10bold"
	"github.com/go-fonts/latin-modern/lmroman10italic"
	"github.com/go-fonts/latin-modern/lmroman10regular"
	"github.com/go-fonts/latin-modern/lmsans10oblique"
	"github.com/go-fonts/latin-modern/lmsans10regular"
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/tdewolff/canvas"
)

const pointsPerMillimetre = 72.0 / 25.4

var (
	fontFamiliesOnce sync.Once
	serifFamily      *canvas.FontFamily
	sansFamily       *canvas.FontFamily
	fontFamilyErr    error
)

func RenderTextSlots(document *Document) error {
	if document == nil || document.ctx == nil {
		return fmt.Errorf("document is not initialized")
	}

	for _, node := range document.nodes {
		if node.SlotType != domain.SlotTypeText {
			continue
		}

		textBox, err := buildTextBox(node)
		if err != nil {
			return fmt.Errorf("render text slot %s: %w", node.SlotID, err)
		}
		if textBox == nil || textBox.Empty() {
			continue
		}

		document.ctx.DrawText(
			float64(node.CanvasRect.XMM),
			float64(node.CanvasRect.YMM+node.CanvasRect.HMM),
			textBox,
		)
	}

	return nil
}

func buildTextBox(node Node) (*canvas.Text, error) {
	content := strings.TrimSpace(node.Text)
	if content == "" {
		if node.Required {
			return nil, fmt.Errorf("required text slot %s is empty", node.SlotID)
		}
		return nil, nil
	}

	style := effectiveTextStyle(node)
	face, err := faceForStyle(style)
	if err != nil {
		return nil, err
	}

	maxHeight := float64(node.CanvasRect.HMM)
	if style.MaxLines > 0 {
		lineBound := face.LineHeight() * float64(style.MaxLines)
		maxHeight = math.Min(maxHeight, lineBound)
	}

	opts := &canvas.TextOptions{
		LineStretch: math.Max(style.LineHeight-1.0, 0),
	}
	text := canvas.NewTextBox(
		face,
		content,
		float64(node.CanvasRect.WMM),
		maxHeight,
		toCanvasTextAlign(style.Align),
		canvas.Top,
		opts,
	)

	if !text.OverflowsX && !text.OverflowsY {
		return text, nil
	}

	switch style.Overflow {
	case domain.OverflowModeError:
		return nil, fmt.Errorf("text overflow for slot %s", node.SlotID)
	case domain.OverflowModeClip:
		return fitText(face, content, float64(node.CanvasRect.WMM), maxHeight, toCanvasTextAlign(style.Align), opts, false)
	default:
		return fitText(face, content, float64(node.CanvasRect.WMM), maxHeight, toCanvasTextAlign(style.Align), opts, true)
	}
}

func fitText(face *canvas.FontFace, content string, width, height float64, align canvas.TextAlign, opts *canvas.TextOptions, ellipsis bool) (*canvas.Text, error) {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil, nil
	}

	low := 0
	high := len(runes)
	best := ""
	for low <= high {
		mid := (low + high) / 2
		candidate := truncateRunes(runes, mid, ellipsis)
		text := canvas.NewTextBox(face, candidate, width, height, align, canvas.Top, opts)
		if text.OverflowsX || text.OverflowsY {
			high = mid - 1
			continue
		}
		best = candidate
		low = mid + 1
	}

	if best == "" {
		return canvas.NewTextBox(face, "", width, height, align, canvas.Top, opts), nil
	}

	return canvas.NewTextBox(face, best, width, height, align, canvas.Top, opts), nil
}

func truncateRunes(runes []rune, length int, ellipsis bool) string {
	if length >= len(runes) {
		return string(runes)
	}

	if length <= 0 {
		if ellipsis {
			return "..."
		}
		return ""
	}

	base := strings.TrimRight(string(runes[:length]), " \n\t")
	if ellipsis {
		return base + "..."
	}
	return base
}

func effectiveTextStyle(node Node) domain.Style {
	style := domain.Style{}
	if node.Style != nil {
		style = *node.Style
	}

	switch strings.ToLower(node.SlotID) {
	case "title", "greeting", "heading":
		if style.Font == "" {
			style.Font = "SerifDisplay"
		}
		if style.SizePt <= 0 {
			style.SizePt = 22
		}
		if style.Align == "" {
			style.Align = domain.TextAlignCenter
		}
		if style.LineHeight <= 0 {
			style.LineHeight = 1.1
		}
	case "signature", "signoff", "closing":
		if style.Font == "" {
			style.Font = "SerifDisplay"
		}
		if style.SizePt <= 0 {
			style.SizePt = 13
		}
		if style.Align == "" {
			style.Align = domain.TextAlignRight
		}
		if style.LineHeight <= 0 {
			style.LineHeight = 1.2
		}
	default:
		if style.Font == "" {
			style.Font = "SansBody"
		}
		if style.SizePt <= 0 {
			style.SizePt = 11
		}
		if style.Align == "" {
			style.Align = domain.TextAlignLeft
		}
		if style.LineHeight <= 0 {
			style.LineHeight = 1.4
		}
	}

	if style.Overflow == "" {
		style.Overflow = domain.OverflowModeTruncate
	}

	return style
}

func faceForStyle(style domain.Style) (*canvas.FontFace, error) {
	if err := ensureFontFamilies(); err != nil {
		return nil, err
	}

	family := serifFamily
	if strings.Contains(strings.ToLower(style.Font), "sans") {
		family = sansFamily
	}

	return family.Face(
		pointsToMillimetres(style.SizePt),
		color.Black,
		canvas.FontRegular,
		canvas.FontNormal,
	), nil
}

func ensureFontFamilies() error {
	fontFamiliesOnce.Do(func() {
		serifFamily = canvas.NewFontFamily("Letterpress Serif")
		serifFamily.MustLoadFont(lmroman10regular.TTF, 0, canvas.FontRegular)
		serifFamily.MustLoadFont(lmroman10bold.TTF, 0, canvas.FontBold)
		serifFamily.MustLoadFont(lmroman10italic.TTF, 0, canvas.FontItalic)

		sansFamily = canvas.NewFontFamily("Letterpress Sans")
		sansFamily.MustLoadFont(lmsans10regular.TTF, 0, canvas.FontRegular)
		sansFamily.MustLoadFont(lmsans10oblique.TTF, 0, canvas.FontItalic)
	})

	return fontFamilyErr
}

func pointsToMillimetres(points float64) float64 {
	return points / pointsPerMillimetre
}

func toCanvasTextAlign(align domain.TextAlign) canvas.TextAlign {
	switch align {
	case domain.TextAlignCenter:
		return canvas.Center
	case domain.TextAlignRight:
		return canvas.Right
	default:
		return canvas.Left
	}
}
