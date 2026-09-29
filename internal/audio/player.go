package audio

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Player names accepted by PlayerConfig.Kind.
const (
	PlayerAuto   = "auto"
	PlayerMPV    = "mpv"
	PlayerFFplay = "ffplay"
	PlayerSilent = "silent"
	PlayerNone   = "none"
)

// PlayerConfig describes how the track should be played back.
type PlayerConfig struct {
	Path     string
	Kind     string
	Start    time.Duration
	Duration time.Duration
	Volume   int
	// Delay shifts the animation against the audio: positive shows it earlier,
	// negative later, which is how to compensate for slow speakers.
	Delay   time.Duration
	Verbose bool
}

// Player plays the track with an external process and keeps a Clock in sync
// with it. When no player is available it falls back to a silent clock, so the
// animation always runs.
type Player struct {
	Kind  string
	Clock Clock

	clock    *WallClock
	log      *log.Logger
	cfg      PlayerConfig
	ipc      *ipcClient
	socket   string
	procDone chan struct{}
	gen      int64

	syncMu      sync.Mutex
	settleUntil time.Time
	volume      atomic.Int64
	haveVolume  atomic.Bool
	delay       atomic.Int64

	mu        sync.Mutex
	cmd       *exec.Cmd
	pauseMu   sync.Mutex
	closeOnce sync.Once
	done      chan struct{}
}

// StartPlayer launches a player for cfg and returns it. The returned Player is
// always usable; Kind reports which backend was selected.
func StartPlayer(cfg PlayerConfig, logger *log.Logger) (*Player, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	p := &Player{clock: NewWallClock(), log: logger, cfg: cfg, done: make(chan struct{})}
	p.clock.Set(cfg.Start)
	p.delay.Store(int64(cfg.Delay))
	p.clock.Offset(cfg.Delay)
	p.Clock = p.clock
	p.Kind = PlayerSilent

	if cfg.Kind == PlayerNone || cfg.Kind == PlayerSilent || cfg.Path == "" {
		return p, nil
	}

	candidates := []string{PlayerMPV, PlayerFFplay}
	if cfg.Kind != "" && cfg.Kind != PlayerAuto {
		candidates = []string{cfg.Kind}
	}
	for _, kind := range candidates {
		bin, err := exec.LookPath(kind)
		if err != nil {
			continue
		}
		if err := p.launch(kind, bin); err != nil {
			logger.Printf("player %s unavailable: %v", kind, err)
			continue
		}
		return p, nil
	}
	if cfg.Kind != PlayerAuto && cfg.Kind != "" {
		return nil, fmt.Errorf("player %q not found in PATH", cfg.Kind)
	}
	logger.Printf("no audio player found, running silently")
	return p, nil
}

func (p *Player) launch(kind, bin string) error {
	switch kind {
	case PlayerMPV:
		return p.launchMPV(bin)
	case PlayerFFplay:
		return p.launchFFplay(bin)
	}
	return fmt.Errorf("unknown player %q", kind)
}

