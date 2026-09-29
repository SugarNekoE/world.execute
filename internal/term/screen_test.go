package term

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"unicode"
)

func newTestScreen(w, h int) (*Screen, *bytes.Buffer) {
	var buf bytes.Buffer
	s := NewScreen(w, h, ColorTrue, bufio.NewWriter(&buf))
	return s, &buf
}

func TestFlushOnlyWritesChanges(t *testing.T) {
	s, buf := newTestScreen(20, 4)
	s.Text(0, 0, "hello", RGB(255, 0, 0), ColorDefault, Bold)
	s.Flush()
	first := buf.Len()
	if first == 0 {
		t.Fatal("first flush wrote nothing")
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("output does not contain the text: %q", buf.String())
	}

	s.Flush()
	if buf.Len() != first {
		t.Errorf("unchanged frame wrote %d extra bytes", buf.Len()-first)
	}

	s.Text(0, 1, "again", ColorDefault, ColorDefault, 0)
	s.Flush()
	if buf.Len() == first {
		t.Error("changed frame wrote nothing")
	}
}

func TestResizeRepaintsEverything(t *testing.T) {
	s, buf := newTestScreen(10, 3)
	s.Text(0, 0, "ab", ColorDefault, ColorDefault, 0)
	s.Flush()
	before := buf.Len()

	s.Resize(12, 3)
	s.Text(0, 0, "ab", ColorDefault, ColorDefault, 0)
	s.Flush()
	if buf.Len() == before {
		t.Error("resize did not repaint")
	}
	if s.W != 12 || s.H != 3 {
		t.Errorf("size = %dx%d", s.W, s.H)
	}
}

func TestWideRuneIsRepairedWhenOverwritten(t *testing.T) {
	s, _ := newTestScreen(10, 2)
	s.Set(1, 0, '世', ColorDefault, ColorDefault, 0)
	if s.At(1, 0).R != '世' {
		t.Fatalf("wide rune not stored: %+v", s.At(1, 0))
	}
	if s.At(2, 0).R != 0 {
		t.Errorf("continuation cell = %q, want 0", s.At(2, 0).R)
	}

	s.Set(1, 0, 'a', ColorDefault, ColorDefault, 0)
	if got := s.At(2, 0).R; got != ' ' {
		t.Errorf("continuation cell = %q, want a blank", got)
	}

	// Painting the second half must clear the whole wide rune.
	s.Set(3, 0, '世', ColorDefault, ColorDefault, 0)
	s.Set(4, 0, 'b', ColorDefault, ColorDefault, 0)
	if got := s.At(3, 0).R; got != ' ' {
		t.Errorf("wide rune head = %q, want a blank", got)
	}
}

func TestWideRuneAtLastColumn(t *testing.T) {
	s, _ := newTestScreen(4, 1)
	s.Set(3, 0, '世', ColorDefault, ColorDefault, 0)
	if got := s.At(3, 0).R; got != ' ' {
		t.Errorf("last column = %q, want a space", got)
	}
}

