package scene

import (
	"fmt"
	"math"
	"time"

	"world.execute/internal/term"
)

// shapeSpec is one stanza of the first verse: the lyric says the shape, the
// screen draws it.
type shapeSpec struct {
	at      time.Duration
	name    string
	code    string
	keyword string
	kwAt    time.Duration
	kind    string
}

// ShapeScene draws the geometry the first verse sings about.
type ShapeScene struct {
	shapes []shapeSpec
	title  string
}

// NewShapeScene returns the geometry scene.
func NewShapeScene(title string, shapes []shapeSpec) *ShapeScene {
	return &ShapeScene{shapes: shapes, title: title}
}

// Draw renders the current shape, its code and the lyric footer.
func (s *ShapeScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	idx := 0
	for i, sh := range s.shapes {
		if sh.at <= ctx.T {
			idx = i
		}
	}
	cur := s.shapes[idx]
	if cur.at > ctx.T {
		return
	}

	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)}, '▌')
	pal := ctx.Palette
	scr := ctx.Screen

	head := "// " + s.title
	scr.Text(area.X+3, area.Y, head, pal.Dim, term.ColorDefault, term.Attr(0))

	box := Rect{X: area.X + 3, Y: area.Y + 1, W: max(area.W-6, 1), H: max(area.H-4, 1)}
	if box.H < 4 {
		s.drawFooter(ctx, area, cur)
		return
	}
	s.drawShape(ctx, box, cur)
	s.drawReadout(ctx, box, cur)

	// The stanza's code sits on the bottom border of the plot.
	code := cur.code
	scr.Text(box.X+1, box.Bottom(), code, pal.Dim, term.ColorDefault, term.Attr(0))
	if cur.keyword != "" && ctx.T >= cur.kwAt && ctx.T < cur.kwAt+900*time.Millisecond {
		label := " " + cur.keyword + " "
		scr.Text(box.X+1+term.StringWidth(code)+2, box.Bottom(), label, pal.Shadow, pal.Accent2, term.Bold)
	}
	s.drawFooter(ctx, area, cur)
}

func (s *ShapeScene) drawFooter(ctx *Context, area Rect, cur shapeSpec) {
	line, _, ok := LyricLineAt(ctx.Lyrics, ctx.T)
	if !ok || line.Text == "" {
		return
	}
	y := area.Bottom() - 1
	if y <= area.Y {
		return
	}
	x := ctx.Screen.Text(area.X+3, y, "// ", ctx.Palette.Dim, term.ColorDefault, term.Attr(0))
	DrawWordLine(ctx.Screen, x, y, line, ctx.T, ctx.Palette, ctx.Palette.Text, area.Right()-2)
}

type readoutRow struct{ label, value string }

func verseReadout(ctx *Context, cur shapeSpec) []readoutRow {
	prog := Progress(ctx.T, cur.at, cur.kwAt)
	t := ctx.Since(cur.at).Seconds()
	switch cur.kind {
	case "points":
		return []readoutRow{
			{"n", "160"},
			{"dim", "2"},
			{"spread", fmt.Sprintf("%.2f", ctx.Energy)},
		}
	case "circle":
		r := 0.5 + 2.5*prog
		return []readoutRow{
			{"r", fmt.Sprintf("%.2f", r)},
			{"C = 2πr", fmt.Sprintf("%.2f", 2*math.Pi*r)},
			{"A = πr²", fmt.Sprintf("%.2f", math.Pi*r*r)},
		}
	case "sine":
		return []readoutRow{
			{"y", "sin(x − 4t)"},
			{"dy/dx", fmt.Sprintf("%+.2f", math.Cos(-4*t))},
			{"tangent", "slope at cursor"},
		}
	case "limit":
		x := math.Pow(10, 1+3*prog)
		return []readoutRow{
			{"x", fmt.Sprintf("%.0f", x)},
			{"1/x", fmt.Sprintf("%.4f", 1/x)},
			{"lim x→∞", "0"},
		}
	}
	return nil
}

// drawReadout prints the live numbers of the current shape in the plot's
// upper corner.
func (s *ShapeScene) drawReadout(ctx *Context, box Rect, cur shapeSpec) {
	rows := verseReadout(ctx, cur)
	const width = 28
	if len(rows) == 0 || box.W < 64 || box.H < 9 {
		return
	}
	pal := ctx.Palette
	scr := ctx.Screen
	x := box.Right() - width - 1
	scr.Fill(x-1, box.Y, width+2, len(rows), ' ', pal.Text, term.ColorDefault, term.Attr(0))
	for i, row := range rows {
		scr.Text(x, box.Y+i, row.label, pal.Dim, term.ColorDefault, term.Attr(0))
		vx := x + width - term.StringWidth(row.value)
		scr.Text(vx, box.Y+i, row.value, pal.Accent2, term.ColorDefault, term.Bold)
	}
}

