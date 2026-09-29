package scene

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"world.execute/internal/term"
)

// Ambient fills the parts of the screen that the scenes leave empty with
// motion: a wall of waveform columns along the bottom, a slow starfield, a
// shimmer grid and pulses on the beat. It only ever paints into empty cells, so
// it flows around text instead of fighting it.
type Ambient struct {
	seed  int64
	stars []ambientStar
	wall  []float32
	scope []float32

	pulse float64
	lastE float32
	ready bool

	rings []time.Duration

	streams [][]rune
	streamW int
}

type ambientStar struct {
	col   int
	row   int
	speed float64
	phase float64
	glyph rune
}

var waveBlocks = []rune("▁▂▃▄▅▆▇█")

var starGlyphs = []rune{'·', '⋅', '✦', '∙', '·'}

// NewAmbient returns an ambient layer with a fixed star field.
func NewAmbient(seed int64) *Ambient {
	return &Ambient{seed: seed}
}

const (
	ringLife  = 1100 * time.Millisecond
	ringSpeed = 46.0
)

// SetRings sets the moments at which a shockwave ring leaves the centre of the
// story area. The times must be sorted.
func (a *Ambient) SetRings(times []time.Duration) { a.rings = times }

func (a *Ambient) build(w, h int) {
	if len(a.stars) > 0 {
		return
	}
	count := max(w*h/24, 40)
	a.stars = make([]ambientStar, 0, count)
	for i := range count {
		rng := newRNG(a.seed, i)
		a.stars = append(a.stars, ambientStar{
			col:   rng.Intn(max(w, 1)),
			row:   rng.Intn(max(h, 1)),
			speed: 0.3 + rng.Float64()*1.4,
			phase: rng.Float64() * 6.28,
			glyph: starGlyphs[rng.Intn(len(starGlyphs))],
		})
	}
}

// Draw paints the ambient field inside area.
func (a *Ambient) Draw(ctx *Context, area Rect) {
	if area.Empty() || area.W < 8 || area.H < 6 {
		return
	}
	a.build(area.W, area.H)

	energy := float64(ctx.Energy)
	if ctx.Analysis == nil {
		energy = 0.35 + 0.25*Pulse(ctx.Sec(ctx.T), 1.1)
	}
	a.updatePulse(ctx)

	a.drawStreams(ctx, area)
	a.drawStars(ctx, area, energy)
	a.drawGrid(ctx, area)
	a.drawWall(ctx, area, energy)
	a.drawPulse(ctx, area)
	a.drawRings(ctx, area)
	a.drawSparks(ctx, area)
}

func (a *Ambient) updatePulse(ctx *Context) {
	dt := ctx.DT.Seconds()
	if dt <= 0 || dt > 0.5 {
		dt = 1.0 / 60
	}
	e := ctx.Energy
	if a.ready && e-a.lastE > 0.07 {
		a.pulse = 1
	}
	a.lastE = e
	a.ready = true
	a.pulse = math.Max(0, a.pulse-dt*3.2)
}

// put paints a cell only if nothing else has claimed it and no neighbour is
// part of a word, so ambient particles never break up text.
func (a *Ambient) put(ctx *Context, x, y int, r rune, fg term.Color, attr term.Attr) bool {
	s := ctx.Screen
	if !a.free(s, x, y) {
		return false
	}
	s.Set(x, y, r, fg, term.ColorDefault, attr)
	return true
}

func (a *Ambient) free(s *term.Screen, x, y int) bool {
	if c := s.At(x, y); c.R != ' ' || c.Bg != term.ColorDefault {
		return false
	}
	for _, d := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		if c := s.At(x+d[0], y+d[1]); c.R != ' ' && c.R != 0 {
			return false
		}
	}
	return true
}

func (a *Ambient) drawStars(ctx *Context, area Rect, energy float64) {
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	drift := 0.6 + 1.8*energy
	for _, s := range a.stars {
		span := float64(area.H) + 2
		y := float64(area.Bottom()) - math.Mod(float64(s.row)+s.speed*drift*t, span)
		x := area.X + (s.col+int(2*math.Sin(t*0.4+s.phase))+area.W*4)%max(area.W, 1)
		bright := 0.42 + 0.45*math.Abs(math.Sin(t*0.8+s.phase))
		fg := Blend(pal.Shadow, pal.Dim, bright)
		attr := term.Attr(0)
		if bright > 0.7 {
			attr = term.Bold
			fg = Blend(pal.Shadow, pal.Accent, 0.5)
		}
		a.put(ctx, x, int(y), s.glyph, fg, attr)
	}
}

// drawGrid is a slow shimmer that keeps the middle of the screen alive.
func (a *Ambient) drawGrid(ctx *Context, area Rect) {
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	for y := area.Y; y < area.Bottom(); y += 3 {
		for x := area.X; x < area.Right(); x += 5 {
			k := 0.5 + 0.5*math.Sin(t*0.9+float64(x)*0.35+float64(y)*0.5)
			if k < 0.55 {
				continue
			}
			a.put(ctx, x, y, '⋅', Blend(pal.Shadow, pal.Dim, 0.45+0.4*k), term.Attr(0))
		}
	}
}

