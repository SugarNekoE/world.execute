package term

import (
	"bufio"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// KeyKind classifies a key press.
type KeyKind int

// Key kinds reported by ReadKeys.
const (
	KeyRune KeyKind = iota
	KeyEscape
	KeyEnter
	KeySpace
	KeyBackspace
	KeyTab
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
	KeyCtrlC
	// KeyUnknown is an escape sequence that carries no action.
	KeyUnknown
)

// Key is a decoded key press.
type Key struct {
	Kind KeyKind
	Rune rune
}

// Terminal owns the controlling terminal: raw mode, the alternate screen
// buffer, cursor visibility and size polling.
type Terminal struct {
	in    *os.File
	out   *os.File
	state *term.State
	isTTY bool

	width, height int
	buf           *bufio.Writer
	entered       bool
}

// Open prepares a terminal bound to in and out. When in is not a character
// device the terminal runs in headless mode: no raw mode, and the size falls
// back to the given default.
func Open(in, out *os.File, defWidth, defHeight int) *Terminal {
	t := &Terminal{
		in:     in,
		out:    out,
		isTTY:  term.IsTerminal(int(in.Fd())) && term.IsTerminal(int(out.Fd())),
		width:  defWidth,
		height: defHeight,
		buf:    bufio.NewWriterSize(out, 1<<16),
	}
	t.width, t.height = t.Size()
	return t
}

// IsTTY reports whether the terminal is attached to a real character device.
func (t *Terminal) IsTTY() bool { return t.isTTY }

// Writer returns the buffered writer to hand to a Screen.
func (t *Terminal) Writer() *bufio.Writer { return t.buf }

// Size returns the current terminal size in cells.
func (t *Terminal) Size() (int, int) {
	if w, h, err := term.GetSize(int(t.out.Fd())); err == nil && w > 0 && h > 0 {
		t.width, t.height = w, h
		return w, h
	}
	return t.width, t.height
}

// Enter switches to raw mode, hides the cursor and activates the alternate
// screen buffer.
func (t *Terminal) Enter() error {
	if t.isTTY {
		state, err := term.MakeRaw(int(t.in.Fd()))
		if err != nil {
			return err
		}
		t.state = state
	}
	t.buf.WriteString("\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H")
	t.buf.Flush()
	t.entered = true
	return nil
}

// Restore returns the terminal to its original state. It is safe to call more
// than once.
func (t *Terminal) Restore() {
	if !t.entered {
		return
	}
	t.entered = false
	t.buf.WriteString("\x1b[0m\x1b[?25h\x1b[?1049l")
	t.buf.Flush()
	if t.state != nil {
		term.Restore(int(t.in.Fd()), t.state)
		t.state = nil
	}
}

// Flush pushes buffered output to the device.
func (t *Terminal) Flush() { t.buf.Flush() }

// NotifyResize delivers SIGWINCH notifications until stop is closed.
func (t *Terminal) NotifyResize(stop <-chan struct{}) <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	if !t.isTTY {
		return ch
	}
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		<-stop
		signal.Stop(ch)
		close(ch)
	}()
	return ch
}

// ReadKeys decodes key presses from the input until the terminal is closed.
// A read failure stops the decoder but leaves the channel open, so losing the
// keyboard never takes the animation down with it.
func (t *Terminal) ReadKeys() <-chan Key {
	ch := make(chan Key, 16)
	if !t.isTTY {
		return ch
	}
	go func() {
		defer func() {
			t.in.SetReadDeadline(time.Time{})
			recover()
		}()
		r := bufio.NewReader(t.in)
		for {
			b, err := r.ReadByte()
			if err != nil {
				if isTimeout(err) {
					continue
				}
				return
			}
			switch {
			case b == 0x03:
				ch <- Key{Kind: KeyCtrlC}
			case b == 0x1b:
				k := t.readEscape(r)
				if k.Kind != KeyUnknown {
					ch <- k
				}
			case b == '\r' || b == '\n':
				ch <- Key{Kind: KeyEnter}
			case b == 0x7f || b == 0x08:
				ch <- Key{Kind: KeyBackspace}
			case b == '\t':
				ch <- Key{Kind: KeyTab}
			case b == ' ':
				ch <- Key{Kind: KeySpace}
			case b < 0x20:
			default:
				if b >= utf8.RuneSelf {
					full := []byte{b}
					for len(full) < utf8.UTFMax {
						next, err := r.ReadByte()
						if err != nil {
							break
						}
						full = append(full, next)
						if utf8.FullRune(full) {
							break
						}
					}
					if rr, _ := utf8.DecodeRune(full); rr != utf8.RuneError {
						ch <- Key{Kind: KeyRune, Rune: rr}
					}
					continue
				}
				ch <- Key{Kind: KeyRune, Rune: rune(b)}
			}
		}
	}()
	return ch
}

// escapeBudget is how long the decoder waits for the rest of an escape
// sequence. Terminals, PTYs and SSH sessions all split sequences, so this must
// be generous: bailing out early would turn an arrow key into a bare Escape.
const escapeBudget = 120 * time.Millisecond

// readEscape consumes a full escape sequence and decodes it. It returns
// KeyUnknown for sequences that carry no action instead of pretending they
// were a bare Escape.
func (t *Terminal) readEscape(r *bufio.Reader) Key {
	var seq []byte
	deadline := time.Now().Add(escapeBudget)
	for len(seq) < 16 {
		if r.Buffered() == 0 {
			wait := time.Until(deadline)
			if wait <= 0 {
				break
			}
			if err := t.in.SetReadDeadline(time.Now().Add(wait)); err != nil {
				break
			}
		}
		b, err := r.ReadByte()
		if err != nil {
			if r.Buffered() > 0 {
				continue
			}
			break
		}
		seq = append(seq, b)
		if len(seq) == 1 {
			if b != '[' && b != 'O' {
				break
			}
			continue
		}
		// A CSI sequence ends at the first byte in 0x40..0x7e.
		if b >= 0x40 && b <= 0x7e {
			break
		}
	}
	t.in.SetReadDeadline(time.Time{})

	if len(seq) == 0 {
		return Key{Kind: KeyEscape}
	}
	if seq[0] != '[' && seq[0] != 'O' {
		return Key{Kind: KeyUnknown}
	}
	params := string(seq[1:])
	switch params {
	case "A":
		return Key{Kind: KeyUp}
	case "B":
		return Key{Kind: KeyDown}
	case "C":
		return Key{Kind: KeyRight}
	case "D":
		return Key{Kind: KeyLeft}
	case "H", "1~", "7~":
		return Key{Kind: KeyHome}
	case "F", "4~", "8~":
		return Key{Kind: KeyEnd}
	case "5~":
		return Key{Kind: KeyPageUp}
	case "6~":
		return Key{Kind: KeyPageDown}
	case "3~":
		return Key{Kind: KeyDelete}
	}
	return Key{Kind: KeyUnknown}
}

func isTimeout(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded)
}

// WriteTitle sets the terminal window title when supported.
func (t *Terminal) WriteTitle(s string) {
	t.buf.WriteString("\x1b]0;")
	t.buf.WriteString(Sanitize(s))
	t.buf.WriteString("\x07")
}

var _ io.Writer = (*Terminal)(nil)

// Write implements io.Writer against the buffered output.
func (t *Terminal) Write(p []byte) (int, error) { return t.buf.Write(p) }
