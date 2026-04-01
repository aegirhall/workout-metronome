package timer

import (
	"context"
	"testing"
	"time"
)

func TestTimerCountdown(t *testing.T) {
	config := Config{
		Reps:               1,
		ConcentricDuration: 100 * time.Millisecond,
		EccentricDuration:  100 * time.Millisecond,
		CountdownDuration:  2 * time.Second,
		NoCountdown:        false,
	}

	timer := NewTimer(config)
	ctx := context.Background()

	// Start timer in goroutine
	done := make(chan error, 1)
	go func() {
		done <- timer.Start(ctx)
	}()

	// Collect events
	events := []Event{}
	for event := range timer.Events() {
		events = append(events, event)
	}

	// Wait for completion
	if err := <-done; err != nil {
		t.Fatalf("timer error: %v", err)
	}

	// Verify countdown events
	countdownCount := 0
	for _, event := range events {
		if event.Type == EventCountdown {
			countdownCount++
		}
	}
	if countdownCount != 2 {
		t.Errorf("expected 2 countdown events, got %d", countdownCount)
	}

	// Verify event sequence
	expectedTypes := []EventType{
		EventCountdown,
		EventCountdown,
		EventWorkoutStart,
		EventPhaseStart, // Concentric
		EventPhaseStart, // Eccentric
		EventRepComplete,
		EventWorkoutComplete,
	}

	if len(events) != len(expectedTypes) {
		t.Errorf("expected %d events, got %d", len(expectedTypes), len(events))
	}

	for i, expected := range expectedTypes {
		if i >= len(events) {
			break
		}
		if events[i].Type != expected {
			t.Errorf("event %d: expected type %v, got %v", i, expected, events[i].Type)
		}
	}
}

func TestTimerNoCountdown(t *testing.T) {
	config := Config{
		Reps:               1,
		ConcentricDuration: 100 * time.Millisecond,
		EccentricDuration:  100 * time.Millisecond,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	timer := NewTimer(config)
	ctx := context.Background()

	// Start timer in goroutine
	done := make(chan error, 1)
	go func() {
		done <- timer.Start(ctx)
	}()

	// Collect events
	events := []Event{}
	for event := range timer.Events() {
		events = append(events, event)
	}

	// Wait for completion
	if err := <-done; err != nil {
		t.Fatalf("timer error: %v", err)
	}

	// Verify no countdown events
	for _, event := range events {
		if event.Type == EventCountdown {
			t.Error("unexpected countdown event when NoCountdown is true")
		}
	}

	// Should start with WorkoutStart
	if len(events) > 0 && events[0].Type != EventWorkoutStart {
		t.Errorf("expected first event to be WorkoutStart, got %v", events[0].Type)
	}
}

func TestTimerMultipleReps(t *testing.T) {
	config := Config{
		Reps:               3,
		ConcentricDuration: 50 * time.Millisecond,
		EccentricDuration:  50 * time.Millisecond,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	timer := NewTimer(config)
	ctx := context.Background()

	// Start timer in goroutine
	done := make(chan error, 1)
	go func() {
		done <- timer.Start(ctx)
	}()

	// Collect events
	events := []Event{}
	for event := range timer.Events() {
		events = append(events, event)
	}

	// Wait for completion
	if err := <-done; err != nil {
		t.Fatalf("timer error: %v", err)
	}

	// Count phase start events (should be 2 per rep: concentric + eccentric)
	phaseStartCount := 0
	repCompleteCount := 0
	for _, event := range events {
		if event.Type == EventPhaseStart {
			phaseStartCount++
		}
		if event.Type == EventRepComplete {
			repCompleteCount++
		}
	}

	if phaseStartCount != 6 { // 3 reps * 2 phases
		t.Errorf("expected 6 phase starts, got %d", phaseStartCount)
	}

	if repCompleteCount != 3 {
		t.Errorf("expected 3 rep completes, got %d", repCompleteCount)
	}
}

func TestTimerCancellation(t *testing.T) {
	config := Config{
		Reps:               10,
		ConcentricDuration: 1 * time.Second,
		EccentricDuration:  1 * time.Second,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	timer := NewTimer(config)
	ctx, cancel := context.WithCancel(context.Background())

	// Start timer in goroutine
	done := make(chan error, 1)
	go func() {
		done <- timer.Start(ctx)
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Drain events
	for range timer.Events() {
	}

	// Wait for completion
	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseConcentric, "Concentric"},
		{PhaseEccentric, "Eccentric"},
		{Phase(999), "Unknown"},
	}

	for _, tt := range tests {
		result := tt.phase.String()
		if result != tt.expected {
			t.Errorf("Phase(%d).String() = %s, expected %s", tt.phase, result, tt.expected)
		}
	}
}
