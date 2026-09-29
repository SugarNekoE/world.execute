package scene

import (
	"math"
	"time"

	"world.execute/internal/term"
)

func (rs *RegisterScene) drawAnimation(ctx *Context, r Rect) {
	if r.W < 24 || r.H < 6 || len(rs.regs) == 0 {
		return
	}
	ctx.Panel(r)
	switch rs.regs[0].label {
	case "current":
		drawCurrent(ctx, r, rs.regs)
	case "position":
		drawTravel(ctx, r, rs.regs)
	case "gender":
		drawIdentity(ctx, r, rs.regs)
	}
}

func motionCanvas(ctx *Context, r Rect) *Canvas {
	p := ctx.Palette
	return NewCanvas(ctx.Screen, r, p.Dim, p.Accent, p.Accent2, p.Kind)
}

func motionLabel(ctx *Context, r Rect, text string) {
	ctx.Screen.TextWidth(r.X, r.Y, r.W, text, ctx.Palette.Text, term.ColorDefault, term.Bold)
}

func ease(k float64) float64 { return k * k * (3 - 2*k) }

func drawCurrent(ctx *Context, r Rect, regs []regEntry) {
	switchAt := regs[1].at
	k := ease(Progress(ctx.T, switchAt, switchAt+450*time.Millisecond))
	mode := "AC ~  alternating current"
	if ctx.T >= switchAt {
		mode = "DC =  direct current"
	}
	motionLabel(ctx, r, mode)
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(regs[0].at).Seconds()
	cy := 0.25 + 0.08*(1-k)
	c.Polyline([]Point{{0.12, 0.33}, {0.12, 0.08}, {0.43, 0.08}}, 0)
	c.Polyline([]Point{{0.59, 0.08}, {0.9, 0.08}, {0.9, 0.45}, {0.12, 0.45}, {0.12, 0.33}}, 0)
	c.Circle(0.12, 0.25, 0.07, 1.6, 1)
	c.Line(0.06, 0.25, 0.18, 0.25, 1)
	c.Circle(0.43, 0.08, 0.012, 1.8, 2)
	c.Circle(0.59, 0.08, 0.012, 1.8, 2)
	c.Line(0.43, 0.08, 0.59, 0.08+0.19*(1-k), 2)
	c.Circle(0.9, 0.26, 0.045, 2.0, 2)
	c.Line(0.87, 0.20, 0.93, 0.32, 2)
	c.Line(0.87, 0.32, 0.93, 0.20, 2)
	flow := (1-k)*0.08*math.Sin(t*7) + k*t*0.24
	for i := range 20 {
		u := math.Mod(float64(i)/20+flow+10, 1)
		x, y := circuitPoint(u)
		c.Disc(x, y, 0.009, 1.6, 1+i%2)
	}
	c.Line(0.03, 0.76, 0.97, 0.76, 0)
	c.Func(func(x float64) float64 {
		wave := 0.76 - 0.18*math.Sin(x*math.Pi*6-t*7)
		return (1-k)*wave + k*0.59
	}, 1, r.W*4)
	if ctx.T >= regs[2].at && ctx.T < regs[3].at {
		shutter := Progress(ctx.T, regs[2].at, regs[2].at+time.Second)
		for i := range 12 {
			x := float64(i) / 11
			c.Line(x, 0, x, 0.40*shutter, 0)
			c.Line(x, 1, x, 1-0.4*shutter, 0)
		}
	}
	if ctx.T >= regs[3].at {
		for i := range 4 {
			a := ctx.Since(regs[3].at).Seconds()*4 + float64(i)*math.Pi/2
			c.Arc(0.5, cy, 0.22, 0.65, a, a+0.9, 3)
		}
	}
	c.Flush()
	ctx.Screen.TextWidth(r.X, r.Bottom()-1, r.W, "AC  ~~~~~    [ switch ]    ━━━━━  DC", ctx.Palette.Dim, term.ColorDefault, term.Attr(0))
}

func circuitPoint(u float64) (float64, float64) {
	switch {
	case u < 0.4:
		return 0.12 + 0.78*u/0.4, 0.08
	case u < 0.5:
		return 0.9, 0.08 + 0.37*(u-0.4)/0.1
	case u < 0.9:
		return 0.9 - 0.78*(u-0.5)/0.4, 0.45
	default:
		return 0.12, 0.45 - 0.37*(u-0.9)/0.1
	}
}

