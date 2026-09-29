// Package control maps key presses to playback actions and tracks the small
// amount of UI state that comes with them, such as whether the key hints are
// still visible.
package control

import (
	"time"

	"world.execute/internal/term"
)

// Action is something the viewer asked the runner to do.
type Action int

// Playback actions.
const (
	None Action = iota
	Quit
	TogglePause
	SeekForward
	SeekBackward
	SeekForwardSmall
	SeekBackwardSmall
	Restart
	SeekStart
	SeekEnd
	VolumeUp
	VolumeDown
	ToggleMute
	ToggleInfo
	ToggleHelp
	DelayLater
	DelayEarlier
	DelayLaterBig
	DelayEarlierBig
	ChapterFirst
)

// ChapterCount is how many chapter keys there are, 1 to 9. Chapter actions
// follow ChapterFirst in order.
const ChapterCount = 9

// Step sizes used by the seek actions.
const (
	SeekStep      = 5 * time.Second
	SeekSmallStep = 1 * time.Second
)

// Step sizes used by the sync actions.
const (
	DelayStep    = 10 * time.Millisecond
	DelayBigStep = 50 * time.Millisecond
	DelayLimit   = 500 * time.Millisecond
)

// HintTimeout is how long the key hint row stays visible after the last input.
const HintTimeout = 5 * time.Second

// Controller turns keys into actions and remembers the visible chrome.
type Controller struct {
	lastInput time.Time
	info      bool
	help      bool
}

// New returns a controller with the info rows visible.
func New() *Controller {
	return &Controller{info: true, lastInput: time.Now()}
}

// Map translates a key press into an action.
func Map(k term.Key) Action {
	if k.Kind != term.KeyRune {
		switch k.Kind {
		case term.KeyEscape, term.KeyCtrlC:
			return Quit
		case term.KeySpace:
			return TogglePause
		case term.KeyRight:
			return SeekForward
		case term.KeyLeft:
			return SeekBackward
		case term.KeyUp:
			return VolumeUp
		case term.KeyDown:
			return VolumeDown
		case term.KeyEnter:
			return TogglePause
		case term.KeyHome:
			return SeekStart
		case term.KeyEnd:
			return SeekEnd
		}
		return None
	}
	switch k.Rune {
	case 'q', 'Q':
		return Quit
	case ' ', 'p', 'P':
		return TogglePause
	case 'l', 'L':
		return SeekForward
	case 'h', 'H':
		return SeekBackward
	case '.', '>':
		return SeekForwardSmall
	case ',', '<':
		return SeekBackwardSmall
	case '0':
		return SeekStart
	case 'r', 'R':
		return Restart
	case '+', '=':
		return VolumeUp
	case '-', '_':
		return VolumeDown
	case 'm', 'M':
		return ToggleMute
	case 's', 'S':
		return ToggleInfo
	case '?', '/':
		return ToggleHelp
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return ChapterFirst + Action(k.Rune-'1')
	case '[':
		return DelayLater
	case ']':
		return DelayEarlier
	case '{':
		return DelayLaterBig
	case '}':
		return DelayEarlierBig
	}
	return None
}

// Handle applies the bookkeeping for an action and returns it unchanged.
// Actions that only affect chrome are resolved here and reported as None.
func (c *Controller) Handle(a Action) Action {
	c.lastInput = time.Now()
	switch a {
	case ToggleInfo:
		c.info = !c.info
		return None
	case ToggleHelp:
		c.help = !c.help
		return None
	}
	return a
}

// Info reports whether the transport and status rows should be drawn.
func (c *Controller) Info() bool { return c.info }

// Help reports whether the help overlay should be drawn.
func (c *Controller) Help() bool { return c.help }

// HintsVisible reports whether the key hint row is still within the timeout.
func (c *Controller) HintsVisible() bool {
	return c.info && time.Since(c.lastInput) < HintTimeout
}

// HintAlpha fades the hint row out over its last second. It returns 0..1.
func (c *Controller) HintAlpha() float64 {
	left := HintTimeout - time.Since(c.lastInput)
	switch {
	case left >= time.Second:
		return 1
	case left <= 0:
		return 0
	}
	return float64(left) / float64(time.Second)
}

// ShiftDelay applies a sync action to delay, keeping it within DelayLimit.
// Later actions move the animation behind the audio, earlier ones ahead.
func ShiftDelay(delay time.Duration, a Action) time.Duration {
	switch a {
	case DelayLater:
		delay -= DelayStep
	case DelayEarlier:
		delay += DelayStep
	case DelayLaterBig:
		delay -= DelayBigStep
	case DelayEarlierBig:
		delay += DelayBigStep
	}
	return min(max(delay, -DelayLimit), DelayLimit)
}

// ChapterIndex reports which chapter, counted from zero, a chapter action
// asks for.
func ChapterIndex(a Action) (int, bool) {
	if a < ChapterFirst || a >= ChapterFirst+ChapterCount {
		return 0, false
	}
	return int(a - ChapterFirst), true
}
