package metronome

import (
	"context"
	"fmt"
	"time"

	"github.com/aegirhall/workout-metronome/pkg/audio"
	"github.com/aegirhall/workout-metronome/pkg/timer"
)

// Observer receives notifications about workout progress
type Observer interface {
	// OnCountdown is called during countdown
	OnCountdown(remaining int)
	// OnWorkoutStart is called when the workout begins
	OnWorkoutStart()
	// OnRepPhaseStart is called when a new phase starts
	OnRepPhaseStart(rep int, phase timer.Phase, duration time.Duration)
	// OnRepComplete is called when a rep is complete
	OnRepComplete(rep int, totalReps int)
	// OnWorkoutComplete is called when the workout finishes
	OnWorkoutComplete(totalDuration time.Duration)
	// OnError is called when an error occurs
	OnError(err error)
}

// Config holds the metronome configuration
type Config struct {
	Reps               int
	ConcentricDuration time.Duration // Used for split timing mode
	EccentricDuration  time.Duration // Used for split timing mode
	TotalDuration      time.Duration // Used for whole rep timing mode
	CountdownDuration  time.Duration
	NoCountdown        bool
}

// Metronome orchestrates the workout timing and audio cues
type Metronome struct {
	config   Config
	player   audio.Player
	observer Observer
}

// New creates a new Metronome
func New(config Config, player audio.Player, observer Observer) *Metronome {
	return &Metronome{
		config:   config,
		player:   player,
		observer: observer,
	}
}

// Start begins the workout metronome
func (m *Metronome) Start(ctx context.Context) error {
	startTime := time.Now()

	// Create timer configuration
	timerConfig := timer.Config{
		Reps:               m.config.Reps,
		ConcentricDuration: m.config.ConcentricDuration,
		EccentricDuration:  m.config.EccentricDuration,
		TotalDuration:      m.config.TotalDuration,
		CountdownDuration:  m.config.CountdownDuration,
		NoCountdown:        m.config.NoCountdown,
	}

	// Create and start timer
	t := timer.NewTimer(timerConfig)

	// Run timer in goroutine
	timerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	timerDone := make(chan error, 1)
	go func() {
		timerDone <- t.Start(timerCtx)
	}()

	// Track the current rep for observer callbacks
	currentRep := 0
	var lastPhase timer.Phase
	isFirstPhase := true // Track if this is the first phase (rep 1, concentric)

	// Process events
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-t.Events():
			if !ok {
				// Channel closed, wait for timer to finish
				err := <-timerDone
				if err != nil && err != context.Canceled {
					m.observer.OnError(err)
					return err
				}
				return nil
			}

			if err := m.handleEvent(event, &currentRep, &lastPhase, &isFirstPhase, startTime); err != nil {
				m.observer.OnError(err)
				return err
			}
		}
	}
}

// handleEvent processes a timer event
func (m *Metronome) handleEvent(event timer.Event, currentRep *int, lastPhase *timer.Phase, isFirstPhase *bool, startTime time.Time) error {
	switch event.Type {
	case timer.EventCountdown:
		m.observer.OnCountdown(event.CountdownRemaining)
		// Play countdown beep
		if err := m.player.Play(audio.BeepRep); err != nil {
			return fmt.Errorf("failed to play countdown beep: %w", err)
		}

	case timer.EventWorkoutStart:
		m.observer.OnWorkoutStart()
		// Play start beep - this begins rep 1's concentric phase
		if err := m.player.Play(audio.BeepStart); err != nil {
			return fmt.Errorf("failed to play start beep: %w", err)
		}

	case timer.EventPhaseStart:
		// Update current rep if this is a new rep
		if event.RepNumber != *currentRep {
			*currentRep = event.RepNumber
		}
		*lastPhase = event.Phase

		m.observer.OnRepPhaseStart(event.RepNumber, event.Phase, event.Duration)

		// Don't play a beep for the first phase (rep 1, concentric)
		// because BeepStart already marked the beginning
		if *isFirstPhase {
			*isFirstPhase = false
		} else {
			// Play rep beep to mark the END of the previous phase
			// and the START of this phase
			if err := m.player.Play(audio.BeepRep); err != nil {
				return fmt.Errorf("failed to play phase beep: %w", err)
			}
		}

	case timer.EventRepComplete:
		m.observer.OnRepComplete(event.RepNumber, m.config.Reps)

	case timer.EventWorkoutComplete:
		totalDuration := time.Since(startTime)
		m.observer.OnWorkoutComplete(totalDuration)
		// Play end beep
		if err := m.player.Play(audio.BeepEnd); err != nil {
			return fmt.Errorf("failed to play end beep: %w", err)
		}
	}

	return nil
}

// Close releases resources
func (m *Metronome) Close() error {
	if m.player != nil {
		return m.player.Close()
	}
	return nil
}