func (s *ShapeScene) drawShape(ctx *Context, box Rect, cur shapeSpec) {
	pal := ctx.Palette
	c := NewCanvas(ctx.Screen, box, pal.Dim, pal.Accent, pal.Accent2, pal.Kind)
	defer c.Flush()

	energy := float64(ctx.Energy)
	if ctx.Analysis == nil {
		energy = 0.4 + 0.3*Pulse(ctx.Sec(ctx.T), 1.2)
	}
	energy = min(max(energy, 0), 1)

	// A baseline makes the plot read as an instrument panel.
	c.Line(0, 0.5, 1, 0.5, 0)

	switch cur.kind {
	case "points":
		s.drawPoints(ctx, c, energy)
	case "circle":
		s.drawCircle(ctx, c, energy, Progress(ctx.T, cur.at, cur.kwAt))
	case "sine":
		s.drawSine(ctx, c, energy)
	case "limit":
		s.drawLimit(ctx, c, Progress(ctx.T, cur.at, cur.kwAt))
	default:
		c.Circle(0.5, 0.5, 0.3, Aspect, 1)
	}
}

// drawPoints scatters points in a disc, the song's "set of points".
func (s *ShapeScene) drawPoints(ctx *Context, c *Canvas, energy float64) {
	count := 160
	rot := ctx.Sec(ctx.T) * 0.25
	radius := 0.22 + 0.22*energy
	golden := math.Pi * (3 - math.Sqrt(5))
	pts := make([]Point, 0, count)
	for i := range count {
		f := float64(i) / float64(count)
		a := golden*float64(i) + rot
		r := radius * math.Sqrt(f)
		jitter := 0.02 * math.Sin(ctx.Sec(ctx.T)*3+float64(i))
		pts = append(pts, Point{
			X: 0.5 + r*math.Cos(a) + jitter,
			Y: 0.5 + r*Aspect*math.Sin(a) - jitter*Aspect,
		})
	}
	c.Points(pts, 1)
	// The line that becomes the circle, drawn through the outermost points.
	c.Circle(0.5, 0.5, radius, Aspect, 2)
}

// drawCircle traces its own circumference as the line is sung.
func (s *ShapeScene) drawCircle(ctx *Context, c *Canvas, energy, progress float64) {
	radius := 0.3 + 0.08*energy
	c.Circle(0.5, 0.5, radius, Aspect, 2)
	c.Circle(0.5, 0.5, radius*0.999, Aspect, 2)
	end := progress * 2 * math.Pi
	c.Arc(0.5, 0.5, radius, Aspect, -math.Pi/2, -math.Pi/2+end, 1)

	// The radius sweeps round with the trace.
	ca := -math.Pi/2 + end
	tipX := 0.5 + radius*math.Cos(ca)
	tipY := 0.5 + radius*Aspect*math.Sin(ca)
	c.Line(0.5, 0.5, tipX, tipY, 1)
	c.Set(0.5, 0.5, 3)
	c.Set(tipX, tipY, 3)
}

// drawSine is a travelling wave whose frequency follows the music.
func (s *ShapeScene) drawSine(ctx *Context, c *Canvas, energy float64) {
	bands := ctx.Analysis.BandsAt(ctx.T, ctx.Bands)
	treble := 0.0
	if len(bands) > 0 {
		for _, b := range bands[len(bands)/2:] {
			treble += float64(b)
		}
		treble /= float64(len(bands) - len(bands)/2)
	}
	freq := 1.6 + 3.4*treble + 1.5*energy
	phase := ctx.Sec(ctx.T) * 1.4
	amp := 0.22 + 0.22*energy
	c.Func(func(x float64) float64 {
		return 0.5 - amp*math.Sin(2*math.Pi*(freq*x+phase))
	}, 1, 420)
	// Tangents at the crests, ready for the next line.
	for i := -2; i <= 2; i++ {
		x := (float64(i)*0.25 + math.Mod(phase, 1)) / 1
		if x < 0 || x > 1 {
			continue
		}
		y := 0.5 - amp*math.Sin(2*math.Pi*(freq*x+phase))
		c.Line(x, y, x, min(y+0.18, 1), 2)
	}
	c.Line(0.5, 0, 0.5, 1, 0)
}

