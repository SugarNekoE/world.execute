package audio

import (
	"math"
	"sync"
)

var twiddles sync.Map

func twiddleTable(n int) []complex128 {
	if t, ok := twiddles.Load(n); ok {
		return t.([]complex128)
	}
	t := make([]complex128, n/2)
	for i := range t {
		ang := 2 * math.Pi * float64(i) / float64(n)
		t[i] = complex(math.Cos(ang), math.Sin(ang))
	}
	actual, _ := twiddles.LoadOrStore(n, t)
	return actual.([]complex128)
}

// Hann returns a Hann window of length n. Windowing is applied before the FFT
// to reduce spectral leakage.
func Hann(n int) []float64 {
	w := make([]float64, n)
	if n == 1 {
		w[0] = 1
		return w
	}
	for i := range w {
		w[i] = 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(n-1)))
	}
	return w
}

// FFT performs an in place radix-2 Cooley-Tukey transform. The length of buf
// must be a power of two. When inverse is true the inverse transform is
// applied and the result is scaled by 1/n.
func FFT(buf []complex128, inverse bool) {
	n := len(buf)
	if n < 2 || n&(n-1) != 0 {
		return
	}
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			buf[i], buf[j] = buf[j], buf[i]
		}
	}
	table := twiddleTable(n)
	for length := 2; length <= n; length <<= 1 {
		half := length / 2
		step := n / length
		for i := 0; i < n; i += length {
			for j := 0; j < half; j++ {
				w := table[j*step]
				if inverse {
					w = complex(real(w), -imag(w))
				}
				u := buf[i+j]
				v := buf[i+j+half] * w
				buf[i+j] = u + v
				buf[i+j+half] = u - v
			}
		}
	}
	if inverse {
		scale := complex(1/float64(n), 0)
		for i := range buf {
			buf[i] *= scale
		}
	}
}

// Magnitudes computes the magnitude spectrum of the real signal in samples,
// reusing the scratch buffer when it is large enough.
func Magnitudes(samples []float64, window []float64, scratch []complex128) []complex128 {
	n := len(samples)
	if cap(scratch) < n {
		scratch = make([]complex128, n)
	}
	buf := scratch[:n]
	for i, s := range samples {
		if i < len(window) {
			buf[i] = complex(s*window[i], 0)
		} else {
			buf[i] = complex(s, 0)
		}
	}
	FFT(buf, false)
	for i := range buf {
		re, im := real(buf[i]), imag(buf[i])
		buf[i] = complex(math.Sqrt(re*re+im*im), 0)
	}
	return buf
}
