# Gemini Workspace (`freqshow`)

## Project Overview

`freqshow` is a command-line Go application designed for audio frequency analysis and manipulation. It processes WAV files by performing Fast Fourier Transforms (FFTs) on the PCM data to visualize and modify audio frequencies. The core functionality is built to support a highly customizable and extensible equalization (EQ) framework.

The project is structured as a CLI tool using the `cobra` library. The main logic reads a WAV file in chunks, applies a Hann window to each chunk, performs an FFT, allows for modification of the frequency data, and then reconstructs the audio signal using an Inverse FFT (IFFT).

### Key Technologies
- **Language:** Go (version 1.22.2)
- **CLI Framework:** `github.com/spf13/cobra`
- **Audio Processing:** `github.com/go-audio/wav` for WAV file I/O and `github.com/mjibson/go-dsp/fft` for FFT calculations.

## Building and Running

### Prerequisites
- Go 1.22.2 or later must be installed.
- Project dependencies are managed with Go Modules.

### Running the application
To run the application, use `go run`. The primary command is `modify`, which processes a WAV file.

```sh
# Download dependencies
go mod tidy

# Run the modify command
go run main.go modify <input.wav> [output.wav]
```
- `<input.wav>`: The path to the source WAV file.
- `[output.wav]`: (Optional) The path for the modified output file. Defaults to `result.wav`.

### Building the application
To build a binary executable:
```sh
go build .
```
This will create a `freqshow` (or `freqshow.exe`) executable in the project root.

```sh
./freqshow modify <input.wav> [output.wav]
```

### Testing
There are no test files in the project.
TODO: Add a test suite to verify the FFT and audio processing logic.

## Development Conventions

- **Code Style:** The code follows standard Go formatting (`gofmt`).
- **Security:** The presence of `.github/instructions/snyk_rules.instructions.md` suggests that security scanning with Snyk is an expected part of the development process. All new code should be scanned for vulnerabilities.
- **Dependencies:** Dependencies are managed via `go.mod`.
