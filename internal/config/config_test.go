package config

import (
	"testing"
	"time"
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
