package scene

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"world.execute/internal/audio"
	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

const total = 211*time.Second + 906*time.Millisecond

func loadTrack(t *testing.T) *lyric.Track {
	t.Helper()
	f, err := os.Open("../../assets/lyrics.lrc")
	if os.IsNotExist(err) {
		t.Skip("assets/lyrics.lrc not present")
	}
	if err != nil {
		t.Fatalf("open lyrics: %v", err)
	}
	defer f.Close()
	track, err := lyric.Parse(f)
	if err != nil {
		t.Fatalf("parse lyrics: %v", err)
	}
	return track
}

func TestSectionAtMatchesStoryboard(t *testing.T) {
	d := &Director{sections: sections}
	tests := []struct {
		at   time.Duration
		want string
	}{
		{-time.Second, "boot"},
		{0, "boot"},
		{5 * time.Second, "boot"},
		{19*time.Second + 110*time.Millisecond, "world.execute(me);"},
		{29*time.Second + 500*time.Millisecond, "world.execute(me);"},
		{29*time.Second + 880*time.Millisecond, "OBJECT CREATION"},
		{35 * time.Second, "OBJECT CREATION"},
		{45 * time.Second, "SWITCH CURRENT"},
		{55 * time.Second, "TRAVEL"},
		{60 * time.Second, "STIMULATIONS"},
		{80 * time.Second, "ABSURD OBJECTS"},
		{90 * time.Second, "SWITCH GENDER"},
		{105 * time.Second, "VIBRATIONS"},
		{115 * time.Second, "ABANDONMENT"},
		{120 * time.Second, "FRAGMENTS"},
		{130 * time.Second, "ILLEGAL ARGUMENTS"},
		{150 * time.Second, "EXECUTION"},
		{170 * time.Second, "EXECUTION II"},
		{180 * time.Second, "LO-O-OVE"},
		{190 * time.Second, "TRAPPED IN LO-O-OVE"},
		{206 * time.Second, "FINAL EXECUTION"},
		{210 * time.Second, "EXIT"},
		{time.Hour, "EXIT"},
	}
	for _, tc := range tests {
		if got := d.SectionAt(tc.at).Name; got != tc.want {
			t.Errorf("SectionAt(%v) = %q, want %q", tc.at, got, tc.want)
		}
	}
}

func TestSectionsAreContiguous(t *testing.T) {
	for i := 1; i < len(sections); i++ {
		if sections[i].Start != sections[i-1].End {
			t.Errorf("gap between %q and %q: %v then %v",
				sections[i-1].Name, sections[i].Name, sections[i-1].End, sections[i].Start)
		}
	}
	if last := sections[len(sections)-1]; last.End != 0 {
		t.Errorf("last section %q has an end time of %v", last.Name, last.End)
	}
}

func TestEverySectionHasAScene(t *testing.T) {
	d := NewDirector(nil, total, Options{Seed: 1})
	for _, s := range sections {
		if _, ok := d.scenes[s.Scene]; !ok {
			t.Errorf("section %q references unknown scene %q", s.Name, s.Scene)
		}
		if _, ok := Palettes[s.Palette]; !ok {
			t.Errorf("section %q references unknown palette %q", s.Name, s.Palette)
		}
	}
}

func TestStampsMatchTheLyricFile(t *testing.T) {
	track := loadTrack(t)
	// Storyboard cues must coincide with a sung line or word.
	check := func(what string, at time.Duration) {
		t.Helper()
		line, _, ok := track.At(at)
		if !ok {
			t.Errorf("%s at %v has no lyric line", what, at)
			return
		}
		if d := line.Time - at; d > 20*time.Millisecond || d < -20*time.Millisecond {
			for _, word := range line.Words {
				if d := word.Time - at; d <= 20*time.Millisecond && d >= -20*time.Millisecond {
					return
				}
			}
			t.Errorf("%s at %v pairs with the line at %v", what, at, line.Time)
		}
	}
	for _, l := range shapeSpecs {
		check("shape stanza", l.at)
	}
	for _, l := range chorusLines {
		check("chorus line", l.at)
	}
	for _, l := range executionLines {
		check("execution line", l.at)
	}
	for _, h := range executionHits {
		check("execution hit", h)
	}
	for _, c := range executionCountdown {
		check("countdown step", c.at)
	}
	for _, r := range currentRegisters {
		check("current register", r.at)
	}
	for _, r := range travelRegisters {
		check("travel register", r.at)
	}
	for _, r := range genderRegisters {
		check("gender register", r.at)
	}
	for _, o := range absurdObjects {
		check("object", o.at)
		check("object keyword", o.kwAt)
	}
	for scene, cues := range lyricVisuals {
		for _, cue := range cues {
			check(scene+" illustration", cue.at)
		}
	}
}

