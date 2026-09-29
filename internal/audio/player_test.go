package audio

import (
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestMPVSeekAndVolume is the integration test for the interactive controls:
// the commands must reach mpv and the clock must follow. It is skipped when mpv
// or the asset is missing.
func TestMPVSeekAndVolume(t *testing.T) {
	const path = "../../assets/world-execute-me.mp3"
	if _, err := exec.LookPath(PlayerMPV); err != nil {
		t.Skip("mpv is not installed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not present", path)
	}

	logger := log.New(io.Discard, "", 0)
	if testing.Verbose() {
		logger = log.New(os.Stderr, "audio: ", 0)
	}
	p, err := StartPlayer(PlayerConfig{Path: path, Kind: PlayerMPV, Start: 10 * time.Second, Volume: 80}, logger)
	if err != nil {
		t.Fatalf("StartPlayer: %v", err)
	}
	defer p.Close()
	if p.Kind != PlayerMPV {
		t.Fatalf("kind = %q, want mpv", p.Kind)
	}

	waitFor(t, 3*time.Second, func() bool {
		_, ok := p.Volume()
		return ok
	}, "mpv to report its volume")

	if v, _ := p.Volume(); v != 80 {
		t.Errorf("initial volume = %d, want 80", v)
	}
	p.SetVolume(35)
	waitFor(t, 2*time.Second, func() bool {
		v, _ := p.Volume()
		return v == 35
	}, "the volume change to reach mpv")

	p.Seek(150 * time.Second)
	waitFor(t, 2*time.Second, func() bool {
		d := p.Clock.Now() - 150*time.Second
		return d < 2*time.Second && d > -2*time.Second
	}, "the clock to follow the seek")

	// Seeking backwards must work too, and must not be undone by the player's
	// stale position reports.
	before := p.Clock.Now()
	p.Seek(20 * time.Second)
	time.Sleep(700 * time.Millisecond)
	after := p.Clock.Now()
	if after < 19*time.Second || after > 23*time.Second {
		t.Errorf("after seeking back the clock is %v (was %v), want about 20s", after, before)
	}

	// The clock must keep running after the seek.
	time.Sleep(300 * time.Millisecond)
	if now := p.Clock.Now(); now <= after {
		t.Errorf("clock stopped after the seek: %v then %v", after, now)
	}

	// Pausing must freeze it and resuming must let it run again.
	if !p.TogglePause() {
		t.Fatal("TogglePause did not report a pause")
	}
	paused := p.Clock.Now()
	time.Sleep(300 * time.Millisecond)
	if d := p.Clock.Now() - paused; d > 50*time.Millisecond {
		t.Errorf("clock advanced while paused by %v", d)
	}
	if p.TogglePause() {
		t.Fatal("TogglePause did not report a resume")
	}
}

func TestSilentPlayerSeekAndVolume(t *testing.T) {
	p, err := StartPlayer(PlayerConfig{Kind: PlayerSilent, Start: 30 * time.Second}, nil)
	if err != nil {
		t.Fatalf("StartPlayer: %v", err)
	}
	defer p.Close()
	if p.Kind != PlayerSilent {
		t.Fatalf("kind = %q, want silent", p.Kind)
	}
	if d := p.Clock.Now() - 30*time.Second; d > 100*time.Millisecond {
		t.Errorf("clock = %v, want to start at 30s", p.Clock.Now())
	}
	p.Seek(90 * time.Second)
	if d := p.Clock.Now() - 90*time.Second; d > 100*time.Millisecond {
		t.Errorf("clock = %v, want 90s after the seek", p.Clock.Now())
	}
	p.Seek(-5 * time.Second)
	if p.Clock.Now() < 0 {
		t.Errorf("clock = %v, want it clamped at zero", p.Clock.Now())
	}
	if _, ok := p.Volume(); ok {
		t.Error("the silent player should not report a volume")
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestSetDelayShiftsTheClockNotThePlayback(t *testing.T) {
	p, err := StartPlayer(PlayerConfig{Kind: PlayerSilent, Start: 10 * time.Second}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	p.Clock.Pause()
	base := p.Clock.Now()
	p.SetDelay(40 * time.Millisecond)
	if got := p.Clock.Now() - base; got != 40*time.Millisecond {
		t.Errorf("clock moved by %v, want 40ms", got)
	}
	if p.Delay() != 40*time.Millisecond {
		t.Errorf("Delay = %v", p.Delay())
	}
	p.SetDelay(-25 * time.Millisecond)
	if got := p.Clock.Now() - base; got != -25*time.Millisecond {
		t.Errorf("clock moved by %v, want -25ms", got)
	}
}
