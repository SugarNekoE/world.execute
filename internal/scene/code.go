package scene

import (
	"math"
	"strings"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// codeLine is one line of the pseudocode that the verses are written as.
type codeLine struct {
	at   time.Duration
	text string
	kw   string
	kwAt time.Duration
}

// CodeScene renders lyrics as a typed out source file with line numbers, a
// highlighted keyword when the song shouts it, and the sung words in a footer.
type CodeScene struct {
	lines []codeLine
	title string
}

// NewCodeScene returns a code scene for the given lines.
func NewCodeScene(title string, lines []codeLine) *CodeScene {
	return &CodeScene{lines: lines, title: title}
}

// Draw renders the file, scrolled so the newest line is visible.
func (c *CodeScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen

	panel := Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-2, 1)}
	ctx.PanelWithBar(panel, '▌')

	body := Rect{X: panel.X + 3, Y: panel.Y, W: max(panel.W-4, 1), H: panel.H}
	if c.title != "" {
		s.Text(body.X, body.Y, "// "+c.title, pal.Dim, term.ColorDefault, term.Attr(0))
		body.Y++
		body.H--
	}
	if body.Empty() {
		return
	}
	if body.W > 64 {
		meter := Rect{X: body.Right() - 14, Y: body.Y, W: 14, H: min(body.H, 14)}
		DrawMeter(ctx, meter)
		body.W -= 17
	}

	const gutter = 4
	codeX := body.X + gutter
	maxX := body.Right()

	visible := 0
	for i, l := range c.lines {
		if l.at <= ctx.T {
			visible = i + 1
		}
	}
	start := max(visible-body.H, 0)
	if start > 0 {
		s.Text(body.X, body.Y, "  ⋮", pal.Dim, term.ColorDefault, term.Attr(0))
	}
	row := body.Y
	for i := start; i < visible; i++ {
		l := c.lines[i]
		current := i == visible-1
		numColor := pal.Dim
		if current {
			numColor = pal.Accent
		}
		s.Text(body.X, row, pad(i+1, gutter-1), numColor, term.ColorDefault, term.Attr(0))
		if l.text != "" {
			flash := l.kw != "" && ctx.T >= l.kwAt && ctx.T < l.kwAt+500*time.Millisecond
			n := Reveal(l.text, ctx.Since(l.at), 200*time.Millisecond, 90)
			WriteCode(s, codeX, row, l.text, n, l.kw, pal, pal.Text, maxX, flash)
		}
		row++
	}
	c.drawFooter(ctx, area, body)
}

func (c *CodeScene) drawFooter(ctx *Context, area, body Rect) {
	y := area.Bottom() - 1
	if y < body.Bottom() || y >= area.Bottom()+1 {
		return
	}
	line, _, ok := LyricLineAt(ctx.Lyrics, ctx.T)
	if !ok || line.Text == "" {
		return
	}
	pal := ctx.Palette
	x := ctx.Screen.Text(area.X+2, y, "// ", pal.Dim, term.ColorDefault, term.Attr(0))
	DrawWordLine(ctx.Screen, x, y, line, ctx.T, pal, pal.Dim, area.Right()-2)
}

// WriteCode draws the first n runes of text, colouring the keyword token when
// it appears and inverting it while the song shouts it.
func WriteCode(s *term.Screen, x, y int, text string, n int, kw string, pal Palette, base term.Color, maxX int, flash bool) int {
	runes := []rune(text)
	kwStart, kwEnd := -1, -1
	if kw != "" {
		if i := strings.Index(text, kw); i >= 0 {
			kwStart = len([]rune(text[:i]))
			kwEnd = kwStart + len([]rune(kw))
		}
	}
	cx := x
	for i, r := range runes {
		if i >= n || cx >= maxX {
			break
		}
		fg, bg, attr := base, term.ColorDefault, term.Attr(0)
		if kwStart >= 0 && i >= kwStart && i < kwEnd {
			fg, bg, attr = pal.Accent2, term.ColorDefault, term.Bold
			if flash {
				fg, bg = pal.Shadow, pal.Accent2
			}
		}
		cx = s.Text(cx, y, string(r), fg, bg, attr)
	}
	return cx
}

