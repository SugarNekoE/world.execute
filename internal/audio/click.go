package audio

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"time"
)

const clickRate = 22050

// WriteClickTrack writes a mono 16 bit WAV of the given length that is silent
// except for a short click at every multiple of period, starting at period. It
// is what the calibration mode plays so the picture can be lined up with it.
func WriteClickTrack(path string, length, period time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeClickTrack(f, length, period)
}

func writeClickTrack(w io.Writer, length, period time.Duration) error {
	samples := int(length.Seconds() * clickRate)
	pcm := make([]int16, samples)
	burst := clickRate * 8 / 1000
	for k := 1; ; k++ {
		start := int(math.Round(float64(k) * period.Seconds() * clickRate))
		if start+burst >= samples {
			break
		}
		for i := range burst {
			env := 1 - float64(i)/float64(burst)
			if i < clickRate/1000 {
				env = float64(i) / float64(clickRate/1000)
			}
			pcm[start+i] = int16(0.8 * env * 32767 * math.Sin(2*math.Pi*1500*float64(i)/clickRate))
		}
	}
	header := make([]byte, 44)
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(36+2*samples))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 1)
	binary.LittleEndian.PutUint32(header[24:], clickRate)
	binary.LittleEndian.PutUint32(header[28:], clickRate*2)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(2*samples))
	if _, err := w.Write(header); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, pcm)
}
