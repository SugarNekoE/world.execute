// Package config resolves how the program should run: which files to use,
// which player to drive, and how the terminal should be painted.
package config

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"world.execute/assets"
)

// Config is the resolved run configuration.
type Config struct {
	AudioPath  string
	LyricsPath string
	Player     string
	Lang       string
	Color      string
	Theme      string
	Charset    string
	Record     string
	Start      time.Duration
	Duration   time.Duration
	Delay      time.Duration
	Volume     int
	Seed       int64
	FPS        int
	Width      int
	Height     int
	Frames     int
	NoAnalysis bool
	Calibrate  bool
	Quiet      bool
	Verbose    bool
}

// Default returns the configuration used when no flags are given.
func Default() *Config {
	return &Config{
		AudioPath:  defaultAudio,
		LyricsPath: defaultLyrics,
		Player:     "auto",
		Lang:       "en",
		Color:      "auto",
		Theme:      "dark",
		Charset:    "ascii",
		Volume:     80,
		FPS:        60,
	}
}

// Register declares every flag on fs.
func (c *Config) Register(fs *flag.FlagSet) {
	fs.StringVar(&c.AudioPath, "audio", c.AudioPath, "audio file to play")
	fs.StringVar(&c.LyricsPath, "lyrics", c.LyricsPath, "LRC lyric file")
	fs.StringVar(&c.Player, "player", c.Player, "audio backend: auto, mpv, ffplay, silent")
	fs.StringVar(&c.Lang, "lang", c.Lang, "lyrics to show: en, zh, both")
	fs.StringVar(&c.Color, "color", c.Color, "color mode: auto, truecolor, 256, 16, none")
	fs.StringVar(&c.Theme, "theme", c.Theme, "color theme: dark or light")
	fs.StringVar(&c.Charset, "charset", c.Charset, "glyphs: ascii for plain terminal art, unicode for blocks and braille")
	fs.StringVar(&c.Record, "record", "", "write frames to a file instead of the terminal")
	fs.IntVar(&c.FPS, "fps", c.FPS, "animation frames per second")
	fs.IntVar(&c.Volume, "volume", c.Volume, "initial volume, 0 to 130")
	fs.IntVar(&c.Width, "width", 0, "force the terminal width in cells")
	fs.IntVar(&c.Height, "height", 0, "force the terminal height in cells")
	fs.IntVar(&c.Frames, "frames", 0, "stop after this many frames, rendered as fast as possible")
	fs.DurationVar(&c.Start, "start", 0, "start offset into the track")
	fs.DurationVar(&c.Duration, "duration", 0, "stop after this much playback")
	fs.DurationVar(&c.Delay, "delay", 0, "shift the animation against the audio: positive shows it earlier, negative later (adjust live with [ and ])")
	fs.Int64Var(&c.Seed, "seed", 0, "seed for the code rain, 0 picks one from the clock")
	fs.BoolVar(&c.NoAnalysis, "no-analysis", false, "skip the ffmpeg spectrum analysis")
	fs.BoolVar(&c.Calibrate, "calibrate", false, "play a click track and a test card to tune the sync with [ and ]")
	fs.BoolVar(&c.Quiet, "quiet", false, "suppress the startup log")
	fs.BoolVar(&c.Verbose, "verbose", false, "let the audio player log to stderr")
}

