package scene

import (
	"fmt"
	"math"
	"strings"
	"time"

	"world.execute/internal/term"
)

// stage draws ASCII figures in cell coordinates, clipped to a rectangle.
type stage struct {
	ctx *Context
	r   Rect
}

func (g stage) set(x, y int, ch rune, col term.Color, attr term.Attr) {
	if x < g.r.X || x >= g.r.Right() || y < g.r.Y || y >= g.r.Bottom() {
		return
	}
	g.ctx.Screen.Set(x, y, ch, col, term.ColorDefault, attr)
}

func (g stage) text(x, y int, s string, col term.Color, attr term.Attr) {
	for _, ch := range s {
		g.set(x, y, ch, col, attr)
		x++
	}
}

// strokeGlyph picks - | / or \ for a segment, allowing for cells being twice as
// tall as they are wide.
func strokeGlyph(dx, dy float64) rune {
	a := math.Abs(math.Atan2(dy*2, dx))
	if a > math.Pi/2 {
		a = math.Pi - a
	}
	switch {
	case a < 0.4:
		return '-'
	case a > 1.17:
		return '|'
	}
	if (dx > 0) == (dy > 0) {
		return '\\'
	}
	return '/'
}

func (g stage) line(x0, y0, x1, y1 float64, glyph rune, col term.Color, attr term.Attr) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)*2)) + 1
	ch := glyph
	if ch == 0 {
		ch = strokeGlyph(x1-x0, y1-y0)
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		g.set(int(math.Round(x0+(x1-x0)*t)), int(math.Round(y0+(y1-y0)*t)), ch, col, attr)
	}
}

func hashUnit(seed int64, a int) float64 {
	return float64(uint64(hashSeed(seed, a))>>11) / float64(1<<53)
}

const piDigits = "3.14159265358979323846264338327950288419716939937510582097494459230781640628620899"

func drawCircleRich(ctx *Context, box Rect, energy, progress float64) {
	if box.W < 30 || box.H < 9 {
		return
	}
	g := stage{ctx, box}
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	cx, cy := float64(box.X)+float64(box.W)/2, float64(box.Y)+float64(box.H)/2
	ry := math.Min(float64(box.H)*0.34, float64(box.W)*0.16) * (1 + 0.05*energy)
	rx := 2 * ry
	end := progress * 2 * math.Pi

	sector := asciiArt{
		inside: func(x, y float64) bool {
			dx, dy := x-float64(box.W)/2, y-float64(box.H)
			if dx*dx+dy*dy > rx*rx {
				return false
			}
			ang := math.Atan2(dy, dx) + math.Pi/2
			if ang < 0 {
				ang += 2 * math.Pi
			}
			return ang <= end
		},
		body: pal.Accent2, edge: pal.Accent, seed: 131, charset: []rune("0123456789ABCDEF"),
	}
	sector.draw(ctx, box, t)

	for k := range 720 {
		a := 2 * math.Pi * float64(k) / 720
		col, attr := pal.Faint(), term.Attr(0)
		if math.Mod(a+math.Pi/2+2*math.Pi, 2*math.Pi) <= end {
			col, attr = pal.Accent, term.Bold
		}
		g.set(int(math.Round(cx+rx*math.Cos(a))), int(math.Round(cy+ry*math.Sin(a))),
			strokeGlyph(-rx*math.Sin(a), ry*math.Cos(a)), col, attr)
	}
	for deg := 0; deg < 360; deg += 5 {
		a := -math.Pi/2 + float64(deg)*math.Pi/180
		ch := '.'
		if deg%30 == 0 {
			ch = '+'
			lx, ly := int(math.Round(cx+rx*1.30*math.Cos(a)))-1, int(math.Round(cy+ry*1.30*math.Sin(a)))
			g.text(lx, ly, fmt.Sprintf("%03d", deg), pal.Dim, term.Attr(0))
		}
		g.set(int(math.Round(cx+rx*1.12*math.Cos(a))), int(math.Round(cy+ry*1.12*math.Sin(a))), ch, pal.Faint(), term.Attr(0))
	}

	tip := -math.Pi/2 + end
	tx, ty := cx+rx*0.96*math.Cos(tip), cy+ry*0.96*math.Sin(tip)
	g.line(cx, cy, tx, ty, 0, pal.Ink(), term.Bold)
	g.set(int(math.Round(cx)), int(math.Round(cy)), '@', pal.Ink(), term.Bold)
	g.set(int(math.Round(tx)), int(math.Round(ty)), 'O', pal.Warn, term.Bold)
	g.text(int(math.Round(tx))+2, int(math.Round(ty))-1, fmt.Sprintf("theta=%03d", int(progress*360)), pal.Warn, term.Bold)
	g.text(int(cx-rx/2), int(cy)+1, "r", pal.Dim, term.Attr(0))

	digits := int(progress * float64(len(piDigits)-10))
	step := 2.0 / (1.30 * rx)
	for k := 0; k < digits && k < len(piDigits); k++ {
		a := -math.Pi/2 + float64(k)*step + t*0.05
		g.set(int(math.Round(cx+rx*1.42*math.Cos(a))), int(math.Round(cy+ry*1.42*math.Sin(a))), rune(piDigits[k]), pal.Accent2, term.Bold)
	}
}

func drawSineRich(ctx *Context, box Rect, energy float64) {
	if box.W < 34 || box.H < 9 {
		return
	}
	g := stage{ctx, box}
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	left, right := box.X+5, box.Right()-2
	top, bottom := box.Y+1, box.Bottom()-2
	w := float64(right - left)
	mid := float64(top+bottom) / 2
	amp := float64(bottom-top) / 2 * 0.86

	treble := 0.0
	if ctx.Analysis != nil {
		bands := ctx.Analysis.BandsAt(ctx.T, ctx.Bands)
		for _, b := range bands[len(bands)/2:] {
			treble += float64(b)
		}
		treble /= float64(len(bands) - len(bands)/2)
	}
	freq := 1.6 + 3.4*treble + 1.5*energy
	phase := t * 1.4
	f := func(x float64) float64 { return math.Sin(2 * math.Pi * (freq*(x-float64(left))/w + phase)) }
	row := func(x float64) float64 { return mid - amp*f(x) }

	for gx := left; gx <= right; gx += 8 {
		for gy := top; gy <= bottom; gy += 3 {
			g.set(gx, gy, '.', pal.Faint(), term.Attr(0))
		}
	}
	for x := left; x <= right; x++ {
		ch := '-'
		if (x-left)%8 == 0 {
			ch = '+'
		}
		g.set(x, int(math.Round(mid)), ch, pal.Dim, term.Attr(0))
	}
	for y := top; y <= bottom; y++ {
		g.set(left-1, y, '|', pal.Dim, term.Attr(0))
	}
	g.text(box.X, top, "+1", pal.Dim, term.Attr(0))
	g.text(box.X, int(math.Round(mid)), " 0", pal.Dim, term.Attr(0))
	g.text(box.X, bottom, "-1", pal.Dim, term.Attr(0))
	for k := 0; k <= 4; k++ {
		g.text(left+int(w*float64(k)/4)-1, bottom+1, fmt.Sprintf("%dpi", k), pal.Faint(), term.Attr(0))
	}

	for x := left; x <= right; x++ {
		yv := row(float64(x))
		lo, hi := int(math.Ceil(math.Min(mid, yv))), int(math.Floor(math.Max(mid, yv)))
		for y := lo; y <= hi; y++ {
			h := uint64(hashSeed(int64(x), y+int(t*9)+x))
			g.set(x, y, codeGlyphs[h%16], Blend(pal.Shadow, pal.Accent2, 0.42), term.Attr(0))
		}
	}
	for x := left; x <= right; x++ {
		g.set(x, int(math.Round(mid-amp*0.5*math.Cos(2*math.Pi*(freq*(float64(x-left))/w+phase)))), '.', pal.Faint(), term.Attr(0))
	}
	for x := left; x < right; x++ {
		g.line(float64(x), row(float64(x)), float64(x+1), row(float64(x+1)), 0, pal.Accent, term.Bold)
	}
	for x := left + 3; x <= right; x += 6 {
		y := int(math.Round(row(float64(x))))
		g.set(x, y, 'o', pal.Ink(), term.Bold)
		for s := min(y, int(mid)) + 1; s < max(y, int(mid)); s++ {
			g.set(x, s, ':', pal.Faint(), term.Attr(0))
		}
	}
	for k := 1; k <= 5; k++ {
		x := float64(left) + w*float64(k)/6
		y := row(x)
		m := row(x+0.5) - row(x-0.5)
		g.line(x-3, y-3*m, x+3, y+3*m, 0, pal.Warn, term.Bold)
		g.text(int(x)-3, int(math.Round(y-3*m))-1, fmt.Sprintf("m=%+.1f", -m*2), pal.Dim, term.Attr(0))
	}
	xc := left + int(math.Mod(t*8, w))
	for y := top; y <= bottom; y++ {
		g.set(xc, y, ':', pal.Warn, term.Attr(0))
	}
	yc := int(math.Round(row(float64(xc))))
	g.set(xc, yc, 'O', pal.Warn, term.Bold)
	label := fmt.Sprintf("[ y=%+.2f ]", f(float64(xc)))
	g.text(min(xc+1, right-len(label)), top, label, pal.Warn, term.Bold)
}