func pad(n, w int) string {
	s := itoa(n)
	for len(s) < w {
		s = " " + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// RegisterScene shows the song's switch statements as a register panel beside
// the lyric log.
type RegisterScene struct {
	log    *LogView
	regs   []regEntry
	labels []string
}

type regEntry struct {
	at     time.Duration
	label  string
	from   string
	to     string
	effect string
}

// NewRegisterScene builds a register scene for the lyrics in [from, to).
func NewRegisterScene(track *lyric.Track, from, to time.Duration, regs []regEntry) *RegisterScene {
	v := NewLogView(LogFromLyrics(track, from, to)...)
	v.Prompt = ">> "
	v.CPS = 60
	var labels []string
	seen := map[string]bool{}
	for _, r := range regs {
		if !seen[r.label] {
			seen[r.label] = true
			labels = append(labels, r.label)
		}
	}
	return &RegisterScene{log: v, regs: regs, labels: labels}
}

// Draw renders the log on the left and the register panel on the right.
func (rs *RegisterScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	full := Rect{X: area.X + 3, Y: area.Y, W: max(area.W-5, 1), H: area.H}
	motion := Rect{}
	if area.W >= 36 && area.H >= 14 {
		full.H = min(7, area.H/3)
		motion = Rect{X: full.X, Y: area.Y + full.H + 1, W: full.W, H: area.H - full.H - 2}
	}
	panelW := min(34, max(area.W/2, 18))
	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: area.W, H: area.H}, '▌')
	if area.W < 60 {
		rs.log.Draw(ctx, full)
		rs.drawAnimation(ctx, motion)
		return
	}
	rs.log.Draw(ctx, Rect{X: full.X, Y: full.Y, W: full.W - panelW - 2, H: full.H})
	rs.drawPanel(ctx, Rect{X: area.Right() - panelW - 2, Y: area.Y, W: panelW, H: full.H})
	rs.drawAnimation(ctx, motion)
}

func (rs *RegisterScene) drawPanel(ctx *Context, rect Rect) {
	if rect.Empty() || len(rs.labels) == 0 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen

	active := -1
	for i, e := range rs.regs {
		if e.at <= ctx.T {
			active = i
		}
	}
	effect := ""
	if active >= 0 {
		effect = rs.regs[active].effect
	}

	panel := Rect{X: rect.X, Y: rect.Y, W: rect.W, H: min(len(rs.labels)+3, rect.H)}
	if panel.H < 3 {
		return
	}
	if effect == "dizzy" {
		panel.X += int(4 * Pulse(ctx.Sec(ctx.T), 0.5))
	}
	s.Box(panel.X, panel.Y, panel.W, panel.H, pal.Dim, term.Attr(0))
	s.Text(panel.X+2, panel.Y, " registers ", pal.Dim, term.ColorDefault, term.Attr(0))

	row := panel.Y + 1
	for _, label := range rs.labels {
		if row >= panel.Bottom()-1 {
			break
		}
		var cur regEntry
		isActive := false
		for _, e := range rs.regs {
			if e.label != label || e.at > ctx.T {
				continue
			}
			cur = e
			if active >= 0 {
				isActive = e.at == rs.regs[active].at
			}
		}
		value := cur.from
		if cur.to != "" && cur.to != cur.from {
			value = cur.from + " → " + cur.to
		}
		fg, attr := pal.Text, term.Attr(0)
		if isActive {
			fg, attr = pal.Accent, term.Bold
		}
		labelX := s.Text(panel.X+2, row, label, pal.Dim, term.ColorDefault, term.Attr(0))
		s.Text(labelX+1, row, value, fg, term.ColorDefault, attr)
		row++
	}
	rs.drawEffect(ctx, panel, effect, active)
}

func (rs *RegisterScene) drawEffect(ctx *Context, panel Rect, effect string, active int) {
	if effect == "" || active < 0 {
		return
	}
	pal := ctx.Palette
	s := ctx.Screen
	age := ctx.Since(rs.regs[active].at).Seconds()
	inner := panel.Inset(1)
	if inner.Empty() {
		return
	}

	switch effect {
	case "blind":
		k := min(age/1.6, 1)
		for i := range int(k * float64(inner.H)) {
			y := inner.Bottom() - 1 - i
			s.Fill(inner.X, y, inner.W, 1, '▒', pal.Dim, term.ColorDefault, term.Attr(0))
		}
		if k >= 1 {
			s.Fill(inner.X, inner.Y, inner.W, inner.H, '▒',
				Fade(pal.Dim, pal.Shadow, 0.4), term.ColorDefault, term.Attr(0))
		}
	case "unite":
		mid := inner.X + inner.W/2
		gap := int(math.Max(0, 6-age))
		y := inner.Bottom() - 1
		s.Text(clamp(mid-gap-4, inner.X, inner.Right()-4), y, "━━━━", pal.Accent2, term.ColorDefault, term.Bold)
		s.Text(clamp(mid+gap, inner.X, inner.Right()-4), y, "━━━━", pal.Accent, term.ColorDefault, term.Bold)
		if age > 5 {
			s.Text(inner.X+1, inner.Y, "united", pal.Accent, term.ColorDefault, term.Bold)
		}
	case "depth":
		n := max(int(age), 1)
		for i := range n {
			box := inner.Pad(i, i/2)
			if box.Empty() {
				break
			}
			s.Box(box.X, box.Y, box.W, box.H, Fade(pal.Accent2, pal.Shadow, 0.3+0.7*float64(i+1)/float64(n)), term.Attr(0))
		}
	case "trance":
		ph := ctx.Sec(ctx.T)
		for i := range inner.H {
			w := int(math.Abs(math.Sin(ph*1.5+float64(i)*0.5)) * float64(inner.W))
			s.HLine(inner.X, inner.Y+i, w, '·', Fade(pal.Accent2, pal.Shadow, 0.55), term.ColorDefault, term.Attr(0))
		}
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	return min(max(v, lo), hi)
}
