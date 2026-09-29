package audio

import (
	"sync"
	"time"
)

// Clock reports the position of the currently playing track. Implementations
// are safe for concurrent use.
type Clock interface {
	// Now returns the playback position.
	Now() time.Duration
	// Set resynchronises the clock onto the position reported by a player.
	Set(d time.Duration)
	// Offset shifts the reported position, compensating for player latency.
	Offset(d time.Duration)
	// Pause freezes the clock. It is idempotent.
	Pause()
	// Resume restarts a paused clock. It is idempotent.
	Resume()
	// Paused reports whether the clock is frozen.
	Paused() bool
	// Done is closed when the underlying player exits.
	Done() <-chan struct{}
}

// WallClock is a monotonic Clock that can be paused, resumed and corrected.
type WallClock struct {
	mu      sync.Mutex
	origin  time.Time
	acc     time.Duration
	offset  time.Duration
	paused  bool
	done    chan struct{}
	stopped bool
}

// NewWallClock returns a clock that starts at zero and is running.
func NewWallClock() *WallClock {
	return &WallClock{origin: time.Now(), done: make(chan struct{})}
}

// Now implements Clock.
func (c *WallClock) Now() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nowLocked()
}

func (c *WallClock) nowLocked() time.Duration {
	d := c.acc
	if !c.paused {
		d += time.Since(c.origin)
	}
	return d + c.offset
}

// Set implements Clock.
func (c *WallClock) Set(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acc = d - c.offset
	c.origin = time.Now()
}

// Offset implements Clock.
func (c *WallClock) Offset(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.offset = d
}

// Pause implements Clock.
func (c *WallClock) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paused {
		return
	}
	c.acc += time.Since(c.origin)
	c.paused = true
}

// Resume implements Clock.
func (c *WallClock) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.paused {
		return
	}
	c.origin = time.Now()
	c.paused = false
}

// Paused implements Clock.
func (c *WallClock) Paused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

// Done implements Clock.
func (c *WallClock) Done() <-chan struct{} { return c.done }

// Stop closes the Done channel exactly once.
func (c *WallClock) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped {
		return
	}
	c.stopped = true
	close(c.done)
}
