package timer

import (
	"context"
	"time"
)

// EventType represents the type of timing event
type EventType int

const (
	// EventCountdown is emitted during countdown
	EventCountdown EventType = iota
	// EventWorkoutStart is emitted when the workout begins
	EventWorkoutStart
	// EventPhaseStart is emitted when a new phase starts (concentric/eccentric)
	EventPhaseStart
	// EventRepComplete is emitted when a rep is complete
	EventRepComplete
	// EventWorkoutComplete is emitted when the workout finishes
	EventWorkoutComplete
)

// Phase represents the exercise phase
type Phase int

const (
	// PhaseConcentric is the concentric (lifting) phase
	PhaseConcentric Phase = iota
	// PhaseEccentric is the eccentric (lowering) phase
	PhaseEccentric
	// PhaseTotal represents a whole rep (not split into phases)
	PhaseTotal
)

func (p Phase) String() string {
	switch p {
	case PhaseConcentric:
		return "Concentric"
	case PhaseEccentric:
		return "Eccentric"
	case PhaseTotal:
		return "Rep"
	default:
		return "Unknown"
	}
}

// Event represents a timing event
type Event struct {
	Type      EventType
	Timestamp time.Time
	// Countdown-specific fields
	CountdownRemaining int
	// Workout-specific fields
	RepNumber int
	Phase     Phase
	Duration  time.Duration
}

// Config holds the timer configuration
type Config struct {
	Reps               int
	ConcentricDuration time.Duration // Used for split timing mode
	EccentricDuration  time.Duration // Used for split timing mode
	TotalDuration      time.Duration // Used for whole rep timing mode
	CountdownDuration  time.Duration
	NoCountdown        bool
}

// Timer manages workout timing
type Timer struct {
	config Config
	events chan Event
}

// NewTimer creates a new timer with the given configuration
func NewTimer(config Config) *Timer {
	return &Timer{
		config: config,
		events: make(chan Event, 10),
	}
}

// Events returns the read-only event channel
func (t *Timer) Events() <-chan Event {
	return t.events
}

// Start begins the timer and emits events
func (t *Timer) Start(ctx context.Context) error {
	defer close(t.events)

	// Countdown phase
	if !t.config.NoCountdown && t.config.CountdownDuration > 0 {
		if err := t.runCountdown(ctx); err != nil {
			return err
		}
	}

	// Emit workout start event
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.events <- Event{
		Type:      EventWorkoutStart,
		Timestamp: time.Now(),
	}:
	}

	// Workout phase
	if err := t.runWorkout(ctx); err != nil {
		return err
	}

	// Emit workout complete event
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.events <- Event{
		Type:      EventWorkoutComplete,
		Timestamp: time.Now(),
	}:
	}

	return nil
}

// runCountdown handles the countdown phase
func (t *Timer) runCountdown(ctx context.Context) error {
	seconds := int(t.config.CountdownDuration.Seconds())

	for i := seconds; i > 0; i-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t.events <- Event{
			Type:               EventCountdown,
			Timestamp:          time.Now(),
			CountdownRemaining: i,
		}:
		}

		// Wait for 1 second
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	return nil
}

// runWorkout handles the workout phase
func (t *Timer) runWorkout(ctx context.Context) error {
	// Check if we're in single-phase mode (whole rep timing)
	singlePhaseMode := t.config.TotalDuration > 0

	for rep := 1; rep <= t.config.Reps; rep++ {
		if singlePhaseMode {
			// Single phase mode - treat entire rep as one phase
			if err := t.runPhase(ctx, rep, PhaseTotal, t.config.TotalDuration); err != nil {
				return err
			}
		} else {
			// Split phase mode - separate concentric and eccentric
			// Concentric phase
			if err := t.runPhase(ctx, rep, PhaseConcentric, t.config.ConcentricDuration); err != nil {
				return err
			}

			// Eccentric phase
			if err := t.runPhase(ctx, rep, PhaseEccentric, t.config.EccentricDuration); err != nil {
				return err
			}
		}

		// Emit rep complete event
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t.events <- Event{
			Type:      EventRepComplete,
			Timestamp: time.Now(),
			RepNumber: rep,
		}:
		}
	}

	return nil
}

// runPhase handles a single phase (concentric or eccentric)
func (t *Timer) runPhase(ctx context.Context, rep int, phase Phase, duration time.Duration) error {
	// Emit phase start event
	select {
	case <-ctx.Done():
		return ctx.Err()
	case t.events <- Event{
		Type:      EventPhaseStart,
		Timestamp: time.Now(),
		RepNumber: rep,
		Phase:     phase,
		Duration:  duration,
	}:
	}

	// Wait for the phase duration
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}

	return nil
}
