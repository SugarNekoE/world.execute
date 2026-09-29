// Package lyric parses LRC lyric files, including the word level timestamps
// used by enhanced LRC, and exposes them as a timeline queried by playback
// time.
package lyric

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// Word is a single lyric word together with the moment it is sung.
type Word struct {
	Text string
	Time time.Duration
	// Space reports whether the source text had a space after this word,
	// which is what lets "world.execute(me);" keep its punctuation tight.
	Space bool
}

// Line is one lyric line.
//
// Text is the primary rendering, normally English. Words carries the per word
// timings when the source provides them. Translation is an optional second
// language rendering of the same line, paired by identical timestamps.
type Line struct {
	Time        time.Duration
	Text        string
	Words       []Word
	Translation string
}

// Track is a parsed lyric timeline.
type Track struct {
	Lines    []Line
	Duration time.Duration
}

// At reports the line playing at d, the index of the word being sung within
// that line, and whether any line is active. The word index is -1 for lines
// that carry no word timings.
func (t *Track) At(d time.Duration) (Line, int, bool) {
	i := t.IndexAt(d)
	if i < 0 {
		return Line{}, -1, false
	}
	line := t.Lines[i]
	return line, activeWord(line, d), true
}

// IndexAt returns the index of the line playing at d, or -1 before the first
// line.
func (t *Track) IndexAt(d time.Duration) int {
	return sort.Search(len(t.Lines), func(i int) bool {
		return t.Lines[i].Time > d
	}) - 1
}

func activeWord(l Line, d time.Duration) int {
	if len(l.Words) == 0 {
		return -1
	}
	return sort.Search(len(l.Words), func(i int) bool {
		return l.Words[i].Time > d
	}) - 1
}

// End returns the moment the line at index i stops being sung.
func (t *Track) End(i int) time.Duration {
	if i < 0 || i >= len(t.Lines) {
		return t.Duration
	}
	if i+1 < len(t.Lines) {
		return t.Lines[i+1].Time
	}
	return t.Duration
}

// Parse reads an LRC stream. Enhanced word timestamps of the form
// [00:12.34]<00:12.34>word are understood, as are plain lines and the
// multi-timestamp form [00:01.00][00:05.00]text.
func Parse(r io.Reader) (*Track, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []Line
	var pending []Line
	for n := 1; sc.Scan(); n++ {
		raw := strings.TrimSpace(strings.TrimSuffix(sc.Text(), "\r"))
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		times, rest, err := splitTags(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if len(times) == 0 {
			continue
		}
		text, words := parseWords(rest)
		if text == "" {
			continue
		}
		for k, at := range times {
			// Word timestamps are absolute, except that a repeated line has
			// to be shifted by the distance between its own timestamps.
			shift := time.Duration(0)
			if k > 0 && len(words) > 0 {
				shift = at - times[0]
			}
			ws := make([]Word, len(words))
			for i, w := range words {
				ws[i] = w
				ws[i].Time = w.Time + shift
			}
			pending = append(pending, Line{Time: at, Text: text, Words: ws})
		}
	}

	lines = mergeTranslations(pending)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })

	t := &Track{Lines: lines}
	if n := len(lines); n > 0 {
		t.Duration = lines[n-1].Time
	}
	return t, sc.Err()
}

// splitTags removes every leading [mm:ss.xx] timestamp and returns them along
// with the remaining text.
func splitTags(s string) ([]time.Duration, string, error) {
	var times []time.Duration
	for strings.HasPrefix(s, "[") {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return nil, "", fmt.Errorf("unterminated timestamp in %q", s)
		}
		d, err := parseStamp(s[1:end])
		if err != nil {
			return nil, "", err
		}
		times = append(times, d)
		s = s[end+1:]
	}
	return times, s, nil
}

// parseWords splits the leading <mm:ss.xx>word sequence from s. Word times are
// absolute, exactly as they appear in the file.
func parseWords(s string) (string, []Word) {
	if !strings.HasPrefix(s, "<") {
		return strings.TrimSpace(s), nil
	}
	var (
		words []Word
		rest  = s
	)
	for strings.HasPrefix(rest, "<") {
		end := strings.IndexByte(rest, '>')
		if end < 0 {
			break
		}
		d, err := parseStamp(rest[1:end])
		if err != nil {
			break
		}
		rest = rest[end+1:]
		next := strings.IndexByte(rest, '<')
		var chunk string
		if next < 0 {
			chunk, rest = rest, ""
		} else {
			chunk, rest = rest[:next], rest[next:]
		}
		if w := strings.TrimSpace(chunk); w != "" {
			words = append(words, Word{
				Text:  w,
				Time:  d,
				Space: len(chunk) > len(strings.TrimRight(chunk, " \t")),
			})
		}
	}
	if len(words) == 0 {
		return strings.TrimSpace(s), nil
	}
	return JoinWords(words), words
}

// JoinWords renders a word list as prose, restoring the spacing of the source
// and attaching punctuation to the word it belongs to.
func JoinWords(words []Word) string {
	var b strings.Builder
	for i, w := range words {
		b.WriteString(w.Text)
		if SpaceAfter(words, i) {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// SpaceAfter reports whether a space belongs between words[i] and words[i+1].
// The parser records this from the source text, which is the only way to keep
// "world.execute(me);" tight while still spacing ordinary prose.
func SpaceAfter(words []Word, i int) bool {
	if i < 0 || i+1 >= len(words) {
		return false
	}
	return words[i].Space
}

// mergeTranslations pairs lines that share a timestamp, promoting the line
// carrying word timings to the primary rendering.
func mergeTranslations(in []Line) []Line {
	out := make([]Line, 0, len(in))
	index := make(map[time.Duration]int, len(in))
	for _, l := range in {
		i, ok := index[l.Time]
		if !ok {
			index[l.Time] = len(out)
			out = append(out, l)
			continue
		}
		cur := out[i]
		switch {
		case len(l.Words) > 0 && len(cur.Words) == 0:
			l.Translation = cur.Text
			out[i] = l
		case l.Text != cur.Text:
			cur.Translation = l.Text
			out[i] = cur
		}
	}
	return out
}

func parseStamp(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("invalid timestamp %q", s)
	}
	parts := strings.Split(s, ":")
	var b strings.Builder
	switch len(parts) {
	case 2:
		b.WriteString(parts[0])
		b.WriteByte('m')
		b.WriteString(parts[1])
		b.WriteByte('s')
	case 3:
		b.WriteString(parts[0])
		b.WriteByte('h')
		b.WriteString(parts[1])
		b.WriteByte('m')
		b.WriteString(parts[2])
		b.WriteByte('s')
	default:
		return 0, fmt.Errorf("invalid timestamp %q", s)
	}
	d, err := time.ParseDuration(b.String())
	if err != nil {
		return 0, fmt.Errorf("invalid timestamp %q", s)
	}
	if d < 0 {
		return 0, fmt.Errorf("negative timestamp %q", s)
	}
	return d, nil
}
