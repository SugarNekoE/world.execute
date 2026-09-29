// Command world.execute plays Mili's world.execute (me) ; and renders a
// synchronised lyric animation in the terminal.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"world.execute/internal/audio"
	"world.execute/internal/config"
	"world.execute/internal/control"
	"world.execute/internal/lyric"
	"world.execute/internal/scene"
	"world.execute/internal/term"
)

const maxKeyLag = 34 * time.Millisecond

// version is reported by -version.
const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin, stdout, stderr *os.File) int {
	cfg, err := config.Load(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(stderr, "world.execute:", err)
		return 2
	}
	logger := log.New(io.Discard, "", 0)
	if cfg.Verbose {
		logger = log.New(stderr, "world.execute: ", 0)
	}
	if !cfg.Quiet {
		fmt.Fprintf(stderr, "world.execute(me); v%s - Mili, Miracle Milk (2015)\n", version)
	}

	if err := cfg.Resolve(); err != nil {
		fmt.Fprintln(stderr, "world.execute:", err)
		return 1
	}
	track, err := loadTrack(cfg.LyricsPath, cfg.Lang)
	if err != nil {
		logger.Printf("lyrics unavailable: %v", err)
	}
	total := totalDuration(cfg, track)

	out, closeOut, err := openOutput(cfg, stdout)
	if err != nil {
		fmt.Fprintln(stderr, "world.execute:", err)
		return 1
	}
	defer closeOut()

	defW, defH := cfg.Size()
	if defW == 0 {
		defW = 100
	}
	if defH == 0 {
		defH = 30
	}
	t := term.Open(stdin, out, defW, defH)
	mode, err := term.ParseColorMode(cfg.Color)
	if err != nil {
		fmt.Fprintln(stderr, "world.execute:", err)
		return 2
	}
	screen := term.NewScreen(defW, defH, mode, t.Writer())
	screen.Resize(t.Size())
	screen.SetASCII(cfg.Charset == "ascii")

	analysis := analyse(cfg, stderr, logger)

	player, err := startPlayer(cfg, logger)
	if err != nil {
		fmt.Fprintln(stderr, "world.execute:", err)
		return 1
	}
	defer player.Close()
	if !cfg.Quiet {
		fmt.Fprintf(stderr, "audio: %s%s\n", player.Kind, backendNote(player.Kind))
	}

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	dir := scene.NewDirector(track, total, scene.Options{Seed: seed, Theme: cfg.Theme})
	ctrl := control.New()
	ctx := scene.Context{
		Screen:   screen,
		Lyrics:   track,
		Analysis: analysis,
		Total:    total,
		Seed:     seed,
		FPS:      cfg.FPS,
		Bands:    make([]float32, audio.BandCount),
		Wave:     make([]float32, audio.WavePoints),
		Player:   player.Kind,
		AudioOK:  player.Kind != audio.PlayerSilent && player.Kind != audio.PlayerNone,
		Volume:   cfg.Volume,
	}

	t.Enter()
	defer t.Restore()
	t.WriteTitle("world.execute(me); - Mili")

	return loop(cfg, t, screen, dir, ctrl, &ctx, player, total, stderr)
}

