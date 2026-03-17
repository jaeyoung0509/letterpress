package render

import (
	"github.com/jaeyoung0509/letterpress/internal/domain"
	"github.com/tdewolff/canvas"
)

type Rect struct {
	XMM domain.Millimetres
	YMM domain.Millimetres
	WMM domain.Millimetres
	HMM domain.Millimetres
}

type Page struct {
	Size        domain.PageSize
	Orientation domain.Orientation
	Dimensions  domain.Dimensions
}

func (p Page) LayoutToCanvasRect(rect Rect) Rect {
	return Rect{
		XMM: rect.XMM,
		YMM: p.Dimensions.HeightMM - rect.YMM - rect.HMM,
		WMM: rect.WMM,
		HMM: rect.HMM,
	}
}

type Node struct {
	SlotID     string
	SlotType   domain.SlotType
	LayoutRect Rect
	CanvasRect Rect
	Style      *domain.Style
	Text       string
	ImagePath  string
	Decoration *domain.Asset
	Required   bool
}

type Document struct {
	templateID string
	page       Page
	nodes      []Node
	surface    *canvas.Canvas
	ctx        *canvas.Context
}

func (d *Document) TemplateID() string {
	return d.templateID
}

func (d *Document) Page() Page {
	return d.page
}

func (d *Document) Nodes() []Node {
	nodes := make([]Node, len(d.nodes))
	copy(nodes, d.nodes)
	return nodes
}

func (d *Document) Canvas() *canvas.Canvas {
	return d.surface
}

func (d *Document) Context() *canvas.Context {
	return d.ctx
}
