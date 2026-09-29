package scene

import (
	"math"
	"math/rand"
	"time"

	"world.execute/internal/term"
)

// Rain is the falling code background. Particle state is derived from the seed
// and the playback time, so seeking lands on the same frame every run.
type Rain struct {
	seed int64
	cols []rainColumn
	w, h int
}

type rainColumn struct {
	token  []rune
	offset float64
	speed  float64
	active bool
	bold   bool
	lane   float64
}

var rainAlphabet = []rune("0123456789ABCDEFabcdef<>{}[]()/*+-=#$%&|;:._,~^!?")

var loveGlyphs = []rune("♥·♥+♥*∙♥")

var rainTokens = []string{
	"world.execute(me);", "self", "return", "if", "else", "null", "true", "false",
	"0x1F", "SIGSEGV", "AC/DC", "love", "SIMULATION", "EXECUTION", "Init", "data",
	"protect", "dizzy", "limit", "∞", "0.1", "yield", "loop", "break", "malloc",
}

// NewRain builds a rain layer seeded with seed.
func NewRain(seed int64) *Rain { return &Rain{seed: seed} }

func hashSeed(seed int64, n int) int64 {
	x := uint64(seed) ^ uint64(n)*0x9E3779B97F4A7C15
	x ^= x >> 30
	x *= 0xBF58476D1CE4E5B9
	x ^= x >> 27
	x *= 0x94D049BB133111EB
	x ^= x >> 31
	return int64(x)
}

// newRNG returns a generator seeded by a hash of seed and n.
func newRNG(seed int64, n int) *rand.Rand { return rand.New(rand.NewSource(hashSeed(seed, n))) }

func (r *Rain) build(w int) {
	if r.w == w && r.cols != nil {
		return
	}
	r.w = w
	r.cols = make([]rainColumn, w)
	for x := range w {
		rng := rand.New(rand.NewSource(hashSeed(r.seed, x)))
		c := rainColumn{
			offset: rng.Float64(),
			speed:  3 + rng.Float64()*11,
			active: rng.Float64() < 0.62,
			bold:   rng.Float64() < 0.12,
			lane:   math.Mod(float64(x+1)*0.6180339887+rng.Float64()*0.05, 1),
		}
		if t := rainTokens[rng.Intn(len(rainTokens))]; rng.Float64() < 0.55 {
			c.token = []rune(t)
		} else {
			n := 4 + rng.Intn(6)
			c.token = make([]rune, n)
			for i := range c.token {
				c.token[i] = rainAlphabet[rng.Intn(len(rainAlphabet))]
			}
		}
		r.cols[x] = c
	}
}

// Draw renders the rain inside area. density scales how many columns are lit,
// speed scales the fall rate.
func (r *Rain) Draw(ctx *Context, area Rect, density, speed float64) {
	if area.Empty() || density <= 0 {
		return
	}
	r.build(area.W)
	energy := float64(ctx.Energy)
	if ctx.Analysis == nil {
		energy = 0.35
	}
	boost := 0.55 + 1.5*energy
	t := ctx.Sec(ctx.T)
	love := ctx.Palette.Name == "love"
	for i, c := range r.cols {
		if !c.active || c.lane > density {
			continue
		}
		x := area.X + i
		n := float64(len(c.token))
		span := float64(area.H) + n
		head := math.Mod(c.offset*span+c.speed*boost*t, span) - n
		for k, ch := range c.token {
			y := area.Y + int(head) - k
			if y < area.Y || y >= area.Bottom() {
				continue
			}
			if love {
				ch = loveGlyphs[(int(ch)+k)%len(loveGlyphs)]
			}
			var fg term.Color
			a := term.Attr(0)
			switch {
			case k == 0:
				fg = Blend(ctx.Palette.Dim, ctx.Palette.Accent, 0.55+0.45*energy)
				a = term.Bold
			case k < 3:
				fg = ctx.Palette.Dim
			default:
				fg = Fade(ctx.Palette.Dim, ctx.Palette.Shadow, 0.45)
			}
			if c.bold {
				a |= term.Bold
			}
			ctx.Screen.Set(x, y, ch, fg, term.ColorDefault, a)
		}
	}
}

// Glitch tears and corrupts the picture on impacts and transients.
type Glitch struct {
	hits []time.Duration
}

// NewGlitch returns a glitch layer that fires at the given moments.
func NewGlitch(hits ...time.Duration) *Glitch { return &Glitch{hits: hits} }

// Amount returns the strength of the glitch at the current time, combining the
// section baseline, the impact table and the loudness of the track.
func (g *Glitch) Amount(ctx *Context, base float64) float64 {
	amount := base
	for _, h := range g.hits {
		age := (ctx.T - h).Seconds()
		if age < 0 || age > 0.6 {
			continue
		}
		amount = math.Max(amount, math.Exp(-age/0.16))
	}
	if ctx.Analysis != nil {
		if e := float64(ctx.Energy); e > 0.72 {
			amount = math.Max(amount, (e-0.72)*1.8)
		}
	}
	return math.Min(amount, 1.6)
}

// Impact is how hard the latest listed hit is still landing at t, from 1 at the
// hit falling to 0 a quarter of a second later.
func (g *Glitch) Impact(t time.Duration) float64 {
	best := 0.0
	for _, h := range g.hits {
		age := (t - h).Seconds()
		if age < 0 || age > 0.25 {
			continue
		}
		best = math.Max(best, math.Exp(-age/0.09))
	}
	return best
}

// Draw applies the glitch to the area. Amounts below a small floor are ignored
// so that ordinary playback is clean and glitches stay punctuation.
func (g *Glitch) Draw(ctx *Context, area Rect, amount float64) {
	if amount < 0.12 || area.Empty() {
		return
	}
	s := ctx.Screen
	bands := 1 + int(amount*5)
	for range bands {
		y := area.Y + ctx.Rand.Intn(area.H)
		span := int(2 + amount*10)
		dx := ctx.Rand.Intn(span*2+1) - span
		if dx == 0 {
			dx = 1
		}
		s.ShiftRow(y, dx)
	}
	if amount > 0.6 {
		for range 1 + int(amount*2) {
			y := area.Y + ctx.Rand.Intn(area.H)
			s.SetAttr(y, area.X, area.Right(), term.Reverse)
		}
	}
	noise := min(int(amount*float64(area.W)*0.18), 24)
	for range noise {
		x := area.X + ctx.Rand.Intn(area.W)
		y := area.Y + ctx.Rand.Intn(area.H)
		ch := rainAlphabet[ctx.Rand.Intn(len(rainAlphabet))]
		fg := ctx.Palette.Err
		if ctx.Rand.Intn(2) == 0 {
			fg = ctx.Palette.Accent2
		}
		s.Set(x, y, ch, fg, term.ColorDefault, term.Attr(0))
	}
}
