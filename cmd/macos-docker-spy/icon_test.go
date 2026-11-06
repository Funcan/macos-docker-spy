package main

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func TestIcon(t *testing.T) {
	for _, bg := range []color.RGBA{green, yellow, red} {
		b, err := icon(bg)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		if got := img.Bounds().Dx(); got != 32 {
			t.Errorf("width %d, want 32", got)
		}
		// Corners are rounded off; the middle of an edge is the background.
		if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
			t.Errorf("corner alpha %d, want 0", a)
		}
		if got := color.RGBAModel.Convert(img.At(16, 0)); got != bg {
			t.Errorf("edge colour %v, want %v", got, bg)
		}
	}
}
