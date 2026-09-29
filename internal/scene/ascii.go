package scene

import (
	"fmt"
	"math"
	"strings"

	"world.execute/internal/term"
)

// The ASCII art engine draws an object as a mask filled with streaming code:
// every column of the shape falls at its own speed through a stock of hex,
// binary and symbols, the outline is drawn in heavy glyphs, a scanner band
// sweeps across, and a few hand placed glyphs give it a face. There is no
// lighting and no shading, only characters.
//
// Shapes live in "unit space": x counts cells across and y counts half rows, so
// a circle of radius r units looks round in the terminal.

var (
	codeGlyphs = []rune("0123456789ABCDEF{}[]<>/\\|=+-*#%&$;:")
	edgeGlyphs = []rune("@#%&8")
)

type asciiFeature func(x, y float64) (rune, term.Color, bool)

type asciiArt struct {
	inside  func(x, y float64) bool
	feature asciiFeature
	body    term.Color
	edge    term.Color
	charset []rune
	seed    int64
}

func pickRune(set []rune, a, b int) rune {
	return set[int(uint(a*73856093^b*19349663))%len(set)]
}

// draw paints the art into r at time t and returns the box it covered.
func (a asciiArt) draw(ctx *Context, r Rect, t float64) Rect {
	s := ctx.Screen
	pal := ctx.Palette
	set := a.charset
	if len(set) == 0 {
		set = codeGlyphs
	}
	minX, minY, maxX, maxY := r.Right(), r.Bottom(), r.X-1, r.Y-1
	for cy := r.Y; cy < r.Bottom(); cy++ {
		for cx := r.X; cx < r.Right(); cx++ {
			ux, uy := float64(cx-r.X)+0.5, float64(cy-r.Y)*2+1
			if a.feature != nil {
				if ch, col, ok := a.feature(ux, uy); ok {
					s.Set(cx, cy, ch, col, term.ColorDefault, term.Bold)
					minX, maxX = min(minX, cx), max(maxX, cx)
					minY, maxY = min(minY, cy), max(maxY, cy)
					continue
				}
			}
			if !a.inside(ux, uy) {
				continue
			}
			minX, maxX = min(minX, cx), max(maxX, cx)
			minY, maxY = min(minY, cy), max(maxY, cy)
			if !a.inside(ux-1, uy) || !a.inside(ux+1, uy) || !a.inside(ux, uy-2) || !a.inside(ux, uy+2) {
				s.Set(cx, cy, pickRune(edgeGlyphs, cx, cy+int(t*6)), a.edge, term.ColorDefault, term.Bold)
				continue
			}
			col := cx - r.X
			speed := 3 + float64(uint64(hashSeed(a.seed, col))%9)
			k := int(t*speed) + cy
			h := uint64(hashSeed(a.seed+int64(col)*131, k))
			phase := (k + col*5) % 14
			b := 1 - float64(phase)/14
			fg := Blend(pal.Shadow, a.body, 0.42+0.58*b*b)
			attr := term.Attr(0)
			if phase == 0 {
				fg, attr = Blend(a.body, pal.Ink(), 0.55), term.Bold
			}
			if v := math.Mod(ux+uy*0.6-t*22, 46); (v >= 0 && v < 5) || v < -41 {
				fg, attr = pal.Ink(), term.Bold
			}
			s.Set(cx, cy, set[h%uint64(len(set))], fg, term.ColorDefault, attr)
		}
	}
	if maxX < minX {
		return Rect{X: r.X + r.W/2, Y: r.Y + r.H/2}
	}
	return Rect{X: minX, Y: minY, W: maxX - minX + 1, H: maxY - minY + 1}
}

func ellipseMask(cx, cy, rx, ry float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return dx*dx+dy*dy <= 1
	}
}

func anyMask(masks ...func(x, y float64) bool) func(x, y float64) bool {
	return func(x, y float64) bool {
		for _, m := range masks {
			if m(x, y) {
				return true
			}
		}
		return false
	}
}

func polyMask(pts [][2]float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		in := false
		for i := range pts {
			a, b := pts[i], pts[(i+1)%len(pts)]
			if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])/(b[1]-a[1])*(b[0]-a[0]) {
				in = !in
			}
		}
		return in
	}
}

func heartMask(cx, cy, R float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		px := (x - cx) / R * 1.15
		py := -(y-cy)/R*1.15 + 0.15
		a := px*px + py*py - 1
		return a*a*a-px*px*py*py*py <= 0
	}
}

