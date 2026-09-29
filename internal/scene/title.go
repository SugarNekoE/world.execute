package scene

import (
	"fmt"
	"strings"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// TitleBanner draws the song title. It lives in two forms: a large banner that
// assembles itself while the first line is sung, and a one line header that
// stays with the rest of the piece.
type TitleBanner struct {
	line1 []bannerChar
	line2 []bannerChar

	revealAt  time.Duration
	settleAt  time.Duration
	headerAt  time.Duration
	subtitle  string
	hasBanner bool
}

type bannerChar struct {
	ch rune
	at time.Duration
}

// NewTitleBanner builds the banner from the lyric line that spells the title.
func NewTitleBanner(track *lyric.Track) *TitleBanner {
	tb := &TitleBanner{
		revealAt: 19*time.Second + 110*time.Millisecond,
		settleAt: 29*time.Second + 880*time.Millisecond,
		headerAt: 30*time.Second + 400*time.Millisecond,
		subtitle: "Mili — Miracle Milk (2015)",
	}
	line, _, ok := LyricLineAt(track, tb.revealAt)
	if !ok || len(line.Words) < 3 {
		return tb
	}
	tb.hasBanner = true
	chars := spreadChars(line.Words)
	split := len(chars)
	if len(line.Words) >= 3 {
		split = len([]rune(line.Words[0].Text)) + len([]rune(line.Words[1].Text))
	}
	tb.line1 = chars[:min(split, len(chars))]
	tb.line2 = chars[min(split, len(chars)):]
	return tb
}

// spreadChars assigns every character of a word the moment it appears,
// interpolating between the word onsets.
func spreadChars(words []lyric.Word) []bannerChar {
	var out []bannerChar
	for i, w := range words {
		runes := []rune(strings.ToUpper(w.Text))
		span := 120 * time.Millisecond
		if i+1 < len(words) {
			span = words[i+1].Time - w.Time
		}
		if span <= 0 || len(runes) == 0 {
			span = 100 * time.Millisecond
		}
		for k, r := range runes {
			out = append(out, bannerChar{
				ch: r,
				at: w.Time + time.Duration(float64(span)*float64(k)/float64(len(runes))),
			})
		}
	}
	return out
}

// BannerHeight is the height of the two banner lines including the gap.
const BannerHeight = 2*term.FontHeight + 4

// Active reports whether the large banner owns the screen at t.
func (tb *TitleBanner) Active(t time.Duration) bool {
	return tb.hasBanner && t >= tb.revealAt && t < tb.headerAt
}

// DrawFull renders the large banner over the whole story area, sliding up and
// away once it settles.
func (tb *TitleBanner) DrawFull(ctx *Context) {
	if !tb.hasBanner || ctx.T < tb.revealAt || ctx.T >= tb.headerAt {
		return
	}
	s := ctx.Screen
	pal := ctx.Palette
	area := ctx.Area
	if area.Empty() {
		return
	}

	rows := BannerHeight
	y := area.CenterY(rows)
	if ctx.T >= tb.settleAt {
		slide := Progress(ctx.T, tb.settleAt, tb.headerAt)
		y -= int(slide * float64(area.Y+term.FontHeight+6))
	}

	w1 := term.Measure(string(charsOf(tb.line1)))
	w2 := term.Measure(string(charsOf(tb.line2)))
	x1 := area.CenterX(w1)
	x2 := area.CenterX(w2)

	bg := Blend(pal.Shadow, term.Hex(0x101828), 0.5)
	bandY := max(y-2, area.Y)
	bandH := min(rows+4, area.Bottom()-bandY)
	s.Fill(area.X, bandY, area.W, bandH, ' ', pal.Text, bg, term.Attr(0))

	drawLine := func(chars []bannerChar, x, row int, fg term.Color) {
		if row < area.Y || row+term.FontHeight > area.Bottom() {
			return
		}
		term.DrawGlyphs(s, x, row, string(charsOf(chars)), fg, term.ColorDefault, term.Attr(0), func(i int) bool {
			return ctx.T >= chars[i].at
		})
	}
	drawLine(tb.line1, x1, y, pal.Accent)
	tb.drawGhost(ctx, x2, y+term.FontHeight+2)
	drawLine(tb.line2, x2, y+term.FontHeight+2, pal.Accent2)
	tb.drawLink(ctx, area, y+term.FontHeight*2+4)

	if ctx.T < tb.settleAt {
		sub := tb.subtitle
		subX := area.CenterX(term.StringWidth(sub))
		k := Progress(ctx.T, tb.revealAt, tb.revealAt+1200*time.Millisecond)
		s.Text(subX, y+term.FontHeight*2+3, sub, Fade(pal.Dim, pal.Shadow, k), term.ColorDefault, term.Attr(0))
		if Pulse(ctx.Sec(ctx.T), 0.5) > 0.5 {
			s.Set(subX+term.StringWidth(sub)+1, y+term.FontHeight*2+3, '█', pal.Accent, term.ColorDefault, term.Attr(0))
		}
	}
	floorY := max(y+term.FontHeight*2+5, area.Bottom()-6)
	drawHorizon(ctx, Rect{X: area.X, Y: floorY, W: area.W, H: area.Bottom() - floorY})
}

func (tb *TitleBanner) ghostProgress(t time.Duration) float64 {
	if len(tb.line2) == 0 {
		return 0
	}
	return Progress(t, tb.revealAt+1500*time.Millisecond, tb.line2[0].at)
}

// drawGhost fades the second title line in as flickering static before its
// words are sung, so the long instrumental gap reads as a signal locking on.
func (tb *TitleBanner) drawGhost(ctx *Context, x, row int) {
	p := tb.ghostProgress(ctx.T)
	area := ctx.Area
	if p <= 0 || row < area.Y || row+term.FontHeight > area.Bottom() {
		return
	}
	slot := int(ctx.T / (70 * time.Millisecond))
	fg := Fade(ctx.Palette.Dim, ctx.Palette.Shadow, 0.3+0.5*p)
	term.DrawGlyphs(ctx.Screen, x, row, string(charsOf(tb.line2)), fg, term.ColorDefault, term.Attr(0), func(i int) bool {
		if ctx.T >= tb.line2[i].at {
			return false
		}
		h := uint64(hashSeed(ctx.Seed, i*7919+slot))
		return float64(h>>11)/float64(1<<53) < p*0.85
	})
}

var linkSteps = []string{"locating self", "resolving execute", "binding me", "linking ;"}

func (tb *TitleBanner) drawLink(ctx *Context, area Rect, row int) {
	p := tb.ghostProgress(ctx.T)
	if p <= 0 || row >= area.Bottom() || row < area.Y {
		return
	}
	pal := ctx.Palette
	const barW = 24
	label := "symbol resolved"
	if p < 1 {
		label = linkSteps[min(int(p*float64(len(linkSteps))), len(linkSteps)-1)] + " …"
	}
	text := fmt.Sprintf(" %3d%%  %s", int(p*100), label)
	x := area.CenterX(barW + 2 + term.StringWidth(text))
	filled := int(p * barW)
	s := ctx.Screen
	s.Set(x, row, '▕', pal.Dim, term.ColorDefault, term.Attr(0))
	s.HLine(x+1, row, filled, '█', pal.Accent, term.ColorDefault, term.Attr(0))
	s.HLine(x+1+filled, row, barW-filled, '░', pal.Dim, term.ColorDefault, term.Attr(0))
	s.Set(x+1+barW, row, '▏', pal.Dim, term.ColorDefault, term.Attr(0))
	s.Text(x+2+barW, row, text, pal.Dim, term.ColorDefault, term.Attr(0))
}

// DrawHeader renders the header that stays for the whole piece: the title, a
// live waveform and the current section. It is drawn before the banner so the
// banner can cover it while it owns the screen.
func (tb *TitleBanner) DrawHeader(ctx *Context, r Rect, section string) {
	if r.Empty() {
		return
	}
	s := ctx.Screen
	pal := ctx.Palette
	fade := Progress(ctx.T, 500*time.Millisecond, 1400*time.Millisecond)
	if fade <= 0 {
		return
	}

	title := "world.execute(me);"
	x := s.Text(r.X+1, r.Y, title, Fade(pal.Accent, pal.Shadow, fade), term.ColorDefault, term.Bold)
	right := "Mili — Miracle Milk (2015)"
	rx := r.Right() - 1 - term.StringWidth(right)
	if rx > x+2 {
		s.Text(rx, r.Y, right, Fade(pal.Dim, pal.Shadow, fade), term.ColorDefault, term.Attr(0))
	}
	if r.H < 2 {
		return
	}

	label := "▸ " + scramble(strings.ToUpper(section), ctx.SectionAge, ctx.Seed)
	labelW := term.StringWidth(label)
	s.Text(r.X+1, r.Y+1, label, Fade(pal.Accent, pal.Shadow, fade), term.ColorDefault, term.Bold)

	scope := Rect{X: r.X + labelW + 3, Y: r.Y + 1, W: r.W - labelW - 5, H: r.H - 1}
	if scope.W >= 24 && scope.H >= 1 {
		DrawScope(ctx, scope, 0.85)
	}
}

func charsOf(chars []bannerChar) []rune {
	out := make([]rune, len(chars))
	for i, c := range chars {
		out[i] = c.ch
	}
	return out
}

// scramble decodes text into place: each letter cycles through noise until its
// turn comes, left to right, a fraction of a second after a section begins.
func scramble(text string, age time.Duration, seed int64) string {
	const settle = 140 * time.Millisecond
	const perLetter = 26 * time.Millisecond
	if age >= settle+time.Duration(len([]rune(text)))*perLetter {
		return text
	}
	slot := int(age / (45 * time.Millisecond))
	runes := []rune(text)
	for i, r := range runes {
		if r == ' ' || age >= settle+time.Duration(i)*perLetter {
			continue
		}
		h := uint64(hashSeed(seed, slot*97+i))
		runes[i] = rainAlphabet[h%uint64(len(rainAlphabet))]
	}
	return string(runes)
}
