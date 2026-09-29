package lyric

import (
	"os"
	"strings"
	"testing"
	"time"
)

const sample = `[00:00.00]<00:00.00>Switch <00:00.63>on <00:00.96>the <00:01.08>power <00:01.44>line
[00:00.00]接上电源
[00:19.11]<00:19.11>world<00:19.80>.<00:29.25>execute<00:29.76>（<00:29.76>me<00:29.88>）<00:29.88>;
[00:19.11]间奏
[00:53.37]<00:53.37>To <00:53.76>A<00:53.91>.<00:53.91>D <00:54.30>to <00:54.60>B<00:54.78>.<00:54.78>C
[00:53.37]从未来到过去
`

func TestParseWordTimingsAndTranslations(t *testing.T) {
	track, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got, want := len(track.Lines), 3; got != want {
		t.Fatalf("lines = %d, want %d", got, want)
	}

	first := track.Lines[0]
	if first.Text != "Switch on the power line" {
		t.Errorf("Text = %q", first.Text)
	}
	if first.Translation != "接上电源" {
		t.Errorf("Translation = %q", first.Translation)
	}
	if len(first.Words) != 5 {
		t.Fatalf("words = %d, want 5", len(first.Words))
	}
	if first.Words[1].Text != "on" || first.Words[1].Time != 630*time.Millisecond {
		t.Errorf("word 1 = %+v", first.Words[1])
	}
	if first.Time != 0 {
		t.Errorf("line time = %v", first.Time)
	}
}

func TestParseKeepsPunctuationTight(t *testing.T) {
	track, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	cases := []struct {
		index int
		want  string
	}{
		{1, "world.execute（me）;"},
		{2, "To A.D to B.C"},
	}
	for _, tc := range cases {
		if got := track.Lines[tc.index].Text; got != tc.want {
			t.Errorf("line %d = %q, want %q", tc.index, got, tc.want)
		}
	}
}

func TestAtFindsLineAndWord(t *testing.T) {
	track, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	tests := []struct {
		at       time.Duration
		wantLine string
		wantWord int
	}{
		{0, "Switch on the power line", 0},
		{700 * time.Millisecond, "Switch on the power line", 1},
		{1500 * time.Millisecond, "Switch on the power line", 4},
		{10 * time.Second, "Switch on the power line", 4},
		{20 * time.Second, "world.execute（me）;", 1},
		{30 * time.Second, "world.execute（me）;", 6},
		{55100 * time.Millisecond, "To A.D to B.C", 7},
	}
	for _, tc := range tests {
		line, word, ok := track.At(tc.at)
		if !ok {
			t.Fatalf("At(%v) reported no line", tc.at)
		}
		if line.Text != tc.wantLine || word != tc.wantWord {
			t.Errorf("At(%v) = %q word %d, want %q word %d", tc.at, line.Text, word, tc.wantLine, tc.wantWord)
		}
	}
}

func TestAtBeforeFirstLine(t *testing.T) {
	track, err := Parse(strings.NewReader("[00:05.00]late\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, _, ok := track.At(time.Second); ok {
		t.Error("At before the first line should report nothing")
	}
}

func TestParseRejectsBadTimestamps(t *testing.T) {
	_, err := Parse(strings.NewReader("[aa:bb]no\n"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Errorf("error should name the line: %v", err)
	}
}

func TestMultiTimestampLine(t *testing.T) {
	track, err := Parse(strings.NewReader("[00:01.00][00:02.00]again\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(track.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(track.Lines))
	}
	if track.Lines[0].Time != time.Second || track.Lines[1].Time != 2*time.Second {
		t.Errorf("times = %v, %v", track.Lines[0].Time, track.Lines[1].Time)
	}
}

// TestRealLyricsParse checks the asset that ships with the program.
func TestRealLyricsParse(t *testing.T) {
	f, err := os.Open("../../assets/lyrics.lrc")
	if os.IsNotExist(err) {
		t.Skip("assets/lyrics.lrc not present")
	}
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	track, err := Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(track.Lines) != 129 {
		t.Errorf("lines = %d, want 129", len(track.Lines))
	}
	first := track.Lines[0]
	if first.Text != "Switch on the power line" || first.Translation != "接上电源" {
		t.Errorf("first line = %q / %q", first.Text, first.Translation)
	}
	last := track.Lines[len(track.Lines)-1]
	if last.Text != "EXECUTION" {
		t.Errorf("last line = %q", last.Text)
	}
	if got := last.Time; got != 205*time.Second+860*time.Millisecond {
		t.Errorf("last line time = %v", got)
	}

	// Every line must carry either word timings or a translation, and the
	// words must reconstruct the line text.
	for i, l := range track.Lines {
		if len(l.Words) > 0 && JoinWords(l.Words) != l.Text {
			t.Errorf("line %d: words join to %q, text is %q", i, JoinWords(l.Words), l.Text)
		}
		for j, w := range l.Words {
			if w.Time < l.Time {
				t.Errorf("line %d word %d starts before the line", i, j)
			}
			if j > 0 && w.Time < l.Words[j-1].Time {
				t.Errorf("line %d word %d goes backwards", i, j)
			}
		}
	}
}
