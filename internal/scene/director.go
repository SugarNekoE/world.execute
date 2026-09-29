package scene

import (
	"sort"
	"time"

	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

// Layer is anything that can paint itself for one frame.
type Layer interface {
	Draw(*Context)
}

// Options configures a Director.
type Options struct {
	// Seed makes the code rain and the glitch reproducible.
	Seed int64
	// Theme selects the colour scheme: "dark" or "light".
	Theme string
}

// Director owns the storyboard: it picks the section for the current playback
// position, then paints the background, the section scene, the title chrome,
// the glitch and the transport, in that order.
type Director struct {
	sections  []section
	scenes    map[string]Layer
	rain      *Rain
	ambient   *Ambient
	sidecar   *Telemetry
	glitch    *Glitch
	transport *Transport
	banner    *TitleBanner
	palettes  map[string]Palette
	total     time.Duration
	snapshot  []term.Cell
}

const transitionLength = 300 * time.Millisecond

var hardCuts = map[string]bool{
	"execute": true,
	"chorus2": true,
	"final":   true,
	"outro":   true,
}

// NewDirector builds every scene from the lyric timeline.
func NewDirector(track *lyric.Track, total time.Duration, opts Options) *Director {
	abandonLog := NewLogView(LogFromLyrics(track, stamp("1:50.94"), stamp("1:58.38"))...)
	abandonLog.Prompt = "> "
	abandonLog.CPS = 55

	scenes := map[string]Layer{
		"boot":      NewBootScene(track, opts.Seed),
		"blank":     BlankScene{},
		"verse":     NewShapeScene("self.kind() — what am I made of?", shapeSpecs),
		"current":   NewRegisterScene(track, stamp("0:44.37"), stamp("0:51.45"), currentRegisters),
		"travel":    NewRegisterScene(track, stamp("0:51.45"), stamp("0:59.46"), travelRegisters),
		"chorus1":   NewCodeScene("self.satisfy(you)", chorusLines),
		"objects":   NewObjectScene(absurdObjects),
		"gender":    NewRegisterScene(track, stamp("1:28.71"), stamp("1:43.50"), genderRegisters),
		"vibrate":   NewCodeScene("self.feel()", vibrateLines),
		"abandon":   NewDeleteScene(abandonLog, "freed 0x1F lines"),
		"fragments": NewFragmentScene(fragmentPool),
		"trace":     NewTraceScene("panic", traceLines),
		"execute":   NewExecutionScene(executionHits, executionCountdown),
		"chorus2":   NewCodeScene("self.execute(them)", executionLines),
		"love":      NewLoveScene(),
		"trap":      NewTrapScene(finalExecution),
		"final":     NewExecutionScene([]time.Duration{finalExecution}, nil),
		"outro":     NewOutroScene(track, total),
	}

	for name, cues := range lyricVisuals {
		scenes[name] = &IllustratedScene{stage: scenes[name], cues: cues}
	}

	hits := append([]time.Duration(nil), executionHits...)
	hits = append(hits,
		finalExecution,
		stamp("0:47.76"),
		stamp("1:09.33"),
		stamp("2:45.39"),
		stamp("2:52.86"),
	)

	ambient := NewAmbient(opts.Seed)
	ambient.SetRings(keywordMoments(track))

	return &Director{
		sections:  sections,
		scenes:    scenes,
		rain:      NewRain(opts.Seed),
		ambient:   ambient,
		glitch:    NewGlitch(hits...),
		transport: &Transport{},
		banner:    NewTitleBanner(track),
		palettes:  Theme(opts.Theme),
		total:     total,
	}
}

// SectionAt returns the storyboard section playing at t.
func (d *Director) SectionAt(t time.Duration) section {
	if len(d.sections) == 0 {
		return section{}
	}
	return d.sections[d.sectionIndex(t)]
}

func (d *Director) sectionIndex(t time.Duration) int {
	i := sort.Search(len(d.sections), func(i int) bool { return d.sections[i].Start > t }) - 1
	return max(i, 0)
}

// telemetryKinds maps a scene to the sidecar it shows, if any.
var telemetryKinds = map[string]string{
	"boot":      TelemetryLoad,
	"outro":     TelemetryLoad,
	"trace":     TelemetryDump,
	"abandon":   TelemetryDump,
	"fragments": TelemetryDump,
}

// Draw paints one frame. The last moments of a section are wiped away by the
// next one, finishing exactly on the boundary so the new section is complete
// on the beat it belongs to.
func (d *Director) Draw(ctx *Context) {
	if len(d.sections) == 0 {
		return
	}
	i := d.sectionIndex(ctx.T)
	sec := d.sections[i]
	wipe := -1.0
	if i+1 < len(d.sections) {
		next := d.sections[i+1]
		if left := next.Start - ctx.T; left > 0 && left <= transitionLength && !hardCuts[next.Scene] {
			wipe = 1 - float64(left)/float64(transitionLength)
			d.paint(ctx, sec)
			d.snapshot = ctx.Screen.Snapshot(d.snapshot)
			sec = next
		}
	}
	d.paint(ctx, sec)
	if wipe >= 0 {
		d.wipe(ctx, ease(wipe))
	}
	if st, ok := crtAt(ctx.T, sec.Scene); ok {
		d.crt(ctx, st)
	}
	d.fadeOut(ctx)
	d.transport.Draw(ctx, ctx.Area, ctx.Transport)
}

// wipe sweeps a slanted edge across the screen: the new frame shows left of
// it, the snapshot of the old frame right of it.
func (d *Director) wipe(ctx *Context, k float64) {
	s := ctx.Screen
	reach := s.W + 2*s.H + 6
	front := int(k * float64(reach))
	edge := [...]rune{'▓', '▒', '░'}
	for y := range s.H {
		e := front - 2*y
		s.RestoreSpan(d.snapshot, y, e, s.W)
		for j, r := range edge {
			x := e - 1 - j
			if x < 0 || x >= s.W {
				continue
			}
			fg := Blend(ctx.Palette.Accent, ctx.Palette.Shadow, float64(j)*0.3)
			s.Set(x, y, r, fg, term.ColorDefault, term.Bold)
		}
	}
}

// paint draws the background, the section scene and every overlay except the
// fade and the transport.
func (d *Director) paint(ctx *Context, sec section) {
	ctx.Section = sec.Name
	ctx.Palette = d.palette(sec)

	ctx.SideW = 0
	kind, hasSide := telemetryKinds[sec.Scene]
	if hasSide && ctx.Area.W >= 84 && ctx.Area.H >= 10 {
		ctx.SideW = min(36, ctx.Area.W/3)
	}

	ctx.Screen.Clear()
	d.rain.Draw(ctx, ctx.Area, sec.Rain, 1)
	if sc, ok := d.scenes[sec.Scene]; ok && sc != nil {
		sc.Draw(ctx)
	}
	d.banner.DrawHeader(ctx, ctx.Header, sec.Name)
	d.banner.DrawFull(ctx)
	// The ambient layer fills whatever the scenes left empty, then the sidecar
	// takes its reserved strip and the glitch has the last word.
	d.ambient.Draw(ctx, ctx.Area)
	if hasSide && ctx.SideW > 0 {
		if d.sidecar == nil || d.sidecar.kind != kind {
			d.sidecar = NewTelemetry(kind, ctx.Seed)
		}
		strip := Rect{
			X: ctx.Area.Right() - ctx.SideW,
			Y: ctx.Area.Y,
			W: ctx.SideW,
			H: max(ctx.Area.H-2, 1),
		}
		d.sidecar.Draw(ctx, strip)
	}
	if !d.banner.Active(ctx.T) {
		d.glitch.Draw(ctx, ctx.Area, d.glitch.Amount(ctx, sec.Glitch))
	}
}

// palette resolves the colour scheme of a section for this director's theme.
func (d *Director) palette(s section) Palette {
	if p, ok := d.palettes[s.Palette]; ok {
		return p
	}
	if p, ok := Palettes[s.Palette]; ok {
		return p
	}
	return Palettes["verse"]
}

// fadeOut dims the whole picture as the track ends.
func (d *Director) fadeOut(ctx *Context) {
	if d.total <= 0 {
		return
	}
	k := Progress(ctx.T, d.total-1200*time.Millisecond, d.total)
	if k <= 0 {
		return
	}
	ctx.Screen.FadeRect(0, 0, ctx.Screen.W, ctx.Screen.H, ctx.Palette.Shadow, k)
}

// keywordMoments lists when each shouted lyric word of four letters or more is
// sung, keeping ring onsets at least a third of a second apart.
func keywordMoments(track *lyric.Track) []time.Duration {
	if track == nil {
		return nil
	}
	var times []time.Duration
	for _, line := range track.Lines {
		for _, w := range line.Words {
			if len(w.Text) < 4 || !IsKeyword(w.Text) {
				continue
			}
			if n := len(times); n > 0 && w.Time-times[n-1] < 330*time.Millisecond {
				continue
			}
			times = append(times, w.Time)
		}
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	return times
}
