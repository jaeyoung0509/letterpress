package domain

import "fmt"

// Millimetres is the shared measurement unit for page and layout dimensions.
type Millimetres float64

// Dimensions captures width and height in millimetres.
type Dimensions struct {
	WidthMM  Millimetres
	HeightMM Millimetres
}

var isoPageSizes = map[PageSize]Dimensions{
	PageSizeA3: {WidthMM: 297, HeightMM: 420},
	PageSizeA4: {WidthMM: 210, HeightMM: 297},
	PageSizeA5: {WidthMM: 148, HeightMM: 210},
	PageSizeA6: {WidthMM: 105, HeightMM: 148},
}

// ISOPage returns the page dimensions in millimetres for the given size and orientation.
func ISOPage(size PageSize, orientation Orientation) (Dimensions, error) {
	base, ok := isoPageSizes[size]
	if !ok {
		return Dimensions{}, fmt.Errorf("unsupported page size: %s", size)
	}

	switch orientation {
	case OrientationPortrait:
		return base, nil
	case OrientationLandscape:
		return Dimensions{WidthMM: base.HeightMM, HeightMM: base.WidthMM}, nil
	default:
		return Dimensions{}, fmt.Errorf("unsupported orientation: %s", orientation)
	}
}
