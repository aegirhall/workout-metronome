# AGENTS.md

This document provides guidelines, architectural context, and operational instructions for AI coding agents (including Claude, OpenAI/Codex, Google Gemini/Antigravity, Cursor, and GitHub Copilot) working within this repository.

---

## 1. Project Overview

**Workout Metronome** is a cross-platform Go application that provides precise audio timing cues for exercise repetitions. It ships as both a command-line interface (CLI) and a native macOS GUI application.

- **Module Name**: `github.com/aegirhall/workout-metronome`
- **Language**: Go (`go.mod` specifies Go version)
- **Primary Deliverables**:
  - CLI binary: `build/metronome` (entry point: `cmd/metronome`)
  - GUI application: `build/metronome-gui` and `Workout Metronome.app` (entry point: `cmd/metronome-gui`)
  - Audio files are embedded at compile time; binaries are fully self-contained.

### Core Workout Timing Modes

1. **Split Phase Mode (`-c` / `-e`)**:
   - Separate durations for concentric and eccentric movement phases.
   - Example: `metronome -r 10 -c 2 -e 4` (10 reps, 2s concentric, 4s eccentric).
   - Event phases: `timer.PhaseConcentric` and `timer.PhaseEccentric`.
   - Output format: `Rep 1: Concentric (2s) ✓ -> Eccentric (4s) ✓`.

2. **Whole Rep Mode (`-t`)**:
   - Single duration for the entire repetition.
   - Example: `metronome -r 10 -t 5` (10 reps, 5s per rep).
   - Event phase: `timer.PhaseTotal`.
   - Output format: `Rep 1 (5s) ✓`.

> **Constraint**: Whole Rep Mode (`-t`) and Split Phase Mode (`-c` / `-e`) are mutually exclusive. Flag validation strictly prevents combining them.

---

## 2. Architecture & Design Principles

The project follows a clean three-layer architecture utilizing the Observer and Dependency Injection patterns.

```
┌───────────────────────────────────────────────────────────┐
│ Layer 3: Entry Points                                     │
│   cmd/metronome/       cmd/metronome-gui/                 │
└─────────────────────────┬─────────────────────────────────┘
                          │
┌─────────────────────────▼─────────────────────────────────┐
│ Layer 2: Interface Adapters                               │
│   internal/cli/             cmd/metronome-gui/ (UI logic) │
│   (Cobra, stdout, signal)   (Fyne progress ring & forms)  │
└─────────────────────────┬─────────────────────────────────┘
                          │ Implements metronome.Observer
┌─────────────────────────▼─────────────────────────────────┐
│ Layer 1: Core Business Logic (UI-Agnostic)                 │
│   pkg/timer        -> Pure timing engine (channels/events)│
│   pkg/audio        -> Audio player abstraction (Oto/MP3)  │
│   pkg/metronome    -> Workout orchestration coordinator   │
└───────────────────────────────────────────────────────────┘
```

### Architectural Invariants for Agents

1. **Keep `pkg/*` UI-Agnostic**: Core domain packages (`pkg/timer`, `pkg/audio`, `pkg/metronome`) MUST NOT import `internal/cli`, Cobra, or GUI packages (Fyne).
2. **Adapter Boundaries**: `internal/cli` is the only package that may import Cobra or write directly to standard output for CLI use.
3. **Observer Pattern**: UIs implement `metronome.Observer` (`pkg/metronome/metronome.go`). Workout state is communicated exclusively via Observer callbacks, decoupling presentation from timing and audio logic.
4. **Channel-Based Timing**: `pkg/timer` emits timing events over Go channels (not callbacks). This enables `select`-based concurrency and robust context cancellation.
5. **Dependency Injection**: Pass dependencies via constructors (`New...`). Avoid global state or package-level singletons.
6. **Graceful Cancellation**: All long-running operations accept and respect `context.Context` for immediate cancellation on interrupt (Ctrl+C) or stop signals.

---

## 3. Audio System & Critical Timing Logic