func segDist(x, y, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	l := dx*dx + dy*dy
	if l == 0 {
		return math.Hypot(x-x0, y-y0)
	}
	k := math.Min(math.Max(((x-x0)*dx+(y-y0)*dy)/l, 0), 1)
	return math.Hypot(x-(x0+k*dx), y-(y0+k*dy))
}

func mergeFeatures(fs ...asciiFeature) asciiFeature {
	return func(x, y float64) (rune, term.Color, bool) {
		for _, f := range fs {
			if f == nil {
				continue
			}
			if ch, col, ok := f(x, y); ok {
				return ch, col, true
			}
		}
		return 0, 0, false
	}
}

// leafFeature draws a small star of leaves, the calyx of a tomato or eggplant.
func leafFeature(cx, cy, outer float64, n int, rot float64, col term.Color) asciiFeature {
	leaves := []rune("VvYyw\\/")
	return func(x, y float64) (rune, term.Color, bool) {
		dx, dy := x-cx, y-cy
		rr := math.Hypot(dx, dy)
		ang := math.Atan2(dy, dx)
		edge := outer * (0.28 + 0.72*math.Pow(math.Abs(math.Cos(float64(n)/2*(ang-rot))), 3))
		if rr > edge {
			return 0, 0, false
		}
		return pickRune(leaves, int(x), int(y)), col, true
	}
}

func tomatoArt(pal Palette, cx, cy, R, t float64) asciiArt {
	rx, ry := R*1.06, R*0.90
	cyb := cy + R*0.08
	topY := cyb - ry*0.82
	sway := 0.12 * math.Sin(t*1.3)
	green := pal.Accent
	return asciiArt{
		inside: ellipseMask(cx, cyb, rx, ry),
		feature: mergeFeatures(
			func(x, y float64) (rune, term.Color, bool) {
				if math.Abs(x-(cx+sway*R*0.3)) < 0.6 && y < topY-R*0.05 && y > topY-R*0.42 {
					return '|', green, true
				}
				return 0, 0, false
			},
			leafFeature(cx, topY+R*0.04, R*0.46, 5, -math.Pi/2+sway, green),
		),
		body: pal.Err, edge: pal.Err, seed: 11,
	}
}

func eggplantArt(pal Palette, cx, cy, R, t float64) asciiArt {
	const n = 26
	type ball struct{ x, y, rad float64 }
	balls := make([]ball, n+1)
	for i := 0; i <= n; i++ {
		s := float64(i) / n
		balls[i] = ball{
			x:   cx - R*0.16 + R*0.38*s*s + R*0.03*math.Sin(t*1.1),
			y:   cy + R*0.80 - s*R*1.64,
			rad: R * (0.14 + 0.42*math.Pow(1-s, 1.2)),
		}
	}
	top := balls[n]
	inside := func(x, y float64) bool {
		for _, b := range balls {
			if sq(x-b.x)+sq(y-b.y) <= b.rad*b.rad {
				return true
			}
		}
		return false
	}
	return asciiArt{
		inside: inside,
		feature: mergeFeatures(
			func(x, y float64) (rune, term.Color, bool) {
				if math.Abs(x-(top.x+R*0.12)) < 0.7 && y < top.y-R*0.02 && y > top.y-R*0.34 {
					return '|', pal.Accent, true
				}
				return 0, 0, false
			},
			leafFeature(top.x, top.y+R*0.04, R*0.38, 6, math.Pi/2*0.2+0.15*math.Sin(t), pal.Accent),
		),
		body: pal.Kind, edge: pal.Kind, seed: 23,
	}
}

func sq(v float64) float64 { return v * v }

