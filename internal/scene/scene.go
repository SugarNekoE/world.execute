// Package scene draws the animation. Every scene is a pure function of the
// playback position, so seeking and frame dumps stay reproducible.
package scene

import (
	"math"
	"math/rand"
	"time"
	"unicode/utf8"

	"world.execute/internal/audio"
	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// Rect is a rectangle of cells. W and H are counts, not coordinates.
type Rect struct {
	X, Y, W, H int
}

// Right returns the first column past the rectangle.
func (r Rect) Right() int { return r.X + r.W }

// Bottom returns the first row past the rectangle.
func (r Rect) Bottom() int { return r.Y + r.H }

// Empty reports whether the rectangle has no drawable area.
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Inset shrinks the rectangle on every side.
func (r Rect) Inset(n int) Rect {
	if 2*n >= r.W || 2*n >= r.H {
		return Rect{X: r.X + n, Y: r.Y + n, W: max(r.W-2*n, 0), H: max(r.H-2*n, 0)}
	}
	return Rect{X: r.X + n, Y: r.Y + n, W: r.W - 2*n, H: r.H - 2*n}
}

// Pad shrinks the rectangle horizontally and vertically by different amounts.
func (r Rect) Pad(x, y int) Rect {
	w := max(r.W-2*x, 0)
	h := max(r.H-2*y, 0)
	return Rect{X: r.X + x, Y: r.Y + y, W: w, H: h}
}

// CenterX returns the column that horizontally centres text of the given
// display width inside the rectangle.
func (r Rect) CenterX(width int) int { return r.X + (r.W-width)/2 }

// CenterY returns the row that vertically centres a block of the given height.
func (r Rect) CenterY(height int) int { return r.Y + (r.H-height)/2 }

// Palette is the colour scheme of a section. Text and Dim are deliberately
// independent of the terminal background: Text inherits the terminal's own
// foreground so the picture is always readable, while Shadow is a real colour
// close to the background, used for fading and for the few places that need a
// solid backdrop.
type Palette struct {
	Name    string
	Shadow  term.Color
	Text    term.Color
	Dim     term.Color
	Accent  term.Color
	Accent2 term.Color
	Warn    term.Color
	Err     term.Color
	Kind    term.Color
}

// Palettes is the default dark scheme. Text uses the terminal's own foreground
// so a themed terminal stays legible.
var Palettes = map[string]Palette{
	"boot": {
		Name: "boot", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0x8b949e), Accent: term.Hex(0x56d364), Accent2: term.Hex(0x39d0d8),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0xff9bd5),
	},
	"title": {
		Name: "title", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0x8b949e), Accent: term.Hex(0x79c0ff), Accent2: term.Hex(0xff9bd5),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0xd2a8ff),
	},
	"verse": {
		Name: "verse", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0x8b949e), Accent: term.Hex(0x7ee787), Accent2: term.Hex(0xffd166),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0x79c0ff),
	},
	"glitch": {
		Name: "glitch", Shadow: term.Hex(0x0c0708), Text: term.ColorDefault,
		Dim: term.Hex(0xb0888f), Accent: term.Hex(0xff6b6b), Accent2: term.Hex(0xffd166),
		Warn: term.Hex(0xffa657), Err: term.Hex(0xff3b30), Kind: term.Hex(0xff9bd5),
	},
	"love": {
		Name: "love", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0xb98fae), Accent: term.Hex(0xff9bd5), Accent2: term.Hex(0xd2a8ff),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0x8fd0ff),
	},
	"void": {
		Name: "void", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0x6b7684), Accent: term.Hex(0x8ab4f8), Accent2: term.Hex(0x6b7684),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0x79c0ff),
	},
	"exit": {
		Name: "exit", Shadow: term.Hex(0x0b0d10), Text: term.ColorDefault,
		Dim: term.Hex(0x7d8898), Accent: term.Hex(0x8b949e), Accent2: term.Hex(0x8b949e),
		Warn: term.Hex(0xe3b341), Err: term.Hex(0xff7b72), Kind: term.Hex(0x79c0ff),
	},
}

