package scene

import (
	"math/rand"
	"time"
)

const RandomHold = 48 * time.Millisecond

type splitMix struct{ state uint64 }

func (m *splitMix) Seed(seed int64) { m.state = uint64(seed) }

func (m *splitMix) Uint64() uint64 {
	m.state += 0x9E3779B97F4A7C15
	z := m.state
	z = (z ^ z>>30) * 0xBF58476D1CE4E5B9
	z = (z ^ z>>27) * 0x94D049BB133111EB
	return z ^ z>>31
}

func (m *splitMix) Int63() int64 { return int64(m.Uint64() >> 1) }

type FrameRand struct {
	*rand.Rand
	src *splitMix
}

func NewFrameRand() *FrameRand {
	src := &splitMix{}
	return &FrameRand{Rand: rand.New(src), src: src}
}

func (f *FrameRand) Reseed(seed int64, t, hold time.Duration) {
	f.src.Seed(hashSeed(seed, int(t/hold)))
}
