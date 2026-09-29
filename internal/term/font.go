package term

import (
	"strings"
	"unicode"
)

// FontHeight is the number of text rows occupied by one glyph.
const FontHeight = 7

const fontWidth = 5

// Glyph returns the 5x7 bitmap of r. Lowercase letters are drawn as capitals.
// Unknown runes render as a blank.
func Glyph(r rune) [FontHeight]string {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	if g, ok := font[r]; ok {
		return g
	}
	return font[' ']
}

// Measure returns the width in cells of s when rendered with the banner font.
func Measure(s string) int {
	if s == "" {
		return 0
	}
	return len([]rune(s))*(fontWidth+1) - 1
}

// DrawGlyphs renders s with the banner font starting at x, y. Only the cells
// where on returns true are painted, which lets callers reveal characters over
// time. The callback receives the rune index, not the byte offset.
func DrawGlyphs(s *Screen, x, y int, text string, fg, bg Color, attr Attr, on func(i int) bool) {
	cx := x
	index := 0
	for _, r := range text {
		if on == nil || on(index) {
			g := Glyph(r)
			for gy, row := range g {
				for gx, ch := range row {
					if ch != '#' {
						continue
					}
					if bg != ColorDefault {
						s.Set(cx+gx, y+gy, ' ', fg, bg, attr)
					} else {
						s.Set(cx+gx, y+gy, '█', fg, bg, attr)
					}
				}
			}
		}
		cx += fontWidth + 1
		index++
	}
}

