package term

import (
	"bufio"
	"math"
	"strconv"
)

// Cell is one character position on the screen.
type Cell struct {
	R  rune
	Fg Color
	Bg Color
	A  Attr
}

var blank = Cell{R: ' ', Fg: ColorDefault, Bg: ColorDefault}

// Screen is a double buffered character canvas that writes only the cells that
// changed since the previous flush. It understands wide (double width) runes,
// so CJK text never tears the layout.
type Screen struct {
	W, H int

	mode  ColorMode
	out   *bufio.Writer
	cells []Cell
	prev  []Cell

	force  bool
	loaded bool

	cur      style
	haveCur  bool
	curX     int
	curY     int
	scratch  []byte
	begun    bool
	ascii    bool
	writeErr error
}

// NewScreen returns a screen of the given size drawing into out.
func NewScreen(w, h int, mode ColorMode, out *bufio.Writer) *Screen {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	s := &Screen{W: w, H: h, mode: mode, out: out}
	s.cells = make([]Cell, w*h)
	s.Clear()
	s.force = true
	return s
}

// Writer returns the underlying buffered writer.
func (s *Screen) Writer() *bufio.Writer { return s.out }

// ColorMode returns the color mode the screen was created with.
func (s *Screen) ColorMode() ColorMode { return s.mode }

// Clear resets every cell to a plain blank.
func (s *Screen) Clear() {
	for i := range s.cells {
		s.cells[i] = blank
	}
}

// Resize reallocates the canvas, preserving nothing.
func (s *Screen) Resize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if w == s.W && h == s.H {
		return
	}
	s.W, s.H = w, h
	s.cells = make([]Cell, w*h)
	s.prev = nil
	s.loaded = false
	s.Clear()
	s.force = true
	s.haveCur = false
}

// Set paints r at x, y with the given style.
func (s *Screen) Set(x, y int, r rune, fg, bg Color, a Attr) {
	if y < 0 || y >= s.H || x < 0 || x >= s.W {
		return
	}
	w := runeWidth(r)
	if w == 0 {
		return
	}
	if w == 2 && x == s.W-1 {
		r, w = ' ', 1
	}
	i := y*s.W + x

	if s.cells[i].R == 0 && x > 0 {
		s.blankOut(i - 1)
	}
	if runeWidth(s.cells[i].R) == 2 && x+1 < s.W {
		s.blankOut(i + 1)
	}
	if w == 2 && x+1 < s.W {
		if runeWidth(s.cells[i+1].R) == 2 && x+2 < s.W {
			s.blankOut(i + 2)
		}
		s.cells[i+1] = Cell{R: 0, Fg: fg, Bg: bg, A: a}
	}
	s.cells[i] = Cell{R: r, Fg: fg, Bg: bg, A: a}
}

// blankOut replaces a cell with a space, keeping its style.
func (s *Screen) blankOut(i int) {
	c := s.cells[i]
	s.cells[i] = Cell{R: ' ', Fg: c.Fg, Bg: c.Bg, A: c.A}
}

// Text paints a string starting at x, y and returns the column just past it.
func (s *Screen) Text(x, y int, text string, fg, bg Color, a Attr) int {
	cx := x
	for _, r := range Sanitize(text) {
		w := runeWidth(r)
		if w == 0 {
			continue
		}
		s.Set(cx, y, r, fg, bg, a)
		cx += w
	}
	return cx
}

// TextWidth paints a string and rewrites it so it occupies exactly w cells,
// truncating with an ellipsis or padding with spaces.
func (s *Screen) TextWidth(x, y, w int, text string, fg, bg Color, a Attr) {
	t := Truncate(text, w)
	end := s.Text(x, y, t, fg, bg, a)
	s.Fill(end, y, x+w-end, 1, ' ', fg, bg, a)
}

// Truncate shortens s to at most w cells, appending an ellipsis when cut.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if StringWidth(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	out := make([]rune, 0, w)
	used := 0
	for _, r := range s {
		rw := runeWidth(r)
		if used+rw > w-1 {
			break
		}
		out = append(out, r)
		used += rw
	}
	return string(out) + "…"
}

// Fill paints a rectangle of r.
func (s *Screen) Fill(x, y, w, h int, r rune, fg, bg Color, a Attr) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			s.Set(i, j, r, fg, bg, a)
		}
	}
}

// HLine paints a horizontal run of r.
func (s *Screen) HLine(x, y, w int, r rune, fg, bg Color, a Attr) {
	s.Fill(x, y, w, 1, r, fg, bg, a)
}