func catArt(pal Palette, cx, cy, R, t float64) asciiArt {
	hx, hy := cx, cy+R*0.16
	rx, ry := R*1.02, R*0.82
	blink := 1.0
	if m := math.Mod(t, 3.1); m > 2.9 {
		blink = math.Abs(m-3.0) * 10
	}
	look := R * 0.05 * math.Sin(t*0.8)
	earMask := func(x, y float64) bool {
		for _, side := range []float64{-1, 1} {
			tri := [][2]float64{
				{hx + side*rx*0.95, hy - ry*0.05},
				{hx + side*rx*0.82, hy - ry*1.38},
				{hx + side*rx*0.20, hy - ry*0.82},
			}
			if polyMask(tri)(x, y) {
				return true
			}
		}
		return false
	}
	eyeR := ellipseMask
	feature := func(x, y float64) (rune, term.Color, bool) {
		for _, side := range []float64{-1, 1} {
			ex, ey := hx+side*rx*0.42, hy-ry*0.08
			erx, ery := rx*0.20, math.Max(ry*0.24*blink, 0.6)
			if eyeR(ex, ey, erx, ery)(x, y) {
				if blink > 0.4 && math.Abs(x-(ex+look)) < 0.7 {
					return '|', pal.Ink(), true
				}
				return '@', pal.Accent, true
			}
			for k := -1; k <= 1; k++ {
				y0 := hy + ry*(0.30+0.07*float64(k))
				y1 := hy + ry*(0.28+0.26*float64(k)) + 0.5*math.Sin(t*3+float64(k))
				if segDist(x, y, hx+side*rx*0.48, y0, hx+side*rx*1.30, y1) < 0.55 {
					return '-', pal.Text, true
				}
			}
		}
		nx, ny := hx, hy+ry*0.24
		if math.Abs(x-nx) < rx*0.10 && math.Abs(y-ny) < ry*0.10 {
			return 'v', pal.Err, true
		}
		if math.Abs(x-nx) < 1.2 && y > ny+ry*0.1 && y < ny+ry*0.2 {
			return 'w', pal.Dim, true
		}
		if ellipseMask(hx, hy+ry*0.42, rx*0.36, ry*0.28)(x, y) && (int(x)+int(y/2))%2 == 0 {
			return '.', pal.Dim, true
		}
		for k := -1; k <= 1; k++ {
			if math.Abs(x-(hx+float64(k)*rx*0.22)) < 0.6 && y < hy-ry*0.3 && y > hy-ry*0.9 {
				return 'I', pal.Dim, true
			}
		}
		return 0, 0, false
	}
	return asciiArt{
		inside:  anyMask(ellipseMask(hx, hy, rx, ry), earMask),
		feature: feature,
		body:    pal.Warn, edge: pal.Warn, seed: 37,
	}
}

func sunArt(pal Palette, cx, cy, R, t, beat float64) asciiArt {
	const rays = 12
	heart := heartMask(cx, cy+R*0.04, R*0.38)
	feature := func(x, y float64) (rune, term.Color, bool) {
		if heart(x, y) {
			if (int(x)+int(y/2))%2 == 0 {
				return '<', pal.Err, true
			}
			return '3', pal.Kind, true
		}
		dx, dy := x-cx, y-cy
		rr := math.Hypot(dx, dy)
		if rr < R*0.70 || rr > R*(1.5+0.15*beat) {
			return 0, 0, false
		}
		ang := math.Atan2(dy, dx) - t*0.3
		step := 2 * math.Pi / rays
		off := math.Mod(ang+math.Pi*2, step)
		if off > step/2 {
			off -= step
		}
		idx := int(math.Round((ang + math.Pi*2) / step))
		length := R * 1.5
		if idx%2 == 1 {
			length = R * 1.15
		}
		if rr > length {
			return 0, 0, false
		}
		width := 0.20 * (1 - (rr-R*0.7)/(length-R*0.7))
		if math.Abs(off) <= width {
			return pickRune([]rune("*+*x"), int(x), int(y)), pal.Warn, true
		}
		return 0, 0, false
	}
	return asciiArt{
		inside:  ellipseMask(cx, cy, R*0.66, R*0.66),
		feature: feature,
		body:    pal.Warn, edge: pal.Warn, seed: 47,
		charset: []rune("01"),
	}
}

func heartArt(pal Palette, cx, cy, R float64, body term.Color) asciiArt {
	return asciiArt{inside: heartMask(cx, cy, R), body: body, edge: body, seed: 59, charset: []rune("01<3")}
}

