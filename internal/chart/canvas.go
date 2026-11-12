package chart

import (
	"image"
	"image/color"
	"strings"
)

const (
	glyphWidth  = 5
	glyphHeight = 7
	glyphGap    = 1
)

type canvas struct {
	img *image.RGBA
}

func newCanvas(w, h int, bg color.RGBA) canvas {
	c := canvas{img: image.NewRGBA(image.Rect(0, 0, w, h))}
	c.rect(0, 0, w, h, bg)

	return c
}

func (c canvas) rect(x0, y0, x1, y1 int, col color.RGBA) {
	for y := max(y0, 0); y < min(y1, c.img.Rect.Dy()); y++ {
		for x := max(x0, 0); x < min(x1, c.img.Rect.Dx()); x++ {
			c.blend(x, y, col)
		}
	}
}

func (c canvas) blend(x, y int, col color.RGBA) {
	if !(image.Point{X: x, Y: y}.In(c.img.Rect)) {
		return
	}

	if col.A == 255 {
		c.img.SetRGBA(x, y, col)

		return
	}

	under := c.img.RGBAAt(x, y)

	c.img.SetRGBA(x, y, color.RGBA{
		R: mix(col.R, under.R, col.A),
		G: mix(col.G, under.G, col.A),
		B: mix(col.B, under.B, col.A),
		A: 255,
	})
}

func mix(top, bottom, alpha uint8) uint8 {
	a := uint32(alpha)

	return uint8((uint32(top)*a + uint32(bottom)*(255-a)) / 255)
}

func (c canvas) line(x0, y0, x1, y1, thickness int, col color.RGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	err := dx + dy

	for {
		c.rect(x0-thickness/2, y0-thickness/2, x0+thickness-thickness/2, y0+thickness-thickness/2, col)

		if x0 == x1 && y0 == y1 {
			return
		}

		e2 := 2 * err

		if e2 >= dy {
			err += dy
			x0 += sx
		}

		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (c canvas) dashedVertical(x, y0, y1 int, col color.RGBA) {
	for y := y0; y < y1; y += 8 {
		c.rect(x, y, x+2, min(y+4, y1), col)
	}
}

func (c canvas) text(x, y, scale int, s string, col color.RGBA) {
	for _, r := range strings.ToUpper(s) {
		glyph, ok := glyphs[r]
		if !ok {
			glyph = glyphs[' ']
		}

		c.glyph(x, y, scale, glyph, col)
		x += (glyphWidth + glyphGap) * scale
	}
}

func (c canvas) glyph(x, y, scale int, glyph [7]string, col color.RGBA) {
	for row, line := range glyph {
		for colIdx, ch := range line {
			if ch == '#' {
				c.rect(x+colIdx*scale, y+row*scale, x+(colIdx+1)*scale, y+(row+1)*scale, col)
			}
		}
	}
}

func textWidth(s string, scale int) int {
	return len([]rune(s)) * (glyphWidth + glyphGap) * scale
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}