// VLine paints a vertical run of r.
func (s *Screen) VLine(x, y, h int, r rune, fg, bg Color, a Attr) {
	s.Fill(x, y, 1, h, r, fg, bg, a)
}

// Box draws a single line border of width w and height h at x, y.
func (s *Screen) Box(x, y, w, h int, fg Color, a Attr) {
	if w < 2 || h < 2 {
		return
	}
	set := func(i, j int, r rune) { s.Set(i, j, r, fg, ColorDefault, a) }
	set(x, y, '┌')
	set(x+w-1, y, '┐')
	set(x, y+h-1, '└')
	set(x+w-1, y+h-1, '┘')
	for i := x + 1; i < x+w-1; i++ {
		set(i, y, '─')
		set(i, y+h-1, '─')
	}
	for j := y + 1; j < y+h-1; j++ {
		set(x, j, '│')
		set(x+w-1, j, '│')
	}
}

// Flush writes every changed cell to the output. It is safe to call with an
// unchanged canvas, in which case nothing is written.
func (s *Screen) Flush() {
	if s.writeErr != nil {
		return
	}
	if !s.loaded || len(s.prev) != len(s.cells) {
		s.force = true
		s.prev = make([]Cell, len(s.cells))
		s.loaded = true
	}
	s.begun = false
	for y := 0; y < s.H; y++ {
		row := y * s.W
		for x := 0; x < s.W; x++ {
			i := row + x
			c := s.cells[i]
			if !s.force && c == s.prev[i] {
				continue
			}
			s.prev[i] = c
			if c.R == 0 {
				continue
			}
			s.put(x, y, c)
		}
	}
	s.force = false
	if s.begun {
		s.out.WriteString(syncEnd)
	}
	s.out.Flush()
}

const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

func (s *Screen) put(x, y int, c Cell) {
	if !s.begun {
		s.out.WriteString(syncBegin)
		s.begun = true
	}
	if !s.haveCur || s.curX != x+1 || s.curY != y+1 {
		s.out.WriteString("\x1b[")
		s.out.WriteString(strconv.Itoa(y + 1))
		s.out.WriteByte(';')
		s.out.WriteString(strconv.Itoa(x + 1))
		s.out.WriteByte('H')
	}
	st := style{fg: c.Fg, bg: c.Bg, attr: c.A}
	if !s.haveCur {
		s.scratch = st.appendSGR(s.scratch[:0], s.mode)
		s.out.Write(s.scratch)
		s.cur = st
	} else if st != s.cur {
		s.scratch = st.appendDelta(s.scratch[:0], s.cur, s.mode)
		s.out.Write(s.scratch)
		s.cur = st
	}
	r := c.R
	if s.ascii {
		r = ASCII(r)
	}
	s.out.WriteRune(r)
	w := runeWidth(c.R)
	s.curX, s.curY = x+1+w, y+1
	s.haveCur = true
}

// At returns the cell at x, y. It is mainly useful for tests and for anything
// that wants to read back what has already been painted.
func (s *Screen) At(x, y int) Cell {
	if x < 0 || x >= s.W || y < 0 || y >= s.H {
		return blank
	}
	return s.cells[y*s.W+x]
}

// ShiftRow moves a row's contents dx cells to the right, blanking the cells
// that fall off the edge. It repairs wide runes so the row stays well formed.
func (s *Screen) ShiftRow(y, dx int) {
	if dx == 0 || y < 0 || y >= s.H {
		return
	}
	row := s.cells[y*s.W : (y+1)*s.W]
	switch {
	case dx >= s.W || dx <= -s.W:
		for i := range row {
			row[i] = blank
		}
	case dx > 0:
		copy(row[dx:], row[:s.W-dx])
		for i := range dx {
			row[i] = blank
		}
	default:
		d := -dx
		copy(row[:s.W-d], row[d:])
		for i := s.W - d; i < s.W; i++ {
			row[i] = blank
		}
	}
	s.repairRow(y)
}

// Snapshot copies every cell into dst, growing it when needed.
func (s *Screen) Snapshot(dst []Cell) []Cell {
	return append(dst[:0], s.cells...)
}

// RestoreSpan copies the cells of row y in [x0, x1) back from a snapshot taken
// at the same size, repairing any wide rune the seam cuts in half.
func (s *Screen) RestoreSpan(snapshot []Cell, y, x0, x1 int) {
	if len(snapshot) != len(s.cells) || y < 0 || y >= s.H {
		return
	}
	x0, x1 = max(x0, 0), min(x1, s.W)
	if x0 >= x1 {
		return
	}
	copy(s.cells[y*s.W+x0:y*s.W+x1], snapshot[y*s.W+x0:y*s.W+x1])
	s.repairRow(y)
}