// TestRenderFrames renders one frame from every section. It is also the
// preview used while working on the visuals: run it with -v to see the frames.
func TestRenderFrames(t *testing.T) {
	track := loadTrack(t)
	d := NewDirector(track, total, Options{Seed: 20250929})

	// Use the real envelope when it is available so the preview shows the
	// waveform and the reactive shapes.
	var an *audio.Analysis
	if audio.Available() {
		if _, err := os.Stat("../../assets/world-execute-me.mp3"); err == nil {
			if got, err := audio.Analyze("../../assets/world-execute-me.mp3", 60); err == nil {
				an = got
			}
		}
	}

	const (
		w = 100
		h = 32
	)
	times := []struct {
		label string
		at    time.Duration
	}{
		{"boot start", 1200 * time.Millisecond},
		{"boot power line", 2 * time.Second},
		{"boot ready", 15 * time.Second},
		{"title WORLD.", 20 * time.Second},
		{"title complete", 29*time.Second + 950*time.Millisecond},
		{"verse points", 33 * time.Second},
		{"verse limit", 44 * time.Second},
		{"switch current", 47 * time.Second},
		{"travel", 55 * time.Second},
		{"chorus stimulations", 62 * time.Second},
		{"chorus simulation", 72*time.Second + 900*time.Millisecond},
		{"objects eggplant", 78 * time.Second},
		{"objects god", 88 * time.Second},
		{"gender", 92 * time.Second},
		{"vibrations", 106 * time.Second},
		{"abandon", 115 * time.Second},
		{"fragments", 122 * time.Second},
		{"trace", 133 * time.Second},
		{"execution hit", 147*time.Second + 950*time.Millisecond},
		{"execution waiting", 150 * time.Second},
		{"countdown", 159*time.Second + 100*time.Millisecond},
		{"chorus two", 168 * time.Second},
		{"love", 181 * time.Second},
		{"trapped", 190 * time.Second},
		{"trapped countdown", 200 * time.Second},
		{"final execution", 205*time.Second + 950*time.Millisecond},
		{"outro", 211 * time.Second},
	}
	for _, tc := range times {
		grid := renderGridWith(t, d, track, an, w, h, tc.at)
		if strings.TrimSpace(grid) == "" {
			t.Errorf("%s at %v rendered an empty frame", tc.label, tc.at)
		}
		t.Logf("--- %s (%v) ---\n%s", tc.label, tc.at, grid)
	}
}

func TestFrameContent(t *testing.T) {
	track := loadTrack(t)
	d := NewDirector(track, total, Options{Seed: 7})
	tests := []struct {
		at   time.Duration
		want string
	}{
		{20 * time.Second, "Mili — Miracle Milk (2015)"},
		{35 * time.Second, "self instanceof Circle"},
		{62 * time.Second, "STIMULATIONS"},
		{78 * time.Second, "world.instantiate"},
		{130 * time.Second, "ILLEGAL ARGUMENTS"},
		{150 * time.Second, "EXECUTION"},
		{190 * time.Second, "I am trapped"},
		{211500 * time.Millisecond, "exit code 0"},
	}
	for _, tc := range tests {
		grid := renderGrid(t, d, track, 120, 36, tc.at)
		if !strings.Contains(grid, tc.want) {
			t.Errorf("frame at %v does not contain %q:\n%s", tc.at, tc.want, grid)
		}
	}
}

func TestTransportBarIsDrawn(t *testing.T) {
	track := loadTrack(t)
	d := NewDirector(track, total, Options{Seed: 3})
	grid := renderGrid(t, d, track, 100, 32, 62*time.Second)
	lines := strings.Split(grid, "\n")
	bar := lines[len(lines)-3]
	if !strings.Contains(bar, "01:02.00") {
		t.Errorf("transport bar = %q, want the elapsed time", bar)
	}
	if !strings.Contains(bar, "03:31.90") {
		t.Errorf("transport bar = %q, want the total time", bar)
	}
	if !strings.Contains(bar, "━") || !strings.Contains(bar, "●") {
		t.Errorf("transport bar = %q, want a filled track and a handle", bar)
	}
}