func drawLimitRich(ctx *Context, box Rect, progress float64) {
	if box.W < 34 || box.H < 9 {
		return
	}
	g := stage{ctx, box}
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	left, right := box.X+5, box.Right()-2
	top, bottom := box.Y+1, box.Bottom()-2
	w := float64(right - left)
	asym := float64(top) + float64(bottom-top)*0.74
	scale := (asym - float64(top)) / 1.0
	at := func(u float64) (float64, float64, float64) {
		xv := math.Pow(10, 3*u)
		val := 1 + 1/xv
		return xv, val, asym - (val-1)*scale
	}

	for x := left; x <= right; x++ {
		g.set(x, bottom, '-', pal.Dim, term.Attr(0))
	}
	for y := top; y <= bottom; y++ {
		g.set(left-1, y, '|', pal.Dim, term.Attr(0))
	}
	for k := 0; k <= 3; k++ {
		x := left + int(w*float64(k)/3)
		g.set(x, bottom, '+', pal.Dim, term.Attr(0))
		g.text(x-1, bottom+1, fmt.Sprintf("1e%d", k), pal.Faint(), term.Attr(0))
	}

	eps := 5.5*(1-progress) + 0.6
	for x := left; x <= right; x++ {
		if (x-left)%2 == 0 {
			g.set(x, int(math.Round(asym)), '-', pal.Accent2, term.Bold)
		}
		if (x-left)%3 == 0 {
			g.set(x, int(math.Round(asym-eps)), '.', pal.Faint(), term.Attr(0))
			g.set(x, int(math.Round(asym+eps)), '.', pal.Faint(), term.Attr(0))
		}
	}
	g.text(right-24, int(math.Round(asym))+1, "y = L   (never reached)", pal.Accent2, term.Bold)
	g.text(left+int(w*0.62), int(math.Round(asym-eps))-1, fmt.Sprintf("eps=%.3f", eps/scale), pal.Dim, term.Attr(0))

	px := int(w * progress)
	var lx, ly float64
	for i := 0; i <= px; i++ {
		u := float64(i) / w
		_, _, y := at(u)
		x := float64(left + i)
		if i > 0 {
			g.line(lx, ly, x, y, 0, pal.Accent, term.Bold)
		}
		lx, ly = x, y
		if i%4 == 0 {
			for yy := int(math.Round(y)) + 1; yy < int(math.Round(asym)); yy++ {
				g.set(left+i, yy, ':', pal.Faint(), term.Attr(0))
			}
		}
	}
	xv, val, y := at(float64(px) / w)
	g.set(left+px, int(math.Round(y)), '@', pal.Warn, term.Bold)
	g.text(min(left+px+2, right-30), int(math.Round(y))-1, fmt.Sprintf("x=%-6.0f f=%.4f d=%.1e", xv, val, val-1), pal.Warn, term.Bold)

	for k := 1; k <= 4; k++ {
		if progress*3.2 < float64(k-1) {
			continue
		}
		row := top + k - 1
		g.text(left+int(w*0.34), row, fmt.Sprintf("f(10^%d) = 1.%s", k, strings.Repeat("0", k-1)+"1"), pal.Dim, term.Attr(0))
	}

	cx, cy := float64(right-14), float64(top+4)
	for k := range 220 {
		a := 2 * math.Pi * float64(k) / 220
		den := 1 + math.Sin(a)*math.Sin(a)
		x, y := 11*math.Cos(a)/den, 3.4*math.Sin(a)*math.Cos(a)/den
		ch := '8'
		col := pal.Accent2
		if math.Mod(a-t*2+4*math.Pi, 2*math.Pi) < 0.16 {
			ch, col = 'O', pal.Ink()
		}
		g.set(int(math.Round(cx+x)), int(math.Round(cy+y)), ch, col, term.Bold)
	}
}

func drawCircuit(ctx *Context, r Rect, regs []regEntry) {
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(regs[0].at).Seconds()
	switchAt := regs[1].at
	k := ease(Progress(ctx.T, switchAt, switchAt+450*time.Millisecond))

	circuitH := max(r.H*11/20, 7)
	x0, x1 := r.X+4, r.Right()-5
	y0, y1 := r.Y+1, r.Y+circuitH-1
	for x := x0; x <= x1; x++ {
		g.set(x, y0, '-', pal.Dim, term.Attr(0))
		g.set(x, y1, '-', pal.Dim, term.Attr(0))
	}
	for y := y0; y <= y1; y++ {
		g.set(x0, y, '|', pal.Dim, term.Attr(0))
		g.set(x1, y, '|', pal.Dim, term.Attr(0))
	}
	for _, p := range [][2]int{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}} {
		g.set(p[0], p[1], '+', pal.Dim, term.Attr(0))
	}
	ym := (y0 + y1) / 2

	source := "(~)"
	if k > 0.5 {
		source = "[=]"
	}
	g.text(x0-1, ym-1, "+", pal.Accent, term.Bold)
	for i, ch := range source {
		g.set(x0-1+i, ym, ch, pal.Accent, term.Bold)
	}
	g.text(x0-1, ym+1, "-", pal.Accent2, term.Bold)

	sx := (x0 + x1) / 2
	g.text(sx-5, y0, "     ", pal.Shadow, term.Attr(0))
	g.set(sx-4, y0, 'o', pal.Accent2, term.Bold)
	g.set(sx+4, y0, 'o', pal.Accent2, term.Bold)
	g.line(float64(sx-4), float64(y0), float64(sx+4), float64(y0)+2.2*(1-k), 0, pal.Accent2, term.Bold)
	g.text(sx-6, y0-1, "SWITCH", pal.Dim, term.Attr(0))
	state := "OPEN"
	if k > 0.5 {
		state = "CLOSED"
	}
	g.text(sx+1, y0-1, state, pal.Warn, term.Bold)

	on := math.Abs(math.Sin(t * 7))
	glyph := 'X'
	if k < 0.5 && on < 0.4 {
		glyph = 'x'
	}
	g.text(x1-1, ym-1, ".-.", pal.Warn, term.Bold)
	g.set(x1, ym, glyph, pal.Warn, term.Bold)
	g.text(x1-1, ym+1, "'-'", pal.Warn, term.Bold)
	g.text(x1+2, ym, "LAMP", pal.Dim, term.Attr(0))

	perim := float64(2*(x1-x0) + 2*(y1-y0)*2)
	flow := (1-k)*0.06*math.Sin(t*7) + k*t*0.18
	for i := range 26 {
		u := math.Mod(float64(i)/26+flow+10, 1) * perim
		var x, y float64
		switch {
		case u < float64(x1-x0):
			x, y = float64(x0)+u, float64(y0)
		case u < float64(x1-x0)+float64(y1-y0)*2:
			x, y = float64(x1), float64(y0)+(u-float64(x1-x0))/2
		case u < 2*float64(x1-x0)+float64(y1-y0)*2:
			x, y = float64(x1)-(u-float64(x1-x0)-float64(y1-y0)*2), float64(y1)
		default:
			x, y = float64(x0), float64(y1)-(u-2*float64(x1-x0)-float64(y1-y0)*2)/2
		}
		ch := '*'
		if i%2 == 1 {
			ch = '+'
		}
		g.set(int(math.Round(x)), int(math.Round(y)), ch, pal.Accent, term.Bold)
	}

	volts := (1-k)*3.3*math.Sin(t*7) + k*5.0
	hz := int(50 * (1 - k))
	g.text(x0+3, y1+1, fmt.Sprintf("V=%+.1fV  f=%dHz  I=%.2fA", volts, hz, math.Abs(volts)/12), pal.Accent2, term.Bold)

	sy0 := y1 + 3
	sy1 := r.Bottom() - 2
	if sy1-sy0 >= 3 {
		smid := float64(sy0+sy1) / 2
		for x := x0; x <= x1; x++ {
			g.set(x, int(math.Round(smid)), '-', pal.Faint(), term.Attr(0))
			if (x-x0)%10 == 0 {
				for y := sy0; y <= sy1; y++ {
					g.set(x, y, ':', pal.Faint(), term.Attr(0))
				}
			}
		}
		amp := float64(sy1-sy0) / 2 * 0.85
		val := func(x float64) float64 {
			wave := math.Sin((x-float64(x0))/float64(x1-x0)*math.Pi*6 - t*7)
			return smid - amp*((1-k)*wave+k*0.75)
		}
		for x := x0; x < x1; x++ {
			g.line(float64(x), val(float64(x)), float64(x+1), val(float64(x+1)), 0, pal.Accent, term.Bold)
		}
		g.text(x0, sy0-1, "V(t)", pal.Dim, term.Attr(0))
	}
}

func drawFragmentsArt(ctx *Context, art Rect, t float64) (Rect, []hudRow) {
	pal := ctx.Palette
	s := ctx.Screen
	spread := Progress(ctx.T, stamp("1:58.38"), stamp("2:04.98"))
	cx, cy := float64(art.W)/2, float64(art.H)
	R := math.Min(float64(art.W)*0.20, float64(art.H)*2*0.40)
	hm := heartMask(cx, cy, R)
	gone := func(x, y float64) float64 { return hashUnit(97, int(x)*131+int(y/2)) }
	heart := asciiArt{
		inside: func(x, y float64) bool { return hm(x, y) && gone(x, y) >= spread*0.94 },
		body:   pal.Accent, edge: pal.Err, seed: 137, charset: []rune("01<3"),
	}
	box := heart.draw(ctx, art, t)
	erased, total := 0, 0
	for cyy := art.Y; cyy < art.Bottom(); cyy++ {
		for cxx := art.X; cxx < art.Right(); cxx++ {
			ux, uy := float64(cxx-art.X)+0.5, float64(cyy-art.Y)*2+1
			if !hm(ux, uy) {
				continue
			}
			total++
			h := gone(ux, uy)
			if h >= spread*0.94 {
				continue
			}
			erased++
			age := (spread - h/0.94) * 6
			if age < 0 || age > 2.2 {
				continue
			}
			dx := int(math.Round(3 * math.Sin(age*3+h*40)))
			dy := int(age * 5)
			x, y := cxx+dx, cyy+dy
			if x >= art.X && x < art.Right() && y >= art.Y && y < art.Bottom() {
				s.Set(x, y, pickRune(codeGlyphs, cxx, cyy+int(age*8)), Fade(pal.Err, pal.Shadow, 1-age/2.4), term.ColorDefault, term.Bold)
			}
		}
	}
	names := []string{"aorta", "atrium", "valve", "ventricle", "vena_cava", "pulse", "memory", "promise"}
	for i := range 5 {
		n := int(spread*60) - i
		if n < 0 {
			break
		}
		line := fmt.Sprintf("rm heart/%s_%03d  [DEL]", names[n%len(names)], n)
		s.Text(art.Right()-len(line)-1, art.Bottom()-2-i, line, Fade(pal.Err, pal.Shadow, 1-0.18*float64(i)), term.ColorDefault, term.Attr(0))
	}
	pct := 0
	if total > 0 {
		pct = 100 * erased / total
	}
	return box, []hudRow{{"ERASED", fmt.Sprintf("%d%%", pct)}, {"FRAGS", fmt.Sprintf("%d", erased)}, {"HEART", "NULL"}}
}

