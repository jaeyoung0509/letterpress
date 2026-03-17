package domain

import "testing"

func TestISOPagePortraitAndLandscape(t *testing.T) {
	tests := []struct {
		name      string
		size      PageSize
		portrait  Dimensions
		landscape Dimensions
	}{
		{
			name:      "A3",
			size:      PageSizeA3,
			portrait:  Dimensions{WidthMM: 297, HeightMM: 420},
			landscape: Dimensions{WidthMM: 420, HeightMM: 297},
		},
		{
			name:      "A4",
			size:      PageSizeA4,
			portrait:  Dimensions{WidthMM: 210, HeightMM: 297},
			landscape: Dimensions{WidthMM: 297, HeightMM: 210},
		},
		{
			name:      "A5",
			size:      PageSizeA5,
			portrait:  Dimensions{WidthMM: 148, HeightMM: 210},
			landscape: Dimensions{WidthMM: 210, HeightMM: 148},
		},
		{
			name:      "A6",
			size:      PageSizeA6,
			portrait:  Dimensions{WidthMM: 105, HeightMM: 148},
			landscape: Dimensions{WidthMM: 148, HeightMM: 105},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"Portrait", func(t *testing.T) {
			got, err := ISOPage(tt.size, OrientationPortrait)
			if err != nil {
				t.Fatalf("ISOPage(..., Portrait) returned error: %v", err)
			}
			if got != tt.portrait {
				t.Fatalf("ISOPage(..., Portrait) = %#v, want %#v", got, tt.portrait)
			}
		})
		t.Run(tt.name+"Landscape", func(t *testing.T) {
			got, err := ISOPage(tt.size, OrientationLandscape)
			if err != nil {
				t.Fatalf("ISOPage(..., Landscape) returned error: %v", err)
			}
			if got != tt.landscape {
				t.Fatalf("ISOPage(..., Landscape) = %#v, want %#v", got, tt.landscape)
			}
		})
	}
}