Audio files are embedded directly into the binary using `//go:embed` in `pkg/audio/beep.go`:
- `sounds/click-01.mp3`: Countdown ticks and repetition phase transitions.
- `sounds/beep-01.mp3`: Workout start and workout completion.

MP3 files are decoded to PCM in-memory at startup and played through Oto (cross-platform audio library).

### Critical Beep Rules & Timing Invariants

> **Crucial Rule**: Beeps mark the **END** of phases (phase transitions), **NOT** the beginning.

1. **Countdown**: Emits `BeepRep` (click) every second of countdown.
2. **Workout Start**: Emits `BeepStart`. This marks workout start **AND** begins Rep 1's concentric/total phase.
3. **Suppressing Redundant Start Beep**: `pkg/metronome/metronome.go` uses an internal `isFirstPhase` flag to skip triggering a transition beep at the immediate start of Rep 1, since `BeepStart` already signaled it.
4. **Phase Transitions**: Each subsequent phase transition plays a `BeepRep` (click).
5. **Workout Completion**: Emits `BeepEnd` immediately after the final phase completes.

### Deterministic Beep Count Calculations

When writing or verifying tests, use these exact formulas:

- **Split Mode** ($N$ reps):
  $$\text{Total Beeps} = 1\text{ (start)} + (2N - 1)\text{ (transitions)} + 1\text{ (end)} = 2N + 1$$
  *Example (2 reps)*: $1 + 3 + 1 = 5$ beeps (`BeepStart` → conc-to-ecc → ecc-to-conc → conc-to-ecc → `BeepEnd`).

- **Whole Rep Mode** ($N$ reps):
  $$\text{Total Beeps} = 1\text{ (start)} + (N - 1)\text{ (transitions)} + 1\text{ (end)} = N + 1$$
  *Example (2 reps)*: $1 + 1 + 1 = 3$ beeps (`BeepStart` → rep1-to-rep2 → `BeepEnd`).

---

## 4. Developer & Build Commands

All standard development tasks are managed via `make`:

```bash
# Build CLI binary (outputs to build/metronome)
make build-cli

# Build GUI binary (outputs to build/metronome-gui)
make build-gui

# Build both CLI and GUI
make build

# Run CLI application
make run

# Run GUI application (macOS)
make run-gui

# Package macOS .app bundle (requires fyne CLI)
make package-mac

# Run all unit tests
make test

# Run tests with HTML coverage report
make test-coverage

# Run a single specific test
go test -v -run <TestName> ./pkg/<package>

# Tidy dependencies
make deps

# Clean build artifacts
make clean

# Cross-compile CLI binaries for Linux, macOS (amd64/arm64), and Windows
make cross-compile
```

---

## 5. Testing & Verification Guidelines for Agents

- **No Real Audio in Tests**: Never initialize real hardware audio drivers in unit tests. Use `audio.MockPlayer` (records beep sequences) or `audio.QuietPlayer` (no-op).
- **Fast Execution**: Set test durations in milliseconds (`time.Millisecond`) so the entire test suite executes in seconds without artificial lag.
- **Test Invariants**:
  - Event sequence: Countdown ticks $\rightarrow$ `EventWorkoutStart` $\rightarrow$ Rep/Phase events $\rightarrow$ `EventWorkoutComplete`.
  - Beep types and counts match the formulas in Section 3.
  - Context cancellation halts timers and coordinators promptly without leaking goroutines.
- **Verification Rule**: Always run `go test ./...` before considering any task complete.

---

## 6. General Guidelines for AI Agents

1. **Documentation Integrity**: Maintain existing comments and docstrings. Update docs whenever changing public API contracts or timing semantics.
2. **Go Idiomatic Conventions**: Follow standard Go naming, error handling (`fmt.Errorf("...: %w", err)`), and formatting (`gofmt`).
3. **Minimal Diff**: Make precise, targeted changes. Do not reformat unrelated files or rewrite working patterns without user request.
4. **Cross-Platform Compatibility**: Ensure code compiles across macOS, Linux, and Windows. Be cautious with OS-specific audio or GUI libraries.
