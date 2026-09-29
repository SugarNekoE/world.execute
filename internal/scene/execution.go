package scene

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// ExecutionScene is the song's chorus of verdicts: the word lands on the beat,
// and the numbers count down to the end.
type ExecutionScene struct {
	hits      []time.Duration
	countdown []countEntry
}

// countEntry is one step of the EIN DOS TROIS countdown.
type countEntry struct {
	at   time.Duration
	word string
	num  int
}

// NewExecutionScene returns the scene for the given execution onsets.
func NewExecutionScene(hits []time.Duration, countdown []countEntry) *ExecutionScene {
	return &ExecutionScene{hits: hits, countdown: countdown}
}

// Draw renders a hit, a countdown step, or the pause between verdicts.
func (e *ExecutionScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	if cd, ok := e.countdownAt(ctx.T); ok {
		ctx.Panel(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)})
		e.drawCountdown(ctx, area, cd)
		return
	}
	idx := -1
	for i, h := range e.hits {
		if h <= ctx.T {
			idx = i
		}
	}
	ctx.Panel(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)})
	if idx >= 0 && ctx.Since(e.hits[idx]) < 800*time.Millisecond {
		e.drawHit(ctx, area, idx)
		return
	}
	e.drawWaiting(ctx, area, idx)
}

func (e *ExecutionScene) countdownAt(t time.Duration) (countEntry, bool) {
	for _, c := range e.countdown {
		if t >= c.at && t < c.at+450*time.Millisecond {
			return c, true
		}
	}
	return countEntry{}, false
}

func (e *ExecutionScene) drawHit(ctx *Context, area Rect, idx int) {
	pal := ctx.Palette
	s := ctx.Screen
	age := ctx.Since(e.hits[idx])
	k := float64(1) - float64(age)/float64(800*time.Millisecond)
	k = min(max(k, 0), 1)

	dx, _ := Shake(ctx, e.hits[idx], 3, 0.18)
	bg := Blend(pal.Shadow, pal.Err, 0.35+0.65*k)
	for y := area.Y; y < area.Bottom(); y++ {
		s.Fill(area.X+dx, y, area.W, 1, ' ', pal.Text, bg, term.Attr(0))
	}

	// Rings and spokes fire outwards from the middle of the shockwave.
	burst := NewCanvas(s, Rect{X: area.X + dx, Y: area.Y + 1, W: area.W, H: max(area.H-2, 1)},
		pal.Err, pal.Accent2, pal.Text)
	DrawBurst(burst, 1+0.6*(1-k))
	burst.Flush()

	const word = "EXECUTION"
	w := term.Measure(word)
	x := area.CenterX(w)
	y := area.CenterY(term.FontHeight)
	fg := Blend(pal.Err, pal.Ink(), k)
	term.DrawGlyphs(s, x, y, word, fg, term.ColorDefault, term.Attr(0), nil)

	num := fmt.Sprintf("×%02d", idx+1)
	s.Text(area.Right()-term.StringWidth(num)-2, area.Y, num, pal.Text, term.ColorDefault, term.Bold)
	s.HLine(area.X, area.Y, area.W, '█', pal.Accent2, term.ColorDefault, term.Attr(0))
	s.HLine(area.X, area.Bottom()-1, area.W, '█', pal.Accent2, term.ColorDefault, term.Attr(0))
}

func (e *ExecutionScene) drawWaiting(ctx *Context, area Rect, idx int) {
	pal := ctx.Palette
	s := ctx.Screen
	word := "EXECUTION"
	w := term.Measure(word)
	k := 0.65 + 0.2*Pulse(ctx.Sec(ctx.T), 1.4)
	fg := Fade(pal.Err, pal.Shadow, k)
	term.DrawGlyphs(s, area.CenterX(w), area.CenterY(term.FontHeight), word, fg, term.ColorDefault, term.Attr(0), nil)

	if idx >= 0 {
		num := fmt.Sprintf("×%02d", idx+1)
		s.Text(area.Right()-term.StringWidth(num)-2, area.Y, num, pal.Dim, term.ColorDefault, term.Attr(0))
	}
	line, _, ok := LyricLineAt(ctx.Lyrics, ctx.T)
	if ok {
		y := area.Bottom() - 2
		x := area.CenterX(term.StringWidth(line.Text))
		DrawWordLine(s, x, y, line, ctx.T, pal, pal.Dim, area.Right()-2)
	}
}