func drawTravel(ctx *Context, r Rect, regs []regEntry) {
	label := "SPACE / TIME   →   HERE ━━━━━ THERE"
	if ctx.T >= regs[1].at {
		label = "TIME REVERSAL   A.D → B.C"
	}
	if ctx.T >= regs[2].at {
		label = "UNITE   TWO → ONE"
	}
	if ctx.T >= regs[3].at {
		label = "DEEP CONNECTION   ∞"
	}
	motionLabel(ctx, r, label)
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(regs[0].at).Seconds()
	phase := t * 0.65
	if ctx.T >= regs[1].at {
		phase = (regs[1].at-regs[0].at).Seconds()*0.65 - ctx.Since(regs[1].at).Seconds()*0.65
	}
	for i := range 10 {
		z := math.Mod(float64(i)/10+phase+20, 1)
		radius := 0.03 + 0.45*z*z
		cx := 0.5 + 0.04*(1-z)*math.Sin(t)
		c.Circle(cx, 0.5, radius, 0.95, i%3)
	}
	for i := range 12 {
		a := float64(i)*math.Pi/6 + t*0.2
		c.Line(0.5+0.04*math.Cos(a), 0.5+0.04*math.Sin(a), 0.5+0.47*math.Cos(a), 0.5+0.45*math.Sin(a), 0)
	}
	merge := ease(Progress(ctx.T, regs[2].at, regs[3].at))
	gap := 0.24 * (1 - merge)
	for _, sign := range []float64{-1, 1} {
		cx := 0.5 + sign*gap
		c.Circle(cx, 0.5, 0.08+0.01*math.Sin(t*5), 1.3, 2)
		c.Disc(cx, 0.5, 0.017, 1.5, 1)
	}
	c.Flush()
}

func drawIdentity(ctx *Context, r Rect, regs []regEntry) {
	active := regs[0]
	for _, reg := range regs {
		if ctx.T >= reg.at {
			active = reg
		}
	}
	motionLabel(ctx, r, active.label+"   [ "+active.from+" ]  →  [ "+active.to+" ]")
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(active.at).Seconds()
	k := ease(Progress(ctx.T, active.at, active.at+650*time.Millisecond))
	if active.label == "clock" {
		c.Circle(0.5, 0.5, 0.29, 1.5, 1)
		for i := range 12 {
			a := float64(i) * math.Pi / 6
			c.Line(0.5+0.25*math.Sin(a), 0.5-0.375*math.Cos(a), 0.5+0.29*math.Sin(a), 0.5-0.435*math.Cos(a), 0)
		}
		for i, speed := range []float64{0.6, 3} {
			a := t*speed - math.Pi/2
			length := 0.16 + float64(i)*0.08
			c.Line(0.5, 0.5, 0.5+length*math.Cos(a), 0.5+length*1.5*math.Sin(a), 1+i)
		}
	} else if active.label == "gender" {
		k = ease(Progress(ctx.T, stamp("1:31.62"), stamp("1:31.92")))
		c.Circle(0.5, 0.40, 0.18, 1.25, 1)
		a := math.Pi/2 - k*math.Pi*0.75
		x0, y0 := 0.5+0.18*math.Cos(a), 0.4+0.225*math.Sin(a)
		x1, y1 := 0.5+0.34*math.Cos(a), 0.4+0.425*math.Sin(a)
		c.Line(x0, y0, x1, y1, 2)
		if k < 0.5 {
			c.Line(0.42, 0.72, 0.58, 0.72, 2)
		} else {
			c.Line(x1, y1, x1-0.1, y1, 2)
			c.Line(x1, y1, x1, y1+0.13, 2)
		}
	} else if active.label == "role" {
		k = ease(Progress(ctx.T, stamp("1:38.94"), stamp("1:39.33")))
		a := math.Pi * k
		c.Circle(0.5, 0.5, 0.3, 0.8, 0)
		for i, sign := range []float64{-1, 1} {
			x, y := 0.5+sign*0.3*math.Cos(a), 0.5+sign*0.24*math.Sin(a)
			c.Circle(x, y, 0.09, 1.5, i+1)
			c.Disc(x, y, 0.02, 1.5, i+1)
		}
	} else {
		for strand := range 2 {
			phase := float64(strand) * math.Pi
			var strandPoints [100]Point
			pts := strandPoints[:]
			for i := range pts {
				y := float64(i) / 99
				pts[i] = Point{0.5 + 0.3*math.Sin(y*math.Pi*4+t*2+phase), 0.08 + y*0.84}
			}
			c.Polyline(pts, 1+strand)
		}
		for i := range 15 {
			y := float64(i) / 14
			x := 0.3 * math.Sin(y*math.Pi*4+t*2)
			c.Line(0.5-x, 0.08+y*0.84, 0.5+x, 0.08+y*0.84, 0)
		}
		c.Disc(0.2+0.6*k, 0.5, 0.035, 1.5, 3)
		for i := range 4 {
			c.Arc(0.5, 0.5, 0.12+float64(i)*0.09, 1, t+float64(i), t+float64(i)+math.Pi, 3)
		}
	}
	c.Flush()
}

