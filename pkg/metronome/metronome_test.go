package metronome

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aegirhall/workout-metronome/pkg/audio"
	"github.com/aegirhall/workout-metronome/pkg/timer"
)

// MockObserver implements Observer for testing
type MockObserver struct {
	CountdownCalls      int
	WorkoutStartCalls   int
	RepPhaseStartCalls  int
	RepCompleteCalls    int
	WorkoutCompleteCalls int
	ErrorCalls          int
	mu                  sync.Mutex
}

func (m *MockObserver) OnCountdown(remaining int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CountdownCalls++
}

func (m *MockObserver) OnWorkoutStart() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WorkoutStartCalls++
}

func (m *MockObserver) OnRepPhaseStart(rep int, phase timer.Phase, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RepPhaseStartCalls++
}

func (m *MockObserver) OnRepComplete(rep int, totalReps int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RepCompleteCalls++
}

func (m *MockObserver) OnWorkoutComplete(totalDuration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WorkoutCompleteCalls++
}

func (m *MockObserver) OnError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCalls++
}

func TestMetronomeBasicWorkout(t *testing.T) {
	config := Config{
		Reps:               2,
		ConcentricDuration: 100 * time.Millisecond,
		EccentricDuration:  100 * time.Millisecond,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	player := audio.NewMockPlayer()
	observer := &MockObserver{}
	m := New(config, player, observer)

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("metronome error: %v", err)
	}

	// Verify observer was called
	if observer.WorkoutStartCalls != 1 {
		t.Errorf("expected 1 workout start call, got %d", observer.WorkoutStartCalls)
	}

	if observer.RepPhaseStartCalls != 4 { // 2 reps * 2 phases
		t.Errorf("expected 4 rep phase start calls, got %d", observer.RepPhaseStartCalls)
	}

	if observer.RepCompleteCalls != 2 {
		t.Errorf("expected 2 rep complete calls, got %d", observer.RepCompleteCalls)
	}

	if observer.WorkoutCompleteCalls != 1 {
		t.Errorf("expected 1 workout complete call, got %d", observer.WorkoutCompleteCalls)
	}

	// Verify beeps were played
	beeps := player.GetBeepsPlayed()
	if len(beeps) == 0 {
		t.Fatal("no beeps were played")
	}

	// Should have: start beep + 3 transition beeps + end beep = 5 total
	// Beeps mark: START, end-of-concentric, end-of-rep1, end-of-concentric, END
	expectedBeeps := 5
	if len(beeps) != expectedBeeps {
		t.Errorf("expected %d beeps, got %d", expectedBeeps, len(beeps))
	}

	// Verify beep sequence
	if beeps[0] != audio.BeepStart {
		t.Errorf("expected first beep to be BeepStart, got %v", beeps[0])
	}

	// Middle beeps should all be BeepRep
	for i := 1; i < len(beeps)-1; i++ {
		if beeps[i] != audio.BeepRep {
			t.Errorf("expected beep %d to be BeepRep, got %v", i, beeps[i])
		}
	}

	if beeps[len(beeps)-1] != audio.BeepEnd {
		t.Errorf("expected last beep to be BeepEnd, got %v", beeps[len(beeps)-1])
	}
}

func TestMetronomeWithCountdown(t *testing.T) {
	config := Config{
		Reps:               1,
		ConcentricDuration: 100 * time.Millisecond,
		EccentricDuration:  100 * time.Millisecond,
		CountdownDuration:  2 * time.Second,
		NoCountdown:        false,
	}

	player := audio.NewMockPlayer()
	observer := &MockObserver{}
	m := New(config, player, observer)

	ctx := context.Background()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("metronome error: %v", err)
	}

	// Verify countdown was called
	if observer.CountdownCalls != 2 {
		t.Errorf("expected 2 countdown calls, got %d", observer.CountdownCalls)
	}

	// Verify countdown beeps were played
	beeps := player.GetBeepsPlayed()
	countdownBeeps := 0
	for i := 0; i < len(beeps)-1; i++ {
		if beeps[i] == audio.BeepRep && i < 2 {
			countdownBeeps++
		}
	}

	if countdownBeeps != 2 {
		t.Errorf("expected 2 countdown beeps, got %d", countdownBeeps)
	}
}

func TestMetronomeCancellation(t *testing.T) {
	config := Config{
		Reps:               10,
		ConcentricDuration: 1 * time.Second,
		EccentricDuration:  1 * time.Second,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	player := audio.NewMockPlayer()
	observer := &MockObserver{}
	m := New(config, player, observer)

	ctx, cancel := context.WithCancel(context.Background())

	// Start in goroutine
	done := make(chan error, 1)
	go func() {
		done <- m.Start(ctx)
	}()

	// Cancel after a short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for completion
	err := <-done
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}

	// Verify we didn't complete all reps
	if observer.RepCompleteCalls >= 10 {
		t.Error("expected cancellation to stop workout before completion")
	}
}

func TestMetronomeClose(t *testing.T) {
	config := Config{
		Reps:               1,
		ConcentricDuration: 100 * time.Millisecond,
		EccentricDuration:  100 * time.Millisecond,
		CountdownDuration:  0,
		NoCountdown:        true,
	}

	player := audio.NewMockPlayer()
	observer := &MockObserver{}
	m := New(config, player, observer)

	// Close should not error
	if err := m.Close(); err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}
