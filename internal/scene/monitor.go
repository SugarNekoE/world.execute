package scene

import (
	"fmt"
	"math"
	"time"

	"world.execute/internal/term"
)

const (
	monitorStart  = 208300 * time.Millisecond
	monitorFail   = 209800 * time.Millisecond
	monitorFlat   = 210900 * time.Millisecond
	monitorBPM    = 72.0
	monitorWindow = 4.2
)

func gauss(x, centre, width float64) float64 {
	d := (x - centre) / width
	return math.Exp(-d * d)
}

func ecgWave(phase float64) float64 {
	return 0.12*gauss(phase, 0.18, 0.035) -
		0.14*gauss(phase, 0.355, 0.010) +
		1.00*gauss(phase, 0.380, 0.012) -
		0.26*gauss(phase, 0.410, 0.012) +
		0.30*gauss(phase, 0.620, 0.055)
}

func monitorGain(t time.Duration) float64 {
	return 1 - ease(Progress(t, monitorFail, monitorFlat))
}

func drawHeartMonitor(ctx *Context, r Rect) {
	if r.W < 30 || r.H < 5 || ctx.T < monitorStart {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	fade := 0.4 + 0.6*Progress(ctx.T, monitorStart, monitorStart+500*time.Millisecond)
	gain := monitorGain(ctx.T)
	bpm := int(math.Round(monitorBPM * gain))

	label := "♥ ECG  lead II"
	s.Text(r.X, r.Y, label, Fade(pal.Dim, pal.Shadow, fade), term.ColorDefault, term.Attr(0))
	reading := fmt.Sprintf("%03d bpm", bpm)
	col := pal.Kind
	if bpm == 0 {
		reading, col = "asystole", pal.Err
	}
	s.Text(r.Right()-term.StringWidth(reading), r.Y, reading, Fade(col, pal.Shadow, fade), term.ColorDefault, term.Bold)

	plot := Rect{X: r.X, Y: r.Y + 1, W: r.W, H: r.H - 1}
	c := NewCanvas(s, plot, pal.Dim, pal.Kind, pal.Err)
	c.Line(0, 0.62, 1, 0.62, 0)
	period := 60 / monitorBPM
	now := ctx.T.Seconds()
	key := 1
	if bpm == 0 {
		key = 2
	}
	prevX, prevY, havePrev := 0.0, 0.0, false
	for i := range plot.W * 2 {
		x := float64(i) / float64(plot.W*2-1)
		sample := now - (1-x)*monitorWindow
		if sample < monitorStart.Seconds() {
			continue
		}
		g := monitorGain(time.Duration(sample * float64(time.Second)))
		y := 0.62 - 0.55*g*ecgWave(math.Mod(sample/period, 1))
		if havePrev {
			c.Line(prevX, prevY, x, y, key)
		}
		prevX, prevY, havePrev = x, y, true
	}
	c.Disc(1, 0.62-0.55*gain*ecgWave(math.Mod(now/period, 1)), 0.012, 1, key)
	c.Flush()
}
