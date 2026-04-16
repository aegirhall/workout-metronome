package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/aegirhall/workout-metronome/pkg/audio"
	"github.com/aegirhall/workout-metronome/pkg/metronome"
	"github.com/aegirhall/workout-metronome/pkg/timer"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Workout Metronome")
	myWindow.Resize(fyne.NewSize(500, 600))

	gui := NewGUI(myWindow)
	myWindow.SetContent(gui.Build())

	// Clean up audio players when window closes
	myWindow.SetOnClosed(func() {
		if gui.otoPlayer != nil {
			gui.otoPlayer.Close()
		}
		if gui.quietPlayer != nil {
			gui.quietPlayer.Close()
		}
	})

	myWindow.ShowAndRun()
}

// GUI manages the workout metronome GUI
type GUI struct {
	window fyne.Window

	// Input fields
	repsEntry         *widget.Entry
	totalEntry        *widget.Entry
	concentricEntry   *widget.Entry
	eccentricEntry    *widget.Entry
	countdownEntry    *widget.Entry
	splitCheck        *widget.Check
	quietCheck        *widget.Check

	// Controls
	playPauseButton *widget.Button
	stopButton      *widget.Button

	// Progress display
	progressRing *CircularProgress
	statusLabel  *widget.Label

	// Audio players (reused across workouts)
	otoPlayer   audio.Player
	quietPlayer audio.Player

	// State
	mu              sync.Mutex
	isRunning       bool
	isPaused        bool
	cancel          context.CancelFunc
	metronomeHandle *metronome.Metronome

	// Timing state for display
	currentRep    int
	totalReps     int
	currentPhase  timer.Phase
	phaseDuration time.Duration
	phaseStarted  time.Time

	// Animation control
	animationCancel context.CancelFunc
	animationGen    int     // incremented on each new phase; goroutines skip updates from old generations
	progressStart   float64 // starting progress for current phase (0.0 or 0.5 in split mode)
}

// NewGUI creates a new GUI instance
func NewGUI(window fyne.Window) *GUI {
	return &GUI{
		window: window,
	}
}

// Build constructs the GUI layout
func (g *GUI) Build() fyne.CanvasObject {
	// Input fields
	g.repsEntry = widget.NewEntry()
	g.repsEntry.SetPlaceHolder("e.g., 10")
	g.repsEntry.Validator = validation.NewRegexp(`^\d+$`, "Must be a number")

	g.totalEntry = widget.NewEntry()
	g.totalEntry.SetPlaceHolder("e.g., 5")
	g.totalEntry.Validator = validation.NewRegexp(`^\d+\.?\d*$`, "Must be a number")

	g.concentricEntry = widget.NewEntry()
	g.concentricEntry.SetPlaceHolder("e.g., 2")
	g.concentricEntry.Validator = validation.NewRegexp(`^\d+\.?\d*$`, "Must be a number")
	g.concentricEntry.Disable()

	g.eccentricEntry = widget.NewEntry()
	g.eccentricEntry.SetPlaceHolder("e.g., 4")
	g.eccentricEntry.Validator = validation.NewRegexp(`^\d+\.?\d*$`, "Must be a number")
	g.eccentricEntry.Disable()

	g.countdownEntry = widget.NewEntry()
	g.countdownEntry.SetText("5")
	g.countdownEntry.Validator = validation.NewRegexp(`^\d+\.?\d*$`, "Must be a number")

	// Checkboxes
	g.splitCheck = widget.NewCheck("Split reps (Concentric/Eccentric)", func(checked bool) {
		if checked {
			g.concentricEntry.Enable()
			g.eccentricEntry.Enable()
			g.totalEntry.Disable()
		} else {
			g.concentricEntry.Disable()
			g.eccentricEntry.Disable()
			g.totalEntry.Enable()
		}
	})

	g.quietCheck = widget.NewCheck("Quiet mode (no audio)", nil)

	// Form layout
	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("Reps:"), g.repsEntry,
		widget.NewLabel("Seconds per rep:"), g.totalEntry,
		widget.NewLabel(""), g.splitCheck,
		widget.NewLabel("Concentric (s):"), g.concentricEntry,
		widget.NewLabel("Eccentric (s):"), g.eccentricEntry,
		widget.NewLabel("Countdown (s):"), g.countdownEntry,
		widget.NewLabel(""), g.quietCheck,
	)

	// Progress display
	g.progressRing = NewCircularProgress()

	g.statusLabel = widget.NewLabel("Ready")
	g.statusLabel.Alignment = fyne.TextAlignCenter

	progressContainer := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(g.progressRing),
		container.NewCenter(g.statusLabel),
		layout.NewSpacer(),
	)

	// Control buttons
	g.playPauseButton = widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), g.onPlayPause)
	g.stopButton = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), g.onStop)
	g.stopButton.Disable()

	controls := container.NewHBox(
		layout.NewSpacer(),
		g.playPauseButton,
		g.stopButton,
		layout.NewSpacer(),
	)

	// Main layout
	content := container.NewBorder(
		container.NewVBox(form, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), controls),
		nil,
		nil,
		progressContainer,
	)

	return content
}

