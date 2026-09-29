package scene

import (
	"fmt"
	"time"

	"world.execute/internal/term"
)

// Transport draws the playback chrome: the progress bar, the equaliser, the
// fading key hints and the help overlay.
type Transport struct{}

var (
	barFilled  = '━'
	barTrack   = '─'
	barHandle  = '●'
	barTick    = '·'
	barChapter = '┃'
	pausedMark = '⏸'
	playMark   = '▶'
)

var equalizerBlocks = []rune(" ▁▂▃▄▅▆▇█")

type hint struct {
	keys  string
	label string
}

var transportHints = []hint{
	{"space", "pause"}, {"←→", "seek"}, {"↑↓", "volume"},
	{"m", "mute"}, {"r", "restart"}, {"?", "help"}, {"q", "quit"},
}

var helpRows = []hint{
	{"space / p", "pause or resume"},
	{"→ / l", "seek forward 5s"},
	{"← / h", "seek back 5s"},
	{". / >", "seek forward 1s"},
	{", / <", "seek back 1s"},
	{"1 - 9", "jump to a chapter"},
	{"0 / Home", "restart from the beginning"},
	{"End", "jump to the last second"},
	{"↑ / +", "volume up"},
	{"↓ / -", "volume down"},
	{"m", "mute or unmute"},
	{"[ ]", "animation later / earlier 10ms"},
	{"{ }", "animation later / earlier 50ms"},
	{"r", "restart animation and audio"},
	{"s", "toggle these rows"},
	{"?", "close this help"},
	{"q / Esc", "quit"},
}

// FormatClock renders a playback position as mm:ss.cc.
func FormatClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := d.Milliseconds()
	cs := total / 10 % 100
	sec := total / 1000 % 60
	min := total / 60000
	return fmt.Sprintf("%02d:%02d.%02d", min, sec, cs)
}

// Draw renders the transport rows at the bottom of the screen. area is the
// story area, used to place the help overlay.
func (t *Transport) Draw(ctx *Context, area Rect, r Rect) {
	if r.Empty() {
		return
	}
	if ctx.ShowHelp {
		t.drawHelp(ctx, area)
	}
	if !ctx.ShowInfo {
		return
	}
	t.drawBar(ctx, r)
	if r.H >= 2 {
		t.drawBottomRow(ctx, Rect{X: r.X, Y: r.Y + 1, W: r.W, H: 1})
	}
}

func (t *Transport) drawBar(ctx *Context, r Rect) {
	s := ctx.Screen
	pal := ctx.Palette
	y := r.Y
	total := ctx.Total
	if total <= 0 {
		total = time.Second
	}
	progress := float64(ctx.T) / float64(total)
	progress = min(max(progress, 0), 1)

	mark := playMark
	if ctx.Paused {
		mark = pausedMark
	}
	if ctx.ShowHelp || r.W < 40 {
		line := fmt.Sprintf("%c %s / %s", mark, FormatClock(ctx.T), FormatClock(total))
		s.TextWidth(r.X, y, r.W, line, pal.Text, term.ColorDefault, term.Attr(0))
		return
	}

	x := s.Text(r.X, y, string(mark), pal.Accent, term.ColorDefault, term.Bold)
	x = s.Text(x+1, y, FormatClock(ctx.T), pal.Text, term.ColorDefault, term.Attr(0))

	right := fmt.Sprintf("%s  %d fps  %s", volumeLabel(ctx), ctx.FPS, ctx.Player)
	if ctx.Delay != 0 {
		right = fmt.Sprintf("sync %+dms  %s", ctx.Delay.Milliseconds(), right)
	}
	tw := term.StringWidth(right)
	rightX := r.Right() - tw

	totalLabel := FormatClock(total)
	barX := x + 2
	barEnd := rightX - term.StringWidth(totalLabel) - 2
	if barEnd-barX < 10 {
		line := fmt.Sprintf("%c %s / %s", mark, FormatClock(ctx.T), FormatClock(total))
		s.TextWidth(r.X, y, r.W, line, pal.Text, term.ColorDefault, term.Attr(0))
		return
	}
	barW := barEnd - barX
	fill := int(progress * float64(barW))
	if fill >= barW {
		fill = barW - 1
	}

	dim := pal.Dim
	if ctx.Paused && Pulse(ctx.Sec(ctx.Clock), 1.1) < 0.5 {
		dim = Fade(pal.Dim, pal.Shadow, 0.4)
	}
	for i := range barW {
		ch := barTrack
		fg := dim
		switch {
		case i == fill:
			ch, fg = barHandle, pal.Accent2
		case isChapterMark(i, barW, total):
			ch, fg = barChapter, pal.Accent2
		case i < fill:
			ch, fg = barFilled, pal.Accent
		case t.isLineTick(ctx, i, barW, total):
			ch, fg = barTick, pal.Dim
		}
		s.Set(barX+i, y, ch, fg, term.ColorDefault, term.Attr(0))
	}
	s.Text(barEnd+1, y, totalLabel, pal.Dim, term.ColorDefault, term.Attr(0))
	s.Text(rightX, y, right, pal.Dim, term.ColorDefault, term.Attr(0))
}

