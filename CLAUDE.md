# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Workout Metronome is a cross-platform CLI tool that provides audio timing cues for exercise repetitions. The application plays distinct beep sounds at configurable intervals to guide users through workout phases.

### Two Timing Modes

1. **Split Phase Mode** (`-c`/`-e`): Separate timing for concentric and eccentric phases
   - Example: `metronome -r 10 -c 2 -e 4` (2s concentric, 4s eccentric)
   - Output: `Rep 1: Concentric (2s) ✓ -> Eccentric (4s) ✓`
   - Uses `timer.PhaseConcentric` and `timer.PhaseEccentric`

2. **Whole Rep Mode** (`-t`): Single timer for entire rep
   - Example: `metronome -r 10 -t 5` (5s total per rep)
   - Output: `Rep 1 (5s) ✓`
   - Uses `timer.PhaseTotal`

**Mutual Exclusivity**: The `-t` and `-c`/`-e` flags cannot be used together. CLI validation enforces this.

## Build and Test Commands

```bash
# Build CLI for current platform
make build

# Build GUI for current platform (macOS)
make build-gui

# Run GUI application
make run-gui

# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run a specific test
go test -v -run TestName ./pkg/path

# Clean build artifacts
make clean

# Cross-compile CLI for all platforms
make cross-compile
```

Binaries are built to `build/metronome` (CLI) and `build/metronome-gui` (GUI). Audio files are embedded at compile time, so binaries are self-contained.

## Architecture

### Layered Design with Observer Pattern

The codebase uses a clean three-layer architecture designed for extensibility:

**Layer 1: Pure Business Logic (pkg/)**
- `pkg/timer` - Pure timing engine, emits events via Go channels
- `pkg/audio` - Audio playback abstraction with interface-based design
- `pkg/metronome` - Orchestration layer that coordinates timer + audio

**Layer 2: Interface Adapters**
- `internal/cli/` - CLI-specific code (Cobra framework, terminal output)
  - Implements `metronome.Observer` interface for console output
- `cmd/metronome-gui/` - GUI application (Fyne framework, macOS)
  - Also implements `metronome.Observer` for visual updates
  - Custom circular progress widget
  - Could be replaced with web/mobile UI without changing core packages

**Layer 3: Entry Points**
- `cmd/metronome/` - CLI application entry point
  - Wires everything together
  - Signal handling for graceful shutdown
- `cmd/metronome-gui/` - GUI application entry point
  - Fyne-based GUI with form inputs and visual progress
  - Circular progress ring matching iOS timer aesthetic

### Key Design Patterns

**Observer Pattern**: The core uses an Observer interface (`pkg/metronome/metronome.go`) to decouple workout events from UI presentation. The CLI implements `CLIObserver` for terminal output, and the GUI implements the interface for visual updates (circular progress, status labels). Future UIs (web, mobile) can provide their own implementations.

**Event-Driven Timer**: `pkg/timer` emits events over channels rather than using callbacks. This enables:
- Clean context-based cancellation
- Composability with Go's select statement
- Easy testing with buffered channels

**Interface-Based Audio**: The `audio.Player` interface allows:
- `OtoPlayer` for real audio playback (cross-platform via Oto library)
- `MockPlayer` for testing (records all beeps played)
- `QuietPlayer` for display-only mode

**Dependency Injection**: All components receive dependencies via constructors (no globals or singletons). Makes testing straightforward.

### Audio System

Audio files are embedded in the binary using `//go:embed` directives in `pkg/audio/beep.go`:
- `sounds/click-01.mp3` - Used for countdown and rep intervals (phase transitions)
- `sounds/beep-01.mp3` - Used for workout start/end

MP3 files are decoded to PCM at initialization using `github.com/hajimehoshi/go-mp3`, then played via Oto (cross-platform audio library). This approach:
- Eliminates external file dependencies
- Provides professional workout sounds
- Keeps binary size reasonable (~2.7 MB)

### Beep Timing Logic

**Critical Design Decision**: Beeps mark the **END** of phases (i.e., phase transitions), not the beginning.

