package term

import "math"

// brailleBits gives the dot position, as column and row, of each braille bit.
var brailleBits = [8][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 1}, {1, 2}, {0, 3}, {1, 3}}

var asciiTable = map[rune]rune{
	'─': '-', '━': '=', '═': '=', '│': '|', '┃': '|', '║': '|', '▏': '|', '▕': '|', '▌': '|', '▍': '|', '▎': '|',
	'┌': '+', '┐': '+', '└': '+', '┘': '+', '├': '+', '┤': '+', '┬': '+', '┴': '+', '┼': '+',
	'█': '#', '▓': '%', '▒': ':', '░': '.', '▀': '"',
	'▁': '_', '▂': '.', '▃': ':', '▄': '-', '▅': '=', '▆': '+', '▇': '#',
	'▶': '>', '⏸': '|', '●': 'o', '○': 'o', '∘': 'o', '•': '.',
	'∙': '.', '⋅': '.', '·': '.', '…': '.',
	'✦': '*', '✧': '*', '★': '*', '☆': '*', '♥': '3', '♡': '3',
	'←': '<', '→': '>', '↑': '^', '↓': 'v', '≈': '~', '—': '-', '–': '-', '×': 'x',
	'∞': '8', 'π': 'p', '∫': 'S', '³': '3', '²': '2', '▸': '>', '◂': '<',
	'‘': '\'', '’': '\'', '“': '"', '”': '"',
}

// ASCII returns a plain ASCII stand-in for a single width glyph. Wide glyphs and
// control characters are returned unchanged so that text in other scripts
// survives.
func ASCII(r rune) rune {
	switch {
	case r < 0x80:
		return r
	case r >= 0x2800 && r <= 0x28ff:
		return brailleASCII(byte(r - 0x2800))
	}
	if a, ok := asciiTable[r]; ok {
		return a
	}
	if runeWidth(r) == 1 {
		return '?'
	}
	return r
}

// brailleASCII picks the ASCII character that best matches a pattern of
// braille dots: a stroke for a line, a dot for a speck, and a denser glyph the
// more dots there are.
func brailleASCII(bits byte) rune {
	var pts [][2]float64
	for i, pos := range brailleBits {
		if bits&(1<<i) != 0 {
			pts = append(pts, [2]float64{float64(pos[0]), float64(pos[1])})
		}
	}
	n := len(pts)
	switch {
	case n == 0:
		return ' '
	case n >= 7:
		return '#'
	case n == 6:
		return '%'
	case n == 5:
		return '&'
	case n == 1:
		if pts[0][1] <= 1 {
			return '\''
		}
		return '.'
	}
	var mx, my float64
	for _, p := range pts {
		mx += p[0]
		my += p[1]
	}
	mx, my = mx/float64(n), my/float64(n)
	var vx, vy, cov float64
	for _, p := range pts {
		vx += (p[0] - mx) * (p[0] - mx)
		vy += (p[1] - my) * (p[1] - my)
		cov += (p[0] - mx) * (p[1] - my)
	}
	switch {
	case n == 4 && vx > 0 && vy > 0 && math.Abs(cov) < 0.01:
		return '='
	case vx == 0:
		return '|'
	case vy == 0:
		switch {
		case my <= 0.5:
			return '"'
		case my >= 2.5:
			return '_'
		}
		return '-'
	case vy > 3*vx:
		return '|'
	case vy < vx/3:
		return '-'
	case cov > 0:
		return '\\'
	}
	return '/'
}

// SetASCII makes the screen write ASCII stand-ins for decorative glyphs.
func (s *Screen) SetASCII(on bool) { s.ascii = on }