// onPlayPause handles the play/pause button
func (g *GUI) onPlayPause() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.isRunning {
		// Start workout
		if err := g.startWorkout(); err != nil {
			g.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
	} else {
		// Note: Pause functionality would require core metronome changes
		// For now, we don't support pause
	}
}

// onStop handles the stop button
func (g *GUI) onStop() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cancel != nil {
		g.cancel()
	}
	g.resetUI()
}

// startWorkout begins a new workout
func (g *GUI) startWorkout() error {
	// Validate inputs
	reps, err := strconv.Atoi(g.repsEntry.Text)
	if err != nil || reps <= 0 {
		return fmt.Errorf("invalid reps value")
	}

	countdown, err := strconv.ParseFloat(g.countdownEntry.Text, 64)
	if err != nil || countdown < 0 {
		return fmt.Errorf("invalid countdown value")
	}

	// Build config based on split mode
	config := metronome.Config{
		Reps:              reps,
		CountdownDuration: time.Duration(countdown * float64(time.Second)),
		NoCountdown:       countdown == 0,
	}

	if g.splitCheck.Checked {
		// Split mode
		concentric, err := strconv.ParseFloat(g.concentricEntry.Text, 64)
		if err != nil || concentric <= 0 {
			return fmt.Errorf("invalid concentric value")
		}
		eccentric, err := strconv.ParseFloat(g.eccentricEntry.Text, 64)
		if err != nil || eccentric <= 0 {
			return fmt.Errorf("invalid eccentric value")
		}
		config.ConcentricDuration = time.Duration(concentric * float64(time.Second))
		config.EccentricDuration = time.Duration(eccentric * float64(time.Second))
	} else {
		// Total mode
		total, err := strconv.ParseFloat(g.totalEntry.Text, 64)
		if err != nil || total <= 0 {
			return fmt.Errorf("invalid total duration value")
		}
		config.TotalDuration = time.Duration(total * float64(time.Second))
	}

	// Get or create audio player
	var player audio.Player
	if g.quietCheck.Checked {
		// Use quiet player
		if g.quietPlayer == nil {
			g.quietPlayer = audio.NewQuietPlayer()
		}
		player = g.quietPlayer
	} else {
		// Use Oto player (create once, reuse for all workouts)
		if g.otoPlayer == nil {
			g.otoPlayer, err = audio.NewOtoPlayer()
			if err != nil {
				return fmt.Errorf("failed to initialize audio: %w", err)
			}
		}
		player = g.otoPlayer
	}

	// Create metronome
	g.totalReps = reps
	m := metronome.New(config, player, g)
	g.metronomeHandle = m

	// Start metronome
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel

	go func() {
		err := m.Start(ctx)
		if err != nil && err != context.Canceled {
			g.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
		}
	}()

	// Update UI state
	g.isRunning = true
	g.playPauseButton.Disable() // Disable for now (no pause support)
	g.stopButton.Enable()
	g.disableInputs()

	return nil
}

// resetUI resets the UI to initial state
func (g *GUI) resetUI() {
	// Cancel any running animation
	if g.animationCancel != nil {
		g.animationCancel()
		g.animationCancel = nil
	}

	g.isRunning = false
	g.isPaused = false
	g.cancel = nil
	g.currentRep = 0
	g.totalReps = 0

	g.playPauseButton.SetText("Start")
	g.playPauseButton.SetIcon(theme.MediaPlayIcon())
	g.playPauseButton.Enable()
	g.stopButton.Disable()
	g.enableInputs()

	g.progressRing.SetProgressAndLabel(0, "")
	g.statusLabel.SetText("Ready")

	// Don't close the metronome since we're reusing the audio players
	// Just clear the reference - it will be garbage collected
	g.metronomeHandle = nil
}

