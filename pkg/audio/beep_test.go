package audio

import (
	"testing"
)

func TestLoadMP3Samples(t *testing.T) {
	tests := []struct {
		name    string
		mp3Data []byte
	}{
		{
			name:    "Rep beep MP3",
			mp3Data: repBeepMP3,
		},
		{
			name:    "Start/Stop beep MP3",
			mp3Data: startStopBeepMP3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			samples, err := LoadMP3Samples(tt.mp3Data)
			if err != nil {
				t.Fatalf("failed to load MP3: %v", err)
			}

			if len(samples) == 0 {
				t.Error("expected non-empty samples")
			}

			// Verify samples are not all zero
			hasNonZero := false
			for _, sample := range samples {
				if sample != 0 {
					hasNonZero = true
					break
				}
			}
			if !hasNonZero {
				t.Error("all samples are zero, expected audio data")
			}
		})
	}
}

func TestLoadBeepSamples(t *testing.T) {
	samples, err := LoadBeepSamples()
	if err != nil {
		t.Fatalf("failed to load beep samples: %v", err)
	}

	// Verify all beep types have samples
	expectedTypes := []BeepType{BeepStart, BeepEnd, BeepRep}
	for _, beepType := range expectedTypes {
		if _, ok := samples[beepType]; !ok {
			t.Errorf("missing samples for beep type %v", beepType)
		}
		if len(samples[beepType]) == 0 {
			t.Errorf("empty samples for beep type %v", beepType)
		}
	}

	// Verify start and end beeps use the same sound
	if len(samples[BeepStart]) != len(samples[BeepEnd]) {
		t.Error("start and end beeps should have the same length (same sound)")
	}

	// Verify different sounds are used for rep vs start/end
	if len(samples[BeepRep]) == len(samples[BeepStart]) {
		// This might be coincidental, so let's check content too
		identical := true
		for i := 0; i < len(samples[BeepRep]) && i < len(samples[BeepStart]); i++ {
			if samples[BeepRep][i] != samples[BeepStart][i] {
				identical = false
				break
			}
		}
		if identical {
			t.Error("rep and start beeps should use different sounds")
		}
	}
}