func TestWideRunesInLyricsDoNotBreakTheScreen(t *testing.T) {
	track := loadTrack(t)
	for i := range track.Lines {
		track.Lines[i].Text = track.Lines[i].Translation
		track.Lines[i].Words = nil
	}
	d := NewDirector(track, total, Options{Seed: 4})
	grid := renderGrid(t, d, track, 100, 32, 35*time.Second)
	if strings.Contains(grid, "\x00") {
		t.Error("frame contains a NUL cell")
	}
	if strings.TrimSpace(grid) == "" {
		t.Error("frame is empty")
	}
}

// TestHeaderScopeDraws checks that the live waveform really paints into the
// header when an envelope is available. It needs ffmpeg.
func TestHeaderScopeDraws(t *testing.T) {
	if !audio.Available() {
		t.Skip("ffmpeg and ffprobe are not installed")
	}
	if _, err := os.Stat("../../assets/world-execute-me.mp3"); err != nil {
		t.Skip("the audio asset is not present")
	}
	an, err := audio.Analyze("../../assets/world-execute-me.mp3", 60)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	track := loadTrack(t)
	d := NewDirector(track, total, Options{Seed: 5})

	for _, at := range []time.Duration{62 * time.Second, 150 * time.Second} {
		grid := renderGridWith(t, d, track, an, 100, 32, at)
		lines := strings.Split(grid, "\n")
		if len(lines) < 2 {
			t.Fatalf("frame at %v is too short", at)
		}
		if !strings.ContainsAny(lines[1], "⠀⠁⠂⠄⡀⢀⠈⠐⠠") {
			t.Errorf("header row at %v has no waveform:\n%s", at, lines[1])
		}
		if got := an.WaveAt(at, nil); len(got) != audio.WavePoints {
			t.Errorf("WaveAt returned %d points, want %d", len(got), audio.WavePoints)
		}
	}
}

// TestShapesReactToTheBeat checks that the storyboards really are driven by the
// track rather than being static.
func TestShapesReactToTheBeat(t *testing.T) {
	if !audio.Available() {
		t.Skip("ffmpeg and ffprobe are not installed")
	}
	if _, err := os.Stat("../../assets/world-execute-me.mp3"); err != nil {
		t.Skip("the audio asset is not present")
	}
	an, err := audio.Analyze("../../assets/world-execute-me.mp3", 60)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	// The chorus is loud and the isolation is quiet, so the frames must differ.
	loud := renderGridWith(t, nil, loadTrack(t), an, 100, 32, 150*time.Second)
	quiet := renderGridWith(t, nil, loadTrack(t), an, 100, 32, 117*time.Second)
	if loud == quiet {
		t.Error("loud and quiet frames are identical, the visuals are not reactive")
	}
}

// TestHintRowIsReadable guards the bug where the key hints were drawn in the
// background colour, which made them invisible on a dark terminal.
func TestHintRowIsReadable(t *testing.T) {
	track := loadTrack(t)
	d := NewDirector(track, total, Options{Seed: 3})
	for _, theme := range []string{"dark", "light"} {
		d = NewDirector(track, total, Options{Seed: 3, Theme: theme})
		screen := renderScreen(t, d, track, nil, 100, 32, 62*time.Second)
		row := 32 - 1
		seen, checked := 0, 0
		for x := range 100 {
			c := screen.At(x, row)
			if c.R == ' ' || c.R == 0 {
				continue
			}
			seen++
			if c.Fg == term.ColorDefault {
				continue
			}
			checked++
			if l := luminance(c.Fg); l < 0.3 {
				t.Errorf("%s theme: hint cell %q at %d has fg %#06x, luminance %.2f — too dark to read",
					theme, string(c.R), x, uint32(c.Fg), l)
			}
		}
		if seen < 20 {
			t.Errorf("%s theme: the hint row only drew %d cells", theme, seen)
		}
		if checked == 0 {
			t.Errorf("%s theme: the hint row has no explicit colours at all", theme)
		}
	}
}

