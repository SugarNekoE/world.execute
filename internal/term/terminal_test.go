package term

import (
	"bufio"
	"os"
	"testing"
	"time"
)

// escapeTest wires a Terminal to a pipe so the real decoder can be exercised,
// including sequences that arrive in pieces.
func escapeTest(t *testing.T, chunks []string, gap time.Duration) *Terminal {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() {
		w.Close()
		r.Close()
	})
	go func() {
		for _, c := range chunks {
			if _, err := w.WriteString(c); err != nil {
				return
			}
			time.Sleep(gap)
		}
	}()
	return &Terminal{in: r}
}

func TestReadEscape(t *testing.T) {
	tests := []struct {
		name string
		seq  string
		want KeyKind
	}{
		{"right", "\x1b[C", KeyRight},
		{"left", "\x1b[D", KeyLeft},
		{"up", "\x1b[A", KeyUp},
		{"down", "\x1b[B", KeyDown},
		{"ss3 up", "\x1bOA", KeyUp},
		{"home", "\x1b[H", KeyHome},
		{"home tilde", "\x1b[1~", KeyHome},
		{"end", "\x1b[F", KeyEnd},
		{"end tilde", "\x1b[8~", KeyEnd},
		{"page up", "\x1b[5~", KeyPageUp},
		{"delete", "\x1b[3~", KeyDelete},
		{"unknown csi", "\x1b[15~", KeyUnknown},
		{"unknown csi with modifier", "\x1b[1;5C", KeyUnknown},
		{"bare escape", "\x1b", KeyEscape},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := escapeTest(t, []string{tc.seq}, 0)
			r := bufio.NewReader(term.in)
			if _, err := r.ReadByte(); err != nil {
				t.Fatalf("read the escape byte: %v", err)
			}
			got := term.readEscape(r)
			if got.Kind != tc.want {
				t.Errorf("readEscape(%q) = %v, want %v", tc.seq, got.Kind, tc.want)
			}
		})
	}
}

// TestReadEscapeSplitDelivery is the regression test for arrow keys turning
// into a bare Escape, which used to quit the animation.
func TestReadEscapeSplitDelivery(t *testing.T) {
	term := escapeTest(t, []string{"\x1b", "[", "C"}, 40*time.Millisecond)
	r := bufio.NewReader(term.in)
	if _, err := r.ReadByte(); err != nil {
		t.Fatalf("read the escape byte: %v", err)
	}
	got := term.readEscape(r)
	if got.Kind != KeyRight {
		t.Fatalf("split arrow key decoded as %v, want KeyRight", got.Kind)
	}
}

func TestReadEscapeLeavesTheStreamUsable(t *testing.T) {
	term := escapeTest(t, []string{"\x1b[Z", "x"}, 10*time.Millisecond)
	r := bufio.NewReader(term.in)
	if _, err := r.ReadByte(); err != nil {
		t.Fatalf("read the escape byte: %v", err)
	}
	if k := term.readEscape(r); k.Kind != KeyUnknown {
		t.Errorf("unknown sequence = %v", k.Kind)
	}
	// The decoder must not have eaten the following key, and reads must keep
	// working after a deadline was set and cleared.
	waitForByte(t, term, r, 'x')
}

func waitForByte(t *testing.T, term *Terminal, r *bufio.Reader, want byte) {
	t.Helper()
	if err := term.in.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	defer term.in.SetReadDeadline(time.Time{})
	for {
		b, err := r.ReadByte()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if b == want {
			return
		}
	}
}