// loop drives the animation, either in real time or as a fast frame dump.
func loop(cfg *config.Config, t *term.Terminal, screen *term.Screen, dir *scene.Director,
	ctrl *control.Controller, ctx *scene.Context, player *audio.Player, total time.Duration, stderr *os.File) int {

	frameDur := time.Second / time.Duration(cfg.FPS)
	deadline := time.Duration(0)
	if cfg.Duration > 0 {
		deadline = cfg.Start + cfg.Duration
	}
	if deadline == 0 || deadline > total {
		deadline = total
	}

	var (
		frame    int
		last     = cfg.Start
		started  = time.Now()
		volume   = cfg.Volume
		delay    = cfg.Delay
		muted    = false
		rendered = 0
		lastDiag = time.Duration(-1)
		rng      = scene.NewFrameRand()
	)

	render := func(now time.Duration) {
		if now < 0 {
			now = 0
		}
		shown := now
		if cfg.Realtime() && !player.Clock.Paused() {
			shown += frameDur
		}
		ctx.T = shown
		ctx.DT = shown - last
		last = shown
		ctx.Frame = frame
		ctx.Clock = time.Since(started)
		rng.Reseed(ctx.Seed, shown, scene.RandomHold)
		ctx.Rand = rng.Rand
		ctx.Paused = player.Clock.Paused()
		ctx.Muted = muted
		ctx.Delay = delay
		ctx.Volume = volume
		if v, ok := player.Volume(); ok && v != volume {
			volume = v
			ctx.Volume = v
		}
		ctx.ShowInfo = ctrl.Info()
		ctx.ShowHelp = ctrl.Help()
		ctx.HintAlpha = ctrl.HintAlpha()
		ctx.Header, ctx.Area, ctx.Transport = layout(screen.W, screen.H, ctx.ShowInfo)
		if ctx.Analysis != nil {
			ctx.Energy = ctx.Analysis.EnergyAt(shown)
		}
		dir.Draw(ctx)
		screen.Flush()
		frame++
		rendered++

		if cfg.Verbose && now/(1*time.Second) != lastDiag {
			lastDiag = now / (1 * time.Second)
			fmt.Fprintf(stderr, "world.execute: t=%s frame=%d vol=%d muted=%t paused=%t player=%s section=%q\n",
				scene.FormatClock(now), frame, volume, muted, player.Clock.Paused(), player.Kind, ctx.Section)
		}
	}

	if !cfg.Realtime() {
		now := cfg.Start
		for range cfg.Frames {
			render(now)
			now += frameDur
			if now > total {
				break
			}
		}
		fmt.Fprintf(stderr, "wrote %d frames to %s\n", rendered, cfg.Record)
		return 0
	}

	keys := t.ReadKeys()
	stop := make(chan struct{})
	defer close(stop)
	resized := t.NotifyResize(stop)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	ticker := time.NewTicker(frameDur)
	defer ticker.Stop()

	defer func() {
		if delay != cfg.Delay {
			config.SaveDelay(delay)
		}
	}()

	playerDone := player.Done()
	for {
		select {
		case <-ticker.C:
			now := player.Clock.Now()
			if now >= deadline {
				return 0
			}
			render(now)

		case key, ok := <-keys:
			if !ok {
				return 0
			}
			act := ctrl.Handle(control.Map(key))
			if idx, ok := control.ChapterIndex(act); ok && idx < len(scene.Chapters) {
				player.Seek(min(scene.Chapters[idx].At, max(total-200*time.Millisecond, 0)))
				act = control.None
			}
			switch act {
			case control.Quit:
				return 0
			case control.TogglePause:
				player.TogglePause()
			case control.SeekForward:
				player.Seek(seekTarget(player.Clock.Now(), control.SeekStep, total))
			case control.SeekBackward:
				player.Seek(seekTarget(player.Clock.Now(), -control.SeekStep, total))
			case control.SeekForwardSmall:
				player.Seek(seekTarget(player.Clock.Now(), control.SeekSmallStep, total))
			case control.SeekBackwardSmall:
				player.Seek(seekTarget(player.Clock.Now(), -control.SeekSmallStep, total))
			case control.SeekStart, control.Restart:
				player.Seek(cfg.Start)
			case control.SeekEnd:
				player.Seek(max(total-time.Second, 0))
			case control.VolumeUp:
				volume = min(volume+5, 130)
				player.SetVolume(volume)
			case control.VolumeDown:
				volume = max(volume-5, 0)
				player.SetVolume(volume)
			case control.ToggleMute:
				muted = !muted
				player.SetMute(muted)
			case control.DelayLater, control.DelayEarlier, control.DelayLaterBig, control.DelayEarlierBig:
				delay = control.ShiftDelay(delay, act)
				player.SetDelay(delay)
			case control.None:
			}
			if frameDur > maxKeyLag {
				render(player.Clock.Now())
			}

		case <-resized:
			screen.Resize(t.Size())
			render(player.Clock.Now())

		case <-playerDone:
			// The player can stop early, for example when there is no audio
			// device. The animation carries on with the wall clock.
			playerDone = nil
			if player.Kind != audio.PlayerSilent {
				fmt.Fprintln(stderr, "audio player stopped, continuing silently")
			}
			render(player.Clock.Now())

		case <-signals:
			return 0
		}
	}
}