func (g *GUI) disableInputs() {
	g.repsEntry.Disable()
	g.totalEntry.Disable()
	g.concentricEntry.Disable()
	g.eccentricEntry.Disable()
	g.countdownEntry.Disable()
	g.splitCheck.Disable()
	g.quietCheck.Disable()
}

func (g *GUI) enableInputs() {
	g.repsEntry.Enable()
	if !g.splitCheck.Checked {
		g.totalEntry.Enable()
	}
	if g.splitCheck.Checked {
		g.concentricEntry.Enable()
		g.eccentricEntry.Enable()
	}
	g.countdownEntry.Enable()
	g.splitCheck.Enable()
	g.quietCheck.Enable()
}

// Observer interface implementation

func (g *GUI) OnCountdown(remaining int) {
	fyne.Do(func() {
		g.statusLabel.SetText(fmt.Sprintf("Countdown: %d", remaining))
		g.progressRing.SetLabel(fmt.Sprintf("%d", remaining))
	})
}

func (g *GUI) OnWorkoutStart() {
	fyne.Do(func() {
		g.statusLabel.SetText("Workout in progress")
	})
}

func (g *GUI) OnRepPhaseStart(rep int, phase timer.Phase, duration time.Duration) {
	g.mu.Lock()
	// Cancel any existing animation
	if g.animationCancel != nil {
		g.animationCancel()
	}

	g.currentRep = rep
	g.currentPhase = phase
	g.phaseDuration = duration
	g.phaseStarted = time.Now()

	// In split mode, concentric starts at 0 and eccentric starts at 0.5
	var start float64
	if phase == timer.PhaseEccentric {
		start = 0.5
	}
	g.progressStart = start
	g.animationGen++
	gen := g.animationGen

	// Create new animation context
	ctx, cancel := context.WithCancel(context.Background())
	g.animationCancel = cancel
	g.mu.Unlock()

	fyne.Do(func() {
		g.progressRing.SetProgressAndLabel(start, fmt.Sprintf("%d", rep))
	})

	go g.animateProgress(ctx, gen, duration)
}

func (g *GUI) OnRepComplete(rep int, totalReps int) {
	// Rep complete notification
}

func (g *GUI) OnWorkoutComplete(totalDuration time.Duration) {
	fyne.Do(func() {
		g.statusLabel.SetText(fmt.Sprintf("Complete! Total: %s", formatDuration(totalDuration)))
		g.progressRing.SetProgressAndLabel(1.0, "✓")
	})

	// Reset after a delay
	time.AfterFunc(2*time.Second, func() {
		fyne.Do(func() {
			g.mu.Lock()
			defer g.mu.Unlock()
			g.resetUI()
		})
	})
}

func (g *GUI) OnError(err error) {
	fyne.Do(func() {
		g.statusLabel.SetText(fmt.Sprintf("Error: %v", err))
	})
}

// animateProgress animates the circular progress ring.
// In split mode, concentric animates 0→0.5 and eccentric animates 0.5→1.0.
// In whole-rep mode, each rep animates 0→1.0.
// gen is a generation counter; stale goroutines skip their updates.
func (g *GUI) animateProgress(ctx context.Context, gen int, duration time.Duration) {
	wallStart := time.Now()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	g.mu.Lock()
	phaseStart := g.progressStart
	g.mu.Unlock()

	// Each phase fills half the ring in split mode (0.5 range), or the full ring otherwise.
	rangeSize := 1.0 - phaseStart

	isCurrent := func() bool {
		g.mu.Lock()
		defer g.mu.Unlock()
		return g.animationGen == gen
	}

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if !isCurrent() {
				return
			}
			elapsed := time.Since(wallStart)
			if elapsed >= duration {
				fyne.Do(func() { g.progressRing.SetProgress(phaseStart + rangeSize) })
				return
			}

			fraction := float64(elapsed) / float64(duration)
			progress := phaseStart + fraction*rangeSize
			fyne.Do(func() { g.progressRing.SetProgress(progress) })
		}
	}
}

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