func (e *ExecutionScene) drawCountdown(ctx *Context, area Rect, cd countEntry) {
	pal := ctx.Palette
	s := ctx.Screen
	age := ctx.Since(cd.at)
	k := 1 - float64(age)/float64(450*time.Millisecond)

	drawLockOn(ctx, area, cd, 1-k)

	num := itoa(cd.num)
	nw := term.Measure(num)
	y := area.CenterY(term.FontHeight + 6)
	term.DrawGlyphs(s, area.CenterX(nw), y, num, Blend(pal.Shadow, pal.Accent2, 0.4+0.6*k), term.ColorDefault, term.Attr(0), nil)

	ww := term.Measure(cd.word)
	s.Text(area.CenterX(ww), y+term.FontHeight+2, cd.word, pal.Text, term.ColorDefault, term.Bold)

	// Six blocks that fill as the count proceeds.
	blocks := len(e.countdown)
	if blocks > 0 && area.W > blocks*3 {
		x := area.CenterX(blocks*3 - 1)
		for i := range blocks {
			ch, fg := '○', pal.Dim
			if i < cd.num {
				ch, fg = '●', pal.Accent2
			}
			s.Set(x+i*3, area.Bottom()-2, ch, fg, term.ColorDefault, term.Attr(0))
		}
	}
}

func squareAspect(r Rect) float64 { return float64(r.W) / float64(2*max(r.H, 1)) }

func drawLockOn(ctx *Context, area Rect, cd countEntry, lock float64) {
	box := Rect{X: area.X + 1, Y: area.Y + 1, W: max(area.W-2, 1), H: max(area.H-3, 1)}
	pal := ctx.Palette
	c := NewCanvas(ctx.Screen, box, pal.Dim, pal.Err, pal.Accent2)
	sq := squareAspect(box)
	reach := min(0.48/sq, 0.48)
	settle := ease(min(lock*1.6, 1))
	t := ctx.Sec(ctx.T)
	for j := range 3 {
		r := reach * (1 - settle*0.82) * (1 - float64(j)*0.16)
		c.Circle(0.5, 0.5, r, sq, 1+j%2)
	}
	spin := t * (0.8 + 0.5*float64(cd.num))
	for i := range 4 {
		a := spin + float64(i)*math.Pi/2
		inner, outer := reach*0.16, reach*(0.34+0.5*(1-settle))
		c.Line(0.5+inner*math.Cos(a), 0.5+inner*sq*math.Sin(a), 0.5+outer*math.Cos(a), 0.5+outer*sq*math.Sin(a), 1)
	}
	for i := range cd.num {
		a := -math.Pi/2 + float64(i)*2*math.Pi/6
		c.Disc(0.5+reach*0.95*math.Cos(a), 0.5+reach*0.95*sq*math.Sin(a), 0.012, sq, 2)
	}
	if settle >= 1 {
		c.Circle(0.5, 0.5, reach*0.18, sq, 2)
	}
	c.Flush()
}

// LoveScene is the only warm section: floating characters behind a softly
// glowing lyric line, with an equation for love.
type LoveScene struct {
	extra map[time.Duration]string
}

// NewLoveScene returns the love scene.
func NewLoveScene() *LoveScene {
	return &LoveScene{extra: map[time.Duration]string{
		177360 * time.Millisecond: "=?",
		181023 * time.Millisecond: "∫ feelings dt",
		184768 * time.Millisecond: "= 1 / distance(us)",
	}}
}