// PalettesLight is for terminals with a light background: the tints are much
// darker so they stand out against white.
var PalettesLight = map[string]Palette{
	"boot": {
		Name: "boot", Shadow: term.Hex(0xf4f5f7), Text: term.ColorDefault,
		Dim: term.Hex(0x6e7781), Accent: term.Hex(0x0a7d32), Accent2: term.Hex(0x0b6f75),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0xa02e86),
	},
	"title": {
		Name: "title", Shadow: term.Hex(0xf4f5f7), Text: term.ColorDefault,
		Dim: term.Hex(0x6e7781), Accent: term.Hex(0x0b57d0), Accent2: term.Hex(0xa02e86),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0x6b21a8),
	},
	"verse": {
		Name: "verse", Shadow: term.Hex(0xf4f5f7), Text: term.ColorDefault,
		Dim: term.Hex(0x6e7781), Accent: term.Hex(0x116329), Accent2: term.Hex(0x8a6100),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0x0b57d0),
	},
	"glitch": {
		Name: "glitch", Shadow: term.Hex(0xfdf3f3), Text: term.ColorDefault,
		Dim: term.Hex(0x8c6f73), Accent: term.Hex(0xb3261e), Accent2: term.Hex(0x8a6100),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0x9c1d13), Kind: term.Hex(0xa02e86),
	},
	"love": {
		Name: "love", Shadow: term.Hex(0xfcf3f9), Text: term.ColorDefault,
		Dim: term.Hex(0x8c6f85), Accent: term.Hex(0x9c2a6b), Accent2: term.Hex(0x6b21a8),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0x0b57d0),
	},
	"void": {
		Name: "void", Shadow: term.Hex(0xf4f5f7), Text: term.ColorDefault,
		Dim: term.Hex(0x7d8898), Accent: term.Hex(0x3b5bdb), Accent2: term.Hex(0x7d8898),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0x0b57d0),
	},
	"exit": {
		Name: "exit", Shadow: term.Hex(0xf4f5f7), Text: term.ColorDefault,
		Dim: term.Hex(0x6e7781), Accent: term.Hex(0x57606a), Accent2: term.Hex(0x57606a),
		Warn: term.Hex(0x8a6100), Err: term.Hex(0xc2402c), Kind: term.Hex(0x0b57d0),
	},
}

// Theme returns the palette set for a theme name. Unknown names fall back to
// the dark scheme.
func Theme(name string) map[string]Palette {
	switch name {
	case "light":
		return PalettesLight
	default:
		return Palettes
	}
}

// Context carries everything a scene needs for one frame.
type Context struct {
	Screen    *term.Screen
	Lyrics    *lyric.Track
	Analysis  *audio.Analysis
	Palette   Palette
	Section   string
	Area      Rect
	Header    Rect
	Transport Rect

	T      time.Duration
	DT     time.Duration
	Frame  int
	Clock  time.Duration
	Total  time.Duration
	Rand   *rand.Rand
	Seed   int64
	Paused bool
	FPS    int

	Energy  float32
	Bands   []float32
	Wave    []float32
	AudioOK bool
	Player  string
	Volume  int
	Muted   bool
	Delay   time.Duration

	ShowInfo  bool
	ShowHelp  bool
	HintAlpha float64
	// SideW is the width a section has reserved for the telemetry sidecar, so
	// scenes keep their text out of that strip.
	SideW int
}

// Sec converts a duration to fractional seconds.
func (c *Context) Sec(d time.Duration) float64 { return d.Seconds() }

// Since returns the time elapsed since d, clamped at zero.
func (c *Context) Since(d time.Duration) time.Duration {
	if c.T < d {
		return 0
	}
	return c.T - d
}

// Beat returns a pulse in 0..1 that decays after each pronounced transient.
func (c *Context) Beat() float32 {
	if c.Analysis == nil {
		return 0
	}
	e := c.Analysis.EnergyAt(c.T)
	return e
}

