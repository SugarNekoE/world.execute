package scene

import (
	"math"
	"math/rand"
	"time"

	"world.execute/internal/term"
)

// objectSpec describes one absurd instantiation from the song.
type objectSpec struct {
	at      time.Duration
	arg     string
	name    string
	gift    string
	icon    [IconHeight]string
	kwAt    time.Duration
	destiny string
}

// IconHeight is the height of the small object icons.
const IconHeight = 4

// ObjectScene instantiates the song's vegetables, cats and gods, one icon per
// object.
type ObjectScene struct {
	objects []objectSpec
}

// NewObjectScene returns an object scene.
func NewObjectScene(objects []objectSpec) *ObjectScene {
	return &ObjectScene{objects: objects}
}

// Draw renders the instantiations, with the newest one expanded.
func (o *ObjectScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() || len(o.objects) == 0 {
		return
	}

	type job struct {
		spec     objectSpec
		expanded bool
	}
	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)}, '▌')
	motion := Rect{}
	if area.W >= 84 && area.H >= 12 {
		width := min(area.W/2, area.W-52)
		motion = Rect{X: area.Right() - width, Y: area.Y + 1, W: width - 2, H: area.H - 2}
		area.W -= width + 1
	} else if area.W >= 36 && area.H >= 22 {
		motion = Rect{X: area.X + 3, Y: area.Y + area.H/2, W: area.W - 6, H: area.H/2 - 1}
		area.H /= 2
	}
	var jobs []job
	for i, spec := range o.objects {
		if spec.at > ctx.T {
			break
		}
		newest := i == len(o.objects)-1 || o.objects[i+1].at > ctx.T
		jobs = append(jobs, job{spec: spec, expanded: newest && ctx.Since(spec.at) < 6*time.Second})
	}

	rows := func(j job) int {
		if j.expanded {
			return IconHeight + 4
		}
		return 2
	}
	total, start := 0, len(jobs)
	for start > 0 && total+rows(jobs[start-1]) <= area.H {
		start--
		total += rows(jobs[start])
	}

	y := area.Y
	for _, j := range jobs[start:] {
		if y >= area.Bottom() {
			break
		}
		y = o.drawObject(ctx, area, y, j.spec, j.expanded)
	}
	if len(jobs) > 0 {
		drawObjectAnimation(ctx, motion, jobs[len(jobs)-1].spec)
	}
}

func (o *ObjectScene) drawObject(ctx *Context, area Rect, y int, spec objectSpec, expanded bool) int {
	pal := ctx.Palette
	s := ctx.Screen
	x := area.X + 2

	cx := s.Text(x, y, ">>> ", pal.Accent, term.ColorDefault, term.Bold)
	s.Text(cx, y, "world.instantiate("+spec.arg+")", pal.Text, term.ColorDefault, term.Attr(0))
	y++

	if !expanded {
		s.Text(x, y, "    new "+spec.name+"() · yield "+spec.gift, pal.Dim, term.ColorDefault, term.Attr(0))
		return y + 2
	}
	if y+IconHeight > area.Bottom() {
		return y
	}

	const boxW = 8
	s.Box(x, y, boxW, IconHeight+2, pal.Dim, term.Attr(0))
	for i, row := range spec.icon {
		for k, ch := range row {
			var fg term.Color
			switch ch {
			case '#':
				fg = pal.Accent2
			case '+':
				fg = pal.Kind
			case '.':
				fg = pal.Text
			default:
				continue
			}
			s.Set(x+2+k, y+i+1, ch, fg, term.ColorDefault, term.Bold)
		}
	}

	tx := x + boxW + 2
	s.Text(tx, y, "new "+spec.name+"()", pal.Text, term.ColorDefault, term.Bold)
	if spec.destiny != "" && y+IconHeight-1 < area.Bottom() {
		s.Text(tx, y+IconHeight-1, spec.destiny, pal.Dim, term.ColorDefault, term.Attr(0))
	}

	flash := spec.kwAt > 0 && ctx.T >= spec.kwAt && ctx.T < spec.kwAt+600*time.Millisecond
	label := "yield " + spec.gift
	if flash {
		s.Fill(tx, y+2, term.StringWidth(label), 1, ' ', pal.Shadow, pal.Accent, term.Attr(0))
		s.Text(tx, y+2, label, pal.Shadow, pal.Accent, term.Bold)
	} else {
		s.Text(tx, y+2, label, pal.Accent, term.ColorDefault, term.Bold)
	}
	return y + IconHeight + 3
}

const fragmentRotation = 2400 * time.Millisecond

var (
	fragmentsStart = stamp("1:58.38")
	fragmentsEnd   = stamp("2:05.67")
)

// DecayScene erases the log after the song's abandonment, or tears fragments of
// earlier frames across the screen.
type DecayScene struct {
	mode  string
	log   *LogView
	pool  []string
	freed string
}

// NewDeleteScene erases the lyric log as the singer is left alone.
func NewDeleteScene(log *LogView, freed string) *DecayScene {
	return &DecayScene{mode: "delete", log: log, freed: freed}
}

// NewFragmentScene scatters fragments of the source across the screen.
func NewFragmentScene(pool []string) *DecayScene {
	return &DecayScene{mode: "fragment", pool: pool}
}

// Draw renders the scene.
func (d *DecayScene) Draw(ctx *Context) {
	switch d.mode {
	case "delete":
		d.drawDelete(ctx)
	case "fragment":
		d.drawFragments(ctx)
	}
}

