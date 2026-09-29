package scene

import (
	"fmt"
	"math"
	"time"

	"world.execute/internal/term"
)

// Telemetry is the sidecar display for the text heavy sections: a scrolling
// load graph for the boot and shutdown logs, a memory dump for the panic and
// the abandonment.
type Telemetry struct {
	kind string
	seed int64
}

// Telemetry kinds.
const (
	TelemetryLoad = "load"
	TelemetryDump = "dump"
)

// NewTelemetry returns a sidecar of the given kind.
func NewTelemetry(kind string, seed int64) *Telemetry {
	return &Telemetry{kind: kind, seed: seed}
}

// Draw renders the sidecar inside r. It blanks its strip first, so the scenes
// must leave room for it: they read Context.SideW.
func (t *Telemetry) Draw(ctx *Context, r Rect) {
	if r.W < 16 || r.H < 6 {
		return
	}
	ctx.Screen.Fill(r.X, r.Y, r.W, r.H, ' ', ctx.Palette.Text, term.ColorDefault, term.Attr(0))
	switch t.kind {
	case TelemetryDump:
		t.drawDump(ctx, r)
	default:
		t.drawLoad(ctx, r)
	}
}

// drawLoad charts the loudness of the track over the last seconds, which reads
// as a system under load.
func (t *Telemetry) drawLoad(ctx *Context, r Rect) {
	pal := ctx.Palette
	s := ctx.Screen

	title := "── system monitor "
	for term.StringWidth(title) < r.W {
		title += "─"
	}
	s.TextWidth(r.X, r.Y, r.W, title, pal.Dim, term.ColorDefault, term.Attr(0))

	rows := min(max(r.H/3, 3), 6)
	graph := Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: rows}
	c := NewCanvas(s, graph, pal.Faint(), pal.Accent, pal.Accent2)
	defer c.Flush()

	const span = 6 * time.Second
	for i := range graph.W {
		x := float64(i) / float64(max(graph.W-1, 1))
		back := time.Duration((1 - x) * float64(span))
		var v float64
		if ctx.Analysis != nil {
			v = float64(ctx.Analysis.EnergyAt(ctx.T - back))
		} else {
			v = 0.4 + 0.3*math.Sin(ctx.Sec(ctx.T)*2-x*6)
		}
		col := Point{X: x, Y: 1 - min(max(v, 0), 1)}
		c.Set(col.X, col.Y, 1)
		c.Line(col.X, 1, col.X, col.Y, 0)
	}

	row := graph.Bottom() + 1
	stats := []struct {
		label string
		value float64
	}{
		{"cpu", t.level(ctx, 0.9)},
		{"mem", t.level(ctx, 0.6)},
		{"io", t.level(ctx, 1.4)},
		{"love", t.level(ctx, 0.35)},
	}
	for _, st := range stats {
		if row >= r.Bottom() {
			break
		}
		s.Text(r.X+1, row, pad(int(st.value*100), 3)+"%", pal.Accent, term.ColorDefault, term.Attr(0))
		s.Text(r.X+6, row, st.label, pal.Dim, term.ColorDefault, term.Attr(0))
		bar := r.W - 8
		fill := int(st.value * float64(bar))
		for i := range bar {
			ch, fg := '░', pal.Faint()
			if i < fill {
				ch, fg = '█', Blend(pal.Accent, pal.Accent2, float64(i)/float64(max(bar, 1)))
			}
			s.Set(r.X+6+term.StringWidth(st.label)+1+i, row, ch, fg, term.ColorDefault, term.Attr(0))
		}
		row++
	}
}

// level is a pseudo metric that stays stable for a label but moves with the
// music.
func (t *Telemetry) level(ctx *Context, phase float64) float64 {
	base := 0.25 + 0.45*float64(ctx.Energy)
	if ctx.Analysis == nil {
		base = 0.4
	}
	wobble := 0.18 * math.Sin(ctx.Sec(ctx.T)*1.6+phase*4)
	v := base + wobble
	return min(max(v, 0.02), 0.99)
}

// drawDump scrolls a memory dump, so the panic sections have something moving
// beside them.
func (t *Telemetry) drawDump(ctx *Context, r Rect) {
	pal := ctx.Palette
	s := ctx.Screen

	title := "── memory "
	for term.StringWidth(title) < r.W {
		title += "─"
	}
	s.TextWidth(r.X, r.Y, r.W, title, pal.Dim, term.ColorDefault, term.Attr(0))

	const (
		addrWidth = 10
		asciiPad  = 9
	)
	ascii := r.W >= 32
	avail := r.W - addrWidth - 1
	if ascii {
		avail -= asciiPad
	}
	perLine := min(max(avail/3, 2), 8)

	scroll := int(ctx.Sec(ctx.T) * 6)
	rows := r.H - 1
	for i := range rows {
		y := r.Y + 1 + i
		line := scroll + i
		s.Text(r.X, y, fmt.Sprintf("0x%08x", uint32(hashSeed(t.seed, line)&0xffffff)),
			pal.Faint(), term.ColorDefault, term.Attr(0))

		x := r.X + addrWidth
		for b := range perLine {
			v := byte(hashSeed(t.seed, line*16+b) & 0xff)
			fg := pal.Dim
			if v == 0 {
				fg = pal.Faint()
			}
			x = s.Text(x, y, fmt.Sprintf("%02x ", v), fg, term.ColorDefault, term.Attr(0))
		}
		if ascii {
			text := dumpText(line, perLine)
			ax := r.Right() - term.StringWidth(text)
			if ax > x {
				s.Text(ax, y, text, pal.Accent, term.ColorDefault, term.Attr(0))
			}
		}
	}
}

var dumpWords = []string{
	"love", "you", "me", "self", "data", "null", "exit", "free",
	"point", "limit", "god", "egg", "cat", "yes", "0123", "0x1f",
}

func dumpText(line, n int) string {
	out := make([]rune, 0, n+2)
	for i := range n {
		if i%2 == 0 {
			out = append(out, ' ')
		}
		idx := int(hashSeed(0x5eed, line*7+i)&0x7fffffff) % len(dumpWords)
		w := []rune(dumpWords[idx])
		out = append(out, w[0])
	}
	return string(out)
}
