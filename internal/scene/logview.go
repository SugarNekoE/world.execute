package scene

import (
	"strings"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// LogStyle selects the stamp and colour of a log row.
type LogStyle int

// Log row styles.
const (
	LogPlain LogStyle = iota
	LogOK
	LogWarn
	LogErr
	LogInfo
)

// LogEntry is one row of a console style scene.
type LogEntry struct {
	At    time.Duration
	End   time.Duration
	Line  lyric.Line
	Text  string
	Style LogStyle
	Stamp string
	Keep  time.Duration
}

// LogView renders entries as a scrolling console. Entries written from lyric
// lines reveal themselves word by word at the times they are sung.
type LogView struct {
	Entries  []LogEntry
	Prompt   string
	CPS      float64
	RightPad int
}

// LogFromLyrics builds entries from every lyric line starting in [from, to).
func LogFromLyrics(track *lyric.Track, from, to time.Duration) []LogEntry {
	var out []LogEntry
	if track == nil {
		return out
	}
	for i, l := range track.Lines {
		if l.Time < from || l.Time >= to {
			continue
		}
		out = append(out, LogEntry{
			At:   l.Time,
			End:  track.End(i),
			Line: l,
			Text: l.Text,
		})
	}
	return out
}

// NewLogView returns a console view with sensible defaults.
func NewLogView(entries ...LogEntry) *LogView {
	return &LogView{Entries: entries, Prompt: "> ", CPS: 45}
}

// Append adds entries to the view.
func (v *LogView) Append(entries ...LogEntry) *LogView {
	v.Entries = append(v.Entries, entries...)
	return v
}

// Draw renders the entries that are alive at the current time, scrolled so the
// newest row sits at the bottom of r.
func (v *LogView) Draw(ctx *Context, r Rect) {
	if r.Empty() {
		return
	}
	t := ctx.T
	alive := make([]int, 0, len(v.Entries))
	for i, e := range v.Entries {
		if e.At > t {
			break
		}
		if e.Keep > 0 && t > e.At+e.Keep {
			continue
		}
		alive = append(alive, i)
	}
	if len(alive) > r.H {
		alive = alive[len(alive)-r.H:]
	}
	y := r.Y
	for _, i := range alive {
		v.drawEntry(ctx, r, y, v.Entries[i])
		y++
	}
}

func (v *LogView) drawEntry(ctx *Context, r Rect, y int, e LogEntry) {
	s := ctx.Screen
	pal := ctx.Palette
	base := pal.Text
	t := ctx.T
	x := r.X
	if v.Prompt != "" {
		x = s.Text(x, y, v.Prompt, pal.Accent, term.ColorDefault, term.Attr(0))
	}

	switch e.Style {
	case LogOK, LogInfo:
		base = pal.Accent
	case LogWarn:
		base = pal.Warn
	case LogErr:
		base = pal.Err
	}

	limit := r.Right() - v.RightPad
	if e.Stamp != "" && t >= e.End {
		w := term.StringWidth(e.Stamp)
		if limit > w+2 {
			limit -= w + 2
		}
	}

	if len(e.Line.Words) > 0 {
		DrawWordLine(s, x, y, e.Line, t, pal, base, limit)
	} else {
		text := e.Text
		if text == "" {
			text = e.Line.Text
		}
		n := Reveal(text, ctx.Since(e.At), 0, v.CPS)
		WriteRunes(s, x, y, text, n, base, term.ColorDefault, term.Attr(0))
	}

	if e.Stamp != "" && t >= e.End {
		fg := pal.Accent
		switch e.Style {
		case LogWarn:
			fg = pal.Warn
		case LogErr:
			fg = pal.Err
		case LogPlain, LogOK, LogInfo:
			fg = pal.Dim
		}
		w := term.StringWidth(e.Stamp)
		s.Text(r.Right()-w, y, e.Stamp, fg, term.ColorDefault, term.Attr(0))
	}
}

// DrawWordLine renders a lyric line up to time t, revealing words as they are
// sung. The word being sung is emphasised, as are the song's capitalised
// keywords. It stops at maxX and returns the column just past the last word.
func DrawWordLine(s *term.Screen, x, y int, line lyric.Line, t time.Duration, pal Palette, base term.Color, maxX int) int {
	words := line.Words
	if len(words) == 0 {
		if t < line.Time {
			return x
		}
		return s.Text(x, y, line.Text, base, term.ColorDefault, term.Attr(0))
	}
	current := -1
	for i, w := range words {
		if w.Time <= t {
			current = i
		}
	}
	cx := x
	for i := 0; i <= current; i++ {
		w := words[i]
		if i > 0 && lyric.SpaceAfter(words, i-1) {
			cx++
		}
		fg, attr := base, term.Attr(0)
		switch {
		case i == current:
			fg, attr = pal.Accent, term.Bold
		case IsKeyword(w.Text):
			fg, attr = pal.Kind, term.Bold
		}
		if cx+term.StringWidth(w.Text) > maxX {
			return cx
		}
		cx = s.Text(cx, y, w.Text, fg, term.ColorDefault, attr)
	}
	return cx
}

// IsKeyword reports whether w is one of the song's shouted keywords: fully
// upper case, at least two characters, containing a letter.
func IsKeyword(w string) bool {
	if len(w) < 2 || w != strings.ToUpper(w) {
		return false
	}
	for _, r := range w {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

// LyricLineAt returns the lyric line playing at t.
func LyricLineAt(track *lyric.Track, t time.Duration) (lyric.Line, int, bool) {
	if track == nil {
		return lyric.Line{}, -1, false
	}
	return track.At(t)
}