func drawJudgeArt(ctx *Context, art Rect, t float64) (Rect, []hudRow) {
	pal := ctx.Palette
	g := stage{ctx, art}
	cx := art.X + art.W/2
	beamY := float64(art.Y + 3)
	bottom := art.Bottom() - 3
	bw := math.Min(float64(art.W)*0.30, 28)
	tilt := math.Sin(t*1.7) * 2.6
	lx, ly := float64(cx)-bw, beamY-tilt
	rx, ry := float64(cx)+bw, beamY+tilt

	for y := int(beamY) + 1; y < bottom; y++ {
		for i, ch := range "[  ]" {
			c := ch
			if i == 1 || i == 2 {
				c = pickRune(codeGlyphs, y+i, int(t*9)+i)
			}
			g.set(cx-2+i, y, c, Blend(pal.Shadow, pal.Accent2, 0.6), term.Bold)
		}
	}
	g.text(cx-8, bottom, "/===============\\", pal.Accent2, term.Bold)
	g.text(cx-12, bottom+1, "/#######################\\", pal.Accent2, term.Bold)

	g.line(lx, ly, rx, ry, '=', pal.Warn, term.Bold)
	g.set(cx, int(beamY), '@', pal.Ink(), term.Bold)
	g.text(cx-1, int(beamY)-1, "/^\\", pal.Warn, term.Bold)

	type pan struct {
		x, y  float64
		label string
		col   term.Color
	}
	pans := []pan{
		{lx, ly + 7, "YOUR GOD", pal.Err},
		{rx, ry + 7, "ILLEGAL ARGUMENTS", pal.Accent},
	}
	for i, p := range pans {
		pw := 9.0
		g.line(p.x, p.y-7+0.5, p.x-pw, p.y, 0, pal.Dim, term.Attr(0))
		g.line(p.x, p.y-7+0.5, p.x+pw, p.y, 0, pal.Dim, term.Attr(0))
		g.text(int(p.x)-int(pw)-1, int(p.y), "\\"+strings.Repeat("_", int(pw)*2-1)+"/", pal.Warn, term.Bold)
		g.text(int(p.x)-len(p.label)/2, int(p.y)-6, p.label, p.col, term.Bold)
		heavy := (i == 0) == (tilt > 0)
		stack := 2
		if heavy {
			stack = 4
		}
		for h := range stack {
			g.text(int(p.x)-3, int(p.y)-1-h, "[####]", p.col, term.Bold)
		}
	}
	verdicts := []string{"GUILTY", "UNDEFINED", "GUILTY", "OVERFLOW"}
	verdict := verdicts[int(t*1.3)%len(verdicts)]
	box := Rect{X: int(lx) - 15, Y: int(beamY) - 2, W: int(2*bw) + 30, H: bottom + 2 - int(beamY) + 2}
	g.text(cx-len(verdict)/2-4, bottom+2, "["+verdict+"]", pal.Ink(), term.Bold)
	return box, []hudRow{{"PLAINT", "self"}, {"DEFEND", "god"}, {"VERDICT", verdict}}
}

type pathPt struct{ x, y float64 }

func pathPoint(path []pathPt, u float64) pathPt {
	total := 0.0
	seg := make([]float64, len(path)-1)
	for i := range seg {
		seg[i] = math.Hypot(path[i+1].x-path[i].x, (path[i+1].y-path[i].y)*2)
		total += seg[i]
	}
	d := math.Mod(u+8, 1) * total
	for i, l := range seg {
		if d <= l {
			f := d / l
			return pathPt{path[i].x + (path[i+1].x-path[i].x)*f, path[i].y + (path[i+1].y-path[i].y)*f}
		}
		d -= l
	}
	return path[len(path)-1]
}

func gauge(value, lo, hi float64, width int) string {
	pos := int(math.Round((value - lo) / (hi - lo) * float64(width-1)))
	pos = min(max(pos, 0), width-1)
	b := []byte(strings.Repeat(".", width))
	b[width/2] = ':'
	b[pos] = '|'
	return "[" + string(b) + "]"
}