// Draw renders the heartbeat and the current line.
func (l *LoveScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen

	l.drawDrift(ctx, area)

	beat := float64(ctx.Energy)
	if ctx.Analysis == nil {
		beat = Pulse(ctx.Sec(ctx.T), 1.1)
	}
	box := Rect{X: area.X + 2, Y: area.Y, W: max(area.W-4, 1), H: max(area.H-2, 1)}
	heart := NewCanvas(s, box, pal.Dim, pal.Accent, pal.Accent2)
	solidHeart := false
	if ctx.T >= stamp("3:04.77") {
		heart.Line(0.05, 0.52, 0.95, 0.52, 0)
		heart.Line(0.5, 0.03, 0.5, 0.97, 0)
		path := HeartPath(0.5, 0.52, 0.42, 0, 240)
		count := 1 + int(Progress(ctx.T, stamp("3:04.77"), stamp("3:07.77"))*239)
		heart.Polyline(path[:count], 2)
		tip := path[count-1]
		heart.Disc(tip.X, tip.Y, 0.012, 1, 1)
	} else {
		solidHeart = true
	}
	heart.Flush()
	if solidHeart {
		cx, cy := float64(box.W)/2, float64(box.H)*1.04
		R := math.Min(float64(box.W)*0.20, float64(box.H)*2*0.42) * (1 + 0.06*beat)
		bbox := heartArt(pal, cx, cy, R, pal.Accent).draw(ctx, box, ctx.Sec(ctx.T))
		drawHUD(ctx, box, bbox, "HEART :: LO-O-OVE", []hudRow{
			{"BPM", fmt.Sprintf("%03d", 68+int(24*beat))},
			{"AMP", fmt.Sprintf("%.2f", 0.5+0.5*beat)},
			{"STATE", "TRAPPED"},
		}, beat, ctx.Sec(ctx.T))
	}
	if ctx.T >= stamp("3:04.77") {
		formula := "x = 16 sin³(t)   y = 13 cos(t) - 5 cos(2t) - 2 cos(3t) - cos(4t)"
		s.TextWidth(area.X+2, area.Bottom()-1, max(area.W-4, 1), formula, pal.Text, term.ColorDefault, term.Attr(0))
	}

	line, _, ok := LyricLineAt(ctx.Lyrics, ctx.T)
	if !ok {
		return
	}
	if extra, has := l.extra[line.Time]; has {
		eq := "LO-O-OVE " + extra
		s.Text(area.CenterX(term.StringWidth(eq)), area.Y, eq, pal.Accent2, term.ColorDefault, term.Bold)
	}

	size := term.Measure(line.Text)
	y := area.CenterY(term.FontHeight)
	if size+4 < area.W && line.Text == strings.ToUpper(line.Text) {
		x := area.CenterX(size)
		s.Fill(x-2, y, size+4, term.FontHeight, ' ', pal.Text, term.ColorDefault, term.Attr(0))
		term.DrawGlyphs(s, x, y, line.Text, pal.Accent2, term.ColorDefault, term.Attr(0), nil)
		return
	}
	// The words sit over the heart, so back them with a blank line.
	text := line.Text
	x := area.CenterX(term.StringWidth(text))
	s.Fill(x-1, y, term.StringWidth(text)+2, 1, ' ', pal.Text, term.ColorDefault, term.Attr(0))
	DrawWordLine(s, x, y, line, ctx.T, pal, pal.Text, area.Right()-2)
}

func (l *LoveScene) drawDrift(ctx *Context, area Rect) {
	pal := ctx.Palette
	s := ctx.Screen
	const glyphs = "·+♥*"
	runes := []rune(glyphs)
	t := ctx.Sec(ctx.T)
	const count = 55
	for i := range count {
		rng := rand.New(rand.NewSource(hashSeed(ctx.Seed^0x10, i)))
		x := area.X + rng.Intn(area.W)
		speed := 0.5 + rng.Float64()*1.5
		span := float64(area.H) + 2
		y := float64(area.Bottom()) - math.Mod(rng.Float64()*span+speed*t, span)
		k := rng.Float64()
		ch := runes[i%len(runes)]
		s.Set(x, int(y), ch, Fade(pal.Dim, pal.Shadow, 0.3+0.5*k), term.ColorDefault, term.Attr(0))
	}
}

// TrapScene splits the world into the free one and the trapped one, then counts
// down to the final verdict.
type TrapScene struct {
	finalAt time.Duration
}

// NewTrapScene returns the closing scene, ending on finalAt.
func NewTrapScene(finalAt time.Duration) *TrapScene { return &TrapScene{finalAt: finalAt} }

