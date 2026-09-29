package scene

import (
	"math"

	"world.execute/internal/term"
)

// Canvas draws smooth curves by packing two by four dots into every character
// cell using braille patterns. Coordinates are normalised to 0..1 within the
// canvas, with y growing downwards, so a scene never has to think about cells.
type Canvas struct {
	screen *term.Screen
	rect   Rect
	cells  []uint8
	colour []uint8
	keys   []term.Color
	w, h   int
}

// brailleBits maps a dot inside a cell to its braille bit.
var brailleBits = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

// NewCanvas returns a canvas covering r. The colours are the drawing keys
// passed to the plot calls: key 0 is the first colour, key 1 the second, and
// so on.
func NewCanvas(s *term.Screen, r Rect, colours ...term.Color) *Canvas {
	if r.W < 1 || r.H < 1 {
		return &Canvas{screen: s, keys: colours}
	}
	c := &Canvas{
		screen: s,
		rect:   r,
		w:      r.W * 2,
		h:      r.H * 4,
		keys:   colours,
	}
	c.cells = make([]uint8, r.W*r.H)
	c.colour = make([]uint8, r.W*r.H)
	c.Clear()
	return c
}

// Rect returns the cell rectangle the canvas covers.
func (c *Canvas) Rect() Rect { return c.rect }

// Dots returns the dot resolution of the canvas.
func (c *Canvas) Dots() (int, int) { return c.w, c.h }

// Clear empties the canvas.
func (c *Canvas) Clear() {
	if c.cells == nil {
		return
	}
	for i := range c.cells {
		c.cells[i] = 0
		c.colour[i] = 0
	}
}

// SetDot lights the dot at integer dot coordinates.
func (c *Canvas) SetDot(x, y int, key int) {
	if c.cells == nil || x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	cx, cy := x/2, y/4
	i := cy*c.rect.W + cx
	c.cells[i] |= brailleBits[x%2][y%4]
	c.colour[i] = uint8(key)
}

// Set lights the dot at normalised coordinates. Values outside 0..1 are drawn
// on the nearest edge, which keeps curves continuous when they leave the box.
func (c *Canvas) Set(x, y float64, key int) {
	dx := int(math.Round(x * float64(c.w-1)))
	dy := int(math.Round(y * float64(c.h-1)))
	c.SetDot(clampInt(dx, 0, c.w-1), clampInt(dy, 0, c.h-1), key)
}

// Line draws a straight line between two normalised points.
func (c *Canvas) Line(x0, y0, x1, y1 float64, key int) {
	steps := int(math.Max(math.Abs(x1-x0)*float64(c.w), math.Abs(y1-y0)*float64(c.h))) + 1
	if steps > 8*c.w+8*c.h {
		steps = 8*c.w + 8*c.h
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		c.Set(x0+(x1-x0)*t, y0+(y1-y0)*t, key)
	}
}

// Polyline draws a chain of normalised points.
func (c *Canvas) Polyline(pts []Point, key int) {
	for i := 1; i < len(pts); i++ {
		c.Line(pts[i-1].X, pts[i-1].Y, pts[i].X, pts[i].Y, key)
	}
}

// Func plots y=f(x) for x over 0..1. f is expected to return 0..1.
func (c *Canvas) Func(f func(x float64) float64, key int, samples int) {
	if samples < 2 {
		samples = c.w
	}
	for i := range samples {
		x := float64(i) / float64(samples-1)
		c.Set(x, f(x), key)
	}
}

// Circle draws a circle of radius r around cx, cy, all in normalised units of
// the x axis. corners scales r on the y axis: pass 0.5 for a visually round
// circle in a cell grid whose dots are twice as tall as they are wide.
func (c *Canvas) Circle(cx, cy, r, corners float64, key int) {
	c.Arc(cx, cy, r, corners, 0, 2*math.Pi, key)
}

// Arc draws the part of a circle between two angles in radians. A negative
// corners value keeps the aspect ratio square instead of matching the cells.
func (c *Canvas) Arc(cx, cy, r, corners, from, to float64, key int) {
	if math.Abs(corners) < 1e-9 {
		corners = 0.5
	}
	steps := int(math.Abs(to-from)*math.Max(r*float64(c.w), 24)) + 8
	if steps > 4096 {
		steps = 4096
	}
	for i := 0; i <= steps; i++ {
		a := from + (to-from)*float64(i)/float64(steps)
		c.Set(cx+r*math.Cos(a), cy+r*corners*math.Sin(a), key)
	}
}

// Disc fills a circle far enough to read as a solid blob at this resolution.
func (c *Canvas) Disc(cx, cy, r, corners float64, key int) {
	for dy := -r; dy <= r; dy += 1 / float64(c.h) {
		half := math.Sqrt(math.Max(r*r-dy*dy, 0))
		y := cy + dy*corners
		c.Line(cx-half, y, cx+half, y, key)
	}
}

// Points lights a list of normalised points, optionally with a glow of
// neighbouring dots.
func (c *Canvas) Points(pts []Point, key int) {
	for _, p := range pts {
		c.Set(p.X, p.Y, key)
	}
}

// Point is a normalised position on a canvas.
type Point struct {
	X, Y float64
}

// Flush writes the canvas into the screen.
func (c *Canvas) Flush() {
	if c.cells == nil || c.screen == nil {
		return
	}
	for cy := range c.rect.H {
		for cx := range c.rect.W {
			bits := c.cells[cy*c.rect.W+cx]
			if bits == 0 {
				continue
			}
			fg := term.ColorDefault
			if int(c.colour[cy*c.rect.W+cx]) < len(c.keys) {
				fg = c.keys[c.colour[cy*c.rect.W+cx]]
			}
			c.screen.Set(c.rect.X+cx, c.rect.Y+cy, rune(0x2800)+rune(bits), fg, term.ColorDefault, term.Attr(0))
		}
	}
}

// Aspect is the ratio between the dot grid and a visually square shape: a
// braille dot is about twice as tall as it is wide.
const Aspect = 0.5

func clampInt(v, lo, hi int) int { return min(max(v, lo), hi) }

// wavePath returns a normalised polyline for a waveform slice, centred in the
// canvas and scaled by gain.
func wavePath(wave []float32, gain float64) []Point {
	pts := make([]Point, len(wave))
	for i, v := range wave {
		x := float64(i) / float64(max(len(wave)-1, 1))
		y := 0.5 - float64(v)*gain*0.5
		pts[i] = Point{X: x, Y: min(max(y, 0), 1)}
	}
	return pts
}