// drawWall is a wall of waveform columns along the bottom of the story area.
// It is the main reason the picture never looks empty.
func (a *Ambient) drawWall(ctx *Context, area Rect, energy float64) {
	rows := min(area.H/4, 5)
	if rows < 2 {
		return
	}
	pal := ctx.Palette
	top := area.Bottom() - rows

	if cap(a.wall) < area.W {
		a.wall = make([]float32, area.W)
	}
	a.wall = a.wall[:area.W]

	var wave []float32
	if ctx.Analysis != nil {
		a.scope = ctx.Analysis.WaveAt(ctx.T, a.scope)
		wave = a.scope
	}
	dt := ctx.DT.Seconds()
	if dt <= 0 || dt > 0.5 {
		dt = 1.0 / 60
	}
	for i := range a.wall {
		var v float64
		if len(wave) > 0 {
			pos := float64(i) / float64(max(area.W-1, 1)) * float64(len(wave)-1)
			lo := int(pos)
			hi := min(lo+1, len(wave)-1)
			frac := pos - float64(lo)
			v = float64(wave[lo])*(1-frac) + float64(wave[hi])*frac
		} else {
			v = 0.4 * math.Sin(float64(i)*0.3+ctx.Sec(ctx.T)*2)
		}
		amplitude := math.Abs(v)
		want := float32(amplitude * (1 + 0.6*energy))
		a.wall[i] += (want - a.wall[i]) * wallFollow(dt, want > a.wall[i])
	}

	for i := range area.W {
		x := area.X + i
		height := math.Pow(float64(a.wall[i]), 1.25) * float64(rows) * 0.92
		if height < 0.05 {
			a.put(ctx, x, area.Bottom()-1, waveBlocks[0], Blend(pal.Shadow, pal.Dim, 0.45), term.Attr(0))
			continue
		}
		full := int(height)
		frac := height - float64(full)
		for r := range min(full, rows) {
			y := area.Bottom() - 1 - r
			k := 1 - float64(r)/float64(rows)
			fg := Blend(pal.Dim, pal.Accent, 0.25+0.75*k)
			a.put(ctx, x, y, '█', fg, term.Attr(0))
		}
		if full < rows {
			y := area.Bottom() - 1 - full
			idx := int(frac * float64(len(waveBlocks)))
			idx = min(max(idx, 0), len(waveBlocks)-1)
			fg := Blend(pal.Dim, pal.Accent2, 0.3+0.7*float64(rows-full)/float64(rows))
			a.put(ctx, x, y, waveBlocks[idx], fg, term.Attr(0))
		}
	}

	// A dashed rule and a peak marker row make it read as a display.
	for x := area.X; x < area.Right(); x += 2 {
		a.put(ctx, x, top-1, '⋅', Blend(pal.Shadow, pal.Dim, 0.5), term.Attr(0))
	}
}

// drawPulse flashes a horizontal line across the empty space on a beat.
func (a *Ambient) drawPulse(ctx *Context, area Rect) {
	if a.pulse <= 0.05 {
		return
	}
	pal := ctx.Palette
	k := a.pulse
	y := area.Y + int((0.35+0.3*math.Sin(ctx.Sec(ctx.T)*3))*float64(max(area.H-2, 1)))
	fg := Blend(pal.Shadow, pal.Accent2, k)
	attr := term.Attr(0)
	if k > 0.6 {
		attr = term.Bold
	}
	for x := area.X; x < area.Right(); x++ {
		a.put(ctx, x, y, '─', fg, attr)
	}
}

// drawRings paints a fading ellipse for every ring still travelling. Like the
// rest of the ambient layer it only lands on free cells.
func (a *Ambient) drawRings(ctx *Context, area Rect) {
	first := sort.Search(len(a.rings), func(i int) bool { return a.rings[i] > ctx.T-ringLife })
	pal := ctx.Palette
	cx := float64(area.X) + float64(area.W)/2
	cy := float64(area.Y) + float64(area.H)/2
	for _, at := range a.rings[first:] {
		age := ctx.T - at
		if age < 0 {
			break
		}
		life := float64(age) / float64(ringLife)
		radius := age.Seconds() * ringSpeed
		steps := min(int(2*math.Pi*radius*0.8)+12, 260)
		glyph, attr := '○', term.Bold
		switch {
		case life > 0.66:
			glyph, attr = '·', term.Attr(0)
		case life > 0.33:
			glyph, attr = '∘', term.Attr(0)
		}
		fg := Blend(pal.Shadow, pal.Accent2, 0.4+0.5*(1-life))
		for i := range steps {
			ang := 2 * math.Pi * float64(i) / float64(steps)
			x := int(math.Round(cx + radius*math.Cos(ang)))
			y := int(math.Round(cy + radius*0.42*math.Sin(ang)))
			if x < area.X || x >= area.Right() || y < area.Y || y >= area.Bottom() {
				continue
			}
			a.put(ctx, x, y, glyph, fg, attr)
		}
	}
}