// drawCircuitRich is a working schematic: a spinning generator feeds a knife
// switch that throws between a direct AC line and a DC branch made of a rectifier
// bridge and a smoothing capacitor, with a phasor, meters, a spectrum and an
// oscilloscope showing the change.
func drawCircuitRich(ctx *Context, r Rect, regs []regEntry) {
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(regs[0].at).Seconds()
	switchAt := regs[1].at
	k := ease(Progress(ctx.T, switchAt, switchAt+450*time.Millisecond))
	phase := t * 7
	acLive := k < 0.5
	groupW := 58.0
	switch {
	case r.W >= 104:
		groupW = 103
	case r.W >= 80:
		groupW = 80
	}
	ox, oy := float64(r.X)+math.Max(2, math.Floor((float64(r.W)-groupW)/2)), float64(r.Y+1)
	wire := func(x0, y0, x1, y1 float64, active bool) {
		col, attr := pal.Faint(), term.Attr(0)
		if active {
			col, attr = pal.Accent, term.Bold
		}
		g.line(ox+x0, oy+y0, ox+x1, oy+y1, 0, col, attr)
	}
	text := func(x, y float64, s string, col term.Color, attr term.Attr) {
		g.text(int(ox+x), int(oy+y), s, col, attr)
	}

	gcx, gcy, grx, gry := 5.5, 5.0, 5.5, 2.6
	for a := 0.0; a < 2*math.Pi; a += 0.07 {
		g.set(int(math.Round(ox+gcx+grx*math.Cos(a))), int(math.Round(oy+gcy+gry*math.Sin(a))),
			strokeGlyph(-grx*math.Sin(a), gry*math.Cos(a)), pal.Accent2, term.Bold)
	}
	for i := range 3 {
		a := phase*0.6 + float64(i)*2*math.Pi/3
		g.line(ox+gcx, oy+gcy, ox+gcx+grx*0.82*math.Cos(a), oy+gcy+gry*0.82*math.Sin(a), 0, pal.Accent, term.Bold)
	}
	g.set(int(ox+gcx), int(oy+gcy), '@', pal.Ink(), term.Bold)
	text(0, 9.2, "AC GEN 50Hz", pal.Dim, term.Attr(0))
	wire(11, 5, 16, 5, true)

	g.set(int(ox+16), int(oy+5), 'O', pal.Ink(), term.Bold)
	g.set(int(ox+24), int(oy+2), 'o', pal.Accent2, term.Bold)
	g.set(int(ox+24), int(oy+8), 'o', pal.Accent2, term.Bold)
	tipY := 2 + 6*k
	g.line(ox+16, oy+5, ox+24, oy+tipY, 0, pal.Ink(), term.Bold)
	state := "AC"
	if !acLive {
		state = "DC"
	}
	text(15, 9.2, "SPDT ["+state+"]", pal.Warn, term.Bold)

	wire(25, 2, 52, 2, acLive)
	text(29, 0.4, "AC LINE  ~ ~ ~", pal.Dim, term.Attr(0))
	wire(25, 8, 27, 8, !acLive)
	boxCol := pal.Faint()
	if !acLive {
		boxCol = pal.Accent2
	}
	text(27, 7, "+-[ BRIDGE ]--+", boxCol, term.Bold)
	text(27, 8, "| >| |< >| |< |", boxCol, term.Bold)
	text(27, 9, "+-------------+", boxCol, term.Bold)
	wire(42, 8, 44, 8, !acLive)
	text(44, 8, "-||-", boxCol, term.Bold)
	wire(48, 8, 52, 8, !acLive)
	charge := 0.0
	if !acLive {
		charge = k * (0.86 + 0.06*(1-math.Abs(math.Sin(phase))))
	}
	filled := int(charge * 8)
	text(43, 6.4, "C 470uF", pal.Dim, term.Attr(0))
	text(43, 9.2, "["+strings.Repeat("#", filled)+strings.Repeat(".", 8-filled)+"]", pal.Accent, term.Bold)

	wire(52, 2, 52, 3.6, acLive)
	wire(52, 6.4, 52, 8, !acLive)
	bright := math.Abs(math.Sin(phase))
	lamp := 'X'
	switch {
	case acLive && bright < 0.25:
		lamp = '.'
	case acLive && bright < 0.55:
		lamp = 'x'
	}
	text(51, 4, ".-.", pal.Warn, term.Bold)
	text(51, 6, "'-'", pal.Warn, term.Bold)
	g.text(int(ox+51), int(oy+5), "(", pal.Warn, term.Bold)
	g.set(int(ox+52), int(oy+5), lamp, pal.Warn, term.Bold)
	g.text(int(ox+53), int(oy+5), ")", pal.Warn, term.Bold)
	text(55, 4, "LOAD", pal.Dim, term.Attr(0))
	text(55, 5, "12V", pal.Dim, term.Attr(0))
	if lamp == 'X' {
		for _, o := range [][2]float64{{-3, 3}, {3, 3}, {0, 3.6}, {-2.5, 6.6}, {2.5, 6.6}} {
			text(51+o[0]+1, o[1]+0.4, "*", pal.Warn, term.Bold)
		}
	}

	var path []pathPt
	if acLive {
		path = []pathPt{{11, 5}, {16, 5}, {24, 2}, {52, 2}, {52, 3.6}}
	} else {
		path = []pathPt{{11, 5}, {16, 5}, {24, 8}, {52, 8}, {52, 6.4}}
	}
	for i := range 18 {
		u := float64(i) / 18
		if acLive {
			u += 0.035 * math.Sin(phase)
		} else {
			u += t * 0.22
		}
		p := pathPoint(path, u)
		ch := '*'
		if i%2 == 1 {
			ch = '+'
		}
		g.set(int(math.Round(ox+p.x)), int(math.Round(oy+p.y)), ch, pal.Accent, term.Bold)
	}

	age := ctx.Since(switchAt).Seconds()
	if ctx.T >= switchAt && age < 0.8 {
		tx, ty := ox+24, oy+tipY
		for i := range 10 {
			h := hashUnit(ctx.Seed, i*17+int(age*30))
			a := h * 2 * math.Pi
			d := age * 16 * (0.4 + hashUnit(ctx.Seed, i))
			g.set(int(math.Round(tx+d*math.Cos(a))), int(math.Round(ty+d*0.5*math.Sin(a))), pickRune([]rune("*+.x"), i, int(age*20)), pal.Warn, term.Bold)
		}
		if k < 0.98 {
			for s := 0.0; s <= 1; s += 0.12 {
				g.set(int(math.Round(tx)), int(math.Round(ty+(oy+8-ty)*s)), '~', pal.Ink(), term.Bold)
			}
		}
	}

	if r.W >= 80 {
		pcx, pcy, prx, pry := ox+69, oy+5, 8.5, 4.2
		text(58, 0.2, "PHASOR  V = sin(wt)", pal.Dim, term.Attr(0))
		for a := 0.0; a < 2*math.Pi; a += 0.06 {
			g.set(int(math.Round(pcx+prx*math.Cos(a))), int(math.Round(pcy+pry*math.Sin(a))), strokeGlyph(-prx*math.Sin(a), pry*math.Cos(a)), pal.Faint(), term.Attr(0))
		}
		for x := -prx; x <= prx; x += 2 {
			g.set(int(math.Round(pcx+x)), int(math.Round(pcy)), '.', pal.Faint(), term.Attr(0))
		}
		for y := -pry; y <= pry; y += 1 {
			g.set(int(math.Round(pcx)), int(math.Round(pcy+y)), '.', pal.Faint(), term.Attr(0))
		}
		ang := -(1 - k) * phase
		tipx, tipy := pcx+prx*0.92*math.Cos(ang), pcy+pry*0.92*math.Sin(ang)
		g.line(pcx, pcy, tipx, tipy, 0, pal.Accent, term.Bold)
		g.set(int(math.Round(tipx)), int(math.Round(tipy)), 'O', pal.Warn, term.Bold)
		for x := tipx + 1; x < pcx+prx+8; x += 2 {
			g.set(int(math.Round(x)), int(math.Round(tipy)), '.', pal.Accent2, term.Attr(0))
		}
		value := (1-k)*math.Sin(phase) + k
		note := fmt.Sprintf("sin=%+.2f", value)
		if k > 0.5 {
			note = "phase locked"
		}
		g.text(int(pcx)-5, int(oy)+9, note, pal.Warn, term.Bold)
	}

	if r.W >= 104 {
		mx := 80.0
		volts := (1-k)*3.3*math.Sin(phase) + k*5
		amps := volts / 12
		watts := volts * amps
		text(mx, 0.2, "METERS", pal.Dim, term.Attr(0))
		text(mx, 1.2, "V "+gauge(volts, -5, 5, 13)+fmt.Sprintf(" %+.1fV", volts), pal.Accent2, term.Bold)
		text(mx, 2.2, "I "+gauge(amps, -0.5, 0.5, 13)+fmt.Sprintf(" %+.2fA", amps), pal.Accent2, term.Bold)
		text(mx, 3.2, "P "+gauge(watts, 0, 2.2, 13)+fmt.Sprintf(" %.2fW", watts), pal.Accent2, term.Bold)
		text(mx, 4.6, "SPECTRUM", pal.Dim, term.Attr(0))
		for bin := range 10 {
			h := 0.05 + 0.03*math.Abs(math.Sin(phase*0.7+float64(bin)))
			switch bin {
			case 0:
				h += 0.95 * k
			case 1:
				h += 0.95 * (1 - k)
			case 3:
				h += 0.25 * (1 - k)
			case 2:
				h += 0.18 * k
			}
			rows := int(math.Min(h, 1) * 4)
			for row := range 4 {
				ch := ' '
				if row < rows {
					ch = '#'
				}
				text(mx+float64(bin*2), 8.4-float64(row), string(ch), pal.Accent, term.Bold)
			}
		}
		text(mx, 9.4, "0  50 100 150 Hz", pal.Faint(), term.Attr(0))
	}

	sy0, sy1 := int(oy)+12, r.Bottom()-2
	x0, x1 := r.X+2, r.Right()-3
	if sy1-sy0 < 3 {
		return
	}
	mid := float64(sy0+sy1) / 2
	amp := float64(sy1-sy0) / 2 * 0.86
	window := 3.4
	for x := x0; x <= x1; x++ {
		g.set(x, int(math.Round(mid)), '-', pal.Faint(), term.Attr(0))
		if (x-x0)%12 == 0 {
			for y := sy0; y <= sy1; y++ {
				g.set(x, y, ':', pal.Faint(), term.Attr(0))
			}
		}
	}
	g.text(x0, sy0-1, "OSCILLOSCOPE", pal.Dim, term.Attr(0))
	g.text(x0+14, sy0-1, "VIN", pal.Faint(), term.Bold)
	g.text(x0+19, sy0-1, "VRECT", pal.Accent2, term.Bold)
	g.text(x0+26, sy0-1, "VOUT", pal.Accent, term.Bold)
	sample := func(x int) (vin, vrect, vout, kx float64) {
		tabs := ctx.T - time.Duration(window*float64(x1-x)/float64(x1-x0)*float64(time.Second))
		kx = ease(Progress(tabs, switchAt, switchAt+450*time.Millisecond))
		tt := (tabs - regs[0].at).Seconds()
		vin = math.Sin(tt * 7)
		vrect = math.Abs(vin)
		vout = (1-kx)*vin + kx*(0.72+0.05*(1-vrect))
		return
	}
	var lx, ly float64
	for x := x0; x <= x1; x++ {
		vin, vrect, vout, kx := sample(x)
		g.set(x, int(math.Round(mid-amp*vin)), '.', pal.Faint(), term.Attr(0))
		if kx > 0.05 {
			g.set(x, int(math.Round(mid-amp*vrect*kx)), ':', pal.Accent2, term.Attr(0))
		}
		y := mid - amp*vout
		if x > x0 {
			g.line(lx, ly, float64(x), y, 0, pal.Accent, term.Bold)
		}
		lx, ly = float64(x), y
	}
}

func glyphFeature(ch rune, cx, cy float64, col term.Color) asciiFeature {
	rows := term.Glyph(ch)
	h := len(rows)
	w := len(rows[0])
	return func(x, y float64) (rune, term.Color, bool) {
		gx := int(math.Floor(x - (cx - float64(w)/2)))
		gy := int(math.Floor((y-cy)/2 + float64(h)/2))
		if gx >= 0 && gx < w && gy >= 0 && gy < h && rows[gy][gx] == '#' {
			return '#', col, true
		}
		return 0, 0, false
	}
}

func drawClockRich(ctx *Context, r Rect, active regEntry) {
	if r.W >= 150 && r.H >= 28 {
		drawClockWide(ctx, r, active)
		return
	}
	if r.W < 60 || r.H < 12 {
		return
	}
	g := stage{ctx, r}
	pal := ctx.Palette
	s := ctx.Screen
	t := ctx.Since(active.at).Seconds()
	sim := 12*3600.0 - 1320 + t*1000
	h24 := math.Mod(sim/3600, 24)
	hour12 := math.Mod(h24, 12)
	minute := math.Mod(sim/60, 60)
	second := math.Mod(sim, 60)
	pm := h24 >= 12

	ry := math.Min(float64(r.H-4)/2-0.5, 7.5)
	rx := 2 * ry
	const readoutW = 56.0
	left := float64(r.X) + math.Max(float64(r.W)-(2*rx+1+6+readoutW), 4)/2
	cx, cy := left+rx, float64(r.Y)+float64(r.H-2)/2
	for a := 0.0; a < 2*math.Pi; a += 0.05 {
		g.set(int(math.Round(cx+rx*math.Cos(a))), int(math.Round(cy+ry*math.Sin(a))), strokeGlyph(-rx*math.Sin(a), ry*math.Cos(a)), pal.Accent2, term.Bold)
	}
	for m := range 60 {
		a := float64(m)/60*2*math.Pi - math.Pi/2
		ch := '.'
		if m%5 == 0 {
			ch = '+'
		}
		g.set(int(math.Round(cx+rx*0.91*math.Cos(a))), int(math.Round(cy+ry*0.91*math.Sin(a))), ch, pal.Dim, term.Attr(0))
	}
	for h := 1; h <= 12; h++ {
		a := float64(h)/12*2*math.Pi - math.Pi/2
		label := fmt.Sprintf("%d", h)
		g.text(int(math.Round(cx+rx*0.74*math.Cos(a)))-len(label)/2, int(math.Round(cy+ry*0.74*math.Sin(a))), label, pal.Faint(), term.Bold)
	}
	hand := func(frac, length float64, col term.Color) {
		a := frac*2*math.Pi - math.Pi/2
		g.line(cx, cy, cx+rx*length*math.Cos(a), cy+ry*length*math.Sin(a), 0, col, term.Bold)
	}
	hand(hour12/12, 0.48, pal.Accent)
	hand(minute/60, 0.70, pal.Ink())
	hand(second/60, 0.84, pal.Warn)
	g.set(int(cx), int(math.Round(cy)), '@', pal.Ink(), term.Bold)

	dx := int(left+2*rx) + 7
	h12 := int(hour12)
	if h12 == 0 {
		h12 = 12
	}
	digits := fmt.Sprintf("%2d:%02d", h12, int(minute))
	term.DrawGlyphs(s, dx, r.Y+1, digits, pal.Accent, term.ColorDefault, term.Attr(0), nil)
	suffix := "AM"
	if pm {
		suffix = "PM"
	}
	flip := math.Abs(t-1.32) < 0.25
	col := pal.Warn
	if flip && int(t*30)%2 == 0 {
		col = pal.Ink()
	}
	term.DrawGlyphs(s, dx+31, r.Y+1, suffix, col, term.ColorDefault, term.Attr(0), nil)
	g.text(dx, r.Y+9, fmt.Sprintf(":%02d.%02d   %s -> %s", int(second), int(math.Mod(sim*100, 100)), active.from, active.to), pal.Dim, term.Attr(0))

	sun := []string{`  \ | /  `, ` -- ( ) --`, `  / | \  `}
	moon := []string{`   _..   `, `  (  ,)  `, `   '-'   `}
	icon, iconCol := sun, pal.Warn
	if pm {
		icon, iconCol = moon, pal.Accent2
	}
	for i, line := range icon {
		g.text(dx+46, r.Y+2+i, line, iconCol, term.Bold)
	}
	g.text(dx+46, r.Y+6, map[bool]string{false: "DAYLIGHT", true: "EVENING "}[pm], pal.Dim, term.Attr(0))

	bcd := []int{h12 / 10, h12 % 10, int(minute) / 10, int(minute) % 10, int(second) / 10, int(second) % 10}
	g.text(dx, r.Y+11, "BCD  h h : m m : s s", pal.Faint(), term.Attr(0))
	for row, weight := range []int{8, 4, 2, 1} {
		g.text(dx-3, r.Y+12+row, fmt.Sprintf("%d", weight), pal.Faint(), term.Attr(0))
		x := dx + 5
		for i, d := range bcd {
			ch, c := 'o', pal.Accent
			if d&weight == 0 {
				ch, c = '.', pal.Faint()
			}
			g.set(x, r.Y+12+row, ch, c, term.Bold)
			x += 2
			if i == 1 || i == 3 {
				x += 2
			}
		}
	}

	ty := r.Bottom() - 2
	x0, x1 := r.X+2, r.Right()-3
	for x := x0; x <= x1; x++ {
		hh := float64(x-x0) / float64(x1-x0) * 24
		ch := '='
		if hh < 6 || hh >= 18 {
			ch = '.'
		}
		g.set(x, ty, ch, pal.Dim, term.Attr(0))
	}
	for _, h := range []int{0, 6, 12, 18, 24} {
		x := x0 + int(float64(x1-x0)*float64(h)/24)
		g.set(x, ty, '+', pal.Ink(), term.Bold)
		g.text(x-1, ty+1, fmt.Sprintf("%02d", h), pal.Faint(), term.Attr(0))
	}
	g.set(x0+int(float64(x1-x0)*h24/24), ty-1, 'v', pal.Warn, term.Bold)
}

