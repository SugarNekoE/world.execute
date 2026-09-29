package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// AnalysisRate is the sample rate used for the amplitude envelope. It is
	// low enough to keep the decode quick and high enough to resolve the
	// frequency bands we draw.
	AnalysisRate = 11025
	// BandCount is the number of frequency bands exposed by Analysis.
	BandCount = 12
	// WavePoints is how many samples of the waveform each frame keeps. The
	// oscilloscope and the shape scenes draw it directly.
	WavePoints = 96
	fftWindow  = 1024
)

// Analysis is a precomputed amplitude, frequency and waveform envelope of a
// track. Amplitude and bands are normalised to 0..1; the waveform is scaled so
// its loudest peak is 1 and keeps its sign.
type Analysis struct {
	FPS      int
	Hop      int
	Frames   int
	Duration time.Duration
	Energy   []float32
	Bands    []float32
	Wave     []float32
}

// FrameAt returns the frame index covering playback position t.
func (a *Analysis) FrameAt(t time.Duration) int {
	if a == nil || a.Frames == 0 {
		return 0
	}
	f := int(float64(t) / float64(time.Second) * float64(a.FPS))
	if f < 0 {
		return 0
	}
	if f >= a.Frames {
		return a.Frames - 1
	}
	return f
}

// framePos returns the two frames surrounding t and the blend between them.
func (a *Analysis) framePos(t time.Duration) (int, int, float32) {
	pos := float64(t) / float64(time.Second) * float64(a.FPS)
	if pos <= 0 {
		return 0, 0, 0
	}
	last := a.Frames - 1
	f0 := int(pos)
	if f0 >= last {
		return last, last, 0
	}
	return f0, f0 + 1, float32(pos - float64(f0))
}

// EnergyAt returns the loudness of the track at t, interpolated between the
// two neighbouring frames.
func (a *Analysis) EnergyAt(t time.Duration) float32 {
	if a == nil || len(a.Energy) == 0 {
		return 0
	}
	f0, f1, k := a.framePos(t)
	return a.Energy[f0] + (a.Energy[f1]-a.Energy[f0])*k
}

// BandsAt fills dst with the normalised band magnitudes at t and returns it.
// dst is resliced to BandCount when it is too small.
func (a *Analysis) BandsAt(t time.Duration, dst []float32) []float32 {
	if cap(dst) < BandCount {
		dst = make([]float32, BandCount)
	}
	dst = dst[:BandCount]
	for i := range dst {
		dst[i] = 0
	}
	if a == nil || len(a.Bands) == 0 {
		return dst
	}
	f0, f1, k := a.framePos(t)
	lo, hi := f0*BandCount, f1*BandCount
	for i := range dst {
		dst[i] = a.Bands[lo+i] + (a.Bands[hi+i]-a.Bands[lo+i])*k
	}
	return dst
}

// Available reports whether the external tools needed for analysis are
// installed.
func Available() bool {
	_, errF := exec.LookPath("ffmpeg")
	_, errP := exec.LookPath("ffprobe")
	return errF == nil && errP == nil
}

// WaveAt fills dst with the waveform at t, interpolating between the two
// neighbouring frames so a moving scope stays smooth. Values are in -1..1.
func (a *Analysis) WaveAt(t time.Duration, dst []float32) []float32 {
	if cap(dst) < WavePoints {
		dst = make([]float32, WavePoints)
	}
	dst = dst[:WavePoints]
	if a == nil || len(a.Wave) == 0 {
		for i := range dst {
			dst[i] = 0
		}
		return dst
	}
	pos := float64(t) / float64(time.Second) * float64(a.FPS)
	f0 := int(pos)
	if f0 < 0 {
		f0 = 0
	}
	if f0 >= a.Frames-1 {
		f0 = a.Frames - 2
	}
	if f0 < 0 {
		f0 = 0
	}
	frac := float32(pos - float64(f0))
	frac = min(max(frac, 0), 1)
	lo := f0 * WavePoints
	hi := lo + WavePoints
	next := min(hi+WavePoints, len(a.Wave))
	for i := range dst {
		x := a.Wave[lo+i]
		if next > hi+i {
			x += (a.Wave[hi+i] - x) * frac
		}
		dst[i] = x
	}
	return dst
}

// Duration reads the playing time of an audio file with ffprobe.
func Duration(path string) (time.Duration, error) {
	bin, err := exec.LookPath("ffprobe")
	if err != nil {
		return 0, err
	}
	out, err := exec.Command(bin,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=nw=1:nk=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe %s: %w", path, err)
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe duration %q: %w", out, err)
	}
	return time.Duration(secs * float64(time.Second)), nil
}

// Analyze decodes path with ffmpeg and computes the envelope at fps frames per
// second.
func Analyze(path string, fps int) (*Analysis, error) {
	if fps <= 0 {
		fps = 60
	}
	samples, err := decode(path, AnalysisRate)
	if err != nil {
		return nil, err
	}
	hop := max(AnalysisRate/fps, 1)

	a := &Analysis{
		FPS:      fps,
		Hop:      hop,
		Frames:   len(samples) / hop,
		Duration: time.Duration(float64(len(samples)) / float64(AnalysisRate) * float64(time.Second)),
	}
	if a.Frames == 0 {
		return a, nil
	}
	a.Energy = make([]float32, a.Frames)
	a.Bands = make([]float32, a.Frames*BandCount)
	a.Wave = make([]float32, a.Frames*WavePoints)

	window := Hann(fftWindow)
	edges := bandEdges(BandCount, AnalysisRate, fftWindow)

	workers := min(runtime.GOMAXPROCS(0), max(a.Frames/256, 1))
	var wg sync.WaitGroup
	for w := range workers {
		lo := a.Frames * w / workers
		hi := a.Frames * (w + 1) / workers
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.analyseFrames(samples, lo, hi, window, edges)
		}()
	}
	wg.Wait()

	normalize(a.Energy)
	if peak := percentile(a.Wave); peak > 0 {
		for i, w := range a.Wave {
			a.Wave[i] = w / peak
			if a.Wave[i] > 1 {
				a.Wave[i] = 1
			}
			if a.Wave[i] < -1 {
				a.Wave[i] = -1
			}
		}
	}
	col := make([]float32, a.Frames)
	for b := range BandCount {
		for f := range a.Frames {
			col[f] = a.Bands[f*BandCount+b]
		}
		normalize(col)
		for f := range a.Frames {
			a.Bands[f*BandCount+b] = col[f]
		}
	}
	return a, nil
}