func TestWideRuneSurvivesFlush(t *testing.T) {
	s, buf := newTestScreen(10, 1)
	s.Set(0, 0, '世', RGB(1, 2, 3), ColorDefault, 0)
	s.Flush()
	if !strings.Contains(buf.String(), "世") {
		t.Errorf("output missing the wide rune: %q", buf.String())
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in   string
		w    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 4, "hel…"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
		{"日本語", 3, "日…"},
	}
	for _, tc := range tests {
		if got := Truncate(tc.in, tc.w); got != tc.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.w, got, tc.want)
		}
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"hello", 5},
		{"日本", 4},
		{"", 0},
		{"a\u0301", 1},
	}
	for _, tc := range tests {
		if got := StringWidth(tc.in); got != tc.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestMeasureAndGlyphCase(t *testing.T) {
	if got := Measure("AB"); got != 11 {
		t.Errorf("Measure(AB) = %d, want 11", got)
	}
	if got := Measure(""); got != 0 {
		t.Errorf("Measure(empty) = %d, want 0", got)
	}
	if Glyph('a') != Glyph('A') {
		t.Error("lowercase should render as the capital glyph")
	}
	if Glyph('A')[1] != "#   #" {
		t.Errorf("glyph A row 1 = %q", Glyph('A')[1])
	}
}

func TestDrawGlyphsRevealsInOrder(t *testing.T) {
	s, _ := newTestScreen(20, 8)
	shown := func(i int) bool { return i < 1 }
	DrawGlyphs(s, 0, 0, "AB", ColorDefault, ColorDefault, 0, shown)
	painted := false
	for y := range 7 {
		for x := range 5 {
			if s.At(x, y).R == '█' {
				painted = true
			}
		}
	}
	if !painted {
		t.Error("the first glyph should be visible")
	}
	for y := range 7 {
		for x := 6; x < 11; x++ {
			if s.At(x, y).R == '█' {
				t.Error("the second glyph should still be hidden")
			}
		}
	}
}

func TestColorQuantisation(t *testing.T) {
	if got := to16(RGB(255, 0, 0)); got != 9 {
		t.Errorf("pure red -> %d, want 9", got)
	}
	if got := to256(RGB(255, 255, 255)); got != 231 {
		t.Errorf("white -> %d, want 231", got)
	}
	if got := to256(RGB(128, 128, 128)); got < 232 || got > 255 {
		t.Errorf("mid grey -> %d, want a grey ramp entry", got)
	}
	mixed := Mix(RGB(0, 0, 0), RGB(255, 255, 255), 0.5)
	if mixed.r() != 127 || mixed.g() != 127 || mixed.b() != 127 {
		t.Errorf("Mix = %#v", mixed)
	}
	if got := Mix(RGB(1, 2, 3), RGB(9, 9, 9), 5); got != RGB(9, 9, 9) {
		t.Errorf("Mix should clamp: %v", got)
	}
}

func TestSGRSequences(t *testing.T) {
	st := style{fg: RGB(255, 0, 0), bg: RGB(0, 0, 255), attr: Bold}
	got := st.sgr(ColorTrue)
	want := "\x1b[0;1;38;2;255;0;0;48;2;0;0;255m"
	if got != want {
		t.Errorf("sgr = %q, want %q", got, want)
	}
	if got := (style{fg: ColorDefault, bg: ColorDefault}).sgr(ColorNone); got != "\x1b[0m" {
		t.Errorf("mono sgr = %q", got)
	}
	if got := (style{fg: RGB(255, 0, 0), bg: ColorDefault}).sgr(Color256); got != "\x1b[0;38;5;196m" {
		t.Errorf("256 sgr = %q", got)
	}
}

func TestParseColorMode(t *testing.T) {
	tests := []struct {
		in      string
		want    ColorMode
		wantErr bool
	}{
		{"truecolor", ColorTrue, false},
		{"256", Color256, false},
		{"none", ColorNone, false},
		{"weird", ColorNone, true},
	}
	for _, tc := range tests {
		got, err := ParseColorMode(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseColorMode(%q) error = %v", tc.in, err)
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("ParseColorMode(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestShiftRow(t *testing.T) {
	s, _ := newTestScreen(6, 1)
	s.Text(0, 0, "abcdef", ColorDefault, ColorDefault, 0)
	s.ShiftRow(0, 2)
	row := ""
	for x := range 6 {
		row += string(s.At(x, 0).R)
	}
	if row != "  abcd" {
		t.Errorf("shift right = %q, want %q", row, "  abcd")
	}
	s.ShiftRow(0, -1)
	row = ""
	for x := range 6 {
		row += string(s.At(x, 0).R)
	}
	if row != " abcd " {
		t.Errorf("shift left = %q, want %q", row, " abcd ")
	}
}

func TestFadeRectSkipsDefaultColors(t *testing.T) {
	s, _ := newTestScreen(4, 1)
	s.Set(0, 0, 'x', RGB(255, 255, 255), ColorDefault, 0)
	s.Set(1, 0, 'y', ColorDefault, ColorDefault, 0)
	s.FadeRect(0, 0, 4, 1, RGB(0, 0, 0), 1)
	if got := s.At(0, 0).Fg; got != RGB(0, 0, 0) {
		t.Errorf("explicit colour = %v, want black", got)
	}
	if got := s.At(1, 0).Fg; got != ColorDefault {
		t.Errorf("default colour was modified: %v", got)
	}
}

func TestSetAttrMergesFlags(t *testing.T) {
	s, _ := newTestScreen(4, 1)
	s.Fill(0, 0, 1, 1, 'z', ColorDefault, ColorDefault, Bold)
	s.SetAttr(0, 0, 4, Reverse)
	if got := s.At(0, 0).A; got != Bold|Reverse {
		t.Errorf("attrs = %b, want %b", got, Bold|Reverse)
	}
}

func TestRestoreSpanCopiesAndRepairsWideRunes(t *testing.T) {
	var buf bytes.Buffer
	s := NewScreen(6, 1, ColorTrue, bufio.NewWriter(&buf))
	s.Text(0, 0, "abcdef", ColorDefault, ColorDefault, 0)
	old := s.Snapshot(nil)
	s.Clear()
	s.Text(0, 0, "世界世", ColorDefault, ColorDefault, 0)
	s.RestoreSpan(old, 0, 3, 6)
	if got := s.At(3, 0).R; got != 'd' {
		t.Errorf("cell 3 = %q, want d", got)
	}
	if got := s.At(2, 0).R; got != ' ' {
		t.Errorf("split wide rune left %q at cell 2, want a space", got)
	}
	s.RestoreSpan(old[:2], 0, 0, 6)
}

func TestSyncedOutputBracketsChangedFrames(t *testing.T) {
	var buf bytes.Buffer
	s := NewScreen(4, 1, ColorTrue, bufio.NewWriter(&buf))
	s.Set(0, 0, 'x', RGB(1, 2, 3), ColorDefault, 0)
	s.Flush()
	out := buf.String()
	if !strings.HasPrefix(out, "\x1b[?2026h") || !strings.HasSuffix(out, "\x1b[?2026l") {
		t.Errorf("changed frame not bracketed: %q", out)
	}
	buf.Reset()
	s.Flush()
	if buf.Len() != 0 {
		t.Errorf("unchanged frame wrote %q", buf.String())
	}
}

func TestStyleDeltaOnlyEmitsWhatChanged(t *testing.T) {
	red := style{fg: RGB(255, 0, 0), bg: ColorDefault}
	blue := style{fg: RGB(0, 0, 255), bg: ColorDefault}
	if got := string(blue.appendDelta(nil, red, ColorTrue)); got != "\x1b[38;2;0;0;255m" {
		t.Errorf("fg delta = %q", got)
	}
	plain := style{fg: ColorDefault, bg: ColorDefault}
	if got := string(plain.appendDelta(nil, red, ColorTrue)); got != "\x1b[39m" {
		t.Errorf("reset to default fg = %q", got)
	}
	bold := style{fg: RGB(255, 0, 0), bg: ColorDefault, attr: Bold}
	if got := string(bold.appendDelta(nil, red, ColorTrue)); got != "\x1b[0;1;38;2;255;0;0m" {
		t.Errorf("attr change should reset: %q", got)
	}
	if got := string(red.appendDelta(nil, red, ColorTrue)); got != "" {
		t.Errorf("identical style emitted %q", got)
	}
	if got := string(blue.appendDelta(nil, red, ColorNone)); got != "" {
		t.Errorf("mono delta emitted %q", got)
	}
}

func TestRuneWidthFastPathAgreesWithTheTable(t *testing.T) {
	for r := rune(0); r < 0x300; r++ {
		want := 1
		switch {
		case r == 0, r < 0x20, r >= 0x7f && r < 0xa0:
			want = 0
		case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
			want = 0
		}
		if got := RuneWidth(r); got != want {
			t.Fatalf("RuneWidth(%U) = %d, want %d", r, got, want)
		}
	}
}

func TestScaleCollapsesTowardsTheCentre(t *testing.T) {
	s, _ := newTestScreen(9, 9)
	for y := range 9 {
		s.Text(0, y, "abcdefghi", ColorDefault, ColorDefault, 0)
	}
	snap := s.Snapshot(nil)
	s.Scale(snap, 1, 1)
	if got := s.At(0, 0).R; got != 'a' {
		t.Errorf("identity scale changed cell to %q", got)
	}
	s.Scale(snap, 0.2, 1)
	filled := 0
	for y := range 9 {
		if s.At(4, y).R != ' ' {
			filled++
		}
	}
	if filled == 0 || filled > 3 {
		t.Errorf("vertical squash kept %d rows, want 1..3", filled)
	}
	s.Scale(snap, 1, 0.25)
	if s.At(0, 4).R != ' ' || s.At(4, 4).R == ' ' {
		t.Error("horizontal squash should empty the edges and keep the middle")
	}
	s.Scale(snap[:3], 0.5, 0.5)
}

func TestASCIIStandIns(t *testing.T) {
	cases := map[rune]rune{
		'a': 'a', '─': '-', '━': '=', '│': '|', '┌': '+', '█': '#', '░': '.', '▁': '_', '▶': '>',
		'✦': '*', '♥': '3', '⏸': '|', '⋅': '.', '→': '>', '⠀': ' ',
	}
	for in, want := range cases {
		if got := ASCII(in); got != want {
			t.Errorf("ASCII(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ASCII('世'); got != '世' {
		t.Errorf("wide glyphs must survive, got %q", got)
	}
	if got := ASCII('☃'); got != '?' {
		t.Errorf("unknown narrow glyph = %q, want ?", got)
	}
}

func TestBrailleBecomesStrokes(t *testing.T) {
	braille := func(dots ...int) rune {
		var bits rune
		for _, d := range dots {
			bits |= 1 << d
		}
		return 0x2800 + bits
	}
	cases := []struct {
		name string
		r    rune
		want rune
	}{
		{"vertical", braille(0, 1, 2, 6), '|'},
		{"horizontal middle", braille(1, 4), '-'},
		{"horizontal bottom", braille(6, 7), '_'},
		{"horizontal top", braille(0, 3), '"'},
		{"falling diagonal", braille(0, 4), '\\'},
		{"rising diagonal", braille(1, 3), '/'},
		{"speck low", braille(2), '.'},
		{"speck high", braille(0), '\''},
		{"full", braille(0, 1, 2, 3, 4, 5, 6, 7), '#'},
	}
	for _, tc := range cases {
		if got := ASCII(tc.r); got != tc.want {
			t.Errorf("%s: ASCII(%U) = %q, want %q", tc.name, tc.r, got, tc.want)
		}
	}
}

func TestScreenWritesASCIIWhenAsked(t *testing.T) {
	s, buf := newTestScreen(8, 1)
	s.SetASCII(true)
	s.Text(0, 0, "─█▶⣿ok", ColorDefault, ColorDefault, 0)
	s.Flush()
	for _, b := range buf.Bytes() {
		if b >= 0x80 {
			t.Fatalf("non-ASCII byte %#x in %q", b, buf.String())
		}
	}
	if !strings.Contains(buf.String(), "-#>#ok") {
		t.Errorf("output %q lacks the stand-ins", buf.String())
	}
	if s.At(0, 0).R != '─' {
		t.Error("the buffer must keep the original glyph")
	}
}