func drawGenderRich(ctx *Context, r Rect, active regEntry) {
	if r.W >= 150 && r.H >= 26 {
		drawGenderWide(ctx, r, active)
		return
	}
	if r.W < 60 || r.H < 12 {
		return
	}
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	k := ease(Progress(ctx.T, stamp("1:31.62"), stamp("1:31.92")))
	R := math.Min(float64(r.W)*0.10, float64(r.H)*2*0.21)
	const panelW = 44.0
	left := math.Max(float64(r.W)-(2.8*R+10+panelW), 4) / 2
	cxc, cyc := left+1.4*R, float64(r.H)-0.5*R
	body := Blend(pal.Err, pal.Accent2, k)
	disk := asciiArt{inside: ellipseMask(cxc, cyc, R, R), body: body, edge: body, seed: 149, charset: []rune("01XYxy")}
	disk.feature = glyphFeature([]rune("FM")[int(k+0.5)], cxc, cyc, pal.Ink())
	disk.draw(ctx, r, t)

	cx, cy := float64(r.X)+cxc, float64(r.Y)+cyc/2
	theta := math.Pi/2 - k*3*math.Pi/4
	dirx, diry := math.Cos(theta), math.Sin(theta)
	tip := func(d float64) (float64, float64) { return cx + dirx*d, cy + diry*d/2 }
	stem := R * 1.55
	x0, y0 := tip(R * 1.02)
	x1, y1 := tip(R + stem*0.62)
	g.line(x0, y0, x1, y1, 0, body, term.Bold)
	g.line(x0+1, y0, x1+1, y1, 0, body, term.Bold)
	bx, by := tip(R + stem*0.30)
	half := (1 - k) * R * 0.42
	px, py := -diry, dirx
	g.line(bx-px*half, by-py*half/2, bx+px*half, by+py*half/2, 0, body, term.Bold)
	ax, ay := tip(R + stem*0.62)
	head := k * R * 0.5
	for _, sgn := range []float64{-1, 1} {
		a := theta + math.Pi + sgn*0.6
		g.line(ax, ay, ax+head*math.Cos(a), ay+head*math.Sin(a)/2, 0, body, term.Bold)
	}
	g.text(int(cx)-6, r.Bottom()-1, fmt.Sprintf("[ %s -> %s ]", active.from, active.to), pal.Ink(), term.Bold)

	x := r.X + int(left+2.8*R+10)
	y := r.Y + max((r.H-13)/2, 1)
	g.text(x, y, "GENOME EDIT", pal.Dim, term.Bold)
	chrom := "XX"
	if k > 0.5 {
		chrom = "XY"
	}
	if k > 0.02 && k < 0.98 {
		chrom = string(pickRune([]rune("XY01"), int(t*40), 1)) + string(pickRune([]rune("XY01"), int(t*40), 2))
	}
	g.text(x, y+2, "KARYOTYPE  "+chrom, pal.Accent, term.Bold)
	gene := "SRY   OFF"
	if k > 0.5 {
		gene = "SRY   ON "
	}
	g.text(x, y+3, "GENE       "+gene, pal.Warn, term.Bold)
	bar := func(label string, v float64, col term.Color, row int) {
		n := int(v * 20)
		g.text(x, y+row, fmt.Sprintf("%-13s[%s%s]%3d%%", label, strings.Repeat("#", n), strings.Repeat(".", 20-n), int(v*100)), col, term.Bold)
	}
	bar("ESTROGEN", 0.85*(1-k)+0.10, pal.Err, 5)
	bar("TESTOSTERONE", 0.10+0.80*k, pal.Accent2, 6)
	bar("SWITCH", k, pal.Warn, 7)
	for i := range 4 {
		n := int(t*12) - i
		g.text(x, y+9+i, fmt.Sprintf("0x%04X  %s", uint16(hashSeed(ctx.Seed, n)&0xffff), [...]string{"transcribe", "splice", "express", "fold"}[(n+4)%4]), Fade(pal.Dim, pal.Shadow, 1-0.2*float64(i)), term.Attr(0))
	}
}

func drawRoleRich(ctx *Context, r Rect, active regEntry) {
	if r.W >= 150 && r.H >= 26 {
		drawRoleWide(ctx, r, active)
		return
	}
	if r.W < 60 || r.H < 12 {
		return
	}
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	k := ease(Progress(ctx.T, stamp("1:38.94"), stamp("1:39.33")))
	cx, cy := float64(r.W)/2, float64(r.H)
	Ro := math.Min(float64(r.W)*0.24, float64(r.H)*2*0.30)
	a := math.Pi*k + 0.15*math.Sin(t)
	Rd := math.Min(Ro*0.42, float64(r.H)*2*0.24)
	pos := func(off float64) (float64, float64) {
		return cx + Ro*math.Cos(a+off), cy + Ro*math.Sin(a+off)
	}
	ax, ay := pos(0)
	bx, by := pos(math.Pi)
	abs := func(ux, uy float64) (float64, float64) { return float64(r.X) + ux, float64(r.Y) + uy/2 }

	for i := 0.0; i < 2*math.Pi; i += 0.045 {
		x, y := abs(cx+Ro*math.Cos(i), cy+Ro*math.Sin(i))
		g.set(int(math.Round(x)), int(math.Round(y)), '.', pal.Faint(), term.Attr(0))
	}
	for i := 1; i <= 6; i++ {
		back := a - float64(i)*0.12*(1-k*0.5)
		for _, off := range []float64{0, math.Pi} {
			x, y := abs(cx+Ro*math.Cos(back+off), cy+Ro*math.Sin(back+off))
			g.set(int(math.Round(x)), int(math.Round(y)), '*', Fade(pal.Dim, pal.Shadow, 1-float64(i)/7), term.Bold)
		}
	}
	x0, y0 := abs(ax, ay)
	x1, y1 := abs(bx, by)
	g.line(x0, y0, x1, y1, '=', pal.Warn, term.Bold)
	g.text(int((x0+x1)/2)-6, int((y0+y1)/2)-1, fmt.Sprintf("tension %.2f", 0.5+0.5*math.Sin(t*3)), pal.Dim, term.Attr(0))

	letters := [2]rune{'S', 'M'}
	cols := [2]term.Color{pal.Accent, pal.Accent2}
	flip := k > 0.5
	for i, p := range [][2]float64{{ax, ay}, {bx, by}} {
		idx := i
		if flip {
			idx = 1 - i
		}
		disc := asciiArt{inside: ellipseMask(p[0], p[1], Rd, Rd), body: cols[idx], edge: cols[idx], seed: int64(151 + idx), charset: []rune("01")}
		disc.feature = glyphFeature(letters[idx], p[0], p[1], pal.Ink())
		disc.draw(ctx, r, t)
	}
	roleA, roleB := active.from, active.to
	if flip {
		roleA, roleB = active.to, active.from
	}
	for i, line := range []struct {
		text string
		col  term.Color
	}{
		{fmt.Sprintf("role_a = %s   role_b = %s", roleA, roleB), pal.Warn},
		{fmt.Sprintf("swap   [%s%s] %3d%%", strings.Repeat("#", int(k*16)), strings.Repeat(".", 16-int(k*16)), int(k*100)), pal.Accent},
		{fmt.Sprintf("angle  %03d deg   omega %.2f", int(a*180/math.Pi)%360, 1+2*(1-k)), pal.Dim},
	} {
		g.text(r.X+(r.W-len(line.text))/2, r.Y+i, line.text, line.col, term.Bold)
	}
}