func seekTarget(now, delta, total time.Duration) time.Duration {
	return min(max(now+delta, 0), max(total-200*time.Millisecond, 0))
}

// backendNote explains what the viewer can expect from the chosen player, so a
// silent fallback never looks like a broken control.
func backendNote(kind string) string {
	switch kind {
	case audio.PlayerMPV:
		return " - seeking and volume are enabled"
	case audio.PlayerFFplay:
		return " - seeking restarts playback, volume is fixed"
	default:
		return " - silent: seeking and volume move the animation only (install mpv for sound)"
	}
}

// layout splits the screen into the header, the story area and the transport.
func layout(w, h int, info bool) (header, area, transport scene.Rect) {
	transportH := 0
	if info {
		transportH = 2
	}
	headerH := 3
	switch {
	case h < 12:
		headerH = 0
	case h < 16:
		headerH = 2
	}
	if h-transportH-headerH < 3 {
		transportH = 0
		headerH = min(headerH, max(h-3, 0))
	}
	header = scene.Rect{X: 0, Y: 0, W: w, H: headerH}
	transport = scene.Rect{X: 0, Y: h - transportH, W: w, H: transportH}
	area = scene.Rect{X: 0, Y: headerH, W: w, H: max(h-headerH-transportH, 1)}
	return header, area, transport
}

func openOutput(cfg *config.Config, stdout *os.File) (*os.File, func(), error) {
	if cfg.Record == "" {
		return stdout, func() {}, nil
	}
	f, err := os.Create(cfg.Record)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func loadTrack(path, lang string) (*lyric.Track, error) {
	if path == "" {
		return nil, errors.New("no lyric file configured")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	track, err := lyric.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	localize(track, lang)
	return track, nil
}

// localize selects which language is shown. Word timings only exist for the
// primary English lines, so the other modes reveal whole lines.
func localize(track *lyric.Track, lang string) {
	switch lang {
	case "zh":
		for i := range track.Lines {
			if track.Lines[i].Translation == "" {
				continue
			}
			track.Lines[i].Text = track.Lines[i].Translation
			track.Lines[i].Words = nil
			track.Lines[i].Translation = ""
		}
	case "both":
		for i := range track.Lines {
			l := &track.Lines[i]
			if l.Translation == "" || l.Translation == l.Text {
				continue
			}
			l.Text = l.Text + "  ·  " + l.Translation
			l.Words = nil
		}
	}
}

func totalDuration(cfg *config.Config, track *lyric.Track) time.Duration {
	if d, err := audio.Duration(cfg.AudioPath); err == nil && d > 0 {
		return d
	}
	if track != nil && track.Duration > 0 {
		return track.Duration + 12*time.Second
	}
	return 211 * time.Second
}

func analyse(cfg *config.Config, stderr *os.File, logger *log.Logger) *audio.Analysis {
	if cfg.NoAnalysis || !cfg.Realtime() {
		return nil
	}
	if !audio.Available() {
		logger.Printf("ffmpeg not found, using a synthesised pulse")
		return nil
	}
	if !cfg.Quiet {
		fmt.Fprintln(stderr, "analysing audio ...")
	}
	a, err := audio.Analyze(cfg.AudioPath, cfg.FPS)
	if err != nil {
		logger.Printf("analysis failed: %v", err)
		return nil
	}
	return a
}

func startPlayer(cfg *config.Config, logger *log.Logger) (*audio.Player, error) {
	kind := cfg.Player
	if !cfg.Realtime() {
		kind = audio.PlayerSilent
	}
	return audio.StartPlayer(audio.PlayerConfig{
		Path:     cfg.AudioPath,
		Kind:     kind,
		Start:    cfg.Start,
		Duration: cfg.Duration,
		Volume:   cfg.Volume,
		Delay:    cfg.Delay,
		Verbose:  cfg.Verbose,
	}, logger)
}
