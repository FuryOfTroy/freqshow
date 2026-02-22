//go:build gui

package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
)

const (
	chunkSize       = 1024
	overlap         = 512
	sampleRate      = 44100
	numChannels     = 2
	cutoffFrequency = 1000.0
)

// App struct to hold methods callable from JavaScript
type App struct{}

// RunEQCommand processes a WAV file and applies equalization.
// This function will be callable from the JavaScript frontend.
func (a *App) RunEQCommand(inputFilePath, outputFilePath string, freqStart, freqEnd, gain float64) error {
	log.Printf("{\n\tInput file: %s\n\tOutput file: %s\n\tFrequency Start: %f Hz\n\tFrequency End: %f Hz\n\tGain: %f dB\n}\n",
		inputFilePath, outputFilePath, freqStart, freqEnd, gain)

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return fmt.Errorf("failed to open WAV file: %w", err)
	}
	defer inputFile.Close()

	decoder := wav.NewDecoder(inputFile)
	if !decoder.IsValidFile() {
		return fmt.Errorf("invalid WAV file")
	}

	decoder.ReadMetadata()
	decoder.Rewind()
	log.Printf("{\n\tPCM size: %d\n\tSample rate: %d\n\tBit depth: %d\n\tChannels: %d\n}\n",
		decoder.PCMSize, decoder.SampleRate, decoder.BitDepth, decoder.NumChans)

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		return fmt.Errorf("failed to read PCM data: %w", err)
	}
	allPCMData := buf.Data

	if int(decoder.NumChans) == 0 {
		return fmt.Errorf("number of channels cannot be zero")
	}
	numSamples := len(allPCMData) / int(decoder.NumChans)

	channelsData := make([][]float64, decoder.NumChans)
	for ch := range channelsData {
		channelsData[ch] = make([]float64, numSamples)
	}

	for i := 0; i < numSamples; i++ {
		for ch := 0; ch < int(decoder.NumChans); ch++ {
			channelsData[ch][i] = float64(allPCMData[i*int(decoder.NumChans)+ch]) / math.MaxInt16
		}
	}

	outputBuffer := make([][]float64, decoder.NumChans)

	stepSize := chunkSize - overlap

	for ch := 0; ch < int(decoder.NumChans); ch++ {
		log.Printf("Processing channel %d of %d\n", ch+1, decoder.NumChans)
		outputBuffer[ch] = make([]float64, numSamples+chunkSize) // Initialize per-channel output buffer with enough space

		for i := 0; i < numSamples; i += stepSize {
			if i > 0 && i%(numSamples/10) == 0 {
				log.Printf("  ...channel %d progress: %d%%", ch+1, int(float64(i)/float64(numSamples)*100))
			}

			end := i + chunkSize
			if end > len(channelsData[ch]) {
				end = len(channelsData[ch])
			}
			chunk := channelsData[ch][i:end]

			if len(chunk) < chunkSize {
				paddedChunk := make([]float64, chunkSize)
				copy(paddedChunk, chunk)
				chunk = paddedChunk
			}

			windowedChunk := applyHannWindow(chunk)
			fftData := performFFT(windowedChunk)
			applyEQ(fftData, int(decoder.SampleRate), chunkSize, freqStart, freqEnd, gain)
			ifftResult := performIFFT(fftData)

			writePos := i
			for j := 0; j < len(ifftResult); j++ {
				if writePos+j < len(outputBuffer[ch]) {
					outputBuffer[ch][writePos+j] += ifftResult[j]
				}
			}
		}
	}

	outputIntBufferData := make([]int, numSamples*int(decoder.NumChans))
	for i := 0; i < numSamples; i++ {
		for ch := 0; ch < int(decoder.NumChans); ch++ {
			sample := outputBuffer[ch][i] * math.MaxInt16
			if sample > math.MaxInt16 {
				sample = math.MaxInt16
			} else if sample < math.MinInt16 {
				sample = math.MinInt16
			}
			outputIntBufferData[i*int(decoder.NumChans)+ch] = int(sample)
		}
	}

	log.Println("Saving result...")
	return a.saveWav(outputFilePath, outputIntBufferData, decoder)
}

// saveWav function is now a method of App
func (a *App) saveWav(filePath string, pcmData []int, decoder *wav.Decoder) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sampleRate := int(decoder.SampleRate)
	bitDepth := int(decoder.BitDepth)
	numChans := int(decoder.NumChans)

	encoder := wav.NewEncoder(file, sampleRate, bitDepth, numChans, 1) // Using 1 for audio.FormatPCM
	intBuffer := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: sampleRate, NumChannels: numChans},
		Data:   pcmData,
	}

	if err := encoder.Write(intBuffer); err != nil {
		return err
	}
	return encoder.Close()
}

// applyHannWindow applies a Hann window to the given data.
func applyHannWindow(data []float64) []float64 {
	N := len(data)
	windowedData := make([]float64, N)
	for n := 0; n < N; n++ {
		windowedData[n] = data[n] * 0.5 * (1 - math.Cos(2*math.Pi*float64(n)/float64(N-1)))
	}
	return windowedData
}

// performFFT performs a Fast Fourier Transform on the PCM data.
func performFFT(pcmData []float64) []complex128 {
	return fft.FFTReal(pcmData)
}

// performIFFT performs an Inverse Fast Fourier Transform on the FFT data.
func performIFFT(fftData []complex128) []float64 {
	ifftResult := fft.IFFT(fftData)
	pcmData := make([]float64, len(ifftResult))
	for i, value := range ifftResult {
		pcmData[i] = real(value)
	}
	return pcmData
}

// applyEQ applies equalization (gain) to the frequency data.
func applyEQ(fftData []complex128, sampleRate, numSamples int, freqStart, freqEnd, gain float64) {
	// Convert gain from dB to a linear amplitude multiplier
	gainLinear := math.Pow(10, gain/20.0)

	for i := range fftData {
		// Calculate the frequency for the current FFT bin
		freq := float64(i) * float64(sampleRate) / float64(numSamples)

		// Apply gain if the frequency is within the specified range
		if freq >= freqStart && freq <= freqEnd {
			fftData[i] *= complex(gainLinear, 0)
		}
	}
}

// chunkData splits the data into overlapping chunks.
func chunkData(data []float64, chunkSize, overlap int) [][]float64 {
	var chunks [][]float64
	stepSize := chunkSize - overlap
	for start := 0; start < len(data)-chunkSize; start += stepSize {
		chunk := data[start : start+chunkSize]
		chunks = append(chunks, chunk)
	}
	return chunks
}
