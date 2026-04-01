package audio

import (
	"errors"
	"testing"
)

func TestMockPlayer(t *testing.T) {
	player := NewMockPlayer()

	// Test playing different beep types
	beeps := []BeepType{BeepStart, BeepRep, BeepRep, BeepEnd}
	for _, beep := range beeps {
		if err := player.Play(beep); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// Verify recorded beeps
	recorded := player.GetBeepsPlayed()
	if len(recorded) != len(beeps) {
		t.Errorf("expected %d beeps, got %d", len(beeps), len(recorded))
	}

	for i, expected := range beeps {
		if recorded[i] != expected {
			t.Errorf("beep %d: expected %v, got %v", i, expected, recorded[i])
		}
	}

	// Test reset
	player.Reset()
	recorded = player.GetBeepsPlayed()
	if len(recorded) != 0 {
		t.Errorf("expected 0 beeps after reset, got %d", len(recorded))
	}

	// Test error handling
	expectedErr := errors.New("test error")
	player.PlayError = expectedErr
	if err := player.Play(BeepStart); err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestQuietPlayer(t *testing.T) {
	player := NewQuietPlayer()

	// Test that playing doesn't error
	beeps := []BeepType{BeepStart, BeepRep, BeepEnd}
	for _, beep := range beeps {
		if err := player.Play(beep); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	// Test close
	if err := player.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestPlayerInterface(t *testing.T) {
	// Verify all player types implement the Player interface
	var _ Player = (*MockPlayer)(nil)
	var _ Player = (*QuietPlayer)(nil)
	var _ Player = (*OtoPlayer)(nil)
}
