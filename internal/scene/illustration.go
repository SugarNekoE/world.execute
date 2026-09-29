package scene

import (
	"fmt"
	"math"
	"time"

	"world.execute/internal/term"
)

type visualCue struct {
	at          time.Duration
	kind, label string
}

var lyricVisuals = map[string][]visualCue{
	"boot": {
		{stamp("0:00.00"), "power", "POWER LINE / connecting"},
		{stamp("0:03.12"), "shield", "PROTECTION / shield online"},
		{stamp("0:06.42"), "assemble", "OBJECT CREATION / assembling"},
		{stamp("0:10.32"), "assemble", "INITIALIZATION / loading parameters"},
		{stamp("0:13.89"), "world", "SIMULATION / world online"},
	},
	"chorus1": {
		{stamp("0:59.46"), "network", "STIMULATIONS / signal propagation"},
		{stamp("1:05.61"), "satisfy", "SATISFACTION / target acquired"},
		{stamp("1:09.33"), "execute", "EXECUTION / instruction fired"},
		{stamp("1:10.35"), "cage", "SIMULATION / no way out"},
	},
	"vibrate": {
		{stamp("1:43.50"), "vibrate", "VIBRATIONS / seeking resonance"},
		{stamp("1:50.16"), "complete", "COMPLETION / in phase"},
	},
	"abandon":   {{stamp("1:50.94"), "isolate", "CONNECTION LOST / isolation"}},
	"fragments": {{stamp("1:58.38"), "fragments", "FRAGMENTS / erasing the heart"}},
	"trace": {
		{stamp("2:05.67"), "judge", "CHALLENGING YOUR GOD / judgment"},
		{stamp("2:11.22"), "error", "ILLEGAL ARGUMENTS / rejected"},
	},
	"chorus2": {
		{stamp("2:41.67"), "execute", "EXECUTION / all targets"},
		{stamp("2:45.39"), "execute", "EXECUTION / all targets"},
		{stamp("2:49.11"), "execute", "EXECUTION / only target"},
		{stamp("2:52.86"), "execute", "EXECUTION / return to me"},
		{stamp("2:53.76"), "cage", "TRAPPED / recursive simulation"},
	},
}

type IllustratedScene struct {
	stage Layer
	cues  []visualCue
}

func (s *IllustratedScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.H < 15 || area.W-ctx.SideW < 34 {
		s.stage.Draw(ctx)
		return
	}
	top := *ctx
	top.Area.H = max(6, area.H/3)
	s.stage.Draw(&top)
	r := Rect{X: area.X + 3, Y: area.Y + top.Area.H + 1, W: area.W - ctx.SideW - 6, H: area.H - top.Area.H - 2}
	ctx.Panel(r)
	cue := s.cues[0]
	for _, next := range s.cues {
		if next.at <= ctx.T {
			cue = next
		}
	}
	motionLabel(ctx, r, cue.label)
	r.Y++
	r.H--
	c := motionCanvas(ctx, r)
	t := ctx.Since(cue.at).Seconds()
	drawIllustration(ctx, c, cue.kind, t)
	c.Flush()
	if cue.kind == "isolate" {
		drawPingLog(ctx, r, cue)
	}
}

// drawPingLog lists the pings sent to the one who left, each timing out, and a
// packet loss gauge that climbs to a hundred percent.
func drawPingLog(ctx *Context, r Rect, cue visualCue) {
	if r.W < 40 || r.H < 6 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	t := ctx.Since(cue.at).Seconds()
	seq := int(t * 1.6)
	lines := min(4, r.H-2)
	for i := range lines {
		n := seq - i
		if n < 0 {
			break
		}
		fg := Fade(pal.Err, pal.Shadow, 1-0.22*float64(i))
		s.Text(r.X, r.Bottom()-1-i, fmt.Sprintf("ping you  seq=%02d  timeout", n), fg, term.ColorDefault, term.Attr(0))
	}
	loss := Progress(ctx.T, cue.at, stamp("1:57.51"))
	const bar = 10
	filled := int(loss * bar)
	label := fmt.Sprintf("packet loss %3d%%", int(loss*100))
	x := r.Right() - bar - 1 - term.StringWidth(label) - 1
	s.Text(x, r.Y, label, pal.Dim, term.ColorDefault, term.Attr(0))
	s.HLine(x+term.StringWidth(label)+1, r.Y, filled, '█', pal.Err, term.ColorDefault, term.Attr(0))
	s.HLine(x+term.StringWidth(label)+1+filled, r.Y, bar-filled, '░', pal.Dim, term.ColorDefault, term.Attr(0))
}

