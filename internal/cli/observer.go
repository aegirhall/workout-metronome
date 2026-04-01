package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/aegirhall/workout-metronome/pkg/timer"
)

// CLIObserver implements the metronome.Observer interface for console output
type CLIObserver struct {
	writer io.Writer
}

// NewCLIObserver creates a new CLI observer
func NewCLIObserver(writer io.Writer) *CLIObserver {
	return &CLIObserver{
		writer: writer,
	}
}

// OnCountdown is called during countdown
func (o *CLIObserver) OnCountdown(remaining int) {
	fmt.Fprintf(o.writer, "%d...", remaining)
}

// OnWorkoutStart is called when the workout begins
func (o *CLIObserver) OnWorkoutStart() {
	// Print START! synchronized with the start beep
	fmt.Fprintf(o.writer, "START!\n")
}

// OnRepPhaseStart is called when a new phase starts
func (o *CLIObserver) OnRepPhaseStart(rep int, phase timer.Phase, duration time.Duration) {
	if phase == timer.PhaseTotal {
		// Single-phase mode - just show rep number and duration
		fmt.Fprintf(o.writer, "Rep %d (%s) ", rep, duration)
	} else if phase == timer.PhaseConcentric {
		// Start of concentric phase - print rep number and phase
		fmt.Fprintf(o.writer, "Rep %d: %s (%s) ", rep, phase, duration)
	} else {
		// Start of eccentric phase - print checkmark for completed concentric, then eccentric info
		fmt.Fprintf(o.writer, "✓ -> %s (%s) ", phase, duration)
	}
}

// OnRepComplete is called when a rep is complete
func (o *CLIObserver) OnRepComplete(rep int, totalReps int) {
	// Print checkmark for completed eccentric phase and newline
	fmt.Fprintf(o.writer, "✓\n")
}

// OnWorkoutComplete is called when the workout finishes
func (o *CLIObserver) OnWorkoutComplete(totalDuration time.Duration) {
	fmt.Fprintf(o.writer, "\nWORKOUT COMPLETE! Total time: %s\n", formatDuration(totalDuration))
}

// OnError is called when an error occurs
func (o *CLIObserver) OnError(err error) {
	fmt.Fprintf(o.writer, "\nError: %v\n", err)
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
