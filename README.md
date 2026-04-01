# Workout Metronome

A CLI tool that provides audio cues to help maintain timing during workout exercises. It beeps at specified intervals to guide you through repetitions, acting as a "metronome" for your workouts.

## Features

- **Precise Timing**: Alternating intervals for concentric and eccentric phases (e.g., 2-second lift, 4-second lower)
- **Audio Cues**: Distinct beeps for start, end, and rep intervals
- **Countdown Timer**: Optional countdown before starting your exercise
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Quiet Mode**: Test timing without audio
- **Clean Interruption**: Graceful handling of Ctrl+C

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/aegirhall/workout-metronome.git
cd workout-metronome

# Build the binary
make build

# Install to GOPATH/bin
make install
```

### Using Go Install

```bash
go install github.com/aegirhall/workout-metronome/cmd/metronome@latest
```

## Usage

### Two Timing Modes

**Split Phase Mode** - Separate timing for concentric and eccentric phases:
```bash
# 12 reps with 2-second concentric, 4-second eccentric
metronome --reps 12 --concentric 2 --eccentric 4

# Short form
metronome -r 12 -c 2 -e 4
```

**Whole Rep Mode** - Single timer for entire rep:
```bash
# 10 reps with 5 seconds per rep total
metronome --reps 10 --total 5

# Short form
metronome -r 10 -t 5
```

**Note**: The `-t/--total` and `-c/--concentric`/`-e/--eccentric` flags are mutually exclusive. Use either `-t` alone, or both `-c` and `-e` together.

### Custom Countdown

```bash
# 3-second countdown instead of default 5 seconds
metronome -r 12 -c 2 -e 4 --countdown 3

# No countdown
metronome -r 12 -c 2 -e 4 --no-countdown
```

### Quiet Mode

```bash
# Test timing without audio (display only)
metronome -r 12 -c 2 -e 4 --quiet
```

### Decimal Durations

```bash
# 1.5-second concentric, 3.5-second eccentric
metronome -r 10 -c 1.5 -e 3.5

# 4.5 seconds per rep
metronome -r 10 -t 4.5
```

## Example Workouts

### Bench Press (Slow Eccentric)
```bash
# 8 reps: 2s up, 4s down
metronome -r 8 -c 2 -e 4
```

### Squats (Explosive)
```bash
# 12 reps: 1s up, 3s down
metronome -r 12 -c 1 -e 3
```

### Pull-ups (Tempo Training)
```bash
# 5 reps: 1s up, 2s hold, 3s down
# Note: For hold phases, use separate sets or time manually
metronome -r 5 -c 1 -e 3
```

### Planks (Interval Training)
```bash
# 10 intervals: 30s work, 15s rest
metronome -r 10 -c 30 -e 15 --no-countdown
```

### Push-ups (Simple Rep Counting)
```bash
# 20 reps: 3 seconds per rep
metronome -r 20 -t 3
```

### Burpees (Timed Reps)
```bash
# 15 reps: 6 seconds per rep
metronome -r 15 -t 6 --countdown 3
```

## Output Format

### Split Phase Mode
The output displays progressively, with checkmarks appearing as each phase completes (synchronized with the beep sounds):

```
Countdown: 5...4...3...2...1...START!
Rep 1: Concentric (2s) ✓ -> Eccentric (4s) ✓
Rep 2: Concentric (2s) ✓ -> Eccentric (4s) ✓
Rep 3: Concentric (2s) ✓ -> Eccentric (4s) ✓
...
Rep 12: Concentric (2s) ✓ -> Eccentric (4s) ✓

WORKOUT COMPLETE! Total time: 1m 17s
```

Each line builds progressively:
1. Start beep → `Rep 1: Concentric (2s) ` appears
2. After 2s, beep → ` ✓ -> Eccentric (4s) ` appears
3. After 4s, beep → ` ✓` appears and moves to next line

### Whole Rep Mode
When using `-t/--total`, each rep is shown as a single unit:

```
Countdown: 3...2...1...START!
Rep 1 (5s) ✓
Rep 2 (5s) ✓
Rep 3 (5s) ✓
...
Rep 10 (5s) ✓

WORKOUT COMPLETE! Total time: 53s
```

Each line builds progressively:
1. Start beep → `Rep 1 (5s) ` appears
2. After 5s, beep → ` ✓` appears and moves to next line

## Audio Cues

The metronome uses professional workout timer sounds embedded in the binary:

- **Countdown Beeps**: Single block sound plays every second during countdown
- **Start Beep**: Distinct beep sound at workout start
- **Rep Beeps**: Single block sound at each phase transition (concentric/eccentric)
- **End Beep**: Same distinct beep sound at workout completion

Audio files are embedded in the binary, so no external sound files are needed.

## Command-Line Options

```
Usage:
  metronome [flags]

Flags:
  -r, --reps int             Number of repetitions (required)

  Timing modes (choose one):
  -t, --total float          Total rep duration in seconds (use alone)
  -c, --concentric float     Concentric phase duration in seconds (use with -e)
  -e, --eccentric float      Eccentric phase duration in seconds (use with -c)

  Optional:
      --countdown float      Countdown duration in seconds (default 5)
      --no-countdown         Skip the countdown phase
  -q, --quiet                Disable audio output (display only)
      --beep-volume int      Beep volume 0-100 (default 80)
  -h, --help                 Help for metronome
```

**Note**: Either use `-t/--total` alone for whole rep timing, or use both `-c/--concentric` and `-e/--eccentric` together for split phase timing. These modes are mutually exclusive.

## Development

### Building

```bash
# Build for current platform
make build

# Cross-compile for all platforms
make cross-compile

# Build for specific platform
make build-darwin
make build-linux
make build-windows
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Project Structure

```
workout-metronome/
├── cmd/metronome/          # CLI entry point
├── pkg/
│   ├── audio/             # Audio playback and MP3 decoding
│   │   └── sounds/        # Embedded MP3 audio files
│   ├── timer/             # Core timing engine
│   └── metronome/         # Orchestration layer
├── internal/cli/          # CLI-specific code
├── Makefile               # Build targets
└── README.md
```

## Architecture

The application is designed with modularity in mind for future UI extensions:

- **pkg/timer**: Pure timing logic, emits events via channels
- **pkg/audio**: Audio abstraction with platform-agnostic interface, uses embedded MP3 files decoded at runtime
- **pkg/metronome**: Core orchestration with Observer pattern
- **internal/cli**: CLI adapter (easily replaceable with web/mobile UI)

This architecture allows the core logic to be reused for:
- Web UI (WebSocket events)
- Mobile apps (Swift/iOS, Kotlin/Android)
- Desktop apps (Electron, native)

## Requirements

- Go 1.24 or later
- Audio output device (for non-quiet mode)

## Platform Notes

### macOS
Audio works out of the box using CoreAudio.

### Linux
Requires ALSA or PulseAudio. Most modern Linux distributions have this installed by default.

### Windows
Audio works out of the box using DirectSound.

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Troubleshooting

### No audio output
- Verify your audio device is working
- Try running with `--quiet` flag to test timing without audio
- Check system volume settings

### Audio initialization fails
- Ensure no other application is exclusively using the audio device
- Try restarting your terminal/shell

### Timing seems off
- Beep generation and playback have minimal latency
- The application uses precise timing internally (±100ms accuracy)
- Verify with a stopwatch for the total workout duration
