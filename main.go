package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/spf13/cobra"
)

const (
	chunkSize       = 1024
	overlap         = 512
	sampleRate      = 44100
	numChannels     = 2
	cutoffFrequency = 1000.0
)

func applyHannWindow(data []float64) []float64 {
	N := len(data)
	windowedData := make([]float64, N)
	for n := 0; n < N; n++ {
		windowedData[n] = data[n] * 0.5 * (1 - math.Cos(2*math.Pi*float64(n)/float64(N-1)))
	}
	return windowedData
}

func performFFT(pcmData []float64) []complex128 {
	return fft.FFTReal(pcmData)
}

func performIFFT(fftData []complex128) []float64 {
	ifftResult := fft.IFFT(fftData)
	pcmData := make([]float64, len(ifftResult))
	for i, value := range ifftResult {
		pcmData[i] = real(value)
	}
	return pcmData
}

func modifyFFT(fftData []complex128, sampleRate int, cutoffFrequency float64) {
	for i := range fftData {
		freq := float64(i) * float64(sampleRate) / float64(len(fftData))
		if freq > cutoffFrequency {
			fftData[i] = 0
		}
	}
}

func chunkData(data []float64, chunkSize, overlap int) [][]float64 {
	var chunks [][]float64
	stepSize := chunkSize - overlap
	for start := 0; start < len(data)-chunkSize; start += stepSize {
		chunk := data[start : start+chunkSize]
		chunks = append(chunks, chunk)
	}
	return chunks
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "freqshow",
		Short: "freqshow - a CLI that SHOWs FREQuencies ",
		Long: `freqshow is CLI that displays the frequencies of sounds in WAV files in 3 dimensions (frequency, amplitude, and time), so you can "see" the sound

This tool hasn't been tested in a professional or academic setting, so it likely has bugs. Please test your results against industry-trusted, battle-tested alternatives`,
	}

	var modifyCmd = &cobra.Command{
		Use:   "modify",
		Short: "Modify the provided WAV file",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			cmdModify(args)
		},
	}

	rootCmd.AddCommand(modifyCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Whoops. There was an error while executing your CLI '%s'\n", err)
		os.Exit(1)
	}
}

func cmdModify(args []string) {
	inputFilePath := args[0]
	outputFilePath := "result.wav"
	if len(args) == 2 {
		outputFilePath = args[1]
	}

	log.Printf(`
{
	Input file: %s
	Output file: %s
}
`, inputFilePath, outputFilePath)

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		log.Fatalf("Failed to open WAV file: %v", err)
	}
	defer inputFile.Close()

	decoder := wav.NewDecoder(inputFile)
	if !decoder.IsValidFile() {
		log.Fatalf("Invalid WAV file")
	}

	decoder.ReadMetadata()
	decoder.Rewind()
	log.Printf(`
{
	PCM size: %d
	Sample rate: %d
	Bit depth: %d
	Channels: %d
}
`, decoder.PCMSize, decoder.SampleRate, decoder.BitDepth, decoder.NumChans)

	buf, err := decoder.FullPCMBuffer()
	if err != nil {
		log.Fatalf("Failed to read PCM data: %v", err)
	}
	allPCMData := buf.Data

	if int(decoder.NumChans) == 0 {
		log.Fatalf("Number of channels cannot be zero")
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

	processedChannelsData := make([][]float64, decoder.NumChans)
	for ch := 0; ch < int(decoder.NumChans); ch++ {
		log.Printf("Processing channel %d of %d\n", ch+1, decoder.NumChans)
		processedChannelsData[ch] = make([]float64, 0, numSamples)

		for i := 0; i < numSamples; i += chunkSize {
			if i > 0 && i%(numSamples/10) == 0 {
				log.Printf("  ...channel %d progress: %d%%", ch+1, int(float64(i)/float64(numSamples)*100))
			}

			end := i + chunkSize
			if end > numSamples {
				end = numSamples
			}
			chunk := channelsData[ch][i:end]

			if len(chunk) < chunkSize {
				paddedChunk := make([]float64, chunkSize)
				copy(paddedChunk, chunk)
				chunk = paddedChunk
			}

			windowedChunk := applyHannWindow(chunk)
			fftData := performFFT(windowedChunk)
			modifyFFT(fftData, int(decoder.SampleRate), cutoffFrequency)
			ifftResult := performIFFT(fftData)

			processedChannelsData[ch] = append(processedChannelsData[ch], ifftResult[:len(chunk)]...)
		}
	}

	outputIntBufferData := make([]int, numSamples*int(decoder.NumChans))
	for i := 0; i < numSamples; i++ {
		for ch := 0; ch < int(decoder.NumChans); ch++ {
			sample := processedChannelsData[ch][i] * math.MaxInt16
			if sample > math.MaxInt16 {
				sample = math.MaxInt16
			} else if sample < math.MinInt16 {
				sample = math.MinInt16
			}
			outputIntBufferData[i*int(decoder.NumChans)+ch] = int(sample)
		}
	}

	log.Println("Saving result...")
	saveWav(outputFilePath, outputIntBufferData, decoder)
}

// Function to save PCM data to a new WAV file
func saveWav(filePath string, pcmData []int, decoder *wav.Decoder) error {
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
