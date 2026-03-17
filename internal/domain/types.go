package domain

const CurrentSchemaVersion = 1

type SchemaVersion int

type PageSize string

const (
	PageSizeA3 PageSize = "A3"
	PageSizeA4 PageSize = "A4"
	PageSizeA5 PageSize = "A5"
	PageSizeA6 PageSize = "A6"
)

type Orientation string

const (
	OrientationPortrait  Orientation = "portrait"
	OrientationLandscape Orientation = "landscape"
)

type ExportFormat string

const (
	ExportFormatPDF ExportFormat = "pdf"
	ExportFormatPNG ExportFormat = "png"
	ExportFormatSVG ExportFormat = "svg"
)

type RenderMode string

const (
	RenderModeCompose RenderMode = "compose"
	RenderModeASCII   RenderMode = "ascii"
)

type ASCIIMode string

const (
	ASCIIModeTone    ASCIIMode = "tone"
	ASCIIModeOutline ASCIIMode = "outline"
	ASCIIModeFill    ASCIIMode = "fill"
	ASCIIModeHybrid  ASCIIMode = "hybrid"
	ASCIIModeVector  ASCIIMode = "vector"
)

type DitherMode string

const (
	DitherModeOff   DitherMode = "off"
	DitherModeFloyd DitherMode = "floyd"
)

type FillFont string

const (
	FillFontPlain  FillFont = "plain"
	FillFontRepeat FillFont = "repeat"
	FillFontBlock  FillFont = "block"
)

type SlotType string

const (
	SlotTypeText       SlotType = "text"
	SlotTypeImage      SlotType = "image"
	SlotTypeDecoration SlotType = "decoration"
)

type TextAlign string

const (
	TextAlignLeft   TextAlign = "left"
	TextAlignCenter TextAlign = "center"
	TextAlignRight  TextAlign = "right"
)

type OverflowMode string

const (
	OverflowModeClip     OverflowMode = "clip"
	OverflowModeTruncate OverflowMode = "truncate"
	OverflowModeError    OverflowMode = "error"
)

type FitMode string

const (
	FitModeContain FitMode = "contain"
	FitModeCover   FitMode = "cover"
)

type AssetKind string

const (
	AssetKindDecoration AssetKind = "decoration"
	AssetKindImage      AssetKind = "image"
	AssetKindFont       AssetKind = "font"
)

type Project struct {
	Version  SchemaVersion  `yaml:"version"`
	Template string         `yaml:"template"`
	Page     ProjectPage    `yaml:"page"`
	Content  Content        `yaml:"content,omitempty"`
	Images   []ImageBinding `yaml:"images,omitempty"`
	Options  ProjectOptions `yaml:"options,omitempty"`
	Export   ExportOptions  `yaml:"export,omitempty"`
}

type ProjectPage struct {
	Size        PageSize    `yaml:"size"`
	Orientation Orientation `yaml:"orientation"`
}

type Content struct {
	Title     string `yaml:"title,omitempty"`
	Body      string `yaml:"body,omitempty"`
	Signature string `yaml:"signature,omitempty"`
}

type ImageBinding struct {
	Slot string `yaml:"slot"`
	Path string `yaml:"path"`
}

type ProjectOptions struct {
	Decorations bool         `yaml:"decorations,omitempty"`
	RenderMode  RenderMode   `yaml:"render_mode,omitempty"`
	ASCII       ASCIIOptions `yaml:"ascii,omitempty"`
}

type ExportOptions struct {
	Format ExportFormat `yaml:"format,omitempty"`
	Out    string       `yaml:"out,omitempty"`
}

type ASCIIOptions struct {
	Mode          ASCIIMode  `yaml:"mode,omitempty"`
	Charset       string     `yaml:"charset,omitempty"`
	ToneCharset   string     `yaml:"tone_charset,omitempty"`
	FillText      string     `yaml:"fill_text,omitempty"`
	FillFont      FillFont   `yaml:"fill_font,omitempty"`
	Density       int        `yaml:"density,omitempty"`
	Threshold     float64    `yaml:"threshold,omitempty"`
	Contrast      float64    `yaml:"contrast,omitempty"`
	Gamma         float64    `yaml:"gamma,omitempty"`
	Invert        bool       `yaml:"invert,omitempty"`
	EdgeWeight    float64    `yaml:"edge_weight,omitempty"`
	EdgeThreshold float64    `yaml:"edge_threshold,omitempty"`
	Dither        DitherMode `yaml:"dither,omitempty"`
	CellAspect    float64    `yaml:"cell_aspect,omitempty"`
}

func (o ASCIIOptions) EffectiveMode() ASCIIMode {
	if o.Mode == "" {
		return ASCIIModeTone
	}
	return o.Mode
}

func (o ASCIIOptions) EffectiveToneCharset() string {
	if o.ToneCharset != "" {
		return o.ToneCharset
	}
	return o.Charset
}

func (o ASCIIOptions) EffectiveFillFont() FillFont {
	if o.FillFont == "" {
		return FillFontPlain
	}
	return o.FillFont
}

type Template struct {
	Version SchemaVersion    `yaml:"version"`
	ID      string           `yaml:"id"`
	Page    TemplatePage     `yaml:"page"`
	Layout  Layout           `yaml:"layout,omitempty"`
	Slots   []Slot           `yaml:"slots"`
	Styles  map[string]Style `yaml:"styles,omitempty"`
	Assets  []Asset          `yaml:"assets,omitempty"`
}

type TemplatePage struct {
	SupportedSizes     []PageSize  `yaml:"supported_sizes"`
	DefaultOrientation Orientation `yaml:"default_orientation"`
}

type Layout struct {
	MarginMM float64 `yaml:"margin_mm,omitempty"`
}

type Slot struct {
	ID       string   `yaml:"id"`
	Type     SlotType `yaml:"type"`
	XMM      float64  `yaml:"x_mm"`
	YMM      float64  `yaml:"y_mm"`
	WMM      float64  `yaml:"w_mm"`
	HMM      float64  `yaml:"h_mm"`
	Style    string   `yaml:"style,omitempty"`
	Fit      FitMode  `yaml:"fit,omitempty"`
	Required bool     `yaml:"required,omitempty"`
}

type Style struct {
	Font       string       `yaml:"font"`
	SizePt     float64      `yaml:"size_pt"`
	LineHeight float64      `yaml:"line_height,omitempty"`
	Align      TextAlign    `yaml:"align,omitempty"`
	MaxLines   int          `yaml:"max_lines,omitempty"`
	Overflow   OverflowMode `yaml:"overflow,omitempty"`
}

type Asset struct {
	ID   string    `yaml:"id"`
	Kind AssetKind `yaml:"kind"`
	Path string    `yaml:"path"`
}