func (p *Player) launchMPV(bin string) error {
	cfg := p.cfg
	dir := os.TempDir()
	socket := filepath.Join(dir, fmt.Sprintf("world-execute-%d.sock", os.Getpid()))
	if len(socket) > 100 {
		socket = filepath.Join("/tmp", fmt.Sprintf("we-%d.sock", os.Getpid()))
	}
	os.Remove(socket)

	args := []string{
		"--no-video",
		"--audio-display=no",
		"--no-terminal",
		"--no-config",
		"--no-input-default-bindings",
		"--input-ipc-server=" + socket,
		fmt.Sprintf("--volume=%d", cfg.Volume),
		fmt.Sprintf("--start=%.3f", cfg.Start.Seconds()),
		"--msg-level=all=error",
		cfg.Path,
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = nil
	cmd.Stderr = io.Discard
	if cfg.Verbose {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	ipc, err := dialIPC(socket, 2*time.Second)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return fmt.Errorf("mpv ipc: %w", err)
	}

	var syncOnce sync.Once
	ipc.OnTime = func(secs float64) {
		p.acceptPosition(time.Duration(secs*float64(time.Second))+p.Delay(), func() {
			syncOnce.Do(func() { p.log.Printf("audio synced at %.3fs", secs) })
		})
	}
	ipc.OnVolume = func(v float64) {
		p.volume.Store(int64(v))
		p.haveVolume.Store(true)
	}
	ipc.Observe("time-pos", 1)
	ipc.Observe("volume", 2)

	p.begin(cmd, PlayerMPV)
	p.ipc = ipc
	p.socket = socket
	go ipc.Read()
	return nil
}

func (p *Player) launchFFplay(bin string) error {
	cfg := p.cfg
	args := []string{
		"-nodisp", "-autoexit", "-loglevel", "quiet",
		"-ss", fmt.Sprintf("%.3f", cfg.Start.Seconds()),
		"-volume", fmt.Sprint(cfg.Volume),
	}
	if cfg.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", cfg.Duration.Seconds()))
	}
	args = append(args, cfg.Path)

	cmd := exec.Command(bin, args...)
	cmd.Stdin = nil
	cmd.Stderr = io.Discard
	if cfg.Verbose {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	p.begin(cmd, PlayerFFplay)
	return nil
}

// begin registers a freshly started process and watches it.
func (p *Player) begin(cmd *exec.Cmd, kind string) {
	gen := atomic.AddInt64(&p.gen, 1)
	done := make(chan struct{})
	p.mu.Lock()
	p.cmd = cmd
	p.procDone = done
	p.Kind = kind
	p.mu.Unlock()
	go p.watch(cmd, done, gen)
}

// acceptPosition folds a position reported by the player into the clock.
// Reports that arrive while a seek is settling, or that would drag the clock
// backwards, are ignored: mpv keeps emitting property changes for a moment
// after a seek and those stale values would undo it.
func (p *Player) acceptPosition(d time.Duration, firstSync func()) {
	p.syncMu.Lock()
	settling := time.Now().Before(p.settleUntil)
	p.syncMu.Unlock()
	if settling {
		return
	}
	if d < p.clock.Now()-500*time.Millisecond {
		return
	}
	firstSync()
	p.clock.Set(d)
}

// Delay reports how far the animation clock is shifted from the audio.
func (p *Player) Delay() time.Duration { return time.Duration(p.delay.Load()) }

// SetDelay shifts the animation clock against the audio: positive shows the
// animation earlier, negative later.
func (p *Player) SetDelay(d time.Duration) {
	p.delay.Store(int64(d))
	p.clock.Offset(d)
}

// Volume reports the volume the player last reported.
func (p *Player) Volume() (int, bool) {
	if !p.haveVolume.Load() {
		return 0, false
	}
	return int(p.volume.Load()), true
}

// watch waits for the process and shuts the clock down unless it was already
// superseded by a restart.
func (p *Player) watch(cmd *exec.Cmd, done chan struct{}, gen int64) {
	cmd.Wait()
	if atomic.LoadInt64(&p.gen) != gen {
		return
	}
	close(done)
	p.clock.Stop()
	if p.socket != "" {
		os.Remove(p.socket)
	}
	p.closeOnce.Do(func() { close(p.done) })
}

// Done is closed when the player process exits.
func (p *Player) Done() <-chan struct{} { return p.done }

// TogglePause pauses or resumes playback and returns whether it is now paused.
func (p *Player) TogglePause() bool {
	p.pauseMu.Lock()
	defer p.pauseMu.Unlock()

	if p.clock.Paused() {
		p.clock.Resume()
		if p.ipc != nil {
			_ = p.ipc.Set("pause", false)
		}
		return false
	}
	p.clock.Pause()
	if p.ipc != nil {
		_ = p.ipc.Set("pause", true)
	}
	return true
}

// Seek moves playback to d. mpv seeks directly; ffplay restarts at the new
// offset; the silent backend simply moves the clock.
func (p *Player) Seek(d time.Duration) {
	if d < 0 {
		d = 0
	}
	p.pauseMu.Lock()
	paused := p.clock.Paused()
	p.pauseMu.Unlock()

	// Stop trusting the player's own position reports for a moment, so the
	// values it queued before the seek cannot pull the clock back.
	p.syncMu.Lock()
	p.settleUntil = time.Now().Add(400 * time.Millisecond)
	p.syncMu.Unlock()

	switch {
	case p.ipc != nil:
		_ = p.ipc.Command("seek", d.Seconds(), "absolute")
	case p.Kind == PlayerFFplay:
		p.restartFFplay(d)
	}
	p.clock.Set(d)
	if paused {
		p.clock.Pause()
		if p.ipc != nil {
			_ = p.ipc.Set("pause", true)
		}
	}
}

func (p *Player) restartFFplay(d time.Duration) {
	bin, err := exec.LookPath(PlayerFFplay)
	if err != nil {
		return
	}
	p.mu.Lock()
	old := p.cmd
	p.mu.Unlock()
	if old != nil && old.Process != nil {
		old.Process.Kill()
	}

	cfg := p.cfg
	cfg.Start = d
	cfg.Duration = 0
	p.cfg = cfg
	if err := p.launchFFplay(bin); err != nil {
		p.log.Printf("restarting ffplay: %v", err)
	}
}

// SetVolume adjusts the playback volume, 0..130.
func (p *Player) SetVolume(v int) {
	if p.ipc != nil {
		_ = p.ipc.Set("volume", v)
	}
}

// SetMute mutes or unmutes the player.
func (p *Player) SetMute(m bool) {
	if p.ipc != nil {
		_ = p.ipc.Set("mute", m)
	}
}

// Close stops playback and releases the player process.
func (p *Player) Close() {
	if p.ipc != nil {
		p.ipc.Close()
		p.ipc = nil
	}
	p.mu.Lock()
	cmd, done := p.cmd, p.procDone
	p.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(400 * time.Millisecond):
			cmd.Process.Kill()
			select {
			case <-done:
			case <-time.After(400 * time.Millisecond):
			}
		}
	}
	p.clock.Stop()
	if p.socket != "" {
		os.Remove(p.socket)
	}
	p.closeOnce.Do(func() { close(p.done) })
}