func drawObjectAnimation(ctx *Context, r Rect, spec objectSpec) {
	if r.W < 22 || r.H < 7 {
		return
	}
	ctx.Panel(r)
	motionLabel(ctx, r, spec.name+" / materializing")
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(spec.at).Seconds()
	appear := ease(Progress(ctx.T, spec.at, spec.at+700*time.Millisecond))
	cy := 0.48 + 0.03*math.Sin(t*3)
	scale := 0.2 + 0.8*appear
	for i := range 3 {
		c.Circle(0.5, 0.86, 0.25+float64(i)*0.06, 0.16, i%3)
	}
	for i := range 30 {
		a := float64(i)*2.39996 + t*0.7
		rad := 0.32 + 0.08*math.Sin(float64(i))
		c.Set(0.5+rad*math.Cos(a), 0.5+0.36*math.Sin(a), i%3)
	}
	switch spec.name {
	case "Tomato":
		c.Circle(0.5, cy, 0.23*scale, 1.15, 2)
		for i := range 5 {
			a := float64(i)*math.Pi*2/5 - t*0.2
			c.Line(0.5, cy-0.25*scale, 0.5+0.12*math.Cos(a), cy-0.25*scale+0.06*math.Sin(a), 1)
		}
	case "Eggplant":
		pts := make([]Point, 121)
		for i := range pts {
			a := float64(i) * math.Pi * 2 / 120
			pts[i] = Point{0.5 + scale*(0.14*math.Cos(a)+0.09*math.Sin(a)), cy + 0.27*scale*math.Sin(a)}
		}
		c.Polyline(pts, 3)
		c.Line(0.41, cy-0.27*scale, 0.47, cy-0.36*scale, 1)
	case "TabbyCat":
		c.Polyline([]Point{{0.29, cy + 0.17}, {0.29, cy - 0.24}, {0.4, cy - 0.11}, {0.6, cy - 0.11}, {0.71, cy - 0.24}, {0.71, cy + 0.17}, {0.5, cy + 0.26}, {0.29, cy + 0.17}}, 2)
		blink := math.Mod(t, 2.7) > 2.5
		for _, x := range []float64{0.4, 0.6} {
			if blink {
				c.Line(x-0.035, cy, x+0.035, cy, 1)
			} else {
				c.Circle(x, cy, 0.025, 1.6, 1)
			}
		}
		for _, sign := range []float64{-1, 1} {
			for i := range 3 {
				c.Line(0.5+sign*0.08, cy+0.10, 0.5+sign*0.32, cy+float64(i)*0.08, 0)
			}
		}
		c.Polyline([]Point{{0.47, cy + 0.07}, {0.5, cy + 0.11}, {0.53, cy + 0.07}}, 3)
	default:
		c.Circle(0.5, cy-0.22, 0.22, 0.23, 2)
		DrawHeart(c, 0.5, cy+0.05, 0.17, float64(ctx.Energy), 1)
		for i := range 12 {
			a := float64(i)*math.Pi/6 + t*0.5
			c.Line(0.5+0.26*math.Cos(a), cy+0.29*math.Sin(a), 0.5+0.32*math.Cos(a), cy+0.36*math.Sin(a), 2)
		}
	}
	scan := math.Mod(t*0.45, 1)
	c.Line(0.16, scan, 0.84, scan, 0)
	if ctx.T >= spec.kwAt {
		age := ctx.Since(spec.kwAt).Seconds()
		for i := range 24 {
			a := float64(i) * 2 * math.Pi / 24
			rad := 0.1 + math.Mod(age*0.3+float64(i%3)*0.08, 0.35)
			c.Set(0.5+rad*math.Cos(a), cy+rad*math.Sin(a), 1+i%3)
		}
	}
	if spec.name == "TabbyCat" && ctx.T >= stamp("1:23.64") {
		for i := range 3 {
			rad := 0.25 + math.Mod(t*0.15+float64(i)*0.08, 0.2)
			c.Arc(0.5, cy, rad, 1, -0.5, 0.5, 1)
			c.Arc(0.5, cy, rad, 1, math.Pi-0.5, math.Pi+0.5, 1)
		}
	}
	c.Flush()
	ctx.Screen.TextWidth(r.X, r.Bottom()-1, r.W, "↑ "+spec.gift, ctx.Palette.Accent, term.ColorDefault, term.Bold)
}

func drawHorizon(ctx *Context, r Rect) {
	if r.W < 24 || r.H < 2 {
		return
	}
	c := motionCanvas(ctx, r)
	for i := range 13 {
		c.Line(0.5, 0, float64(i)/12, 1, 0)
	}
	for i := range 8 {
		z := math.Mod(float64(i)/8+ctx.T.Seconds()*0.25, 1)
		y := z * z
		c.Line(0.5-y*0.5, y, 0.5+y*0.5, y, 1)
	}
	c.Flush()
}