// Blend mixes two colours; k is clamped to 0..1.
func Blend(a, b term.Color, k float64) term.Color { return term.Mix(a, b, k) }

// Fade returns c faded towards bg by 1-k.
func Fade(c, bg term.Color, k float64) term.Color { return Blend(bg, c, k) }

// Pulse oscillates between 0 and 1 with the given period in seconds.
func Pulse(sec, period float64) float64 {
	if period <= 0 {
		return 0
	}
	return 0.5 + 0.5*math.Sin(2*math.Pi*sec/period)
}

// Reveal returns how many runes of s should be visible at time t given a start
// and a per-character rate. A rate of zero reveals everything at once.
func Reveal(s string, elapsed, hold time.Duration, cps float64) int {
	n := utf8.RuneCountInString(s)
	if cps <= 0 {
		return n
	}
	if elapsed < hold {
		return 0
	}
	visible := int((elapsed - hold).Seconds() * cps)
	if visible < 0 {
		return 0
	}
	return min(visible, n)
}

// Progress returns a 0..1 ramp from start to end.
func Progress(t, start, end time.Duration) float64 {
	if end <= start {
		return 1
	}
	if t <= start {
		return 0
	}
	if t >= end {
		return 1
	}
	return float64(t-start) / float64(end-start)
}

// Shake returns a small pseudo random offset used for impact frames. The
// amplitude decays with falloff.
func Shake(c *Context, at time.Duration, amplitude, falloff float64) (int, int) {
	age := c.Since(at).Seconds()
	if age < 0 || amplitude <= 0 {
		return 0, 0
	}
	decay := math.Exp(-age / math.Max(falloff, 0.01))
	if decay < 0.02 {
		return 0, 0
	}
	dx := int(decay * amplitude * (c.Rand.Float64()*2 - 1))
	dy := int(decay * amplitude * 0.5 * (c.Rand.Float64()*2 - 1))
	return dx, dy
}

// WriteRunes writes at most the first count runes of s and returns the column
// just past the last rune written.
func WriteRunes(s *term.Screen, x, y int, text string, count int, fg, bg term.Color, a term.Attr) int {
	cx := x
	for i, r := range text {
		if i >= count {
			break
		}
		s.Set(cx, y, r, fg, bg, a)
		cx += term.RuneWidth(r)
	}
	return cx
}

// Faint is a foreground for chrome that should recede without disappearing.
// Shadow itself is the background colour, so text painted in it is invisible.
func (p Palette) Faint() term.Color { return Blend(p.Shadow, p.Dim, 0.5) }

// Ink is a readable text colour for the palette's theme: light on dark
// palettes, dark on light ones. Text itself is the terminal's own foreground,
// which cannot be blended.
func (p Palette) Ink() term.Color {
	c := uint32(p.Shadow)
	if (c>>16&0xff)*299+(c>>8&0xff)*587+(c&0xff)*114 > 128*1000 {
		return term.Hex(0x14181f)
	}
	return term.Hex(0xf2f6ff)
}

// Panel blanks a rectangle so text stays readable over the code rain. It
// deliberately does not paint a background colour: the terminal's own
// background shows through, which keeps the picture legible whatever theme the
// viewer uses.
func (c *Context) Panel(r Rect) {
	if r.Empty() {
		return
	}
	c.Screen.Fill(r.X, r.Y, r.W, r.H, ' ', c.Palette.Text, term.ColorDefault, term.Attr(0))
}

// PanelWithBar blanks a panel and marks its left edge with a subtle accent bar.
func (c *Context) PanelWithBar(r Rect, bar rune) {
	c.Panel(r)
	if r.W <= 0 || r.H <= 0 {
		return
	}
	fg := Blend(c.Palette.Accent, c.Palette.Shadow, 0.55)
	for y := r.Y; y < r.Bottom(); y++ {
		c.Screen.Set(r.X, y, bar, fg, term.ColorDefault, term.Attr(0))
	}
}