var font = map[rune][FontHeight]string{
	' ':  {"     ", "     ", "     ", "     ", "     ", "     ", "     "},
	'A':  {" ### ", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'B':  {"#### ", "#   #", "#   #", "#### ", "#   #", "#   #", "#### "},
	'C':  {" ####", "#    ", "#    ", "#    ", "#    ", "#    ", " ####"},
	'D':  {"#### ", "#   #", "#   #", "#   #", "#   #", "#   #", "#### "},
	'E':  {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#####"},
	'F':  {"#####", "#    ", "#    ", "#### ", "#    ", "#    ", "#    "},
	'G':  {" ####", "#    ", "#    ", "#  ##", "#   #", "#   #", " ####"},
	'H':  {"#   #", "#   #", "#   #", "#####", "#   #", "#   #", "#   #"},
	'I':  {" ### ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", " ### "},
	'J':  {"   ##", "    #", "    #", "    #", "#   #", "#   #", " ### "},
	'K':  {"#   #", "#  # ", "# #  ", "##   ", "# #  ", "#  # ", "#   #"},
	'L':  {"#    ", "#    ", "#    ", "#    ", "#    ", "#    ", "#####"},
	'M':  {"#   #", "## ##", "# # #", "#   #", "#   #", "#   #", "#   #"},
	'N':  {"#   #", "##  #", "# # #", "#  ##", "#   #", "#   #", "#   #"},
	'O':  {" ### ", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'P':  {"#### ", "#   #", "#   #", "#### ", "#    ", "#    ", "#    "},
	'Q':  {" ### ", "#   #", "#   #", "#   #", "# # #", "#  # ", " ## #"},
	'R':  {"#### ", "#   #", "#   #", "#### ", "# #  ", "#  # ", "#   #"},
	'S':  {" ####", "#    ", "#    ", " ### ", "    #", "    #", "#### "},
	'T':  {"#####", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "},
	'U':  {"#   #", "#   #", "#   #", "#   #", "#   #", "#   #", " ### "},
	'V':  {"#   #", "#   #", "#   #", "#   #", "#   #", " # # ", "  #  "},
	'W':  {"#   #", "#   #", "#   #", "#   #", "# # #", "## ##", "#   #"},
	'X':  {"#   #", "#   #", " # # ", "  #  ", " # # ", "#   #", "#   #"},
	'Y':  {"#   #", "#   #", " # # ", "  #  ", "  #  ", "  #  ", "  #  "},
	'Z':  {"#####", "    #", "   # ", "  #  ", " #   ", "#    ", "#####"},
	'0':  {" ### ", "#   #", "#  ##", "# # #", "##  #", "#   #", " ### "},
	'1':  {"  #  ", " ##  ", "  #  ", "  #  ", "  #  ", "  #  ", " ### "},
	'2':  {" ### ", "#   #", "    #", "   # ", "  #  ", " #   ", "#####"},
	'3':  {"#####", "   # ", "  #  ", "   # ", "    #", "#   #", " ### "},
	'4':  {"   # ", "  ## ", " # # ", "#  # ", "#####", "   # ", "   # "},
	'5':  {"#####", "#    ", "#### ", "    #", "    #", "#   #", " ### "},
	'6':  {"  ## ", " #   ", "#    ", "#### ", "#   #", "#   #", " ### "},
	'7':  {"#####", "    #", "   # ", "  #  ", " #   ", " #   ", " #   "},
	'8':  {" ### ", "#   #", "#   #", " ### ", "#   #", "#   #", " ### "},
	'9':  {" ### ", "#   #", "#   #", " ####", "    #", "   # ", " ##  "},
	'.':  {"     ", "     ", "     ", "     ", "     ", " ##  ", " ##  "},
	',':  {"     ", "     ", "     ", "     ", " ##  ", " ##  ", " #   "},
	';':  {"     ", " ##  ", " ##  ", "     ", " ##  ", " ##  ", " #   "},
	':':  {"     ", " ##  ", " ##  ", "     ", " ##  ", " ##  ", "     "},
	'(':  {"   # ", "  #  ", " #   ", " #   ", " #   ", "  #  ", "   # "},
	')':  {" #   ", "  #  ", "   # ", "   # ", "   # ", "  #  ", " #   "},
	'[':  {"  ###", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  ###"},
	']':  {"###  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "###  "},
	'{':  {"   ##", "  #  ", "  #  ", " #   ", "  #  ", "  #  ", "   ##"},
	'}':  {"##   ", "  #  ", "  #  ", "   # ", "  #  ", "  #  ", "##   "},
	'<':  {"   # ", "  #  ", " #   ", "#    ", " #   ", "  #  ", "   # "},
	'>':  {" #   ", "  #  ", "   # ", "    #", "   # ", "  #  ", " #   "},
	'/':  {"    #", "    #", "   # ", "  #  ", " #   ", "#    ", "#    "},
	'\\': {"#    ", "#    ", " #   ", "  #  ", "   # ", "    #", "    #"},
	'-':  {"     ", "     ", "     ", "#####", "     ", "     ", "     "},
	'_':  {"     ", "     ", "     ", "     ", "     ", "     ", "#####"},
	'+':  {"     ", "  #  ", "  #  ", "#####", "  #  ", "  #  ", "     "},
	'=':  {"     ", "     ", "#####", "     ", "#####", "     ", "     "},
	'*':  {"     ", "# # #", " ### ", "#####", " ### ", "# # #", "     "},
	'!':  {"  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "     ", "  #  "},
	'?':  {" ### ", "#   #", "    #", "   # ", "  #  ", "     ", "  #  "},
	'\'': {"  #  ", "  #  ", "  #  ", "     ", "     ", "     ", "     "},
	'"':  {" # # ", " # # ", " # # ", "     ", "     ", "     ", "     "},
	'|':  {"  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  ", "  #  "},
	'$':  {"  #  ", " ####", "# #  ", " ### ", "  # #", "#### ", "  #  "},
	'%':  {"##  #", "##  #", "   # ", "  #  ", " #   ", "#  ##", "#  ##"},
	'&':  {" ##  ", "#  # ", " ##  ", " ## #", "#  # ", "#   #", " ## #"},
	'#':  {" # # ", "#####", " # # ", " # # ", "#####", " # # ", "     "},
	'@':  {" ### ", "#   #", "# ###", "# # #", "# ###", "#    ", " ### "},
	'^':  {"  #  ", " # # ", "#   #", "     ", "     ", "     ", "     "},
	'~':  {"     ", "     ", " ## #", "#  # ", "     ", "     ", "     "},
}

// runeWidth reports how many terminal cells r occupies.
// RuneWidth returns the number of terminal cells r occupies.
func RuneWidth(r rune) int { return runeWidth(r) }

func runeWidth(r rune) int {
	if r >= 0x20 && r < 0x7f {
		return 1
	}
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case unicode.Is(unicode.Mn, r), unicode.Is(unicode.Me, r), unicode.Is(unicode.Cf, r):
		return 0
	case isWide(r):
		return 2
	}
	return 1
}

func isWide(r rune) bool {
	switch {
	case r < 0x1100:
		return false
	}
	return r <= 0x115f ||
		(r >= 0x2e80 && r <= 0x303e) ||
		(r >= 0x3041 && r <= 0x33ff) ||
		(r >= 0x3400 && r <= 0x4dbf) ||
		(r >= 0x4e00 && r <= 0x9fff) ||
		(r >= 0xa000 && r <= 0xa4cf) ||
		(r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) ||
		(r >= 0xfe30 && r <= 0xfe6f) ||
		(r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) ||
		(r >= 0x1f300 && r <= 0x1f64f) ||
		(r >= 0x1f900 && r <= 0x1f9ff) ||
		(r >= 0x20000 && r <= 0x3fffd)
}

// StringWidth reports how many terminal cells s occupies.
func StringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// Sanitize replaces control characters that would corrupt the screen.
func Sanitize(s string) string {
	if !strings.ContainsFunc(s, unicode.IsControl) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
