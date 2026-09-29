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

func TestChaptersStartOnSections(t *testing.T) {
	if len(Chapters) != 9 {
		t.Fatalf("%d chapters, want the nine number keys", len(Chapters))
	}
	starts := map[time.Duration]bool{}
	for _, s := range sections {
		starts[s.Start] = true
	}
	var last time.Duration = -1
	for _, c := range Chapters {
		if c.At <= last {
			t.Errorf("chapter %q at %v is not after the previous one", c.Name, c.At)
		}
		last = c.At
		if !starts[c.At] {
			t.Errorf("chapter %q at %v does not start a section", c.Name, c.At)
		}
	}
}

func TestBarMarksTheChapters(t *testing.T) {
	track := loadTrack(t)
	frame := renderGrid(t, nil, track, 120, 32, stamp("1:00.00"))
	bar := ""
	for _, row := range strings.Split(frame, "\n") {
		if strings.ContainsAny(row, "▶⏸") {
			bar = row
		}
	}
	if n := strings.Count(bar, "┃"); n < 6 {
		t.Errorf("bar shows %d chapter marks, want most of the nine: %q", n, bar)
	}
	if !isChapterMark(int(float64(Chapters[2].At)/float64(total)*100), 100, total) {
		t.Error("chapter three should be marked on a 100 cell bar")
	}
}

func testScreen(w, h int) *term.Screen {
	var buf bytes.Buffer
	return term.NewScreen(w, h, term.ColorTrue, bufio.NewWriter(&buf))
}

func countBraille(s *term.Screen) int {
	n := 0
	for y := range s.H {
		for x := range s.W {
			if r := s.At(x, y).R; r >= 0x2800 && r <= 0x28ff && r != 0x2800 {
				n++
			}
		}
	}
	return n
}

func TestStreamsFlowAroundText(t *testing.T) {
	s := testScreen(100, 30)
	a := NewAmbient(2)
	ctx := &Context{Screen: s, Palette: Palettes["verse"], T: 4 * time.Second, Area: Rect{0, 3, 100, 24}}
	for y := 4; y < 26; y++ {
		s.Text(30, y, strings.Repeat("W", 40), term.ColorDefault, term.ColorDefault, 0)
	}
	a.drawStreams(ctx, ctx.Area)
	painted := 0
	for y := range 30 {
		for x := range 100 {
			inText := x >= 30 && x < 70 && y >= 4 && y < 26
			c := s.At(x, y)
			if inText && c.R != 'W' {
				t.Fatalf("a stream overwrote text at %d,%d: %q", x, y, c.R)
			}
			if !inText && c.R != ' ' {
				painted++
			}
		}
	}
	if painted < 40 {
		t.Errorf("streams painted only %d cells", painted)
	}
	tiny := testScreen(10, 4)
	a.drawStreams(&Context{Screen: tiny, Palette: Palettes["verse"], Area: Rect{0, 0, 10, 4}}, Rect{0, 0, 10, 4})
	if strings.TrimSpace(gridText(tiny)) != "" {
		t.Error("streams should skip a tiny area")
	}
}

func TestStreamRowsScrollAndAlternate(t *testing.T) {
	area := Rect{0, 3, 100, 24}
	frame := func(at time.Duration) string {
		s := testScreen(100, 30)
		a := NewAmbient(2)
		a.drawStreams(&Context{Screen: s, Palette: Palettes["verse"], T: at, Area: area}, area)
		return gridText(s)
	}
	if frame(time.Second) == frame(1500*time.Millisecond) {
		t.Error("the streams should move")
	}
	if frame(time.Second) != frame(time.Second) {
		t.Error("streams must be deterministic")
	}
}

func TestSparksBurstAndFall(t *testing.T) {
	s := testScreen(100, 30)
	a := NewAmbient(3)
	a.SetRings([]time.Duration{time.Second})
	ctx := &Context{Screen: s, Palette: Palettes["verse"], Area: Rect{0, 3, 100, 24}}
	count := func(at time.Duration) int {
		s.Clear()
		ctx.T = at
		a.drawSparks(ctx, ctx.Area)
		n := 0
		for y := range 30 {
			for x := range 100 {
				if s.At(x, y).R != ' ' {
					n++
				}
			}
		}
		return n
	}
	if n := count(1500 * time.Millisecond); n < 8 {
		t.Errorf("sparks in flight number only %d", n)
	}
	if n := count(500 * time.Millisecond); n != 0 {
		t.Errorf("%d sparks before the keyword", n)
	}
	if n := count(4 * time.Second); n != 0 {
		t.Errorf("%d sparks after they burned out", n)
	}
}

