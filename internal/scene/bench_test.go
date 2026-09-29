package scene

import (
	"bufio"
	"io"
	"testing"
	"time"

	"world.execute/internal/audio"
	"world.execute/internal/lyric"
	"world.execute/internal/term"
)

func BenchmarkFrame(b *testing.B) {
	s := term.NewScreen(160, 45, term.ColorTrue, bufio.NewWriterSize(io.Discard, 1<<16))
	track := &lyric.Track{}
	total := 211 * time.Second
	d := NewDirector(track, total, Options{Seed: 1})
	rng := NewFrameRand()
	ctx := &Context{
		Screen: s, Lyrics: track, Total: total, FPS: 60, Seed: 1,
		Bands: make([]float32, audio.BandCount), Wave: make([]float32, audio.WavePoints),
		Header: Rect{0, 0, 160, 3}, Area: Rect{0, 3, 160, 40}, Transport: Rect{0, 43, 160, 2},
		Rand: rng.Rand,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.T = time.Duration(i%12000) * 16 * time.Millisecond
		ctx.Frame = i
		rng.Reseed(ctx.Seed, ctx.T, RandomHold)
		d.Draw(ctx)
		s.Flush()
	}
}