func (a *Analysis) analyseFrames(samples []float32, from, to int, window []float64, edges []int) {
	var scratch []complex128
	frame := make([]float64, fftWindow)
	bands := make([]float32, BandCount)
	for f := from; f < to; f++ {
		centre := f*a.Hop + a.Hop/2
		start := centre - fftWindow/2
		lo, hi := max(start, 0), min(start+fftWindow, len(samples))
		chunk := samples[lo:hi]

		var sum float64
		for _, s := range chunk {
			sum += float64(s) * float64(s)
		}
		a.Energy[f] = float32(math.Sqrt(sum / float64(len(chunk))))

		clear(frame)
		chunk32to64(chunk, frame[lo-start:])
		downsample(chunk, a.Wave[f*WavePoints:(f+1)*WavePoints])
		scratch = Magnitudes(frame, window, scratch)
		spectrumBands(scratch, edges, bands)
		copy(a.Bands[f*BandCount:], bands)
	}
}

func chunk32to64(chunk []float32, dst []float64) []float64 {
	n := min(len(chunk), len(dst))
	for i := range n {
		dst[i] = float64(chunk[i])
	}
	return dst[:n]
}

func decode(path string, rate int) ([]float32, error) {
	bin, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(bin,
		"-hide_banner", "-loglevel", "error",
		"-i", path,
		"-vn", "-ac", "1", "-ar", strconv.Itoa(rate),
		"-f", "f32le", "-",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var samples []float32
	buf := make([]byte, 1<<18)
	carry := 0
	for {
		n, err := stdout.Read(buf[carry:])
		total := carry + n
		whole := total - total%4
		if whole > 0 {
			samples = slices.Grow(samples, whole/4)
			for i := 0; i < whole; i += 4 {
				samples = append(samples, math.Float32frombits(binary.LittleEndian.Uint32(buf[i:])))
			}
			carry = copy(buf, buf[whole:total])
		} else {
			carry = total
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cmd.Wait()
			return nil, fmt.Errorf("ffmpeg decode: %w", err)
		}
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ffmpeg decode %s: %w: %s", path, err, stderr.String())
	}
	return samples, nil
}

// downsample reduces a window of samples to WavePoints averages, preserving the
// sign so it can be drawn as a waveform.
func downsample(samples []float32, dst []float32) {
	n := len(dst)
	if n == 0 {
		return
	}
	if n > len(samples) {
		n = len(samples)
	}
	for k := range dst {
		dst[k] = 0
	}
	if n == 0 {
		return
	}
	for k := range n {
		lo := k * len(samples) / n
		hi := (k + 1) * len(samples) / n
		if hi <= lo {
			hi = lo + 1
		}
		var sum float32
		for _, s := range samples[lo:hi] {
			sum += s
		}
		dst[k] = sum / float32(hi-lo)
	}
}

// bandEdges returns the band boundaries in FFT bins, log spaced between 40 Hz
// and 5 kHz.
func bandEdges(count, rate, size int) []int {
	const lo, hi = 40.0, 5000.0
	edges := make([]int, count+1)
	for i := range edges {
		freq := lo * math.Pow(hi/lo, float64(i)/float64(count))
		bin := int(freq / (float64(rate) / float64(size)))
		edges[i] = min(max(bin, 1), size/2)
	}
	for i := 1; i < len(edges); i++ {
		if edges[i] <= edges[i-1] {
			edges[i] = edges[i-1] + 1
		}
	}
	return edges
}

func spectrumBands(mag []complex128, edges []int, dst []float32) {
	half := len(mag)
	for b := range dst {
		lo, hi := edges[b], edges[b+1]
		if hi > half {
			hi = half
		}
		if lo >= hi {
			dst[b] = 0
			continue
		}
		var sum float64
		for i := lo; i < hi; i++ {
			sum += real(mag[i])
		}
		dst[b] = float32(sum / float64(hi-lo))
	}
}

// percentile returns the 99th percentile of the magnitude of v, which is a
// steadier reference than the maximum.
func percentile(v []float32) float32 {
	if len(v) == 0 {
		return 0
	}
	sorted := make([]float32, len(v))
	for i, x := range v {
		sorted[i] = float32(math.Abs(float64(x)))
	}
	return reference(sorted)
}

func reference(sorted []float32) float32 {
	slices.Sort(sorted)
	ref := sorted[int(float64(len(sorted)-1)*0.99)]
	if ref <= 0 {
		ref = sorted[len(sorted)-1]
	}
	return ref
}

// normalize scales values so the 99th percentile maps to one, then applies a
// square root curve to make quiet passages visible.
func normalize(v []float32) {
	if len(v) == 0 {
		return
	}
	ref := reference(slices.Clone(v))
	for i, x := range v {
		if ref <= 0 {
			v[i] = 0
			continue
		}
		y := math.Sqrt(float64(x) / float64(ref))
		v[i] = float32(min(y, 1))
	}
}