// Scale redraws a snapshot scaled about the screen centre by v vertically and h
// horizontally, both in (0, 1]. Cells outside the scaled picture are blank.
func (s *Screen) Scale(snapshot []Cell, v, h float64) {
	if len(snapshot) != len(s.cells) {
		return
	}
	v, h = math.Max(v, 0.01), math.Max(h, 0.01)
	s.Clear()
	cx, cy := float64(s.W-1)/2, float64(s.H-1)/2
	for y := range s.H {
		sy := int(math.Round(cy + (float64(y)-cy)/v))
		if sy < 0 || sy >= s.H {
			continue
		}
		for x := range s.W {
			sx := int(math.Round(cx + (float64(x)-cx)/h))
			if sx >= 0 && sx < s.W {
				s.cells[y*s.W+x] = snapshot[sy*s.W+sx]
			}
		}
		s.repairRow(y)
	}
}

// Vignette darkens coloured cells towards bg as they get further from the
// centre, by at most strength, never below a minimum distance from bg. Cells that use the terminal's default colours
// are left alone, so body text keeps its own contrast.
func (s *Screen) Vignette(bg Color, strength float64) {
	if strength <= 0 || s.W < 4 || s.H < 4 {
		return
	}
	cx, cy := float64(s.W-1)/2, float64(s.H-1)/2
	for y := range s.H {
		ny := (float64(y) - cy) / cy
		for x := range s.W {
			nx := (float64(x) - cx) / cx
			r := math.Sqrt(nx*nx+ny*ny) / math.Sqrt2
			if r <= 0.6 {
				continue
			}
			t := math.Min((r-0.6)/0.5, 1)
			k := strength * t * t * (3 - 2*t)
			c := &s.cells[y*s.W+x]
			if c.R == 0 || c.R == ' ' || c.Fg == ColorDefault {
				continue
			}
			if faded := Mix(c.Fg, bg, k); bg == ColorDefault || channelGap(faded, bg) >= minVignetteGap {
				c.Fg = faded
			}
		}
	}
}

// minVignetteGap is the least distance, summed over the colour channels, a
// vignetted cell may keep from the background, so faint text never vanishes.
const minVignetteGap = 150

func channelGap(a, b Color) int {
	return abs(int(a.r())-int(b.r())) + abs(int(a.g())-int(b.g())) + abs(int(a.b())-int(b.b()))
}

// SetAttr merges attribute bits into every cell of a row within [x0, x1).
func (s *Screen) SetAttr(y, x0, x1 int, a Attr) {
	if y < 0 || y >= s.H {
		return
	}
	x0, x1 = max(x0, 0), min(x1, s.W)
	for x := x0; x < x1; x++ {
		s.cells[y*s.W+x].A |= a
	}
}

// FadeRect blends every coloured cell in the given rectangle towards bg. Cells
// that use the terminal default colour are left alone. It is used for the
// closing fade.
func (s *Screen) FadeRect(x, y, w, h int, bg Color, k float64) {
	if k <= 0 {
		return
	}
	x0, y0 := max(x, 0), max(y, 0)
	x1, y1 := min(x+w, s.W), min(y+h, s.H)
	for row := y0; row < y1; row++ {
		for col := x0; col < x1; col++ {
			c := &s.cells[row*s.W+col]
			if c.Fg != ColorDefault {
				c.Fg = Mix(c.Fg, bg, k)
			}
			if c.Bg != ColorDefault {
				c.Bg = Mix(c.Bg, bg, k)
			}
		}
	}
}

// repairRow removes orphaned half cells left behind by row shifts.
func (s *Screen) repairRow(y int) {
	row := s.cells[y*s.W : (y+1)*s.W]
	for x := 0; x < s.W; x++ {
		c := row[x]
		if c.R == 0 {
			if x == 0 || runeWidth(row[x-1].R) != 2 {
				row[x] = Cell{R: ' ', Fg: c.Fg, Bg: c.Bg, A: c.A}
			}
			continue
		}
		if runeWidth(c.R) != 2 {
			continue
		}
		if x+1 >= s.W || row[x+1].R != 0 {
			row[x] = Cell{R: ' ', Fg: c.Fg, Bg: c.Bg, A: c.A}
			continue
		}
		x++
	}
}
