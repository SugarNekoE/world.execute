package scene

import (
	"time"

	"world.execute/internal/term"
)

const (
	crtOnLength  = 900 * time.Millisecond
	crtOffLength = 1540 * time.Millisecond
	crtOffAt     = 206760 * time.Millisecond
)

type crtState struct {
	v, h float64
	dot  float64
}

func crtAt(t time.Duration, scene string) (crtState, bool) {
	switch {
	case scene == "boot" && t >= 0 && t < crtOnLength:
		p := float64(t) / float64(crtOnLength)
		if p < 0.3 {
			return crtState{v: 0.03, h: ease(p / 0.3)}, true
		}
		return crtState{v: 0.03 + 0.97*ease((p-0.3)/0.7), h: 1}, true
	case scene == "final" && t >= crtOffAt && t < crtOffAt+crtOffLength:
		p := float64(t-crtOffAt) / float64(crtOffLength)
		switch {
		case p < 0.55:
			return crtState{v: 1 - 0.97*ease(p/0.55), h: 1}, true
		case p < 0.85:
			return crtState{v: 0.03, h: 1 - ease((p-0.55)/0.3)}, true
		}
		return crtState{v: 0.03, h: 0.01, dot: 1 - (p-0.85)/0.15}, true
	}
	return crtState{}, false
}

func (d *Director) crt(ctx *Context, st crtState) {
	s := ctx.Screen
	d.snapshot = s.Snapshot(d.snapshot)
	s.Scale(d.snapshot, st.v, st.h)
	white := ctx.Palette.Ink()
	cy := s.H / 2
	if st.dot > 0 {
		glyph := '●'
		if st.dot < 0.5 {
			glyph = '·'
		}
		s.Set(s.W/2, cy, glyph, Fade(white, ctx.Palette.Shadow, st.dot), term.ColorDefault, term.Bold)
		return
	}
	if st.v < 0.12 {
		width := int(st.h * float64(s.W))
		s.HLine((s.W-width)/2, cy, width, '━', white, term.ColorDefault, term.Bold)
	}
}
