package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"world.execute/assets"
)

func TestSavedDelayIsUsedUnlessTheFlagIsGiven(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := LoadDelay(); got != 0 {
		t.Fatalf("fresh config delay = %v", got)
	}
	if err := SaveDelay(-40 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if got := LoadDelay(); got != -40*time.Millisecond {
		t.Errorf("LoadDelay = %v, want -40ms", got)
	}
	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Delay != -40*time.Millisecond {
		t.Errorf("Load without a flag used %v, want the saved -40ms", cfg.Delay)
	}
	cfg, err = Load([]string{"-delay", "25ms"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Delay != 25*time.Millisecond {
		t.Errorf("explicit -delay gave %v, want 25ms", cfg.Delay)
	}
}

func TestEmbeddedAssetsBackTheDefaultPaths(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cfg := Default()
	if err := cfg.Resolve(); err != nil {
		t.Fatal(err)
	}
	audio, err := assets.Files.ReadFile("world-execute-me.mp3")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cfg.AudioPath)
	if err != nil || info.Size() != int64(len(audio)) {
		t.Fatalf("extracted audio %s: %v, %v", cfg.AudioPath, info, err)
	}
	if filepath.Base(cfg.LyricsPath) != "lyrics.lrc" {
		t.Errorf("lyrics path = %s", cfg.LyricsPath)
	}
	if _, err := os.Stat(cfg.LyricsPath); err != nil {
		t.Errorf("extracted lyrics missing: %v", err)
	}

	again := Default()
	if err := again.Resolve(); err != nil {
		t.Fatal(err)
	}
	if again.AudioPath != cfg.AudioPath {
		t.Errorf("second run extracted to %s, want the cached %s", again.AudioPath, cfg.AudioPath)
	}
}

func TestOnDiskAssetsWinAndCustomPathsAreNotReplaced(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(dir, "assets", "world-execute-me.mp3")
	if err := os.WriteFile(local, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Default()
	if err := cfg.Resolve(); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(cfg.AudioPath) != "world-execute-me.mp3" || filepath.Dir(cfg.AudioPath) == "" {
		t.Fatalf("audio path = %s", cfg.AudioPath)
	}
	if data, _ := os.ReadFile(cfg.AudioPath); string(data) != "mine" {
		t.Error("the file on disk should win over the embedded copy")
	}

	custom := Default()
	custom.AudioPath = "elsewhere.mp3"
	if err := custom.Resolve(); err == nil {
		t.Error("a missing custom audio path should be an error, not the embedded track")
	}
}