func luminance(c term.Color) float64 {
	r := float64(c>>16&0xff) / 255
	g := float64(c>>8&0xff) / 255
	b := float64(c&0xff) / 255
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func randFor(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

// renderGrid draws one frame and returns it as plain text.
func renderGrid(t *testing.T, d *Director, track *lyric.Track, w, h int, at time.Duration) string {
	t.Helper()
	return renderGridWith(t, d, track, nil, w, h, at)
}

// renderGridWith draws one frame with an optional envelope attached.
func renderGridWith(t *testing.T, d *Director, track *lyric.Track, an *audio.Analysis, w, h int, at time.Duration) string {
	t.Helper()
	return gridText(renderScreen(t, d, track, an, w, h, at))
}

// renderScreen draws one frame and returns the cell buffer.
func renderScreen(t *testing.T, d *Director, track *lyric.Track, an *audio.Analysis, w, h int, at time.Duration) *term.Screen {
	t.Helper()
	var buf bytes.Buffer
	screen := term.NewScreen(w, h, term.ColorTrue, bufio.NewWriter(&buf))

	seed := int64(9)
	if d == nil {
		d = NewDirector(track, total, Options{Seed: seed})
	}
	ctx := Context{
		Screen:    screen,
		Lyrics:    track,
		Analysis:  an,
		Total:     total,
		Seed:      seed,
		FPS:       60,
		T:         at,
		Bands:     make([]float32, audio.BandCount),
		Wave:      make([]float32, audio.WavePoints),
		Volume:    80,
		ShowInfo:  true,
		HintAlpha: 1,
		Rand:      randFor(seed),
	}
	if an != nil {
		ctx.Energy = an.EnergyAt(at)
	}
	ctx.Header = Rect{X: 0, Y: 0, W: w, H: 3}
	ctx.Transport = Rect{X: 0, Y: h - 2, W: w, H: 2}
	ctx.Area = Rect{X: 0, Y: 3, W: w, H: h - 5}

	d.Draw(&ctx)
	screen.Flush()
	return screen
}

func gridText(s *term.Screen) string {
	var b strings.Builder
	for y := range s.H {
		for x := range s.W {
			c := s.At(x, y)
			if c.R == 0 {
				continue
			}
			b.WriteRune(c.R)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestSectionChangeWipesFromTheOldFrame(t *testing.T) {
	track := loadTrack(t)
	start := stamp("0:29.88")
	mid := renderGrid(t, nil, track, 110, 32, start-150*time.Millisecond)
	if n := strings.Count(mid, "▓"); n < 20 {
		t.Errorf("mid transition has %d wipe cells, want an edge on most rows", n)
	}
	after := renderGrid(t, nil, track, 110, 32, start+50*time.Millisecond)
	if strings.Contains(after, "░▒▓") {
		t.Error("wipe edge still visible after the transition")
	}
}

func TestHardCutsSkipTheWipe(t *testing.T) {
	track := loadTrack(t)
	for _, name := range []string{"2:27.87", "3:25.86", "3:28.30"} {
		frame := renderGrid(t, nil, track, 110, 32, stamp(name)-100*time.Millisecond)
		if strings.Contains(frame, "░▒▓") {
			t.Errorf("%s should cut, not wipe", name)
		}
	}
}

func TestFrameRandIsDeterministicAndHeldBetweenSlots(t *testing.T) {
	r := NewFrameRand()
	r.Reseed(7, 100*time.Millisecond, RandomHold)
	a := r.Intn(1 << 20)
	r.Reseed(7, 100*time.Millisecond+RandomHold/4, RandomHold)
	if b := r.Intn(1 << 20); a != b {
		t.Errorf("same slot gave %d then %d", a, b)
	}
	r.Reseed(7, 100*time.Millisecond+RandomHold, RandomHold)
	if c := r.Intn(1 << 20); a == c {
		t.Errorf("next slot repeated %d", a)
	}
}

func TestTitleInstrumentalShowsTheLinkProgress(t *testing.T) {
	track := loadTrack(t)
	frame := renderGrid(t, nil, track, 110, 32, stamp("0:24.00"))
	for _, want := range []string{"▕", "%", "resolving execute"} {
		if !strings.Contains(frame, want) {
			t.Errorf("title instrumental frame lacks %q", want)
		}
	}
	early := renderGrid(t, nil, track, 110, 32, stamp("0:19.50"))
	if strings.Contains(early, "resolving") {
		t.Error("link progress should not start before the ghost line")
	}
}

func TestVerseReadoutFollowsTheShape(t *testing.T) {
	track := loadTrack(t)
	cases := map[string]string{"0:31.00": "spread", "0:35.00": "C = 2πr", "0:38.50": "dy/dx", "0:42.00": "1/x"}
	for at, want := range cases {
		if frame := renderGrid(t, nil, track, 110, 32, stamp(at)); !strings.Contains(frame, want) {
			t.Errorf("verse frame at %s lacks %q", at, want)
		}
	}
	if frame := renderGrid(t, nil, track, 50, 20, stamp("0:35.00")); strings.Contains(frame, "C = 2πr") {
		t.Error("readout should be dropped on a narrow terminal")
	}
}

func TestHeartMonitorFlatlines(t *testing.T) {
	if g := monitorGain(monitorFail - time.Second); g != 1 {
		t.Errorf("gain before failure = %v, want 1", g)
	}
	if g := monitorGain(monitorFlat); g != 0 {
		t.Errorf("gain at flatline = %v, want 0", g)
	}
	track := loadTrack(t)
	if frame := renderGrid(t, nil, track, 110, 32, stamp("3:29.60")); !strings.Contains(frame, "072 bpm") {
		t.Error("beating monitor missing its reading")
	}
	if frame := renderGrid(t, nil, track, 110, 32, stamp("3:31.50")); !strings.Contains(frame, "asystole") {
		t.Error("flatlined monitor missing asystole")
	}
	if frame := renderGrid(t, nil, track, 80, 16, stamp("3:31.50")); strings.Contains(frame, "ECG") {
		t.Error("monitor should be dropped on a short terminal")
	}
}

func TestCountdownLocksOnEachNumber(t *testing.T) {
	track := loadTrack(t)
	early := renderGrid(t, nil, track, 110, 32, stamp("2:39.00"))
	late := renderGrid(t, nil, track, 110, 32, stamp("2:39.30"))
	if early == late {
		t.Error("lock-on rings should shrink between the start and end of a step")
	}
}

func centreRow(frame string) string {
	lines := strings.Split(frame, "\n")
	return lines[(len(lines)-1)/2]
}

func TestCRTPowerOffAndOn(t *testing.T) {
	track := loadTrack(t)
	if _, ok := crtAt(stamp("3:20.00"), "trap"); ok {
		t.Error("crt effect outside the boot and final scenes")
	}
	const line = "━━━━━━━━"
	if row := centreRow(renderGrid(t, nil, track, 110, 32, 300*time.Millisecond)); !strings.Contains(row, line) {
		t.Errorf("power-on frame lacks the bright line: %q", row)
	}
	off := renderGrid(t, nil, track, 110, 32, crtOffAt+crtOffLength*7/10)
	if !strings.Contains(centreRow(off), line) || strings.Contains(off, "world.execute") {
		t.Error("power-off frame should be a bare line with the picture gone")
	}
	dot := renderGrid(t, nil, track, 110, 32, crtOffAt+crtOffLength-30*time.Millisecond)
	if row := centreRow(dot); !strings.ContainsAny(row, "●·") || strings.Contains(row, line) {
		t.Errorf("power-off should end on a single dot: %q", row)
	}
	if row := centreRow(renderGrid(t, nil, track, 110, 32, 2*time.Second)); strings.Contains(row, line) {
		t.Errorf("boot should be steady after the power-on: %q", row)
	}
}

func TestFragmentsNeverShareARow(t *testing.T) {
	var pool []string
	for i := range 26 {
		pool = append(pool, strings.Repeat(string(rune('A'+i)), 5)+strings.Repeat(string(rune('a'+(i*7)%26)), 3))
	}
	for _, at := range []string{"1:58.50", "2:00.20", "2:02.90", "2:04.70"} {
		var buf bytes.Buffer
		screen := term.NewScreen(100, 30, term.ColorTrue, bufio.NewWriter(&buf))
		ctx := Context{Screen: screen, Palette: Palettes["void"], Seed: 4, T: stamp(at), Rand: randFor(1),
			Area: Rect{X: 0, Y: 3, W: 100, H: 9}}
		NewFragmentScene(pool).Draw(&ctx)
		for y := ctx.Area.Y; y < ctx.Area.Bottom(); y++ {
			var row strings.Builder
			for x := range screen.W {
				row.WriteRune(screen.At(x, y).R)
			}
			seen := 0
			for _, p := range pool {
				if strings.Contains(row.String(), p[:5]) {
					seen++
				}
			}
			if seen > 1 {
				t.Errorf("at %s row %d shows %d fragments: %q", at, y, seen, strings.TrimSpace(row.String()))
			}
		}
	}
}

func TestAbandonmentShowsThePingTimeouts(t *testing.T) {
	track := loadTrack(t)
	frame := renderGrid(t, nil, track, 110, 32, stamp("1:55.00"))
	for _, want := range []string{"ping you", "timeout", "packet loss"} {
		if !strings.Contains(frame, want) {
			t.Errorf("abandonment frame lacks %q", want)
		}
	}
}

func contrastLuminance(c uint32) float64 {
	channel := func(v uint32) float64 {
		x := float64(v&0xff) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c>>16) + 0.7152*channel(c>>8) + 0.0722*channel(c)
}

func contrastRatio(a, b uint32) float64 {
	la, lb := contrastLuminance(a), contrastLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestNoTextVanishesIntoTheBackground(t *testing.T) {
	track := loadTrack(t)
	const w, h = 110, 32
	for _, theme := range []string{"dark", "light"} {
		termFg, termBg := uint32(0xe6e6e6), uint32(0x000000)
		if theme == "light" {
			termFg, termBg = 0x1a1a1a, 0xffffff
		}
		var buf bytes.Buffer
		screen := term.NewScreen(w, h, term.ColorTrue, bufio.NewWriter(&buf))
		d := NewDirector(track, total, Options{Seed: 3, Theme: theme})
		rng := NewFrameRand()
		ctx := Context{
			Screen: screen, Lyrics: track, Total: total, Seed: 3, FPS: 60, Rand: rng.Rand,
			Bands: make([]float32, audio.BandCount), Wave: make([]float32, audio.WavePoints),
			Volume: 80, ShowInfo: true, HintAlpha: 1,
			Header: Rect{0, 0, w, 3}, Area: Rect{0, 3, w, h - 5}, Transport: Rect{0, h - 2, w, 2},
		}
		for at := 1500 * time.Millisecond; at < total-1500*time.Millisecond; at += 400 * time.Millisecond {
			ctx.T, ctx.DT = at, 16*time.Millisecond
			rng.Reseed(3, at, RandomHold)
			d.Draw(&ctx)
			hidden := 0
			var first string
			for y := range h {
				for x := range w {
					c := screen.At(x, y)
					if c.R == 0 || c.R == ' ' {
						continue
					}
					fg, bg := termFg, termBg
					if c.Fg != term.ColorDefault {
						fg = uint32(c.Fg)
					}
					if c.Bg != term.ColorDefault {
						bg = uint32(c.Bg)
					}
					if c.A&term.Reverse != 0 {
						fg, bg = bg, fg
					}
					if contrastRatio(fg, bg) < 1.2 {
						if hidden == 0 {
							first = fmt.Sprintf("%q at %d,%d fg=%06x", c.R, x, y, fg)
						}
						hidden++
					}
				}
			}
			if hidden > 6 {
				t.Errorf("%s theme at %v: %d cells are unreadable against the background, first %s (%s)", theme, at, hidden, first, d.SectionAt(at).Name)
			}
		}
	}
}

func TestTrapLyricCompletesBeforeTheCountdown(t *testing.T) {
	track := loadTrack(t)
	frame := renderGrid(t, nil, track, 110, 32, stamp("3:13.40"))
	if n := strings.Count(frame, "LO-O-OVE"); n < 2 {
		t.Errorf("the sung LO-O-OVE should be complete beside the header at 3:13.40, found %d", n)
	}
	if !strings.Contains(frame, "Trapped in") {
		t.Error("the phrase should read Trapped in / LO-O-OVE together")
	}
	late := renderGrid(t, nil, track, 110, 32, stamp("3:20.00"))
	if n := strings.Count(late, "LO-O-OVE"); n != 1 {
		t.Errorf("only the header should still spell LO-O-OVE by 3:20, found %d", n)
	}
}

func TestRainSpansTheWholeWidth(t *testing.T) {
	var buf bytes.Buffer
	screen := term.NewScreen(120, 30, term.ColorTrue, bufio.NewWriter(&buf))
	rain := NewRain(1)
	ctx := &Context{Screen: screen, Palette: Palettes["verse"]}
	for _, density := range []float64{0.18, 0.3, 0.6} {
		var thirds [3]int
		for step := range 40 {
			screen.Clear()
			ctx.T = time.Duration(step) * 700 * time.Millisecond
			rain.Draw(ctx, Rect{0, 0, 120, 30}, density, 1)
			for y := range 30 {
				for x := range 120 {
					if screen.At(x, y).R != ' ' {
						thirds[x/40]++
					}
				}
			}
		}
		for i, n := range thirds {
			if n == 0 {
				t.Errorf("density %.2f leaves the %d/3 of the screen without rain: %v", density, i+1, thirds)
			}
		}
	}
}

func TestKeywordsSendRingsAcrossTheEmptySpace(t *testing.T) {
	var buf bytes.Buffer
	screen := term.NewScreen(100, 30, term.ColorTrue, bufio.NewWriter(&buf))
	a := NewAmbient(5)
	a.SetRings([]time.Duration{time.Second})
	ctx := &Context{Screen: screen, Palette: Palettes["verse"], Area: Rect{0, 3, 100, 24}}
	count := func(at time.Duration) int {
		screen.Clear()
		ctx.T = at
		a.drawRings(ctx, ctx.Area)
		n := 0
		for y := range 30 {
			for x := range 100 {
				if screen.At(x, y).R != ' ' {
					n++
				}
			}
		}
		return n
	}
	if n := count(1400 * time.Millisecond); n < 30 {
		t.Errorf("ring in flight painted only %d cells", n)
	}
	if n := count(5 * time.Second); n != 0 {
		t.Errorf("finished ring left %d cells", n)
	}
	if n := count(500 * time.Millisecond); n != 0 {
		t.Errorf("ring painted %d cells before its keyword", n)
	}
	screen.Text(50, 15, "WORD", term.ColorDefault, term.ColorDefault, 0)
	ctx.T = 1500 * time.Millisecond
	a.drawRings(ctx, ctx.Area)
	if screen.At(50, 15).R != 'W' {
		t.Error("ring overwrote text")
	}
}

func TestLoveRainFallsAsHearts(t *testing.T) {
	var buf bytes.Buffer
	screen := term.NewScreen(120, 30, term.ColorTrue, bufio.NewWriter(&buf))
	rain := NewRain(1)
	ctx := &Context{Screen: screen, Palette: Palettes["love"], T: 5 * time.Second}
	rain.Draw(ctx, Rect{0, 0, 120, 30}, 0.6, 1)
	hearts := 0
	for y := range 30 {
		for x := range 120 {
			if screen.At(x, y).R == '♥' {
				hearts++
			}
		}
	}
	if hearts == 0 {
		t.Error("love rain shows no hearts")
	}
}

func TestRainDoesNotHugTheLeftEdge(t *testing.T) {
	var buf bytes.Buffer
	screen := term.NewScreen(120, 30, term.ColorTrue, bufio.NewWriter(&buf))
	rain := NewRain(1)
	ctx := &Context{Screen: screen, Palette: Palettes["verse"]}
	for _, density := range []float64{0.15, 0.3, 0.6} {
		var column [120]int
		for step := range 60 {
			screen.Clear()
			ctx.T = time.Duration(step) * 500 * time.Millisecond
			rain.Draw(ctx, Rect{0, 0, 120, 30}, density, 1)
			for y := range 30 {
				for x := range 120 {
					if screen.At(x, y).R != ' ' {
						column[x]++
					}
				}
			}
		}
		edge := column[0] + column[1] + column[2]
		total := 0
		for _, n := range column {
			total += n
		}
		if mean := float64(total) / 120 * 3; float64(edge) > 2.2*mean {
			t.Errorf("density %.2f: the three leftmost columns hold %d cells, %.1fx the average", density, edge, float64(edge)/mean)
		}
	}
}

func TestWallRisesFasterThanItFalls(t *testing.T) {
	const frame = 1.0 / 60
	if up := wallFollow(frame, true); up < 0.7 {
		t.Errorf("one frame of attack covers only %.2f of the gap", up)
	}
	if down := wallFollow(frame, false); down > 0.35 {
		t.Errorf("one frame of release covers %.2f of the gap, want a slow glide", down)
	}
	if wallFollow(0, true) != 0 {
		t.Error("no time should mean no movement")
	}
}