func drawTranceRich(ctx *Context, r Rect, active regEntry) {
	if r.W < 50 || r.H < 10 {
		return
	}
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	cx, cy := float64(r.X)+float64(r.W)/2, float64(r.Y)+float64(r.H)/2
	for ring := 0; ring < 9; ring++ {
		f := math.Mod(float64(ring)/9+t*0.35, 1)
		rx := f * float64(r.W) * 0.48
		ry := rx / 2
		for a := 0.0; a < 2*math.Pi; a += 0.09 {
			g.set(int(math.Round(cx+rx*math.Cos(a))), int(math.Round(cy+ry*math.Sin(a))), '.', Fade(pal.Accent2, pal.Shadow, (1-f)*0.8+0.1), term.Attr(0))
		}
	}
	amp := float64(r.H) * 0.36
	pairs := []string{"AT", "TA", "GC", "CG"}
	for x := r.X + 1; x < r.Right()-1; x++ {
		u := float64(x-r.X)*0.22 + t*2.4
		y1 := cy + amp*math.Sin(u)
		y2 := cy - amp*math.Sin(u)
		depth := math.Cos(u)
		front, back := pal.Accent, pal.Faint()
		if depth < 0 {
			front, back = back, front
		}
		if (x-r.X)%3 == 0 {
			steps := int(math.Abs(y2-y1)) + 1
			p := pairs[(x/3)%4]
			for i := 0; i <= steps; i++ {
				f := float64(i) / float64(steps)
				ch := '-'
				if f < 0.12 {
					ch = rune(p[0])
				} else if f > 0.88 {
					ch = rune(p[1])
				}
				if steps > 3 && i > 0 && i < steps {
					ch = ':'
				}
				g.set(x, int(math.Round(y1+(y2-y1)*f)), ch, pal.Dim, term.Attr(0))
			}
		}
		g.set(x, int(math.Round(y2)), 'o', back, term.Attr(0))
		g.set(x, int(math.Round(y1)), '@', front, term.Bold)
	}
	label := "[ ENTER THE TRANCE ]"
	g.text(int(cx)-len(label)/2, int(cy), label, pal.Ink(), term.Bold)
	g.text(r.X+2, r.Y, fmt.Sprintf("theta=%03d  depth=%.2f  loops=%d", int(t*90)%360, 0.5+0.5*math.Sin(t*0.8), int(t*2)), pal.Dim, term.Attr(0))
}

// bigText draws text in the block font with each pixel stretched to sx columns
// by sy rows.
func bigText(g stage, x, y int, text string, sx, sy int, col term.Color) {
	for _, ch := range text {
		rows := term.Glyph(ch)
		for gy, row := range rows {
			for gx, px := range row {
				if px != '#' {
					continue
				}
				for dy := range sy {
					for dx := range sx {
						g.set(x+gx*sx+dx, y+gy*sy+dy, '#', col, term.Bold)
					}
				}
			}
		}
		x += (len(rows[0]) + 1) * sx
	}
}

var worldClocks = []struct {
	name   string
	offset float64
}{{"TOKYO", 9}, {"LONDON", 0}, {"NEW YORK", -5}, {"SYDNEY", 11}}

func drawClockWide(ctx *Context, r Rect, active regEntry) {
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	sim := 12*3600.0 - 1320 + t*1000
	h24 := math.Mod(sim/3600, 24)
	hour12 := math.Mod(h24, 12)
	minute := math.Mod(sim/60, 60)
	second := math.Mod(sim, 60)
	pm := h24 >= 12

	ry := math.Min(float64(r.H-5)/2-0.5, 15)
	rx := 2 * ry
	const digitsW, zoneC, gap = 60.0, 36.0, 9.0
	left := float64(r.X) + math.Max(float64(r.W)-(2*rx+1+gap+digitsW+gap+zoneC), 6)/2
	cx, cy := left+rx, float64(r.Y)+float64(r.H-3)/2
	for a := 0.0; a < 2*math.Pi; a += 0.03 {
		g.set(int(math.Round(cx+rx*math.Cos(a))), int(math.Round(cy+ry*math.Sin(a))), strokeGlyph(-rx*math.Sin(a), ry*math.Cos(a)), pal.Accent2, term.Bold)
		g.set(int(math.Round(cx+rx*1.04*math.Cos(a))), int(math.Round(cy+ry*1.04*math.Sin(a))), '.', pal.Faint(), term.Attr(0))
	}
	for m := range 60 {
		a := float64(m)/60*2*math.Pi - math.Pi/2
		ch := '.'
		if m%5 == 0 {
			ch = '+'
		}
		g.set(int(math.Round(cx+rx*0.92*math.Cos(a))), int(math.Round(cy+ry*0.92*math.Sin(a))), ch, pal.Dim, term.Attr(0))
	}
	for h := 1; h <= 12; h++ {
		a := float64(h)/12*2*math.Pi - math.Pi/2
		label := fmt.Sprintf("%d", h)
		g.text(int(math.Round(cx+rx*0.78*math.Cos(a)))-len(label)/2, int(math.Round(cy+ry*0.78*math.Sin(a))), label, pal.Ink(), term.Bold)
	}
	hand := func(frac, length float64, col term.Color, thick bool) {
		a := frac*2*math.Pi - math.Pi/2
		g.line(cx, cy, cx+rx*length*math.Cos(a), cy+ry*length*math.Sin(a), 0, col, term.Bold)
		if thick {
			g.line(cx+1, cy, cx+1+rx*length*math.Cos(a), cy+ry*length*math.Sin(a), 0, col, term.Bold)
		}
	}
	hand(hour12/12, 0.50, pal.Accent, true)
	hand(minute/60, 0.72, pal.Ink(), true)
	hand(second/60, 0.86, pal.Warn, false)
	g.set(int(cx), int(math.Round(cy)), '@', pal.Ink(), term.Bold)
	g.text(int(cx)-3, int(cy+ry*0.4), "MILI", pal.Faint(), term.Bold)

	bx := int(left + 2*rx + 1 + gap)
	top := int(cy) - 10
	h12 := int(hour12)
	if h12 == 0 {
		h12 = 12
	}
	bigText(g, bx, top, fmt.Sprintf("%2d:%02d", h12, int(minute)), 2, 1, pal.Accent)
	g.text(bx, top+8, fmt.Sprintf(":%02d.%02d   %s -> %s", int(second), int(math.Mod(sim*100, 100)), active.from, active.to), pal.Dim, term.Attr(0))
	suffix := "AM"
	if pm {
		suffix = "PM"
	}
	col := pal.Warn
	if math.Abs(t-1.32) < 0.25 && int(t*30)%2 == 0 {
		col = pal.Ink()
	}
	bigText(g, bx, top+10, suffix, 2, 1, col)
	bcd := []int{h12 / 10, h12 % 10, int(minute) / 10, int(minute) % 10, int(second) / 10, int(second) % 10}
	g.text(bx+30, top+10, "BCD   h h : m m : s s", pal.Faint(), term.Attr(0))
	for row, weight := range []int{8, 4, 2, 1} {
		g.text(bx+30, top+12+row, fmt.Sprintf("%d", weight), pal.Faint(), term.Attr(0))
		x := bx + 36
		for i, d := range bcd {
			ch, c := 'O', pal.Accent
			if d&weight == 0 {
				ch, c = '.', pal.Faint()
			}
			g.set(x, top+12+row, ch, c, term.Bold)
			x += 2
			if i == 1 || i == 3 {
				x += 2
			}
		}
	}
	g.text(bx, top+18, "WORLD CLOCK", pal.Dim, term.Bold)
	for i, city := range worldClocks {
		local := math.Mod(h24+city.offset-8+48, 24)
		lh := int(local)
		day := ""
		for b := range 12 {
			hh := float64(b) * 2
			if hh >= 6 && hh < 18 {
				day += "#"
			} else {
				day += "."
			}
		}
		marker := int(local / 2)
		bar := []byte(day)
		bar[min(marker, 11)] = '|'
		g.text(bx, top+19+i, fmt.Sprintf("%-9s %02d:%02d  [%s]", city.name, lh, int(math.Mod(minute, 60)), string(bar)), pal.Accent2, term.Bold)
	}

	x0 := int(left + 2*rx + 1 + gap + digitsW + gap)
	yh := int(cy) - 3
	acx, arx, ary := float64(x0)+zoneC/2, zoneC/2-1, 7.0
	for a := math.Pi; a <= 2*math.Pi+0.01; a += 0.04 {
		g.set(int(math.Round(acx+arx*math.Cos(a))), int(math.Round(float64(yh)+ary*math.Sin(a))), '.', pal.Dim, term.Attr(0))
	}
	for x := x0; x < x0+int(zoneC); x++ {
		g.set(x, yh, '=', pal.Faint(), term.Attr(0))
	}
	var sx2, sy2 float64
	glyph, gcol := "(O)", pal.Warn
	if h24 >= 6 && h24 < 18 {
		p := (h24 - 6) / 12
		sx2, sy2 = acx-arx*math.Cos(p*math.Pi), float64(yh)+ary*math.Sin(-p*math.Pi)
	} else {
		q := math.Mod(h24-18+24, 24) / 12
		sx2, sy2 = acx-arx*math.Cos(q*math.Pi), float64(yh)+ary*0.7*math.Sin(q*math.Pi)
		glyph, gcol = "( )", pal.Accent2
	}
	g.text(int(sx2)-1, int(math.Round(sy2)), glyph, gcol, term.Bold)
	g.text(x0, yh+ary2(), "SUNRISE 06:00  SUNSET 18:00", pal.Dim, term.Attr(0))
	g.text(x0, yh+ary2()+2, fmt.Sprintf("EPOCH  %d", 1735689600+int(sim)), pal.Accent, term.Bold)
	g.text(x0, yh+ary2()+3, fmt.Sprintf("JULIAN %.4f", 2460676.5+sim/86400), pal.Dim, term.Attr(0))
	g.text(x0, yh+ary2()+4, "TZ     UTC+08:00", pal.Dim, term.Attr(0))
	g.text(x0, int(cy)-12, "SOLAR TRACK", pal.Dim, term.Bold)

	ty := r.Bottom() - 2
	x1 := r.X + 3
	x2 := r.Right() - 4
	for x := x1; x <= x2; x++ {
		hh := float64(x-x1) / float64(x2-x1) * 24
		ch := '='
		if hh < 6 || hh >= 18 {
			ch = '.'
		}
		g.set(x, ty, ch, pal.Dim, term.Attr(0))
	}
	for _, h := range []int{0, 6, 12, 18, 24} {
		x := x1 + int(float64(x2-x1)*float64(h)/24)
		g.set(x, ty, '+', pal.Ink(), term.Bold)
		g.text(x-1, ty+1, fmt.Sprintf("%02d", h), pal.Faint(), term.Attr(0))
	}
	g.set(x1+int(float64(x2-x1)*h24/24), ty-1, 'v', pal.Warn, term.Bold)
}