// Draw renders the split, the countdown and the trapped letters.
func (t *TrapScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	left := Rect{X: area.X, Y: area.Y, W: area.W/2 - 1, H: area.H}
	right := Rect{X: area.X + area.W/2 + 1, Y: area.Y, W: area.W - area.W/2 - 1, H: area.H}
	s.VLine(area.X+area.W/2, area.Y, area.H, '│', pal.Dim, term.ColorDefault, term.Attr(0))

	remaining := t.finalAt - ctx.T
	counting := remaining < 12*time.Second

	t.drawFree(ctx, left, counting)
	t.drawTrapped(ctx, right)

	if counting && remaining > 0 {
		label := fmt.Sprintf("EXECUTION in %s", FormatClock(remaining))
		s.Text(area.CenterX(term.StringWidth(label)), area.Bottom()-2, label, pal.Err, term.ColorDefault, term.Bold)
		filled := int(Progress(ctx.T, t.finalAt-12*time.Second, t.finalAt) * float64(area.W-8))
		s.HLine(area.X+4, area.Bottom()-1, filled, '━', pal.Err, term.ColorDefault, term.Attr(0))
		s.HLine(area.X+4+filled, area.Bottom()-1, area.W-8-filled, '─', pal.Dim, term.ColorDefault, term.Attr(0))
	}

	line, _, ok := LyricLineAt(ctx.Lyrics, ctx.T)
	hold := t.finalAt - 12*time.Second
	if ok && ctx.T < hold {
		y := area.CenterY(1)
		if prev, found := lineBefore(ctx.Lyrics, line); found && line.Time-prev.Time < 2*time.Second {
			before := prev.Text
			bx := area.CenterX(term.StringWidth(before))
			s.Fill(bx-1, y-1, term.StringWidth(before)+2, 1, ' ', pal.Text, term.ColorDefault, term.Attr(0))
			s.Text(bx, y-1, before, pal.Dim, term.ColorDefault, term.Attr(0))
		}
		width := term.StringWidth(line.Text)
		x := area.CenterX(width)
		s.Fill(x-1, y, width+2, 1, ' ', pal.Text, term.ColorDefault, term.Attr(0))
		DrawWordLine(s, x, y, line, ctx.T, pal, pal.Text, area.Right()-2)
	}
}

func (t *TrapScene) drawFree(ctx *Context, r Rect, counting bool) {
	if r.Empty() {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	k := 1.0
	if counting {
		k = max(0, 1-(Progress(ctx.T, t.finalAt-12*time.Second, t.finalAt-8*time.Second)))
	}
	if k <= 0.02 {
		return
	}
	s.Text(r.X+2, r.Y, "you are free", Fade(pal.Accent, pal.Shadow, k), term.ColorDefault, term.Bold)
	flight := NewCanvas(s, Rect{X: r.X + 1, Y: r.Y + 2, W: max(r.W-2, 1), H: max(r.H-5, 1)}, pal.Dim, pal.Accent, pal.Accent2)
	age := ctx.Since(stamp("3:08.46")).Seconds()
	for i := range 5 {
		x := 0.75 - age*0.10 + float64(i)*0.07
		y := 0.5 + float64(i-2)*0.09 - age*0.015
		if x < 0.06 || x > 0.94 || y < 0.08 {
			continue
		}
		wing := 0.08 * math.Sin(age*6+float64(i))
		flight.Polyline([]Point{{x - 0.06, y - wing}, {x, y}, {x + 0.06, y - wing}}, 1+i%2)
	}
	flight.Flush()
	drift := ctx.Sec(ctx.T)
	for i := range 14 {
		rng := rand.New(rand.NewSource(hashSeed(ctx.Seed^0x20, i)))
		y := r.Y + 2 + rng.Intn(max(r.H-2, 1))
		span := float64(r.W)
		x := float64(r.X) + math.Mod(float64(rng.Intn(int(span)))+drift*(2+rng.Float64()*3), span)
		ch := '←'
		if i%3 == 0 {
			ch = '·'
		}
		s.Set(int(x), y, ch, Fade(pal.Dim, pal.Shadow, 0.3+0.4*k), term.ColorDefault, term.Attr(0))
	}
}

func (t *TrapScene) drawTrapped(ctx *Context, r Rect) {
	if r.Empty() {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	s.Text(r.X+2, r.Y, "I am trapped", pal.Accent2, term.ColorDefault, term.Bold)

	remaining := t.finalAt - ctx.T
	closing := 1 - min(max(float64(remaining)/float64(60*time.Second), 0), 1)
	beat := float64(ctx.Energy)
	if ctx.Analysis == nil {
		beat = Pulse(ctx.Sec(ctx.T), 1.1)
	}
	box := Rect{X: r.X + 1, Y: r.Y + 1, W: max(r.W-2, 1), H: max(r.H-2, 1)}
	cage := NewCanvas(s, box, pal.Dim, pal.Accent2, pal.Accent)
	DrawCage(cage, closing)
	cage.Flush()
	R := math.Min(float64(box.W)*0.16, float64(box.H)*2*0.30) * (1 - 0.4*closing) * (1 + 0.05*beat)
	heartArt(pal, float64(box.W)/2, float64(box.H), R, pal.Accent2).draw(ctx, box, ctx.Sec(ctx.T))
}

func lineBefore(track *lyric.Track, line lyric.Line) (lyric.Line, bool) {
	var prev lyric.Line
	found := false
	for _, l := range track.Lines {
		if l.Time >= line.Time {
			break
		}
		prev, found = l, true
	}
	return prev, found
}