// ipcClient is a minimal mpv JSON IPC client. It only understands the handful
// of messages the animation needs.
type ipcClient struct {
	conn   net.Conn
	enc    *json.Encoder
	mu     sync.Mutex
	closed bool

	OnTime   func(secs float64)
	OnVolume func(v float64)
}

func dialIPC(path string, timeout time.Duration) (*ipcClient, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return &ipcClient{conn: conn, enc: json.NewEncoder(conn)}, nil
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	return nil, fmt.Errorf("connect to %s: %w", path, lastErr)
}

func (c *ipcClient) send(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return errors.New("ipc closed")
	}
	return c.enc.Encode(v)
}

// Observe registers a property to be reported on change.
func (c *ipcClient) Observe(name string, id int) error {
	return c.send(map[string]any{"command": []any{"observe_property", id, name}})
}

// Set sets a property.
func (c *ipcClient) Set(name string, value any) error {
	return c.send(map[string]any{"command": []any{"set_property", name, value}})
}

// Command issues an arbitrary mpv command.
func (c *ipcClient) Command(args ...any) error {
	return c.send(map[string]any{"command": args})
}

// Read consumes events until the connection closes.
func (c *ipcClient) Read() {
	sc := bufio.NewScanner(c.conn)
	sc.Buffer(make([]byte, 0, 8192), 1<<20)
	for sc.Scan() {
		var ev struct {
			Event string          `json:"event"`
			Name  string          `json:"name"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Event != "property-change" {
			continue
		}
		switch ev.Name {
		case "time-pos":
			var secs float64
			if err := json.Unmarshal(ev.Data, &secs); err == nil && c.OnTime != nil {
				c.OnTime(secs)
			}
		case "volume":
			var v float64
			if err := json.Unmarshal(ev.Data, &v); err == nil && c.OnVolume != nil {
				c.OnVolume(v)
			}
		}
	}
}

// Close shuts the IPC connection down.
func (c *ipcClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	c.conn.Close()
}
