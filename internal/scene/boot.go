package scene

import (
	"fmt"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// BootScene is the power on sequence: a console log whose lines arrive with
// the sung words, a boot progress bar, and a console prompt.
type BootScene struct {
	log     *LogView
	ready   time.Duration
	handoff time.Duration
	seed    int64
}

// NewBootScene builds the boot log from the opening lyrics.
func NewBootScene(track *lyric.Track, seed int64) *BootScene {
	entries := LogFromLyrics(track, 0, 14*time.Second)
	for i := range entries {
		entries[i].Stamp = "[  OK  ]"
	}
	extra := []LogEntry{
		{At: 14100 * time.Millisecond, End: 15 * time.Second, Text: "linking libmili.so", Stamp: "[  OK  ]"},
		{At: 15300 * time.Millisecond, End: 16 * time.Second, Text: "calibrating sine wave", Stamp: "[  OK  ]"},
		{At: 16500 * time.Millisecond, End: 17 * time.Second,
			Text: fmt.Sprintf("seeding pseudo random (seed %d)", seed), Stamp: "[  OK  ]"},
		{At: 17700 * time.Millisecond, End: 18 * time.Second, Text: "mounting /dev/emotion", Stamp: "[  OK  ]"},
		{At: 18900 * time.Millisecond, End: 10 * time.Second, Text: "awaiting OBJECT CREATION", Stamp: "[  OK  ]"},
	}
	v := NewLogView(append(entries, extra...)...)
	v.Prompt = "$ "
	v.CPS = 90
	v.RightPad = 2
	return &BootScene{log: v, ready: 14 * time.Second, handoff: 19*time.Second + 110*time.Millisecond, seed: seed}
}

// Draw renders the boot sequence.
func (b *BootScene) Draw(ctx *Context) {
	if ctx.T >= b.handoff {
		return
	}
	area := ctx.Area
	if area.Empty() {
		return
	}
	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: max(area.W-ctx.SideW-1, 1), H: max(area.H-1, 1)}, '▌')
	logArea := Rect{X: area.X + 3, Y: area.Y, W: max(area.W-ctx.SideW-6, 1), H: max(area.H-2, 1)}
	b.log.Draw(ctx, logArea)
	b.drawProgress(ctx, Rect{X: area.X + 3, Y: area.Bottom() - 1, W: max(area.W-ctx.SideW-6, 1), H: 1})
}

func (b *BootScene) drawProgress(ctx *Context, r Rect) {
	if r.Empty() {
		return
	}
	s := ctx.Screen
	pal := ctx.Palette
	ready := ctx.T >= b.ready
	p := Progress(ctx.T, 0, b.ready)

	label := "initialising"
	status := fmt.Sprintf("%3d%%", int(p*100))
	if ready {
		label = "ready"
		status = "[ READY ]"
	}
	x := s.Text(r.X, r.Y, label, pal.Text, term.ColorDefault, term.Attr(0))
	statusX := r.Right() - term.StringWidth(status)
	barX := x + 2
	barEnd := statusX - 2
	if barEnd-barX < 8 {
		return
	}
	barW := barEnd - barX
	fill := int(p * float64(barW))
	for i := range barW {
		ch, fg := '░', pal.Dim
		if i < fill {
			ch, fg = '█', pal.Accent
		}
		if !ready && i == fill && Pulse(ctx.Sec(ctx.T), 0.6) > 0.5 {
			ch, fg = '█', pal.Accent2
		}
		s.Set(barX+i, r.Y, ch, fg, term.ColorDefault, term.Attr(0))
	}
	fg := pal.Dim
	if ready {
		fg = pal.Accent
	}
	s.Text(statusX, r.Y, status, fg, term.ColorDefault, term.Attr(0))
	if ready {
		blink := Pulse(ctx.Sec(ctx.T), 1.0) > 0.5
		if blink {
			s.Set(r.Right()-1, r.Y, '█', pal.Accent, term.ColorDefault, term.Attr(0))
		}
	}
}

// BlankScene draws nothing but the background layers. It covers the stretch
// where the title banner owns the screen.
type BlankScene struct{}

// Draw implements Layer.
func (BlankScene) Draw(*Context) {}

// OutroScene is the shutdown log that runs after the final execution.
type OutroScene struct {
	log *LogView
}

// NewOutroScene builds the teardown log.
func NewOutroScene(track *lyric.Track, total time.Duration) *OutroScene {
	entries := LogFromLyrics(track, 205*time.Second, total)
	for i := range entries {
		entries[i].Stamp = "[ VOID ]"
		entries[i].Style = LogErr
	}
	extra := []LogEntry{
		{At: 206600 * time.Millisecond, End: 207400 * time.Millisecond, Text: "freeing simulation", Stamp: "[  OK  ]"},
		{At: 207400 * time.Millisecond, End: 208200 * time.Millisecond, Text: "flushing lyric buffers", Stamp: "[  OK  ]"},
		{At: 208200 * time.Millisecond, End: 209000 * time.Millisecond, Text: "unmounting /dev/emotion", Stamp: "[  OK  ]"},
		{At: 209000 * time.Millisecond, End: 209800 * time.Millisecond, Text: "detaching from OBJECT CREATION", Stamp: "[  OK  ]"},
		{At: 209800 * time.Millisecond, End: 210600 * time.Millisecond, Text: "signal SIGTERM received", Style: LogWarn, Stamp: "[ WARN ]"},
		{At: 210600 * time.Millisecond, End: 211400 * time.Millisecond, Text: "goodbye, world", Style: LogPlain},
	}
	v := NewLogView(append(entries, extra...)...)
	v.Prompt = "$ "
	v.CPS = 60
	return &OutroScene{log: v}
}

// Draw renders the teardown log and the exit status line.
func (o *OutroScene) Draw(ctx *Context) {
	area := ctx.Area
	if area.Empty() {
		return
	}
	ctx.PanelWithBar(Rect{X: area.X, Y: area.Y, W: area.W, H: max(area.H-1, 1)}, '▌')
	o.log.Draw(ctx, Rect{X: area.X + 3, Y: area.Y, W: max(area.W-5, 1), H: max(area.H-2, 1)})

	if area.H >= 22 {
		top := area.Y + area.H/2 + 1
		drawHeartMonitor(ctx, Rect{X: area.X + 3, Y: top, W: max(area.W-ctx.SideW-6, 1), H: area.Bottom() - 2 - top})
	}

	row := area.Bottom() - 1
	status := fmt.Sprintf("world.execute(me); — exit code %d", 0)
	if ctx.T < 211400*time.Millisecond {
		return
	}
	text := status
	if ctx.T > 211600*time.Millisecond {
		text = "process terminated"
	}
	x := area.CenterX(term.StringWidth(text))
	ctx.Screen.Text(x, row, text, ctx.Palette.Dim, term.ColorDefault, term.Attr(0))
	if Pulse(ctx.Sec(ctx.T), 1.0) > 0.4 {
		ctx.Screen.Set(x+term.StringWidth(text)+1, row, '█', ctx.Palette.Accent, term.ColorDefault, term.Attr(0))
	}
}
