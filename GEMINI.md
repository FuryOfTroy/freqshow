# Gemini Workspace (`freqshow`)

## Project Overview

`freqshow` is a command-line Go application designed for audio frequency analysis and manipulation. It processes WAV files by performing Fast Fourier Transforms (FFTs) on the PCM data to visualize and modify audio frequencies. The core functionality is built to support a highly customizable and extensible equalization (EQ) framework.

The project is structured as a CLI tool using the `cobra` library. The main logic reads a WAV file in chunks, applies a Hann window to each chunk, performs an FFT, allows for modification of the frequency data, and then reconstructs the audio signal using an Inverse FFT (IFFT).

In addition to file-based processing, `freqshow` now supports **real-time equalization and playback**, allowing users to hear the EQ effects instantly. A graphical user interface (GUI) has also been added for easier interaction.

### Key Technologies
- **Language:** Go (version 1.22.2)
- **CLI Framework:** `github.com/spf13/cobra`
- **Audio Processing:** `github.com/go-audio/wav` for WAV file I/O and `github.com/mjibson/go-dsp/fft` for FFT calculations.
- **Real-time Audio Playback:** `github.com/ebitengine/oto/v3`
- **Graphical User Interface:** `github.com/webview/webview_go`

## Building and Running

### Prerequisites
- Go 1.22.2 or later must be installed.
- Project dependencies are managed with Go Modules.

### Running the application
To run the application, use `go run`.

#### EQ Command (File-based processing)
Applies equalization to a WAV file and saves the result to a new file.

```sh
# Download dependencies
go mod tidy

# Run the eq command
go run main.go eq --input-file <input.wav> --output-file [output.wav] --freq-start <start_freq> --freq-end <end_freq> --gain <gain_db>
```
- `--input-file <input.wav>`: The path to the source WAV file (required).
- `--output-file [output.wav]`: (Optional) The path for the modified output file. Defaults to `result.wav`.
- `--freq-start <start_freq>`: Starting frequency for the EQ band (Hz). Defaults to 20.0 Hz.
- `--freq-end <end_end>`: Ending frequency for the EQ band (Hz). Defaults to 20000.0 Hz.
- `--gain <gain_db>`: Gain to apply to the frequency band (in dB). Defaults to 0.0 dB.

#### Play Command (Real-time playback)
Plays a WAV file with real-time equalization applied.

```sh
go run main.go play --input-file <input.wav> --freq-start <start_freq> --freq-end <end_freq> --gain <gain_db>
```
- `--input-file <input.wav>`: The path to the source WAV file (required).
- `--freq-start <start_freq>`: Starting frequency for the EQ band (Hz). Defaults to 20.0 Hz.
- `--freq-end <end_freq>`: Ending frequency for the EQ band (Hz). Defaults to 20000.0 Hz.
- `--gain <gain_db>`: Gain to apply to the frequency band (in dB). Defaults to 0.0 dB.

#### GUI Command (Graphical Interface)
Launches a graphical user interface for interactive EQ application and real-time playback.
Requires the `gui` build tag.

```sh
# Run the GUI
go run -tags gui main.go gui
```
The GUI provides fields to enter input/output file paths, frequency range, and gain, along with "Play" and "Save" buttons.

### Building the application
To build a binary executable:
```sh
# Build CLI executable
go build .

# Build GUI executable (requires `gui` tag)
go build -tags gui .
```
This will create a `freqshow` (or `freqshow.exe`) executable in the project root.

```sh
# Run CLI eq command from binary
./freqshow eq --input-file <input.wav> --output-file [output.wav] --freq-start <start_freq> --freq-end <end_freq> --gain <gain_db>

# Run CLI play command from binary
./freqshow play --input-file <input.wav> --freq-start <start_freq> --freq-end <end_freq> --gain <gain_db>

# Run GUI from binary
./freqshow gui
```

### Testing
There are no test files in the project. This has proven challenging during development, highlighting the critical need for a robust test suite to verify the FFT and audio processing logic, especially given the subtle interactions between different components.
TODO: Add a test suite to verify the FFT and audio processing logic.

## Development Conventions

- **Code Style:** The code follows standard Go formatting (`gofmt`).
- **Security:** The presence of `.github/instructions/snyk_rules.instructions.md` suggests that security scanning with Snyk is an expected part of the development process. All new code should be scanned for vulnerabilities.
- **Dependencies:** Dependencies are managed via `go.mod`.