func globeArt(pal Palette, cx, cy, R, t float64) asciiArt {
	a := t * 0.5
	sa, ca := math.Sin(a), math.Cos(a)
	feature := func(x, y float64) (rune, term.Color, bool) {
		nx, ny := (x-cx)/R, (y-cy)/R
		d2 := nx*nx + ny*ny
		if d2 > 0.86 {
			return 0, 0, false
		}
		nz := math.Sqrt(1 - d2)
		px := nx*ca + nz*sa
		pz := -nx*sa + nz*ca
		lat := math.Asin(math.Max(math.Min(-ny, 1), -1))
		lon := math.Atan2(px, pz)
		for _, g := range []float64{lon / (math.Pi / 6), lat / (math.Pi / 6)} {
			if math.Abs(g-math.Round(g)) < 0.05 {
				return ':', pal.Dim, true
			}
		}
		v := math.Sin(2.3*lon+0.7)*math.Cos(1.6*lat) + 0.55*math.Sin(4.1*lat-lon*1.3+1.9) + 0.35*math.Sin(6*lon+3*lat)
		switch {
		case math.Abs(lat) > 1.25:
			return '=', pal.Ink(), true
		case v > 0.45:
			return pickRune([]rune("#%@"), int(x), int(y)), pal.Accent, true
		}
		return pickRune([]rune("~.~-"), int(x+t*6), int(y)), pal.Accent2, true
	}
	return asciiArt{
		inside:  ellipseMask(cx, cy, R, R),
		feature: feature,
		body:    pal.Accent2, edge: pal.Accent2, seed: 71,
	}
}

func shieldArt(pal Palette, cx, cy, R float64) asciiArt {
	outline := [][2]float64{
		{cx - 0.82*R, cy - 0.80*R}, {cx, cy - 1.0*R}, {cx + 0.82*R, cy - 0.80*R},
		{cx + 0.78*R, cy + 0.10*R}, {cx + 0.40*R, cy + 0.66*R}, {cx, cy + 1.0*R},
		{cx - 0.40*R, cy + 0.66*R}, {cx - 0.78*R, cy + 0.10*R},
	}
	for range 3 {
		next := make([][2]float64, 0, 2*len(outline))
		for i := range outline {
			p, q := outline[i], outline[(i+1)%len(outline)]
			next = append(next,
				[2]float64{0.75*p[0] + 0.25*q[0], 0.75*p[1] + 0.25*q[1]},
				[2]float64{0.25*p[0] + 0.75*q[0], 0.25*p[1] + 0.75*q[1]})
		}
		outline = next
	}
	tick := func(x, y float64) (rune, term.Color, bool) {
		if segDist(x, y, cx-0.34*R, cy+0.02*R, cx-0.08*R, cy+0.32*R) < 1.1 ||
			segDist(x, y, cx-0.08*R, cy+0.32*R, cx+0.40*R, cy-0.32*R) < 1.1 {
			return '#', pal.Ink(), true
		}
		return 0, 0, false
	}
	return asciiArt{inside: polyMask(outline), feature: tick, body: pal.Accent2, edge: pal.Accent, seed: 83, charset: []rune("01")}
}

func warningArt(pal Palette, cx, cy, R float64) asciiArt {
	tri := [][2]float64{{cx, cy - 1.1*R}, {cx + 1.15*R, cy + 0.85*R}, {cx - 1.15*R, cy + 0.85*R}}
	bang := func(x, y float64) (rune, term.Color, bool) {
		if math.Abs(x-cx) < 1.1 {
			if y > cy-0.3*R && y < cy+0.28*R {
				return '!', pal.Ink(), true
			}
			if y > cy+0.46*R && y < cy+0.62*R {
				return '.', pal.Ink(), true
			}
		}
		return 0, 0, false
	}
	return asciiArt{inside: polyMask(tri), feature: bang, body: pal.Err, edge: pal.Warn, seed: 97}
}

func cageArt(pal Palette, cx, cy, R, t float64) asciiArt {
	rx, ry := R*1.2, R*1.1
	inner := ellipseMask(cx, cy, rx*0.82, ry*0.82)
	box := func(x, y float64) bool { return math.Abs(x-cx) <= rx && math.Abs(y-cy) <= ry }
	feature := func(x, y float64) (rune, term.Color, bool) {
		if !box(x, y) || inner(x, y) {
			return 0, 0, false
		}
		if int(x)%3 == 0 {
			return '|', pal.Accent2, true
		}
		return 0, 0, false
	}
	return asciiArt{inside: func(x, y float64) bool { return false }, feature: feature, body: pal.Accent2, edge: pal.Accent2, seed: 101}
}

// hudRow is one line of a target readout.
type hudRow struct{ label, value string }