func TestChromaticSplitOnlyLandsOnBlankCells(t *testing.T) {
	s := testScreen(60, 20)
	d := &Director{glitch: NewGlitch(time.Second)}
	ctx := &Context{Screen: s, Palette: Palettes["glitch"], Area: Rect{0, 2, 60, 16}}
	s.Text(20, 10, "HIT", term.ColorDefault, term.ColorDefault, 0)
	s.Text(23, 10, "X", term.ColorDefault, term.ColorDefault, 0)

	ctx.T = 5 * time.Second
	d.chromatic(ctx)
	if s.At(18, 10).R != ' ' {
		t.Error("ghost appeared with no recent hit")
	}

	ctx.T = time.Second + 10*time.Millisecond
	d.chromatic(ctx)
	if got := s.At(18, 10); got.R != 'H' || got.Fg != Palettes["glitch"].Err {
		t.Errorf("left ghost = %q %x, want a red H", got.R, uint32(got.Fg))
	}
	if got := s.At(25, 10); got.R != 'X' || got.Fg != Palettes["glitch"].Accent2 {
		t.Errorf("right ghost = %q %x, want a cyan X", got.R, uint32(got.Fg))
	}
	for i, r := range "HITX" {
		if s.At(20+i, 10).R != r {
			t.Errorf("original text changed at %d: %q", 20+i, s.At(20+i, 10).R)
		}
	}
	if g := (&Glitch{hits: []time.Duration{0}}); g.Impact(0) < 0.99 || g.Impact(400*time.Millisecond) != 0 {
		t.Error("impact should start at 1 and be gone after a quarter second")
	}
}

func TestSungWordFlashes(t *testing.T) {
	s := testScreen(40, 3)
	line := lyric.Line{Time: 0, Text: "go now", Words: []lyric.Word{{Text: "go", Time: 0, Space: true}, {Text: "now", Time: time.Second}}}
	DrawWordLine(s, 0, 0, line, time.Second+40*time.Millisecond, Palettes["verse"], term.ColorDefault, 40)
	if s.At(3, 0).A&term.Reverse == 0 {
		t.Error("a word should flash in reverse as it is sung")
	}
	s.Clear()
	DrawWordLine(s, 0, 0, line, time.Second+400*time.Millisecond, Palettes["verse"], term.ColorDefault, 40)
	if s.At(3, 0).A&term.Reverse != 0 {
		t.Error("the flash should be over after a moment")
	}
}

func TestSectionNameDecodesIntoPlace(t *testing.T) {
	const name = "OBJECT CREATION"
	if got := scramble(name, 0, 1); got == name || len([]rune(got)) != len([]rune(name)) || got[6] != ' ' {
		t.Errorf("scramble at the start = %q", got)
	}
	if got := scramble(name, 2*time.Second, 1); got != name {
		t.Errorf("scramble after settling = %q", got)
	}
	mid := scramble(name, 300*time.Millisecond, 1)
	if !strings.HasPrefix(mid, "OBJ") || mid == name {
		t.Errorf("scramble half way = %q, want the first letters settled", mid)
	}
}

func TestVignetteDarkensTheEdgesButNotTheText(t *testing.T) {
	s := testScreen(60, 20)
	shadow := term.Hex(0x0b0d10)
	bright := term.Hex(0xe0e0e0)
	s.Set(30, 10, 'c', bright, term.ColorDefault, 0)
	s.Set(0, 0, 'e', bright, term.ColorDefault, 0)
	s.Set(59, 19, 'f', term.ColorDefault, term.ColorDefault, 0)
	s.Set(1, 0, 'g', term.Hex(0x101214), term.ColorDefault, 0)
	s.Vignette(shadow, 0.5)
	if s.At(30, 10).Fg != bright {
		t.Error("the centre should be untouched")
	}
	if s.At(0, 0).Fg == bright {
		t.Error("the corner should be darkened")
	}
	if s.At(59, 19).Fg != term.ColorDefault {
		t.Error("default-coloured text must be left alone")
	}
	if s.At(1, 0).Fg != term.Hex(0x101214) {
		t.Error("text already close to the background must not fade further")
	}
}

func TestASCIICharsetWritesOnlyASCII(t *testing.T) {
	track := loadTrack(t)
	var buf bytes.Buffer
	screen := term.NewScreen(110, 32, term.ColorTrue, bufio.NewWriterSize(&buf, 1<<16))
	screen.SetASCII(true)
	d := NewDirector(track, total, Options{Seed: 3})
	rng := NewFrameRand()
	ctx := Context{
		Screen: screen, Lyrics: track, Total: total, Seed: 3, FPS: 60, Rand: rng.Rand,
		Bands: make([]float32, audio.BandCount), Wave: make([]float32, audio.WavePoints),
		Volume: 80, ShowInfo: true, HintAlpha: 1,
		Header: Rect{0, 0, 110, 3}, Area: Rect{0, 3, 110, 27}, Transport: Rect{0, 30, 110, 2},
	}
	for at := time.Duration(0); at < total; at += 1300 * time.Millisecond {
		ctx.T, ctx.DT = at, 16*time.Millisecond
		rng.Reseed(3, at, RandomHold)
		d.Draw(&ctx)
		screen.Flush()
		for _, b := range buf.Bytes() {
			if b >= 0x80 {
				t.Fatalf("non-ASCII byte %#x at %v", b, at)
			}
		}
		buf.Reset()
	}
}

