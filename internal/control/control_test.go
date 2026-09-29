package control

import (
	"testing"
	"time"

	"world.execute/internal/term"
)

func TestMapKeys(t *testing.T) {
	tests := []struct {
		key  term.Key
		want Action
	}{
		{term.Key{Kind: term.KeyRune, Rune: 'q'}, Quit},
		{term.Key{Kind: term.KeyEscape}, Quit},
		{term.Key{Kind: term.KeyCtrlC}, Quit},
		{term.Key{Kind: term.KeySpace}, TogglePause},
		{term.Key{Kind: term.KeyRune, Rune: 'p'}, TogglePause},
		{term.Key{Kind: term.KeyRight}, SeekForward},
		{term.Key{Kind: term.KeyLeft}, SeekBackward},
		{term.Key{Kind: term.KeyUp}, VolumeUp},
		{term.Key{Kind: term.KeyDown}, VolumeDown},
		{term.Key{Kind: term.KeyRune, Rune: 'l'}, SeekForward},
		{term.Key{Kind: term.KeyRune, Rune: 'h'}, SeekBackward},
		{term.Key{Kind: term.KeyRune, Rune: '.'}, SeekForwardSmall},
		{term.Key{Kind: term.KeyRune, Rune: ','}, SeekBackwardSmall},
		{term.Key{Kind: term.KeyRune, Rune: '0'}, SeekStart},
		{term.Key{Kind: term.KeyRune, Rune: 'r'}, Restart},
		{term.Key{Kind: term.KeyRune, Rune: '+'}, VolumeUp},
		{term.Key{Kind: term.KeyRune, Rune: '-'}, VolumeDown},
		{term.Key{Kind: term.KeyRune, Rune: 'm'}, ToggleMute},
		{term.Key{Kind: term.KeyRune, Rune: 's'}, ToggleInfo},
		{term.Key{Kind: term.KeyRune, Rune: '?'}, ToggleHelp},
		{term.Key{Kind: term.KeyRune, Rune: '['}, DelayLater},
		{term.Key{Kind: term.KeyRune, Rune: ']'}, DelayEarlier},
		{term.Key{Kind: term.KeyRune, Rune: '{'}, DelayLaterBig},
		{term.Key{Kind: term.KeyRune, Rune: '}'}, DelayEarlierBig},
		{term.Key{Kind: term.KeyRune, Rune: 'z'}, None},
		{term.Key{Kind: term.KeyRune, Rune: 'x'}, None},
		{term.Key{Kind: term.KeyTab}, None},
	}
	for _, tc := range tests {
		if got := Map(tc.key); got != tc.want {
			t.Errorf("Map(%+v) = %v, want %v", tc.key, got, tc.want)
		}
	}
}

func TestHandleTogglesChrome(t *testing.T) {
	c := New()
	if !c.Info() {
		t.Fatal("info rows should start visible")
	}
	if got := c.Handle(ToggleInfo); got != None {
		t.Errorf("ToggleInfo returned %v, want None", got)
	}
	if c.Info() {
		t.Error("info rows should be hidden now")
	}
	c.Handle(ToggleInfo)
	if !c.Info() {
		t.Error("info rows should be visible again")
	}

	if c.Help() {
		t.Fatal("help should start hidden")
	}
	c.Handle(ToggleHelp)
	if !c.Help() {
		t.Error("help should be visible")
	}
	c.Handle(ToggleHelp)
	if c.Help() {
		t.Error("help should be hidden")
	}
}

func TestHandleKeepsPlaybackActions(t *testing.T) {
	c := New()
	for _, a := range []Action{Quit, TogglePause, SeekForward, VolumeUp, ToggleMute, Restart} {
		if got := c.Handle(a); got != a {
			t.Errorf("Handle(%v) = %v, want it passed through", a, got)
		}
	}
}

func TestHintVisibilityFades(t *testing.T) {
	c := New()
	c.lastInput = time.Now().Add(-HintTimeout + 500*time.Millisecond)
	if !c.HintsVisible() {
		t.Error("hints should still be visible within the timeout")
	}
	if a := c.HintAlpha(); a <= 0 || a >= 1 {
		t.Errorf("alpha = %v, want a partial fade", a)
	}
	c.lastInput = time.Now().Add(-HintTimeout - time.Second)
	if c.HintsVisible() {
		t.Error("hints should have expired")
	}
	if a := c.HintAlpha(); a != 0 {
		t.Errorf("alpha = %v, want 0", a)
	}

	c.info = false
	c.lastInput = time.Now()
	if c.HintsVisible() {
		t.Error("hints should be hidden when the info rows are off")
	}
}

func TestShiftDelayMovesAndClamps(t *testing.T) {
	if got := ShiftDelay(0, DelayEarlier); got != DelayStep {
		t.Errorf("earlier from zero = %v", got)
	}
	if got := ShiftDelay(0, DelayLater); got != -DelayStep {
		t.Errorf("later from zero = %v", got)
	}
	if got := ShiftDelay(DelayStep, DelayLaterBig); got != DelayStep-DelayBigStep {
		t.Errorf("big later = %v", got)
	}
	if got := ShiftDelay(DelayLimit, DelayEarlierBig); got != DelayLimit {
		t.Errorf("delay passed the upper limit: %v", got)
	}
	if got := ShiftDelay(-DelayLimit, DelayLaterBig); got != -DelayLimit {
		t.Errorf("delay passed the lower limit: %v", got)
	}
	if got := ShiftDelay(30*time.Millisecond, SeekForward); got != 30*time.Millisecond {
		t.Errorf("unrelated action changed the delay: %v", got)
	}
}