func ary2() int { return 3 }

func chromosome(g stage, cx, cy float64, kind rune, k float64, col term.Color, label string) {
	up := 4.5
	spread := 7.0
	lowSpread := spread
	upSpread := spread
	if kind == 'y' {
		lowSpread = spread * (1 - k)
		upSpread = spread * (1 - 0.35*k)
	}
	arms := [][4]float64{
		{cx - upSpread, cy - up, cx, cy},
		{cx + upSpread, cy - up, cx, cy},
		{cx, cy, cx - lowSpread, cy + up},
		{cx, cy, cx + lowSpread, cy + up},
	}
	for i, a := range arms {
		g.line(a[0], a[1], a[2], a[3], 0, col, term.Bold)
		g.line(a[0]+1, a[1], a[2]+1, a[3], 0, col, term.Bold)
		for b := 0.2; b < 0.9; b += 0.25 {
			g.set(int(math.Round(a[0]+(a[2]-a[0])*b))+2, int(math.Round(a[1]+(a[3]-a[1])*b)), '=', pal2(i), term.Bold)
		}
	}
	g.set(int(cx), int(math.Round(cy)), '@', col, term.Bold)
	g.text(int(cx)-len(label)/2, int(cy+up)+2, label, col, term.Bold)
}

func pal2(i int) term.Color { return term.Hex(0xffffff) }

func drawGenderWide(ctx *Context, r Rect, active regEntry) {
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	k := ease(Progress(ctx.T, stamp("1:31.62"), stamp("1:31.92")))
	R := math.Min(float64(r.W)*0.11, float64(r.H)*2*0.27)
	const panelW, chartW, gap = 44.0, 40.0, 8.0
	left := math.Max(float64(r.W)-(2.8*R+gap+panelW+gap+chartW), 4) / 2
	cxc, cyc := left+1.4*R, float64(r.H)-0.4*R
	body := Blend(pal.Err, pal.Accent2, k)
	disk := asciiArt{inside: ellipseMask(cxc, cyc, R, R), body: body, edge: body, seed: 149, charset: []rune("01XYxy")}
	disk.feature = glyphFeature([]rune("FM")[int(k+0.5)], cxc, cyc, pal.Ink())
	disk.draw(ctx, r, t)
	cx, cy := float64(r.X)+cxc, float64(r.Y)+cyc/2
	theta := math.Pi/2 - k*3*math.Pi/4
	dirx, diry := math.Cos(theta), math.Sin(theta)
	tip := func(d float64) (float64, float64) { return cx + dirx*d, cy + diry*d/2 }
	stem := R * 1.55
	x0, y0 := tip(R * 1.02)
	x1, y1 := tip(R + stem*0.62)
	g.line(x0, y0, x1, y1, 0, body, term.Bold)
	g.line(x0+1, y0, x1+1, y1, 0, body, term.Bold)
	bx, by := tip(R + stem*0.30)
	half := (1 - k) * R * 0.42
	px, py := -diry, dirx
	g.line(bx-px*half, by-py*half/2, bx+px*half, by+py*half/2, 0, body, term.Bold)
	head := k * R * 0.5
	for _, sgn := range []float64{-1, 1} {
		a := theta + math.Pi + sgn*0.6
		g.line(x1, y1, x1+head*math.Cos(a), y1+head*math.Sin(a)/2, 0, body, term.Bold)
	}
	g.text(int(cx)-6, r.Bottom()-1, fmt.Sprintf("[ %s -> %s ]", active.from, active.to), pal.Ink(), term.Bold)

	px0 := r.X + int(left+2.8*R+gap)
	y := r.Y + max((r.H-13)/2, 1)
	g.text(px0, y, "GENOME EDIT", pal.Dim, term.Bold)
	chrom := "XX"
	if k > 0.5 {
		chrom = "XY"
	}
	if k > 0.02 && k < 0.98 {
		chrom = string(pickRune([]rune("XY01"), int(t*40), 1)) + string(pickRune([]rune("XY01"), int(t*40), 2))
	}
	g.text(px0, y+2, "KARYOTYPE  "+chrom, pal.Accent, term.Bold)
	gene := "SRY   OFF"
	if k > 0.5 {
		gene = "SRY   ON "
	}
	g.text(px0, y+3, "GENE       "+gene, pal.Warn, term.Bold)
	bar := func(label string, v float64, col term.Color, row int) {
		n := int(v * 20)
		g.text(px0, y+row, fmt.Sprintf("%-13s[%s%s]%3d%%", label, strings.Repeat("#", n), strings.Repeat(".", 20-n), int(v*100)), col, term.Bold)
	}
	bar("ESTROGEN", 0.85*(1-k)+0.10, pal.Err, 5)
	bar("TESTOSTERONE", 0.10+0.80*k, pal.Accent2, 6)
	bar("SWITCH", k, pal.Warn, 7)
	for i := range 4 {
		n := int(t*12) - i
		g.text(px0, y+9+i, fmt.Sprintf("0x%04X  %s", uint16(hashSeed(ctx.Seed, n)&0xffff), [...]string{"transcribe", "splice", "express", "fold"}[(n+4)%4]), Fade(pal.Dim, pal.Shadow, 1-0.2*float64(i)), term.Attr(0))
	}

	cxs := r.X + int(left+2.8*R+gap+panelW+gap)
	g.text(cxs, r.Y+1, "KARYOTYPE", pal.Dim, term.Bold)
	chromosome(g, float64(cxs)+8, float64(r.Y)+8, 'x', 0, pal.Err, "X")
	chromosome(g, float64(cxs)+26, float64(r.Y)+8, 'y', k, pal.Accent2, map[bool]string{false: "X", true: "Y"}[k > 0.5])
	cy0 := r.Y + 16
	ch := 9
	g.text(cxs, cy0-1, "HORMONE LEVEL / TIME", pal.Dim, term.Bold)
	for row := 0; row < ch; row++ {
		g.set(cxs, cy0+row, '|', pal.Dim, term.Attr(0))
	}
	for x := 0; x < int(chartW); x++ {
		g.set(cxs+x, cy0+ch, '-', pal.Dim, term.Attr(0))
	}
	var pe, pt [2]float64
	for x := 1; x < int(chartW); x++ {
		when := ctx.T - time.Duration(float64(int(chartW)-x)/chartW*8*float64(time.Second))
		kx := ease(Progress(when, stamp("1:31.62"), stamp("1:31.92")))
		e := 0.85*(1-kx) + 0.10
		tt := 0.10 + 0.80*kx
		ye, yt := float64(cy0+ch-1)-e*float64(ch-1), float64(cy0+ch-1)-tt*float64(ch-1)
		if x > 1 {
			g.line(pe[0], pe[1], float64(cxs+x), ye, 0, pal.Err, term.Bold)
			g.line(pt[0], pt[1], float64(cxs+x), yt, 0, pal.Accent2, term.Bold)
		}
		pe, pt = [2]float64{float64(cxs + x), ye}, [2]float64{float64(cxs + x), yt}
	}
	g.text(cxs+2, cy0, "E", pal.Err, term.Bold)
	g.text(cxs+4, cy0, "T", pal.Accent2, term.Bold)
}