func TestASCIIArtFillsItsMaskWithStreamingCode(t *testing.T) {
	pal := Palettes["verse"]
	box := Rect{0, 0, 60, 20}
	draw := func(at float64) (*term.Screen, Rect) {
		s := testScreen(60, 20)
		ctx := &Context{Screen: s, Palette: pal, Seed: 1}
		art := tomatoArt(pal, 30, 20, 14, at)
		return s, art.draw(ctx, box, at)
	}
	s, bbox := draw(1)
	if bbox.W < 20 || bbox.H < 8 {
		t.Fatalf("tomato box is only %dx%d cells", bbox.W, bbox.H)
	}
	filled, ascii := 0, true
	for y := range 20 {
		for x := range 60 {
			if r := s.At(x, y).R; r != ' ' {
				filled++
				if r > 0x7f {
					ascii = false
				}
			}
		}
	}
	if filled < 250 {
		t.Errorf("tomato filled only %d cells", filled)
	}
	if !ascii {
		t.Error("ASCII art must be made of ASCII glyphs")
	}
	inside := func(x, y int) bool { return s.At(x, y).R != ' ' }
	if inside(0, 0) || inside(59, 19) {
		t.Error("the corners are outside the shape and must stay empty")
	}
	a, _ := draw(1)
	b, _ := draw(1.4)
	if gridText(a) == gridText(b) {
		t.Error("the streams inside the shape should move")
	}
	c, _ := draw(1)
	if gridText(a) != gridText(c) {
		t.Error("the art must be deterministic")
	}
}

func TestEveryArtObjectDrawsInsideItsBox(t *testing.T) {
	pal := Palettes["verse"]
	box := Rect{10, 3, 60, 22}
	cx, cy, R := 30.0, 22.0, 13.0
	arts := map[string]asciiArt{
		"tomato":   tomatoArt(pal, cx, cy, R, 1),
		"eggplant": eggplantArt(pal, cx, cy, R, 1),
		"cat":      catArt(pal, cx, cy, R, 1),
		"sun":      sunArt(pal, cx, cy, R, 1, 0.5),
		"heart":    heartArt(pal, cx, cy, R, pal.Accent),
		"globe":    globeArt(pal, cx, cy, R, 1),
		"shield":   shieldArt(pal, cx, cy, R),
		"warning":  warningArt(pal, cx, cy, R),
		"cage":     cageArt(pal, cx, cy, R, 1),
		"box":      boxArt(pal, cx, cy, R*1.5, R*0.8, 5),
	}
	for name, art := range arts {
		s := testScreen(80, 28)
		ctx := &Context{Screen: s, Palette: pal, Seed: 1}
		bbox := art.draw(ctx, box, 1)
		painted := 0
		for y := range 28 {
			for x := range 80 {
				inside := x >= box.X && x < box.Right() && y >= box.Y && y < box.Bottom()
				if s.At(x, y).R == ' ' {
					continue
				}
				painted++
				if !inside {
					t.Fatalf("%s painted outside its box at %d,%d", name, x, y)
				}
				if s.At(x, y).R > 0x7f {
					t.Fatalf("%s used non-ASCII glyph %q", name, s.At(x, y).R)
				}
			}
		}
		if painted < 40 {
			t.Errorf("%s painted only %d cells", name, painted)
		}
		if bbox.W < 1 || bbox.H < 1 {
			t.Errorf("%s reported an empty box", name)
		}
	}
}

func TestHUDFramesTheTarget(t *testing.T) {
	s := testScreen(100, 28)
	ctx := &Context{Screen: s, Palette: Palettes["verse"], Seed: 1}
	r := Rect{0, 2, 100, 24}
	box := Rect{35, 8, 30, 12}
	drawHUD(ctx, r, box, "TARGET LOCK :: TEST", []hudRow{{"CLASS", "TEST"}, {"CONF", "0.99"}}, 0.5, 1)
	text := gridText(s)
	for _, want := range []string{"[ TARGET LOCK :: TEST ]", "CLASS", "TEST", "0.99", "[########........]  50%", "+----"} {
		if !strings.Contains(text, want) {
			t.Errorf("HUD lacks %q", want)
		}
	}
	if !strings.Contains(text, "0x") && !strings.ContainsAny(text, "ABCDEF") {
		t.Error("HUD should scroll a hex dump")
	}
}

func TestArtSceneFramesUseTheHUD(t *testing.T) {
	track := loadTrack(t)
	for name, at := range map[string]string{
		"tomato": "1:19.50", "cat": "1:23.00", "globe": "0:15.50", "shield": "0:04.60",
		"memory": "0:08.00", "warning": "2:13.00", "love": "3:00.50", "network": "1:03.50",
	} {
		frame := renderGrid(t, nil, track, 120, 36, stamp(at))
		if !strings.Contains(frame, "[ ") || !strings.Contains(frame, "+----") {
			t.Errorf("%s at %s lacks the HUD brackets and title", name, at)
		}
	}
}