// drawHUD wraps an object in the furniture of a targeting system: corner
// brackets that breathe, a crosshair with tick marks, a hex dump scrolling up
// the left, live readouts on the right and a progress bar along the bottom.
func drawHUD(ctx *Context, r, box Rect, title string, rows []hudRow, progress, t float64) {
	s := ctx.Screen
	pal := ctx.Palette
	dim := pal.Faint()
	breathe := int(math.Round(0.6 + 0.6*math.Sin(t*2.4)))
	x0, x1 := max(box.X-2-breathe, r.X), min(box.Right()+1+breathe, r.Right()-1)
	y0, y1 := max(box.Y-1-breathe/2, r.Y+1), min(box.Bottom()+breathe/2, r.Bottom()-1)
	corner := func(x, y, dx, dy int) {
		s.Set(x, y, '+', pal.Accent, term.ColorDefault, term.Bold)
		for i := 1; i <= 4; i++ {
			s.Set(x+dx*i, y, '-', pal.Accent, term.ColorDefault, term.Bold)
		}
		for i := 1; i <= 2; i++ {
			s.Set(x, y+dy*i, '|', pal.Accent, term.ColorDefault, term.Bold)
		}
	}
	corner(x0, y0, 1, 1)
	corner(x1, y0, -1, 1)
	corner(x0, y1, 1, -1)
	corner(x1, y1, -1, -1)

	cx, cy := box.X+box.W/2, box.Y+box.H/2
	for x := r.X; x < r.Right(); x++ {
		if x >= x0-1 && x <= x1+1 {
			continue
		}
		ch := '-'
		if (x-cx)%6 == 0 {
			ch = '+'
		}
		s.Set(x, cy, ch, dim, term.ColorDefault, term.Attr(0))
	}
	for y := r.Y; y < r.Bottom(); y++ {
		if y >= y0-1 && y <= y1+1 {
			continue
		}
		ch := ':'
		if (y-cy)%3 == 0 {
			ch = '+'
		}
		s.Set(cx, y, ch, dim, term.ColorDefault, term.Attr(0))
	}

	head := "[ " + title + " ]"
	s.Text(x0, r.Y, head, pal.Accent, term.ColorDefault, term.Bold)
	rx := x1 + 3
	if r.Right()-rx >= 18 {
		for i, row := range rows {
			if y0+i > r.Bottom()-3 {
				break
			}
			s.Text(rx, y0+i, fmt.Sprintf("%-6s", row.label), dim, term.ColorDefault, term.Attr(0))
			s.Text(rx+7, y0+i, row.value, pal.Accent2, term.ColorDefault, term.Bold)
		}
	}
	if lw := x0 - 3 - r.X; lw >= 12 {
		scroll := int(t * 5)
		for y := r.Y; y < r.Bottom()-1; y++ {
			h := uint64(hashSeed(ctx.Seed, y+scroll))
			line := fmt.Sprintf("%04X %02X %02X %02X %02X", h&0xffff, h>>16&0xff, h>>24&0xff, h>>32&0xff, h>>40&0xff)
			if len(line) > lw {
				line = line[:lw]
			}
			s.Text(r.X, y, line, dim, term.ColorDefault, term.Attr(0))
		}
	}
	if r.H >= 8 && r.W >= 30 {
		const barW = 16
		filled := int(progress * barW)
		bar := "[" + strings.Repeat("#", filled) + strings.Repeat(".", barW-filled) + "]"
		s.Text(r.X, r.Bottom()-2, fmt.Sprintf("%s %3d%%", bar, int(progress*100)), pal.Accent, term.ColorDefault, term.Attr(0))
	}
}

func boxArt(pal Palette, cx, cy, rx, ry float64, seed int64) asciiArt {
	return asciiArt{
		inside: func(x, y float64) bool { return math.Abs(x-cx) <= rx && math.Abs(y-cy) <= ry },
		body:   pal.Accent2, edge: pal.Accent, seed: seed, charset: []rune("01{}[]<>"),
	}
}

// labelFeature stamps text across the middle row of a shape.
func labelFeature(text string, cx, cy float64, col term.Color) asciiFeature {
	runes := []rune(text)
	left := cx - float64(len(runes))/2
	return func(x, y float64) (rune, term.Color, bool) {
		i := int(math.Floor(x - left))
		if math.Abs(y-cy) < 1 && i >= 0 && i < len(runes) && runes[i] != ' ' {
			return runes[i], col, true
		}
		return 0, 0, false
	}
}
