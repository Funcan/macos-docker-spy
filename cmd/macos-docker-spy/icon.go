package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

// whale.png is the menu bar glyph: black on transparent, 32x32 (the menu bar
// shows icons at 16x16pt, i.e. 32x32px on Retina). Edit it freely; it is
// drawn on top of the status background, or shown alone as a template image
// (tinted by macOS to match the menu bar) when Docker Desktop isn't running.
//
//go:embed whale.png
var whalePNG []byte

// Status background colours. These are Apple's system colours, which are
// designed to read well in the menu bar in both light and dark mode.
var (
	green  = color.RGBA{0x34, 0xc7, 0x59, 0xff} // running normally
	yellow = color.RGBA{0xff, 0xcc, 0x00, 0xff}
	red    = color.RGBA{0xff, 0x3b, 0x30, 0xff}
)

// icon returns a PNG of the whale on a rounded square of the given colour.
func icon(bg color.Color) ([]byte, error) {
	whale, err := png.Decode(bytes.NewReader(whalePNG))
	if err != nil {
		return nil, err
	}
	bounds := whale.Bounds()
	img := image.NewRGBA(bounds)
	roundedRect(img, bg, float64(bounds.Dx())/4.5)
	draw.Draw(img, bounds, whale, bounds.Min, draw.Over)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// roundedRect fills img with c, rounding the corners to radius r. Edge pixels
// are anti-aliased by supersampling.
func roundedRect(img *image.RGBA, c color.Color, r float64) {
	const samples = 4
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	cr, cg, cb, ca := c.RGBA()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			inside := 0
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					px := float64(x-b.Min.X) + (float64(sx)+0.5)/samples
					py := float64(y-b.Min.Y) + (float64(sy)+0.5)/samples
					// Distance outside the rectangle shrunk by r.
					dx := math.Max(math.Max(r-px, px-(w-r)), 0)
					dy := math.Max(math.Max(r-py, py-(h-r)), 0)
					if dx*dx+dy*dy <= r*r {
						inside++
					}
				}
			}
			cover := uint32(inside * 0xffff / (samples * samples))
			// Premultiplied alpha, so scale every channel by coverage.
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(cr * cover / 0xffff >> 8),
				G: uint8(cg * cover / 0xffff >> 8),
				B: uint8(cb * cover / 0xffff >> 8),
				A: uint8(ca * cover / 0xffff >> 8),
			})
		}
	}
}