// isChapterMark reports whether a chapter starts under bar cell i.
func isChapterMark(i, barW int, total time.Duration) bool {
	if barW <= 0 || total <= 0 {
		return false
	}
	for _, c := range Chapters {
		if int(float64(c.At)/float64(total)*float64(barW)) == i {
			return true
		}
	}
	return false
}

// isLineTick reports whether the cell under the cursor is close to a lyric
// line, so the bar doubles as a song map.
func (t *Transport) isLineTick(ctx *Context, x, barW int, total time.Duration) bool {
	if ctx.Lyrics == nil || barW <= 0 {
		return false
	}
	at := time.Duration(float64(x) / float64(barW) * float64(total))
	for _, l := range ctx.Lyrics.Lines {
		if d := l.Time - at; d > -120*time.Millisecond && d < 120*time.Millisecond {
			return true
		}
	}
	return false
}

func volumeLabel(ctx *Context) string {
	if ctx.Muted {
		return "vol muted"
	}
	return fmt.Sprintf("vol %d%%", ctx.Volume)
}

// HintColours returns the colours for the key hint row at the given fade. The
// hints grow out of the shadow colour into the accent and the mid grey, so they
// are never drawn in the background colour itself.
func HintColours(p Palette, alpha float64) (keys, labels term.Color) {
	return Blend(p.Shadow, p.Accent, alpha), Blend(p.Shadow, p.Dim, alpha)
}

func (t *Transport) drawBottomRow(ctx *Context, r Rect) {
	s := ctx.Screen
	pal := ctx.Palette
	if ctx.HintAlpha > 0.15 {
		k := ctx.HintAlpha
		keyColour, labelColour := HintColours(pal, k)
		x := r.X
		for _, h := range transportHints {
			w := term.StringWidth(h.keys) + term.StringWidth(h.label) + 3
			if x+w > r.Right() {
				break
			}
			x = s.Text(x, r.Y, h.keys, keyColour, term.ColorDefault, term.Bold)
			x = s.Text(x+1, r.Y, h.label, labelColour, term.ColorDefault, term.Attr(0))
			x += 2
		}
		return
	}
	t.drawEqualizer(ctx, r)
}

// drawEqualizer renders the frequency bands as a compact row of blocks.
func (t *Transport) drawEqualizer(ctx *Context, r Rect) {
	if ctx.Analysis == nil || r.Empty() {
		return
	}
	n := len(equalizerBlocks)
	bands := ctx.Analysis.BandsAt(ctx.T, ctx.Bands)
	tw := len(bands) * 2
	x := r.Right() - tw
	if x < r.X {
		return
	}
	for i, b := range bands {
		idx := int(float64(b) * float64(n-1))
		idx = min(max(idx, 0), n-1)
		ch := equalizerBlocks[idx]
		fg := Blend(ctx.Palette.Accent2, ctx.Palette.Accent, float64(i)/float64(len(bands)))
		ctx.Screen.Set(x+i*2, r.Y, ch, fg, term.ColorDefault, term.Attr(0))
		ctx.Screen.Set(x+i*2+1, r.Y, ch, fg, term.ColorDefault, term.Attr(0))
	}
}

func (t *Transport) drawHelp(ctx *Context, area Rect) {
	if area.Empty() {
		return
	}
	s := ctx.Screen
	pal := ctx.Palette
	w := 44
	h := len(helpRows) + 4
	if w > area.W {
		w = area.W
	}
	if h > area.H {
		h = area.H
	}
	x := area.CenterX(w)
	y := area.CenterY(h)
	s.Fill(x, y, w, h, ' ', pal.Text, term.ColorDefault, term.Attr(0))
	s.Box(x, y, w, h, pal.Accent, term.Attr(0))
	s.TextWidth(x+2, y, w-4, " world.execute(me); — controls", pal.Accent, pal.Shadow, term.Bold)
	row := y + 2
	for _, hh := range helpRows {
		if row >= y+h-1 {
			break
		}
		s.Text(x+2, row, hh.keys, pal.Accent, pal.Shadow, term.Bold)
		s.Text(x+16, row, hh.label, pal.Text, pal.Shadow, term.Attr(0))
		row++
	}
}