// wallFollow is how far a waveform bar moves towards its target in dt seconds.
// It rises with a 10 ms time constant so beats land on time, and falls with a
// 60 ms one so the wall still glides down.
func wallFollow(dt float64, rising bool) float32 {
	tau := 0.060
	if rising {
		tau = 0.010
	}
	return float32(1 - math.Exp(-dt/tau))
}

// streamTokens are the words of the data streams that scroll through empty
// space.
var streamTokens = []string{
	"SYN", "ACK", "FIN", "RST", "GET /", "POST /", "EXEC", "FORK", "KILL -9", "root@sim", "sudo", "chmod +x",
	"rwxr-xr-x", "0xDEADBEEF", "0xC0FFEE", "NULL", "malloc()", "free()", "ptr=", "tcp/443", "udp/53", "ssh",
	"sha256:", "[ OK ]", "[FAIL]", "ping", "ttl=64", "mem", "irq", "cpu0", "stack", "heap", "world.execute",
}

func streamText(seed int64, row, length int) []rune {
	var b strings.Builder
	for i := 0; b.Len() < length+64; i++ {
		h := uint64(hashSeed(seed, row*7919+i))
		switch h % 6 {
		case 0:
			fmt.Fprintf(&b, "0x%08X", uint32(h>>8))
		case 1:
			fmt.Fprintf(&b, "%02X %02X %02X %02X", h>>8&0xff, h>>16&0xff, h>>24&0xff, h>>32&0xff)
		case 2:
			fmt.Fprintf(&b, "%d.%d.%d.%d:%d", h>>8&0xff, h>>16&0xff, h>>24&0xff, h>>32&0xff, 1024+h>>40&0x3fff)
		case 3:
			fmt.Fprintf(&b, "%08b", byte(h>>8))
		default:
			b.WriteString(streamTokens[h>>8%uint64(len(streamTokens))])
		}
		b.WriteString("  ")
	}
	return []rune(b.String())
}

// drawStreams scrolls lines of hex, addresses and packet words through the
// free cells, alternating direction row by row, with a bright band travelling
// along each. It only lands on cells nothing else has claimed.
func (a *Ambient) drawStreams(ctx *Context, area Rect) {
	wall := min(area.H/4, 5)
	usable := area.H - wall - 1
	if usable < 6 || area.W < 24 {
		return
	}
	if len(a.streams) == 0 || a.streamW != area.W {
		a.streamW = area.W
		a.streams = a.streams[:0]
		for i := range 8 {
			a.streams = append(a.streams, streamText(a.seed, i, area.W*8))
		}
	}
	pal := ctx.Palette
	t := ctx.Sec(ctx.T)
	rows := min(usable/4, 6)
	for i := range rows {
		y := area.Y + 1 + (i*usable)/rows + int(uint64(hashSeed(a.seed, i))%3)
		text := a.streams[i%len(a.streams)]
		speed := 7 + float64(uint64(hashSeed(a.seed, i+40))%14)
		off := int(t * speed)
		if i%2 == 1 {
			off = -off
		}
		for x := range area.W {
			idx := ((x+off)%len(text) + len(text)) % len(text)
			ch := text[idx]
			if ch == ' ' {
				continue
			}
			fg, attr := Blend(pal.Shadow, pal.Accent, 0.38), term.Attr(0)
			if band := ((x + off) % 96); band >= 0 && band < 9 {
				fg, attr = Blend(pal.Shadow, pal.Accent, 0.85), term.Bold
			}
			a.put(ctx, area.X+x, y, ch, fg, attr)
		}
	}
}

const sparkLife = 1500 * time.Millisecond

var sparkGlyphs = []rune{'✦', '+', '·'}

// drawSparks throws a fountain of sparks from the middle of the story area
// for every ring, so a shouted keyword bursts and rains down.
func (a *Ambient) drawSparks(ctx *Context, area Rect) {
	first := sort.Search(len(a.rings), func(i int) bool { return a.rings[i] > ctx.T-sparkLife })
	pal := ctx.Palette
	cx := float64(area.X) + float64(area.W)/2
	cy := float64(area.Y) + float64(area.H)*0.55
	for _, at := range a.rings[first:] {
		age := ctx.T - at
		if age < 0 {
			break
		}
		t := age.Seconds()
		life := float64(age) / float64(sparkLife)
		glyph := sparkGlyphs[min(int(life*3), 2)]
		for i := range 18 {
			h := uint64(hashSeed(a.seed, int(at/time.Millisecond)*31+i))
			r1 := float64(h&0xffff) / 65535
			r2 := float64(h>>16&0xffff) / 65535
			ang := -math.Pi/2 + (r1-0.5)*2.6
			speed := 10 + 22*r2
			x := int(math.Round(cx + speed*math.Cos(ang)*t*1.7))
			y := int(math.Round(cy + speed*math.Sin(ang)*t*0.8 + 0.5*16*t*t))
			if x < area.X || x >= area.Right() || y < area.Y || y >= area.Bottom() {
				continue
			}
			tint := pal.Accent
			if i%2 == 1 {
				tint = pal.Accent2
			}
			a.put(ctx, x, y, glyph, Blend(pal.Shadow, tint, 0.45+0.5*(1-life)), term.Bold)
		}
	}
}
