package audio

// BeepType represents the different types of beeps
type BeepType int

const (
	// BeepStart is the beep at workout start
	BeepStart BeepType = iota
	// BeepEnd is the beep at workout end
	BeepEnd
	// BeepRep is a regular rep interval beep (also used for countdown)
	BeepRep
)

// Player is the interface for audio playback
type Player interface {
	// Play plays a beep of the specified type
	Play(beepType BeepType) error
	// Close releases audio resources
	Close() error
}
