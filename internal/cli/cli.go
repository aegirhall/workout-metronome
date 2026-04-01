package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/aegirhall/workout-metronome/pkg/audio"
	"github.com/aegirhall/workout-metronome/pkg/metronome"
)

var (
	reps               int
	concentricDuration float64
	eccentricDuration  float64
	totalDuration      float64
	countdownDuration  float64
	noCountdown        bool
	quiet              bool
	beepVolume         int
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metronome",
		Short: "Workout metronome for timing exercise reps",
		Long: `A CLI tool that provides audio cues to help maintain timing during workout exercises.
It beeps at specified intervals to guide you through repetitions (e.g., 2-second concentric, 4-second eccentric).`,
		RunE: runMetronome,
	}

	// Required flags
	cmd.Flags().IntVarP(&reps, "reps", "r", 0, "Number of repetitions (required)")

	// Timing mode flags (mutually exclusive)
	cmd.Flags().Float64VarP(&concentricDuration, "concentric", "c", 0, "Concentric phase duration in seconds (use with -e)")
	cmd.Flags().Float64VarP(&eccentricDuration, "eccentric", "e", 0, "Eccentric phase duration in seconds (use with -c)")
	cmd.Flags().Float64VarP(&totalDuration, "total", "t", 0, "Total rep duration in seconds (use alone, not with -c/-e)")

	// Optional flags
	cmd.Flags().Float64Var(&countdownDuration, "countdown", 5.0, "Countdown duration in seconds before starting")
	cmd.Flags().BoolVar(&noCountdown, "no-countdown", false, "Skip the countdown phase")
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Disable audio output (display only)")
	cmd.Flags().IntVar(&beepVolume, "beep-volume", 80, "Beep volume (0-100)")

	// Mark required flags
	cmd.MarkFlagRequired("reps")

	return cmd
}

// runMetronome is the main command handler
func runMetronome(cmd *cobra.Command, args []string) error {
	// Validate flags
	if err := validateFlags(); err != nil {
		return err
	}

	// Create audio player
	var player audio.Player
	var err error

	if quiet {
		player = audio.NewQuietPlayer()
	} else {
		player, err = audio.NewOtoPlayer()
		if err != nil {
			return fmt.Errorf("failed to initialize audio: %w", err)
		}
		defer player.Close()
	}

	// Create observer
	observer := NewCLIObserver(os.Stdout)

	// Create metronome configuration
	config := metronome.Config{
		Reps:              reps,
		CountdownDuration: time.Duration(countdownDuration * float64(time.Second)),
		NoCountdown:       noCountdown,
	}

	// Set timing mode based on flags
	if totalDuration > 0 {
		// Total rep timing mode (single phase)
		config.TotalDuration = time.Duration(totalDuration * float64(time.Second))
	} else {
		// Split timing mode (concentric + eccentric)
		config.ConcentricDuration = time.Duration(concentricDuration * float64(time.Second))
		config.EccentricDuration = time.Duration(eccentricDuration * float64(time.Second))
	}

	// Create metronome
	m := metronome.New(config, player, observer)

	// Set up context with cancellation for Ctrl+C
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start metronome in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.Start(ctx)
	}()

	// Wait for completion or interruption
	select {
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			return err
		}
	case <-sigChan:
		fmt.Fprintln(os.Stdout, "\nWorkout interrupted")
		cancel()
		// Wait for cleanup
		<-errChan
	}

	return nil
}

// validateFlags validates the command flags
func validateFlags() error {
	if reps <= 0 {
		return fmt.Errorf("reps must be greater than 0")
	}

	// Check for timing mode: either -t alone, or both -c and -e
	hasConcentric := concentricDuration > 0
	hasEccentric := eccentricDuration > 0
	hasTotal := totalDuration > 0

	if hasTotal && (hasConcentric || hasEccentric) {
		return fmt.Errorf("-t/--total and -c/--concentric/-e/--eccentric flags are mutually exclusive")
	}

	if hasConcentric && !hasEccentric {
		return fmt.Errorf("--concentric requires --eccentric (or use --total for whole rep timing)")
	}

	if hasEccentric && !hasConcentric {
		return fmt.Errorf("--eccentric requires --concentric (or use --total for whole rep timing)")
	}

	if !hasTotal && !hasConcentric && !hasEccentric {
		return fmt.Errorf("must specify either --total for whole rep timing, or both --concentric and --eccentric for split timing")
	}

	if !noCountdown && countdownDuration < 0 {
		return fmt.Errorf("countdown duration cannot be negative")
	}

	if beepVolume < 0 || beepVolume > 100 {
		return fmt.Errorf("beep volume must be between 0 and 100")
	}

	return nil
}

// Execute runs the root command
func Execute() error {
	return NewRootCmd().Execute()
}