func (d *DecayScene) drawDelete(ctx *Context) {
	area := ctx.Area
	if area.Empty() || d.log == nil {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	body := Rect{X: area.X + 3, Y: area.Y, W: max(area.W-5, 1), H: max(area.H-3, 1)}
	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)}, '▌')

	alive := 0
	row := body.Y
	for _, e := range d.log.Entries {
		if e.At > ctx.T {
			break
		}
		erased := (ctx.Since(e.At).Seconds() - 4) / 5
		if erased >= 1 {
			continue
		}
		alive++
		if row >= body.Bottom() {
			continue
		}
		text := e.Line.Text
		if text == "" {
			text = e.Text
		}
		keep := len([]rune(text)) - int(erased*float64(len([]rune(text))))
		attr := term.Attr(0)
		if erased > 0 {
			attr = term.Strike
		}
		x := s.Text(body.X, row, "> ", pal.Dim, term.ColorDefault, term.Attr(0))
		WriteRunes(s, x, row, text, max(keep, 0), pal.Text, term.ColorDefault, attr)
		row++
	}
	if alive == 0 && Pulse(ctx.Sec(ctx.T), 1.2) > 0.4 {
		s.Set(body.CenterX(1), body.CenterY(1), '█', pal.Accent, term.ColorDefault, term.Attr(0))
	}
	if d.freed != "" {
		s.Text(body.Right()-term.StringWidth(d.freed), area.Y, d.freed, pal.Dim, term.ColorDefault, term.Attr(0))
	}
}

func (d *DecayScene) drawFragments(ctx *Context) {
	area := ctx.Area
	if area.Empty() || len(d.pool) == 0 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	usable := Rect{X: area.X, Y: area.Y, W: max(area.W-ctx.SideW, 1), H: area.H}
	t := ctx.Sec(ctx.T)
	corrupt := 0.25 * Progress(ctx.T, fragmentsStart, fragmentsEnd)
	slot := int(ctx.T / (120 * time.Millisecond))
	centre := usable.CenterY(1) - usable.Y
	rows := make([]int, 0, usable.H)
	for _, row := range rand.New(rand.NewSource(hashSeed(ctx.Seed, 0x51))).Perm(usable.H) {
		if row != centre || usable.H == 1 {
			rows = append(rows, row)
		}
	}

	visible := min(len(d.pool), len(rows))
	first := int(t/fragmentRotation.Seconds()) * visible
	for j := range visible {
		i := (first + j) % len(d.pool)
		text := d.pool[i]
		rng := rand.New(rand.NewSource(hashSeed(ctx.Seed, i*7+1)))
		row := rows[j]
		w := term.StringWidth(text)
		span := float64(usable.W + w)
		dir := float64(1 - 2*(i%2))
		pos := math.Mod(float64(rng.Intn(usable.W+w))+dir*(4+9*rng.Float64())*t, span)
		if pos < 0 {
			pos += span
		}
		x := usable.X + int(pos) - w
		k := 0.45 + 0.35*rng.Float64()
		attr := term.Attr(0)
		if rng.Intn(4) == 0 {
			attr = term.Reverse
		}
		if math.Mod(t+float64(i)*0.37, 1.9) < 0.12 {
			x += 3 * int(dir)
			attr = term.Reverse
		}
		shown := text
		if corrupt > 0 {
			shown = corruptText(text, corrupt, ctx.Seed^int64(i), slot)
		}
		clipText(s, usable, x, usable.Y+row, shown, Fade(pal.Dim, pal.Shadow, k), attr)
	}
	keep := d.pool[len(d.pool)/2]
	s.Text(area.CenterX(term.StringWidth(keep)), area.CenterY(1), keep, pal.Accent, term.ColorDefault, term.Bold)
}

func corruptText(text string, amount float64, seed int64, slot int) string {
	runes := []rune(text)
	for k, r := range runes {
		if r == ' ' {
			continue
		}
		h := uint64(hashSeed(seed, slot*131+k))
		if float64(h>>11)/float64(1<<53) < amount {
			runes[k] = rainAlphabet[h%uint64(len(rainAlphabet))]
		}
	}
	return string(runes)
}

func clipText(s *term.Screen, area Rect, x, y int, text string, fg term.Color, attr term.Attr) {
	if y < area.Y || y >= area.Bottom() {
		return
	}
	cx := x
	for _, r := range text {
		if cx >= area.Right() {
			break
		}
		if cx >= area.X {
			s.Set(cx, y, r, fg, term.ColorDefault, attr)
		}
		cx += term.RuneWidth(r)
	}
}

// TraceScene frames the lyric as a panic and a stack trace.
type TraceScene struct {
	code *CodeScene
}

// NewTraceScene returns a stack trace scene.
func NewTraceScene(title string, lines []codeLine) *TraceScene {
	return &TraceScene{code: NewCodeScene(title, lines)}
}

// Draw renders the panic header, the trace and a red frame.
func (t *TraceScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() || len(t.code.lines) == 0 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	s.Box(area.X, area.Y, area.W, area.H, pal.Err, term.Attr(0))
	t.code.Draw(ctx)

	age := ctx.Since(t.code.lines[0].at)
	if age >= 0 && age < 900*time.Millisecond && Pulse(ctx.Sec(ctx.T), 0.3) > 0.4 {
		s.Fill(area.X+1, area.Y, area.W-2, 1, ' ', pal.Text, pal.Err, term.Bold)
	}
}
