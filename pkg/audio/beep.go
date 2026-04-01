package audio

import (
	_ "embed"
	"fmt"
	"io"

	"github.com/hajimehoshi/go-mp3"
)

const (
	// SampleRate is the audio sample rate (44.1 kHz standard)
	SampleRate = 44100
	// ChannelCount is the number of audio channels (stereo for MP3)
	ChannelCount = 2
	// BitDepth is 16-bit audio
	BitDepth = 2 // 2 bytes per sample
)

//go:embed sounds/single_block.mp3
var repBeepMP3 []byte

//go:embed sounds/beep-07.mp3
var startStopBeepMP3 []byte

// LoadMP3Samples decodes an MP3 file and returns PCM samples
func LoadMP3Samples(mp3Data []byte) ([]byte, error) {
	decoder, err := mp3.NewDecoder(io.NopCloser(io.Reader(newBytesReader(mp3Data))))
	if err != nil {
		return nil, fmt.Errorf("failed to create MP3 decoder: %w", err)
	}

	// Read all PCM data
	pcmData, err := io.ReadAll(decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MP3: %w", err)
	}

	return pcmData, nil
}

// bytesReader implements io.Reader for byte slices
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// LoadBeepSamples loads all beep samples from embedded MP3 files
func LoadBeepSamples() (map[BeepType][]byte, error) {
	samples := make(map[BeepType][]byte)

	// Load rep beep (for countdown and rep intervals)
	repSamples, err := LoadMP3Samples(repBeepMP3)
	if err != nil {
		return nil, fmt.Errorf("failed to load rep beep: %w", err)
	}
	samples[BeepRep] = repSamples

	// Load start/stop beep (for workout start and end)
	startStopSamples, err := LoadMP3Samples(startStopBeepMP3)
	if err != nil {
		return nil, fmt.Errorf("failed to load start/stop beep: %w", err)
	}
	samples[BeepStart] = startStopSamples
	samples[BeepEnd] = startStopSamples // Use same sound for start and end

	return samples, nil
}
