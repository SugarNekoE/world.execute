package scene

import (
	"fmt"
	"math"
	"time"

	"world.execute/internal/term"
)

// CalibrationPeriod is the gap between the clicks of the calibration track.
const CalibrationPeriod = 500 * time.Millisecond

// Calibration is the sync test card: a flash and a sweeping bar timed to the
// clicks of the calibration track, so the viewer can nudge the delay until the
// picture and the sound agree.
type Calibration struct{ theme string }

// NewCalibration returns the test card in the given theme.
func NewCalibration(theme string) *Calibration { return &Calibration{theme: theme} }

// Draw paints the test card.
func (c *Calibration) Draw(ctx *Context) {
	s := ctx.Screen
	pal := Theme(c.theme)["verse"]
	ctx.Palette = pal
	ctx.Section = "CALIBRATION"
	area := ctx.Area
	if area.Empty() {
		return
	}
	g := stage{ctx, area}
	s.Clear()

	phase := math.Mod(ctx.T.Seconds(), CalibrationPeriod.Seconds())
	if ctx.T < 0 {
		phase = 0
	}
	since := time.Duration(phase * float64(time.Second))
	beat := int(ctx.T / CalibrationPeriod)
	flash := ctx.T >= CalibrationPeriod && since < 90*time.Millisecond

	bw, bh := min(area.W-8, 44), min(area.H-12, 9)
	bx, by := area.X+(area.W-bw)/2, area.Y+2
	for y := by; y < by+bh; y++ {
		for x := bx; x < bx+bw; x++ {
			edge := y == by || y == by+bh-1 || x == bx || x == bx+bw-1
			switch {
			case flash:
				g.set(x, y, '#', pal.Ink(), term.Bold)
			case edge:
				g.set(x, y, '+', pal.Dim, term.Attr(0))
			}
		}
	}
	label := "[ CLICK ]"
	col := pal.Faint()
	if flash {
		col = pal.Shadow
		label = "#########"
	}
	g.text(bx+(bw-len(label))/2, by+bh/2, label, col, term.Bold)

	trackY := by + bh + 2
	trackX0, trackX1 := area.X+4, area.Right()-5
	for x := trackX0; x <= trackX1; x++ {
		g.set(x, trackY, '-', pal.Faint(), term.Attr(0))
	}
	for i := 0; i <= 10; i++ {
		g.set(trackX0+(trackX1-trackX0)*i/10, trackY, '+', pal.Dim, term.Attr(0))
	}
	mark := trackX0 + int(float64(trackX1-trackX0)*phase/CalibrationPeriod.Seconds())
	g.set(mark, trackY-1, 'v', pal.Warn, term.Bold)
	g.set(mark, trackY, 'O', pal.Warn, term.Bold)
	g.text(trackX1-6, trackY+1, "click |", pal.Dim, term.Attr(0))

	line := func(y int, text string, col term.Color, attr term.Attr) {
		g.text(area.X+(area.W-len(text))/2, y, text, col, attr)
	}
	y := trackY + 3
	line(y, fmt.Sprintf("SYNC  %+d ms   (positive shows the picture earlier)", ctx.Delay.Milliseconds()), pal.Accent, term.Bold)
	line(y+2, "Watch the flash and listen to the click.", pal.Text, term.Attr(0))
	line(y+3, "] picture is late  (flash after the click)      [ picture is early (flash before the click)", pal.Dim, term.Attr(0))
	line(y+4, "{ } move 50 ms      q saves the value and quits", pal.Dim, term.Attr(0))
	line(y+6, fmt.Sprintf("click %04d   period %.2fs", beat, CalibrationPeriod.Seconds()), pal.Faint(), term.Attr(0))
}