// Load builds a configuration from the command line.
func Load(args []string) (*Config, error) {
	c := Default()
	fs := flag.NewFlagSet("world.execute", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	c.Register(fs)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	explicit := false
	fs.Visit(func(f *flag.Flag) { explicit = explicit || f.Name == "delay" })
	if !explicit {
		c.Delay = LoadDelay()
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return c, nil
}

// Validate rejects combinations the renderer cannot honour.
func (c *Config) Validate() error {
	switch c.Lang {
	case "en", "zh", "both":
	default:
		return fmt.Errorf("lang must be en, zh or both, got %q", c.Lang)
	}
	switch c.Player {
	case "auto", "mpv", "ffplay", "silent", "none":
	default:
		return fmt.Errorf("player must be auto, mpv, ffplay or silent, got %q", c.Player)
	}
	switch c.Charset {
	case "ascii", "unicode":
	default:
		return fmt.Errorf("charset must be ascii or unicode, got %q", c.Charset)
	}
	switch c.Theme {
	case "dark", "light":
	default:
		return fmt.Errorf("theme must be dark or light, got %q", c.Theme)
	}
	if c.FPS < 1 || c.FPS > 240 {
		return fmt.Errorf("fps must be between 1 and 240, got %d", c.FPS)
	}
	if c.Volume < 0 || c.Volume > 130 {
		return fmt.Errorf("volume must be between 0 and 130, got %d", c.Volume)
	}
	if c.Width < 0 || c.Height < 0 {
		return errors.New("width and height must not be negative")
	}
	if c.Frames > 0 && c.Record == "" {
		c.Record = "world.execute.ansi"
	}
	return nil
}

// Resolve locates the audio and lyric files, looking next to the working
// directory first and then next to the executable.
func (c *Config) Resolve() error {
	audio, err := locate(c.AudioPath)
	if errors.Is(err, os.ErrNotExist) && c.AudioPath == defaultAudio {
		audio, err = extractEmbedded(filepath.Base(defaultAudio))
	}
	if err != nil {
		return err
	}
	c.AudioPath = audio
	if c.LyricsPath != "" {
		lyr, err := locate(c.LyricsPath)
		if errors.Is(err, os.ErrNotExist) && c.LyricsPath == defaultLyrics {
			lyr, err = extractEmbedded(filepath.Base(defaultLyrics))
		}
		if err == nil {
			c.LyricsPath = lyr
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

var (
	defaultAudio  = filepath.Join("assets", "world-execute-me.mp3")
	defaultLyrics = filepath.Join("assets", "lyrics.lrc")
)

// extractEmbedded writes an embedded asset into the user cache directory, once,
// and returns its path. The directory is named after the content, so a new
// build with a different track never reuses a stale copy.
func extractEmbedded(name string) (string, error) {
	data, err := assets.Files.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("asset %s: %w", name, os.ErrNotExist)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	sum := sha256.Sum256(data)
	dir := filepath.Join(base, "world.execute", hex.EncodeToString(sum[:6]))
	path := filepath.Join(dir, name)
	if info, err := os.Stat(path); err == nil && info.Size() == int64(len(data)) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, name+".*")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return path, nil
}

func locate(path string) (string, error) {
	if path == "" {
		return "", os.ErrNotExist
	}
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("asset %s: %w", path, err)
		}
		return path, nil
	}
	dirs := []string{"."}
	if exe, err := os.Executable(); err == nil {
		d := filepath.Dir(exe)
		for range 4 {
			dirs = append(dirs, d)
			parent := filepath.Dir(d)
			if parent == d {
				break
			}
			d = parent
		}
	}
	for _, d := range dirs {
		candidate := filepath.Join(d, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("asset %s: %w", path, os.ErrNotExist)
}

// Size returns the forced terminal size, or zeroes when the terminal decides.
func (c *Config) Size() (int, int) { return c.Width, c.Height }

// Realtime reports whether playback should follow the clock. Frame dumps run as
// fast as they can and skip audio entirely.
func (c *Config) Realtime() bool { return c.Frames <= 0 }

func delayPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "world.execute", "delay"), nil
}

// LoadDelay returns the sync delay saved by an earlier run, or zero.
func LoadDelay() time.Duration {
	path, err := delayPath()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	d, err := time.ParseDuration(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return d
}

// SaveDelay remembers the sync delay for the next run.
func SaveDelay(d time.Duration) error {
	path, err := delayPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(d.String()+"\n"), 0o644)
}
