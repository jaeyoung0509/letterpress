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
	Decorations bool `yaml:"decorations,omitempty"`
}

type ExportOptions struct {
	Format ExportFormat `yaml:"format,omitempty"`
	Out    string       `yaml:"out,omitempty"`
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