// drawLimit shows a curve that approaches a wall it never reaches.
func (s *ShapeScene) drawLimit(ctx *Context, c *Canvas, progress float64) {
	reach := 0.15 + 0.8*progress
	c.Func(func(x float64) float64 {
		if x > reach {
			return 1.4
		}
		return 0.98 - 0.85/(1+12*x)
	}, 1, 420)
	wall := 0.98
	for y := 0.0; y < 1; y += 0.06 {
		c.Line(wall, y, wall, y+0.03, 2)
	}
	// The gap that shrinks as the section goes on.
	gap := max(0.02, 0.35-0.3*progress)
	x := reach
	y := 0.98 - 0.85/(1+12*x)
	c.Line(max(x-gap, 0), y, min(x+gap, 1), y, 3)
}

// HeartPath returns a parametric heart scaled into 0..1 around cx, cy.
func HeartPath(cx, cy, size float64, phase float64, samples int) []Point {
	pts := make([]Point, 0, samples)
	for i := range samples {
		t := 2 * math.Pi * float64(i) / float64(samples)
		x := math.Pow(math.Sin(t), 3)
		y := 13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t)
		// The parametrisation spans roughly -1..1 in x and -17..12 in y.
		nx := x / 1.1
		ny := -y / 17.0
		pts = append(pts, Point{
			X: cx + nx*size*math.Cos(phase),
			Y: cy + ny*size*Aspect,
		})
	}
	return pts
}

// DrawScope draws a live waveform strip, which is what keeps the whole piece
// moving even while a section is just text.
func DrawScope(ctx *Context, r Rect, gain float64) {
	if r.W < 8 || r.H < 1 || ctx.Analysis == nil {
		return
	}
	pal := ctx.Palette
	c := NewCanvas(ctx.Screen, r, pal.Faint(), pal.Accent, pal.Accent2)
	defer c.Flush()
	c.Line(0, 0.5, 1, 0.5, 0)
	ctx.Wave = ctx.Analysis.WaveAt(ctx.T, ctx.Wave)
	c.Polyline(wavePath(ctx.Wave, gain), 1)
}

// DrawMeter draws a compact spectrum meter, one row per band, so text heavy
// sections still move.
func DrawMeter(ctx *Context, r Rect) {
	if r.Empty() || ctx.Analysis == nil || r.H < 3 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	bands := ctx.Analysis.BandsAt(ctx.T, ctx.Bands)
	rows := min(len(bands), r.H)
	for i := range rows {
		y := r.Y + i
		s.Text(r.X, y, pad(i+1, 2), pal.Dim, term.ColorDefault, term.Attr(0))
		track := r.W - 4
		w := int(float64(bands[i]) * float64(track))
		fg := Blend(pal.Accent, pal.Accent2, float64(i)/float64(max(len(bands)-1, 1)))
		for x := range track {
			if x < w {
				s.Set(r.X+3+x, y, '█', fg, term.ColorDefault, term.Attr(0))
			} else if x == w {
				s.Set(r.X+3+x, y, '▏', Blend(fg, pal.Shadow, 0.5), term.ColorDefault, term.Attr(0))
			} else {
				s.Set(r.X+3+x, y, '·', pal.Faint(), term.ColorDefault, term.Attr(0))
			}
		}
	}
}

// DrawCage draws bars closing in from both sides.
func DrawCage(c *Canvas, close float64) {
	gap := math.Max(1-close, 0) * 0.5
	for i := 0; i < 7; i++ {
		x := float64(i) / 8 * (0.5 - gap)
		c.Line(x, 0, x, 1, 1)
		c.Line(1-x, 0, 1-x, 1, 1)
	}
}

// DrawBurst draws expanding rings and spokes, used for every EXECUTION.
func DrawBurst(c *Canvas, k float64) {
	if k <= 0 {
		return
	}
	for i := range 3 {
		r := (k + float64(i)*0.3) * 0.85
		if r <= 0.01 || r > 1.3 {
			continue
		}
		c.Circle(0.5, 0.5, r, Aspect, 1+i%2)
	}
	spokes := 28
	for i := range spokes {
		a := 2 * math.Pi * float64(i) / float64(spokes)
		r0 := 0.05 + 0.35*k
		r1 := r0 + 0.22*k
		c.Line(0.5+r0*math.Cos(a), 0.5+r0*Aspect*math.Sin(a),
			0.5+r1*math.Cos(a), 0.5+r1*Aspect*math.Sin(a), 2)
	}
}
