package term

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ColorMode describes how much color the output terminal can render.
type ColorMode int

const (
	// ColorNone disables color entirely. Attributes such as bold and reverse
	// video still work.
	ColorNone ColorMode = iota
	// Color16 uses the eight base colors and their bright variants.
	Color16
	// Color256 uses the xterm 256 color cube.
	Color256
	// ColorTrue uses 24 bit direct color.
	ColorTrue
)

// Color is a packed 24 bit RGB value. ColorDefault asks the terminal for its
// own default foreground or background.
type Color uint32

// ColorDefault is the terminal's configured foreground or background.
const ColorDefault Color = 1 << 24

// RGB builds a Color from its components.
func RGB(r, g, b uint8) Color {
	return Color(uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

// Hex builds a Color from a 0xRRGGBB literal.
func Hex(v uint32) Color { return Color(v & 0xFFFFFF) }

func (c Color) r() uint8 { return uint8(c >> 16) }
func (c Color) g() uint8 { return uint8(c >> 8) }
func (c Color) b() uint8 { return uint8(c) }

// Attr is a bit set of text attributes.
type Attr uint16

// Text attributes supported by the renderer.
const (
	Bold Attr = 1 << iota
	Dim
	Italic
	Underline
	Blink
	Reverse
	Strike
)

var attrCodes = []struct {
	attr Attr
	code int
}{
	{Bold, 1}, {Dim, 2}, {Italic, 3}, {Underline, 4},
	{Blink, 5}, {Reverse, 7}, {Strike, 9},
}

// ParseColorMode turns a flag value into a ColorMode. Auto defers to the
// environment.
func ParseColorMode(s string) (ColorMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "auto", "":
		return detectColorMode(), nil
	case "none", "off", "mono":
		return ColorNone, nil
	case "16", "ansi", "basic":
		return Color16, nil
	case "256", "xterm":
		return Color256, nil
	case "truecolor", "true", "24bit", "24", "rgb":
		return ColorTrue, nil
	}
	return ColorNone, fmt.Errorf("unknown color mode %q", s)
}

func detectColorMode() ColorMode {
	if os.Getenv("NO_COLOR") != "" {
		return ColorNone
	}
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	if strings.Contains(ct, "truecolor") || strings.Contains(ct, "24bit") {
		return ColorTrue
	}
	term := strings.ToLower(os.Getenv("TERM"))
	switch {
	case term == "" || term == "dumb":
		return ColorNone
	case strings.Contains(term, "truecolor"), strings.Contains(term, "24bit"):
		return ColorTrue
	case strings.Contains(term, "256"):
		return Color256
	}
	return Color16
}

// Mix blends a towards b by k, which is clamped to 0..1. ColorDefault has no
// known value, so blending with it yields the other colour rather than black.
func Mix(a, b Color, k float64) Color {
	switch {
	case a == ColorDefault:
		return b
	case b == ColorDefault:
		return a
	}
	w := uint32(math.Min(math.Max(k, 0), 1)*256 + 0.5)
	mix := func(x, y uint32) uint32 {
		return (x*(256-w) + y*w) >> 8
	}
	return Color(mix(uint32(a>>16&0xff), uint32(b>>16&0xff))<<16 |
		mix(uint32(a>>8&0xff), uint32(b>>8&0xff))<<8 |
		mix(uint32(a&0xff), uint32(b&0xff)))
}

var ansi16 = [16]Color{
	RGB(0x00, 0x00, 0x00), RGB(0xcd, 0x00, 0x00), RGB(0x00, 0xcd, 0x00), RGB(0xcd, 0xcd, 0x00),
	RGB(0x00, 0x00, 0xee), RGB(0xcd, 0x00, 0xcd), RGB(0x00, 0xcd, 0xcd), RGB(0xe5, 0xe5, 0xe5),
	RGB(0x7f, 0x7f, 0x7f), RGB(0xff, 0x00, 0x00), RGB(0x00, 0xff, 0x00), RGB(0xff, 0xff, 0x00),
	RGB(0x5c, 0x5c, 0xff), RGB(0xff, 0x00, 0xff), RGB(0x00, 0xff, 0xff), RGB(0xff, 0xff, 0xff),
}

func dist2(a, b Color) int {
	dr := int(a.r()) - int(b.r())
	dg := int(a.g()) - int(b.g())
	db := int(a.b()) - int(b.b())
	return dr*dr + dg*dg + db*db
}

func to16(c Color) int {
	best, bestD := 0, 1<<30
	for i, cand := range ansi16 {
		if d := dist2(c, cand); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

func to256(c Color) int {
	r, g, b := int(c.r()), int(c.g()), int(c.b())
	if abs(r-g) <= 8 && abs(g-b) <= 8 {
		switch {
		case r < 8:
			return 16
		case r > 248:
			return 231
		default:
			return 232 + (r-8)*24/247
		}
	}
	return 16 + 36*((r*5+127)/255) + 6*((g*5+127)/255) + ((b*5 + 127) / 255)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func appendInt(dst []byte, n int) []byte { return strconv.AppendInt(dst, int64(n), 10) }

func appendColor(dst []byte, mode ColorMode, bg bool, c Color) []byte {
	switch mode {
	case ColorTrue:
		if bg {
			dst = append(dst, "48;2;"...)
		} else {
			dst = append(dst, "38;2;"...)
		}
		dst = appendInt(dst, int(c.r()))
		dst = append(dst, ';')
		dst = appendInt(dst, int(c.g()))
		dst = append(dst, ';')
		return appendInt(dst, int(c.b()))
	case Color256:
		if bg {
			dst = append(dst, "48;5;"...)
		} else {
			dst = append(dst, "38;5;"...)
		}
		return appendInt(dst, to256(c))
	case Color16:
		n := to16(c)
		base := 30
		if bg {
			base = 40
		}
		if n >= 8 {
			base += 60
			n -= 8
		}
		return appendInt(dst, base+n)
	}
	return dst
}

func appendDefault(dst []byte, mode ColorMode, bg bool) []byte {
	if mode == ColorNone {
		return dst
	}
	if bg {
		return append(dst, "49"...)
	}
	return append(dst, "39"...)
}

type style struct {
	fg, bg Color
	attr   Attr
}

func (st style) appendSGR(dst []byte, mode ColorMode) []byte {
	dst = append(dst, "\x1b[0"...)
	for _, a := range attrCodes {
		if st.attr&a.attr != 0 {
			dst = append(dst, ';')
			dst = appendInt(dst, a.code)
		}
	}
	if st.fg != ColorDefault && mode != ColorNone {
		dst = append(dst, ';')
		dst = appendColor(dst, mode, false, st.fg)
	}
	if st.bg != ColorDefault && mode != ColorNone {
		dst = append(dst, ';')
		dst = appendColor(dst, mode, true, st.bg)
	}
	return append(dst, 'm')
}

func (st style) appendDelta(dst []byte, from style, mode ColorMode) []byte {
	if st.attr != from.attr {
		return st.appendSGR(dst, mode)
	}
	if mode == ColorNone || (st.fg == from.fg && st.bg == from.bg) {
		return dst
	}
	dst = append(dst, "\x1b["...)
	open := len(dst)
	if st.fg != from.fg {
		if st.fg == ColorDefault {
			dst = appendDefault(dst, mode, false)
		} else {
			dst = appendColor(dst, mode, false, st.fg)
		}
	}
	if st.bg != from.bg {
		if len(dst) > open {
			dst = append(dst, ';')
		}
		if st.bg == ColorDefault {
			dst = appendDefault(dst, mode, true)
		} else {
			dst = appendColor(dst, mode, true, st.bg)
		}
	}
	return append(dst, 'm')
}

func (st style) sgr(mode ColorMode) string { return string(st.appendSGR(nil, mode)) }
