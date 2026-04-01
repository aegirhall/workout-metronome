package audio

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/ebitengine/oto/v3"
)

// OtoPlayer implements Player using the Oto audio library
type OtoPlayer struct {
	ctx     *oto.Context
	samples map[BeepType][]byte
	mu      sync.Mutex
}

// NewOtoPlayer creates a new Oto-based audio player
func NewOtoPlayer() (*OtoPlayer, error) {
	// Initialize Oto context
	op := &oto.NewContextOptions{
		SampleRate:   SampleRate,
		ChannelCount: ChannelCount,
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, readyChan, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio context: %w", err)
	}

	// Wait for the audio context to be ready
	<-readyChan

	// Load all beep samples from embedded MP3 files
	samples, err := LoadBeepSamples()
	if err != nil {
		return nil, fmt.Errorf("failed to load beep samples: %w", err)
	}

	return &OtoPlayer{
		ctx:     ctx,
		samples: samples,
	}, nil
}

// Play plays a beep of the specified type
func (p *OtoPlayer) Play(beepType BeepType) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	samples, ok := p.samples[beepType]
	if !ok {
		return fmt.Errorf("unknown beep type: %d", beepType)
	}

	// Create a new player for this beep
	player := p.ctx.NewPlayer(bytes.NewReader(samples))

	// Write the samples to the player
	player.Play()

	// Wait for playback to complete
	for player.IsPlaying() {
		// Small sleep to avoid busy-waiting
		// The beep durations are short (100-800ms) so this is fine
	}

	return nil
}

// Close releases audio resources
func (p *OtoPlayer) Close() error {
	if p.ctx != nil {
		return p.ctx.Suspend()
	}
	return nil
}

// MockPlayer is a mock implementation of Player for testing
type MockPlayer struct {
	BeepsPlayed []BeepType
	PlayError   error
	mu          sync.Mutex
}

// NewMockPlayer creates a new mock player
func NewMockPlayer() *MockPlayer {
	return &MockPlayer{
		BeepsPlayed: make([]BeepType, 0),
	}
}

// Play records the beep type
func (m *MockPlayer) Play(beepType BeepType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.PlayError != nil {
		return m.PlayError
	}

	m.BeepsPlayed = append(m.BeepsPlayed, beepType)
	return nil
}

// Close is a no-op for the mock
func (m *MockPlayer) Close() error {
	return nil
}

// GetBeepsPlayed returns a copy of the beeps played (thread-safe)
func (m *MockPlayer) GetBeepsPlayed() []BeepType {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]BeepType, len(m.BeepsPlayed))
	copy(result, m.BeepsPlayed)
	return result
}

// Reset clears the recorded beeps
func (m *MockPlayer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BeepsPlayed = make([]BeepType, 0)
}

// QuietPlayer is a no-op player that doesn't produce sound
type QuietPlayer struct{}

// NewQuietPlayer creates a new quiet player
func NewQuietPlayer() *QuietPlayer {
	return &QuietPlayer{}
}

// Play is a no-op
func (q *QuietPlayer) Play(beepType BeepType) error {
	return nil
}

// Close is a no-op
func (q *QuietPlayer) Close() error {
	return nil
}

var _ Player = (*OtoPlayer)(nil)
var _ Player = (*MockPlayer)(nil)
var _ Player = (*QuietPlayer)(nil)