func drawRoleWide(ctx *Context, r Rect, active regEntry) {
	g := stage{ctx, r}
	pal := ctx.Palette
	t := ctx.Since(active.at).Seconds()
	k := ease(Progress(ctx.T, stamp("1:38.94"), stamp("1:39.33")))
	cx, cy := float64(r.W)/2, float64(r.H)+2
	Ro := math.Min(float64(r.W)*0.17, float64(r.H)*2*0.30)
	a := math.Pi*k + 0.15*math.Sin(t)
	Rd := math.Min(Ro*0.46, float64(r.H)*2*0.22)
	abs := func(ux, uy float64) (float64, float64) { return float64(r.X) + ux, float64(r.Y) + uy/2 }
	pos := func(off float64) (float64, float64) { return cx + Ro*math.Cos(a+off), cy + Ro*math.Sin(a+off) }
	ax, ay := pos(0)
	bx, by := pos(math.Pi)
	for i := 0.0; i < 2*math.Pi; i += 0.035 {
		for _, m := range []float64{1, 1.22} {
			x, y := abs(cx+Ro*m*math.Cos(i), cy+Ro*m*math.Sin(i))
			g.set(int(math.Round(x)), int(math.Round(y)), '.', pal.Faint(), term.Attr(0))
		}
	}
	for i := 1; i <= 8; i++ {
		back := a - float64(i)*0.10*(1-k*0.5)
		for _, off := range []float64{0, math.Pi} {
			x, y := abs(cx+Ro*math.Cos(back+off), cy+Ro*math.Sin(back+off))
			g.set(int(math.Round(x)), int(math.Round(y)), '*', Fade(pal.Dim, pal.Shadow, 1-float64(i)/9), term.Bold)
		}
	}
	x0, y0 := abs(ax, ay)
	x1, y1 := abs(bx, by)
	g.line(x0, y0, x1, y1, '=', pal.Warn, term.Bold)
	g.text(int((x0+x1)/2)-6, int((y0+y1)/2)-1, fmt.Sprintf("tension %.2f", 0.5+0.5*math.Sin(t*3)), pal.Dim, term.Attr(0))
	letters := [2]rune{'S', 'M'}
	cols := [2]term.Color{pal.Accent, pal.Accent2}
	flip := k > 0.5
	for i, p := range [][2]float64{{ax, ay}, {bx, by}} {
		idx := i
		if flip {
			idx = 1 - i
		}
		disc := asciiArt{inside: ellipseMask(p[0], p[1], Rd, Rd), body: cols[idx], edge: cols[idx], seed: int64(151 + idx), charset: []rune("01")}
		disc.feature = glyphFeature(letters[idx], p[0], p[1], pal.Ink())
		disc.draw(ctx, r, t)
	}
	roleA, roleB := active.from, active.to
	if flip {
		roleA, roleB = active.to, active.from
	}
	for i, line := range []struct {
		text string
		col  term.Color
	}{
		{fmt.Sprintf("role_a = %s   role_b = %s", roleA, roleB), pal.Warn},
		{fmt.Sprintf("swap   [%s%s] %3d%%", strings.Repeat("#", int(k*16)), strings.Repeat(".", 16-int(k*16)), int(k*100)), pal.Accent},
		{fmt.Sprintf("angle  %03d deg   omega %.2f", int(a*180/math.Pi)%360, 1+2*(1-k)), pal.Dim},
	} {
		g.text(r.X+(r.W-len(line.text))/2, r.Y+i, line.text, line.col, term.Bold)
	}

	lx := r.X + 4
	g.text(lx, r.Y+1, "HANDSHAKE", pal.Dim, term.Bold)
	protos := []string{"SYN", "SYN-ACK", "ACK", "LEASE", "BIND", "RELEASE"}
	for i := range 12 {
		n := int(t*5) - i
		if n < 0 {
			break
		}
		from, to := roleA, roleB
		if n%2 == 1 {
			from, to = roleB, roleA
		}
		g.text(lx, r.Y+3+i, fmt.Sprintf("%s -> %s : %-8s seq=%04d", from, to, protos[n%len(protos)], n), Fade(pal.Accent, pal.Shadow, 1-0.07*float64(i)), term.Attr(0))
	}

	rx := r.Right() - 44
	g.text(rx, r.Y+1, "DOMINANCE", pal.Dim, term.Bold)
	dom := func(label string, v float64, col term.Color, row int) {
		n := int(v * 24)
		g.text(rx, r.Y+row, fmt.Sprintf("%s [%s%s] %3d%%", label, strings.Repeat("#", n), strings.Repeat(".", 24-n), int(v*100)), col, term.Bold)
	}
	dom("S", 0.72*(1-k)+0.28*k, pal.Accent, 3)
	dom("M", 0.28*(1-k)+0.72*k, pal.Accent2, 4)
	g.text(rx, r.Y+7, "PHASE DIAGRAM", pal.Dim, term.Bold)
	bw, bh := 36, 12
	for x := range bw {
		g.set(rx+x, r.Y+8+bh/2, '-', pal.Faint(), term.Attr(0))
	}
	for y := range bh {
		g.set(rx+bw/2, r.Y+8+y, '|', pal.Faint(), term.Attr(0))
	}
	for i := range 240 {
		u := float64(i) / 240 * 2 * math.Pi
		x := math.Sin(3*u + t + math.Pi*k)
		y := math.Sin(2 * u)
		g.set(rx+bw/2+int(math.Round(x*float64(bw/2-1))), r.Y+8+bh/2+int(math.Round(y*float64(bh/2-1))), '.', pal.Accent2, term.Bold)
	}
}

func drawPowerArt(ctx *Context, art Rect, t float64) (Rect, []hudRow) {
	pal := ctx.Palette
	g := stage{ctx, art}
	s := ctx.Screen
	k := ease(math.Min(t/1.44, 1))
	connected := t >= 1.44
	W, H := float64(art.W), float64(art.H)
	bw := math.Min(W*0.16, 44)
	bh := math.Min(H*1.5, 36)
	cyU := H
	const prong = 12.0
	gap := (1 - k) * W * 0.24
	cxR := W/2 + bw/2 + 6
	cxL := W/2 - bw/2 - 4 - gap
	rowOf := func(uy float64) int { return art.Y + int(uy/2) }

	plug := boxArt(pal, cxL, cyU, bw/2, bh/2, 161)
	plug.body, plug.edge = pal.Accent, pal.Accent
	plug.feature = func(x, y float64) (rune, term.Color, bool) {
		if x > cxL+bw/2 && x < cxL+bw/2+prong {
			for _, off := range []float64{-0.30, 0.30} {
				if math.Abs(y-(cyU+off*bh)) < 1.4 {
					return '=', pal.Ink(), true
				}
			}
			if math.Abs(y-cyU) < 1.0 && x < cxL+bw/2+prong-2 {
				return '#', pal.Warn, true
			}
		}
		if math.Abs(x-cxL) < bw*0.22 && math.Abs(y-cyU) < 2.2 {
			return pickRune([]rune("PWR"), int(x), 1), pal.Ink(), true
		}
		return 0, 0, false
	}
	socket := boxArt(pal, cxR, cyU, bw/2, bh/2, 163)
	socket.body, socket.edge = pal.Accent2, pal.Accent2
	socket.feature = func(x, y float64) (rune, term.Color, bool) {
		if x > cxR-bw/2+0.5 && x < cxR-bw/2+3.5 {
			for _, off := range []float64{-0.30, 0, 0.30} {
				if math.Abs(y-(cyU+off*bh)) < 1.4 {
					return 'O', pal.Ink(), true
				}
			}
		}
		if math.Abs(x-cxR) < bw*0.22 && math.Abs(y-cyU) < 2.2 {
			return pickRune([]rune("AC~"), int(x), 2), pal.Ink(), true
		}
		return 0, 0, false
	}
	plug.draw(ctx, art, t)
	socket.draw(ctx, art, t)

	row := rowOf(cyU)
	cableEnd := art.X + int(cxL-bw/2)
	for x := art.X + 2; x < cableEnd; x++ {
		phase := float64(x)*0.7 - t*4
		g.set(x, row-1, rune("/\\"[int(math.Abs(phase))%2]), pal.Accent, term.Bold)
		g.set(x, row, pickRune([]rune("%&8#"), x, int(t*8)), pal.Accent, term.Bold)
		g.set(x, row+1, rune("\\/"[int(math.Abs(phase))%2]), pal.Accent, term.Bold)
	}
	g.text(art.X+2, row-3, "CABLE  3 x 1.5mm2  H05VV-F", pal.Dim, term.Attr(0))
	if connected {
		for i := range 16 {
			u := math.Mod(float64(i)/16+t*0.9, 1)
			g.set(art.X+2+int(u*float64(cableEnd-art.X-3)), row, '*', pal.Ink(), term.Bold)
		}
	}

	if connected {
		age := t - 1.44
		mains := art.X + int(cxR+bw/2) + 1
		reach := min(int(age*60), art.Right()-mains-3)
		for x := 0; x < reach; x++ {
			yv := float64(row) - float64(art.H)*0.18*math.Sin(float64(x)*0.28-t*8)
			g.set(mains+x, int(math.Round(yv)), '~', pal.Accent2, term.Bold)
			if x%6 == 0 {
				g.set(mains+x, row, '+', pal.Faint(), term.Attr(0))
			}
		}
		g.text(mains, row+int(float64(art.H)*0.22)+1, "MAINS 230V 50Hz", pal.Dim, term.Attr(0))
		if age < 0.7 {
			cx0, cy0 := float64(art.X)+cxL+bw/2+prong-3, float64(row)
			for i := range 16 {
				h := hashUnit(ctx.Seed, i*13+int(age*40))
				a := h * 2 * math.Pi
				d := age * 26 * (0.3 + hashUnit(ctx.Seed, i))
				s.Set(int(math.Round(cx0+d*math.Cos(a))), int(math.Round(cy0+d*0.5*math.Sin(a))), pickRune([]rune("*+x.%"), i, int(age*25)), pal.Warn, term.ColorDefault, term.Bold)
			}
			g.text(int(cx0)-6, row-int(bh/4)-2, "[ CONNECTED ]", pal.Ink(), term.Bold)
		}
	}

	pinX := art.X + 16
	g.text(pinX, art.Y+1, "PINOUT", pal.Dim, term.Bold)
	for i, p := range [][2]string{{"L", "live      230V"}, {"N", "neutral   0V"}, {"PE", "earth     GND"}} {
		g.text(pinX, art.Y+3+i, fmt.Sprintf("%-2s  %s", p[0], p[1]), pal.Accent, term.Bold)
	}
	checks := []string{"fuse 13A", "ground", "insulation", "mains", "phase lock", "load"}
	stX := art.Right() - 30
	g.text(stX, art.Y+1, "SELF TEST", pal.Dim, term.Bold)
	for i, c := range checks {
		state, col := "[ .. ]", pal.Faint()
		if t > 1.6+float64(i)*0.35 {
			state, col = "[ OK ]", pal.Accent
		}
		g.text(stX, art.Y+3+i, fmt.Sprintf("%s %s", state, c), col, term.Bold)
	}
	gy := art.Bottom() - 4
	volts := 230 * (0.05 + 0.95*k)
	amps := 0.0
	if connected {
		amps = 3.2 * (1 - math.Exp(-(t-1.44)*2))
	}
	g.text(pinX, gy, fmt.Sprintf("V [%s%s] %5.1fV", strings.Repeat("#", int(volts/230*24)), strings.Repeat(".", 24-int(volts/230*24)), volts), pal.Accent2, term.Bold)
	g.text(pinX, gy+1, fmt.Sprintf("I [%s%s] %5.2fA", strings.Repeat("#", int(amps/3.2*24)), strings.Repeat(".", 24-int(amps/3.2*24)), amps), pal.Accent2, term.Bold)
	g.text(pinX, gy+2, fmt.Sprintf("P [%s%s] %5.0fW", strings.Repeat("#", int(volts*amps/736*24)), strings.Repeat(".", 24-int(volts*amps/736*24)), volts*amps), pal.Accent2, term.Bold)

	box := Rect{X: art.X + int(cxL-bw/2) - 2, Y: rowOf(cyU - bh/2), W: int(cxR-cxL+bw) + 4, H: int(bh/2) + 1}
	return box, []hudRow{{"VOLT", fmt.Sprintf("%.0fV", volts)}, {"FREQ", "50Hz"}, {"LOAD", fmt.Sprintf("%.1fA", amps)}, {"GND", "OK"}}
}
