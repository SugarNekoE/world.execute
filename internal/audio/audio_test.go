package audio

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/cmplx"
	"os"
	"testing"
	"time"
)

func TestFFTFindsSineFrequency(t *testing.T) {
	const (
		n    = 1024
		rate = 11025
		freq = 1000
	)
	buf := make([]complex128, n)
	for i := range buf {
		buf[i] = complex(math.Sin(2*math.Pi*freq*float64(i)/rate), 0)
	}
	FFT(buf, false)

	peak, best := 0, 0.0
	for i := 1; i < n/2; i++ {
		if m := cmplx.Abs(buf[i]); m > best {
			peak, best = i, m
		}
	}
	got := float64(peak) * rate / n
	if math.Abs(got-freq) > rate/n {
		t.Errorf("peak at %.1f Hz, want %d Hz", got, freq)
	}
}

func TestFFTRoundTrip(t *testing.T) {
	src := []complex128{1, 2, 3, 4, 5, 6, 7, 8}
	buf := append([]complex128(nil), src...)
	FFT(buf, false)
	FFT(buf, true)
	for i := range src {
		if d := cmplx.Abs(buf[i] - src[i]); d > 1e-9 {
			t.Errorf("element %d off by %v", i, d)
		}
	}
}

func TestFFTIgnoresNonPowerOfTwo(t *testing.T) {
	buf := []complex128{1, 2, 3}
	FFT(buf, false)
	for i, v := range buf {
		if v != complex(float64(i+1), 0) {
			t.Fatalf("buffer modified: %v", buf)
		}
	}
}

func TestHannIsSymmetricAndZeroEnded(t *testing.T) {
	w := Hann(64)
	if w[0] != 0 || w[len(w)-1] != 0 {
		t.Errorf("ends = %v, %v", w[0], w[len(w)-1])
	}
	for i := range w {
		if math.Abs(w[i]-w[len(w)-1-i]) > 1e-12 {
			t.Fatalf("not symmetric at %d", i)
		}
	}
}

func TestNormalizeUsesPercentile(t *testing.T) {
	v := make([]float32, 200)
	for i := range v {
		v[i] = 0.1
	}
	v[199] = 100 // a single outlier must not flatten the rest
	normalize(v)
	if v[0] < 0.9 {
		t.Errorf("typical value normalised to %v, want near 1", v[0])
	}
	if v[199] > 1 {
		t.Errorf("outlier = %v, want clamped to 1", v[199])
	}
}

func TestBandEdgesAreIncreasingAndBounded(t *testing.T) {
	edges := bandEdges(BandCount, AnalysisRate, fftWindow)
	half := fftWindow / 2
	for i, e := range edges {
		if e < 1 || e > half {
			t.Errorf("edge %d = %d, want within 1..%d", i, e, half)
		}
		if i > 0 && e <= edges[i-1] {
			t.Errorf("edges not increasing at %d: %v", i, edges)
		}
	}
}

func TestAnalysisLookup(t *testing.T) {
	a := &Analysis{
		FPS:    10,
		Frames: 5,
		Energy: []float32{0.1, 0.2, 0.3, 0.4, 0.5},
		Bands:  make([]float32, 5*BandCount),
	}
	for i := range 5 {
		for b := range BandCount {
			a.Bands[i*BandCount+b] = float32(i)
		}
	}
	if got := a.EnergyAt(200 * time.Millisecond); got != 0.3 {
		t.Errorf("EnergyAt = %v, want 0.3", got)
	}
	if got := a.EnergyAt(250 * time.Millisecond); math.Abs(float64(got)-0.35) > 1e-5 {
		t.Errorf("EnergyAt between frames = %v, want 0.35", got)
	}
	if got := a.EnergyAt(-time.Second); got != 0.1 {
		t.Errorf("EnergyAt before the start = %v", got)
	}
	if got := a.EnergyAt(time.Hour); got != 0.5 {
		t.Errorf("EnergyAt past the end = %v", got)
	}
	bands := a.BandsAt(200*time.Millisecond, nil)
	if len(bands) != BandCount {
		t.Fatalf("bands = %d, want %d", len(bands), BandCount)
	}
	if bands[0] != 2 {
		t.Errorf("bands[0] = %v, want 2", bands[0])
	}
	if got := a.BandsAt(250*time.Millisecond, nil)[0]; got != 2.5 {
		t.Errorf("bands[0] between frames = %v, want 2.5", got)
	}
	var nilAnalysis *Analysis
	if got := nilAnalysis.EnergyAt(time.Second); got != 0 {
		t.Errorf("nil analysis energy = %v", got)
	}
	if got := nilAnalysis.BandsAt(time.Second, nil); len(got) != BandCount {
		t.Errorf("nil analysis bands = %d", len(got))
	}
}