func drawIllustration(ctx *Context, c *Canvas, kind string, t float64) {
	switch kind {
	case "power":
		k := ease(min(t/1.44, 1))
		gap := 0.15 * (1 - k)
		c.Line(0.03, 0.5, 0.4-gap, 0.5, 1)
		c.Line(0.6+gap, 0.5, 0.97, 0.5, 2)
		for _, y := range []float64{0.4, 0.6} {
			c.Line(0.4-gap, y, 0.5-gap, y, 1)
		}
		c.Polyline([]Point{{0.4 - gap, 0.3}, {0.4 - gap, 0.7}, {0.3 - gap, 0.7}, {0.3 - gap, 0.3}, {0.4 - gap, 0.3}}, 1)
		c.Polyline([]Point{{0.6 + gap, 0.3}, {0.6 + gap, 0.7}, {0.5 + gap, 0.7}, {0.5 + gap, 0.3}, {0.6 + gap, 0.3}}, 2)
		if k == 1 {
			for i := range 18 {
				c.Disc(math.Mod(t*0.5+float64(i)/18, 1), 0.5, 0.008, 2, 2)
			}
		}
	case "shield":
		c.Polyline([]Point{{0.5, 0.07}, {0.76, 0.23}, {0.70, 0.66}, {0.5, 0.94}, {0.30, 0.66}, {0.24, 0.23}, {0.5, 0.07}}, 1)
		c.Polyline([]Point{{0.37, 0.48}, {0.47, 0.66}, {0.65, 0.30}}, 2)
		for i := range 14 {
			x := math.Mod(float64(i)/14+t*0.28, 1)
			y := 0.08 + float64(i%7)*0.14
			if x < 0.23 || x > 0.77 {
				c.Line(x-0.02, y, x, y, 3)
			}
		}
	case "assemble", "cage":
		drawWireCube(c, t, kind == "assemble")
		if kind == "cage" {
			DrawHeart(c, 0.5, 0.5, 0.14, float64(ctx.Energy), 2)
		}
	case "world":
		c.Circle(0.5, 0.5, 0.36, 1.2, 1)
		for i := range 8 {
			a := t*0.6 + float64(i)*math.Pi/8
			c.Circle(0.5, 0.5, max(0.005, math.Abs(math.Cos(a))*0.36), 0.432/max(0.005, math.Abs(math.Cos(a))*0.36), 0)
		}
		for _, y := range []float64{-0.24, 0, 0.24} {
			rx := 0.36 * math.Sqrt(1-y*y/(0.432*0.432))
			c.Circle(0.5, 0.5+y, rx, 0.12, 2)
		}
	case "network", "isolate":
		remaining := 8
		if kind == "isolate" {
			remaining = max(0, 8-int(t*1.25))
		}
		for i := range 8 {
			a := float64(i)*math.Pi/4 + t*0.12
			x, y := 0.5+0.39*math.Cos(a), 0.5+0.40*math.Sin(a)
			if i >= remaining {
				continue
			}
			c.Line(0.5, 0.5, x, y, 0)
			c.Circle(x, y, 0.025, 1.4, 2)
			p := math.Mod(t*0.9+float64(i)/8, 1)
			c.Disc(0.5+(x-0.5)*p, 0.5+(y-0.5)*p, 0.014, 1.4, 1)
		}
		DrawHeart(c, 0.5, 0.5, 0.10, float64(ctx.Energy), 1)
	case "satisfy":
		for i := range 5 {
			r := 0.09 + float64(i)*0.07
			c.Arc(0.5, 0.5, r, 1.05, t+float64(i), t+float64(i)+math.Pi*1.7, 1+i%2)
		}
		c.Polyline([]Point{{0.42, 0.5}, {0.48, 0.59}, {0.6, 0.37}}, 2)
	case "vibrate", "complete":
		align := Progress(ctx.T, stamp("1:47.28"), stamp("1:50.16"))
		for i := range 2 {
			c.Func(func(x float64) float64 {
				return 0.5 + 0.27*math.Sin(x*math.Pi*6-t*5+float64(i)*(1-align)*math.Pi)
			}, 1+i, 500)
		}
		if kind == "complete" {
			c.Circle(0.5, 0.5, 0.32, 1.2, 2)
			c.Polyline([]Point{{0.40, 0.48}, {0.48, 0.60}, {0.64, 0.33}}, 2)
		}
	case "fragments":
		pts := HeartPath(0.5, 0.5, 0.29, 0, 120)
		spread := Progress(ctx.T, stamp("2:00.24"), stamp("2:04.98"))
		for i := 1; i < len(pts); i++ {
			if float64(i%13)/13 < spread*0.8 {
				continue
			}
			p := pts[i]
			dx := (p.X - 0.5) * spread * 0.9
			dy := (p.Y-0.5)*spread + 0.08*spread*math.Sin(t+float64(i/8))
			c.Line(pts[i-1].X+dx, pts[i-1].Y+dy, p.X+dx, p.Y+dy, 1+i/8%3)
		}
	case "judge":
		tilt := 0.14 * math.Sin(t*2)
		c.Line(0.5, 0.1, 0.5, 0.88, 1)
		c.Line(0.35, 0.9, 0.65, 0.9, 1)
		c.Line(0.18, 0.24-tilt, 0.82, 0.24+tilt, 2)
		for _, sign := range []float64{-1, 1} {
			x, y := 0.5+sign*0.28, 0.24+sign*tilt
			c.Line(x, y, x-0.12, y+0.38, 0)
			c.Line(x, y, x+0.12, y+0.38, 0)
			c.Arc(x, y+0.38, 0.12, 0.9, 0, math.Pi, 2)
		}
	case "error":
		c.Polyline([]Point{{0.5, 0.06}, {0.84, 0.87}, {0.16, 0.87}, {0.5, 0.06}}, 2)
		c.Line(0.5, 0.29, 0.5, 0.59, 1)
		c.Disc(0.5, 0.74, 0.025, 1.3, 1)
		c.Arc(0.5, 0.5, 0.46, 1, t, t+1.3, 3)
	case "execute":
		DrawBurst(c, math.Mod(t*1.6, 1))
		c.Line(0.5, 0.05, 0.5, 0.95, 2)
		c.Line(0.08, 0.5, 0.92, 0.5, 2)
		c.Circle(0.5, 0.5, 0.12+0.04*math.Sin(t*5), 1.5, 1)
	}
}

func drawWireCube(c *Canvas, t float64, assembling bool) {
	var points [8]Point
	for i := range points {
		x, y, z := float64(i&1)*2-1, float64(i>>1&1)*2-1, float64(i>>2&1)*2-1
		a := t * 0.6
		rx, rz := x*math.Cos(a)+z*math.Sin(a), z*math.Cos(a)-x*math.Sin(a)
		scale := 1.0
		if assembling {
			scale += 0.6 * math.Exp(-t*2)
		}
		points[i] = Point{0.5 + rx*0.22*scale, 0.5 + (y*0.26+rz*0.10)*scale}
	}
	for i, p := range points {
		for bit := range 3 {
			j := i ^ (1 << bit)
			if j > i {
				c.Line(p.X, p.Y, points[j].X, points[j].Y, 1+bit%2)
			}
		}
		c.Disc(p.X, p.Y, 0.012, 1.4, 3)
	}
}