**Timing Flow**:
1. Countdown beeps (BeepRep) play every second during countdown
2. **Start beep** (BeepStart) marks workout start AND begins Rep 1's concentric phase timer
3. When concentric phase ends, **click** (BeepRep) plays → transitions to eccentric phase
4. When eccentric phase ends, **click** (BeepRep) plays → transitions to next rep's concentric
5. This continues for all reps
6. **End beep** (BeepEnd) plays after final phase completes

**Example - 2 reps, 2s concentric, 2s eccentric**:
- Countdown: click at 2s, click at 1s
- START beep → Rep 1 concentric begins (2s timer)
- After 2s: click → Rep 1 eccentric begins
- After 2s: click → Rep 2 concentric begins
- After 2s: click → Rep 2 eccentric begins
- After 2s: END beep → Workout complete
- **Total**: 5 beeps (1 start + 3 transitions + 1 end)

**Implementation Note**: The `handleEvent` function in `pkg/metronome/metronome.go` tracks `isFirstPhase` to avoid playing a beep for Rep 1's concentric phase start (since BeepStart already marked it). All subsequent phase starts trigger a BeepRep to mark the transition.

### Context and Cancellation

All long-running operations accept `context.Context` for cancellation:
- `timer.Start(ctx)` - Can be interrupted mid-workout
- `metronome.Start(ctx)` - Propagates cancellation to child components
- CLI handles Ctrl+C by canceling the context and waiting for cleanup

### Testing Strategy

Tests use interface implementations to avoid external dependencies:
- `audio.MockPlayer` - Records beep sequences for verification
- `audio.QuietPlayer` - No-op implementation
- Timer tests use short durations (milliseconds) to run quickly
- Metronome tests verify event sequences and beep timing

When modifying core logic, ensure tests verify:
- Correct event sequence (countdown → start → phases → complete)
- Proper beep types (BeepRep vs BeepStart/BeepEnd)
- Correct beep count depends on mode:
  - **Split mode**: For N reps, expect `1 start + (2N-1) transitions + 1 end` beeps
    - Example: 2 reps = 1 + 3 + 1 = 5 beeps (start, conc→ecc, ecc→conc, conc→ecc, end)
  - **Whole rep mode**: For N reps, expect `1 start + (N-1) transitions + 1 end` beeps
    - Example: 2 reps = 1 + 1 + 1 = 3 beeps (start, rep→rep, end)
- Context cancellation behavior

## Package Responsibilities

**pkg/timer**: Generates workout events based on configuration. No knowledge of audio or UI. Emits events via channel when countdown ticks, phases start, reps complete. Detects timing mode by checking if `TotalDuration > 0` (whole rep mode) vs `ConcentricDuration/EccentricDuration > 0` (split mode).

**pkg/audio**: Handles audio playback. Loads embedded MP3s, decodes to PCM, plays via Oto. Provides interface for testing without actual audio.

**pkg/metronome**: Coordinates timer events with audio playback and observer notifications. Translates timer events into audio cues (e.g., EventWorkoutStart → play BeepStart). Uses `isFirstPhase` flag to skip the redundant beep after workout start (since BeepStart already marks the beginning of Rep 1's concentric phase).

**internal/cli**: CLI-specific code. Parses flags (Cobra), implements Observer for terminal output, handles signals. Validates mutual exclusivity of `-t` vs `-c`/`-e` flags. This is the only package that should import Cobra or write to stdout. Observer detects `PhaseTotal` to format output differently for whole rep mode.

## Extending the Application

To add a new UI (web, mobile, etc.):

1. Import `pkg/metronome`, `pkg/audio`, `pkg/timer`
2. Implement the `metronome.Observer` interface for your UI
3. Create appropriate `audio.Player` implementation if needed
4. Wire components together with dependency injection

**Example**: See `cmd/metronome-gui/` for a complete GUI implementation using Fyne. The GUI:
- Implements all Observer methods to update visual elements
- Uses a custom `CircularProgress` widget for the progress ring
- Runs the metronome in a goroutine and updates UI via callbacks
- Manages workout state (running, paused, stopped)

The core packages (`pkg/*`) should remain UI-agnostic. Never import CLI-specific code (`internal/cli`) or GUI code (`cmd/metronome-gui`) into `pkg/`.

## Module Name

The Go module is `github.com/aegirhall/workout-metronome`. When updating imports after moving files or refactoring, use this module path.