func TestClockPauseAndResume(t *testing.T) {
	c := NewWallClock()
	c.Set(time.Second)
	time.Sleep(20 * time.Millisecond)
	paused := c.Now()
	if paused <= time.Second {
		t.Fatalf("clock did not advance: %v", paused)
	}
	c.Pause()
	time.Sleep(20 * time.Millisecond)
	if got := c.Now(); got-paused > time.Millisecond {
		t.Errorf("clock advanced while paused: %v then %v", paused, got)
	}
	if !c.Paused() {
		t.Error("Paused() = false")
	}
	c.Resume()
	time.Sleep(20 * time.Millisecond)
	if got := c.Now(); got <= paused {
		t.Errorf("clock did not resume: %v then %v", paused, got)
	}
	c.Offset(500 * time.Millisecond)
	if got := c.Now(); got < paused+500*time.Millisecond {
		t.Errorf("offset not applied: %v", got)
	}
}

// TestAnalyzeRealFile checks the envelope against the track that ships with the
// program. It is skipped when ffmpeg or the asset is missing.
func TestAnalyzeRealFile(t *testing.T) {
	const path = "../../assets/world-execute-me.mp3"
	if !Available() {
		t.Skip("ffmpeg and ffprobe are not installed")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not present", path)
	}
	dur, err := Duration(path)
	if err != nil {
		t.Fatalf("Duration: %v", err)
	}
	if d := dur - (211*time.Second + 906*time.Millisecond); d > 50*time.Millisecond || d < -50*time.Millisecond {
		t.Errorf("duration = %v, want about 3m31.9s", dur)
	}

	a, err := Analyze(path, 60)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if a.Frames < 10000 {
		t.Fatalf("frames = %d, want about one per 1/60s", a.Frames)
	}
	for i, e := range a.Energy {
		if e < 0 || e > 1 {
			t.Fatalf("energy[%d] = %v, out of range", i, e)
		}
	}
	for i, b := range a.Bands {
		if b < 0 || b > 1 {
			t.Fatalf("band[%d] = %v, out of range", i, b)
		}
	}

	mean := func(from, to time.Duration) float32 {
		start, end := a.FrameAt(from), a.FrameAt(to)
		if end <= start {
			return 0
		}
		var sum float32
		for _, e := range a.Energy[start:end] {
			sum += e
		}
		return sum / float32(end-start)
	}
	intro := mean(0, 3*time.Second)
	chorus := mean(148*time.Second, 155*time.Second)
	if chorus <= intro {
		t.Errorf("chorus energy %v should exceed the intro %v", chorus, intro)
	}
	if hit := a.EnergyAt(148 * time.Second); hit < 0.3 {
		t.Errorf("energy at the first EXECUTION = %v, want a loud hit", hit)
	}
}

func TestAnalysisFramesAreCentredOnTheirTimestamp(t *testing.T) {
	const fps = 60
	hop := AnalysisRate / fps
	samples := make([]float32, AnalysisRate*2)
	for i := AnalysisRate; i < AnalysisRate+AnalysisRate/5; i++ {
		samples[i] = float32(math.Sin(float64(i) * 0.9))
	}
	a := &Analysis{FPS: fps, Hop: hop, Frames: len(samples) / hop}
	a.Energy = make([]float32, a.Frames)
	a.Bands = make([]float32, a.Frames*BandCount)
	a.Wave = make([]float32, a.Frames*WavePoints)
	a.analyseFrames(samples, 0, a.Frames, Hann(fftWindow), bandEdges(BandCount, AnalysisRate, fftWindow))

	peak := 0
	for f, e := range a.Energy {
		if e > a.Energy[peak] {
			peak = f
		}
	}
	centre := int(1.1 * fps)
	if peak < centre-1 || peak > centre+1 {
		t.Errorf("energy peaks at frame %d, want %d±1 for a burst centred on 1.1 s", peak, centre)
	}
	if a.Energy[0] != 0 || a.Energy[a.Frames-1] != 0 {
		t.Error("silence at the edges should have no energy")
	}
}

func TestClickTrackClicksOnEveryPeriod(t *testing.T) {
	var buf bytes.Buffer
	if err := writeClickTrack(&buf, 3*time.Second, 500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if string(b[0:4]) != "RIFF" || string(b[8:16]) != "WAVEfmt " || string(b[36:40]) != "data" {
		t.Fatalf("not a WAV header: %q", b[:44])
	}
	if got := binary.LittleEndian.Uint32(b[24:]); got != clickRate {
		t.Errorf("sample rate = %d", got)
	}
	samples := (len(b) - 44) / 2
	if want := 3 * clickRate; samples != want {
		t.Errorf("%d samples, want %d", samples, want)
	}
	pcm := func(at time.Duration, span time.Duration) (peak int) {
		start := int(at.Seconds() * clickRate)
		for i := range int(span.Seconds() * clickRate) {
			v := int(int16(binary.LittleEndian.Uint16(b[44+2*(start+i):])))
			if v < 0 {
				v = -v
			}
			peak = max(peak, v)
		}
		return
	}
	if p := pcm(0, 400*time.Millisecond); p != 0 {
		t.Errorf("the lead in should be silent, peak %d", p)
	}
	for _, k := range []int{1, 2, 3, 4, 5} {
		at := time.Duration(k) * 500 * time.Millisecond
		if p := pcm(at, 8*time.Millisecond); p < 10000 {
			t.Errorf("click %d is too quiet or missing: peak %d", k, p)
		}
		if p := pcm(at+20*time.Millisecond, 400*time.Millisecond); p != 0 {
			t.Errorf("silence after click %d has peak %d", k, p)
		}
	}
}
