package main

import (
	"fmt"
	"log"
	"math"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/spf13/cobra"
)




var rootCmd = &cobra.Command{
	Use:   "freqshow",
	Short: "freqshow - a CLI that SHOWs FREQuencies ",
	Long: `freqshow is CLI that displays the frequencies of sounds in WAV files in 3 dimensions (frequency, amplitude, and time), so you can "see" the sound

This tool hasn't been tested in a professional or academic setting, so it likely has bugs. Please test your results against industry-trusted, battle-tested alternatives`,
}

func main() {

	var (
		freqStart      float64
		freqEnd        float64
		gain           float64
		inputFilePath  string
		outputFilePath string
	)

	var eqCmd = &cobra.Command{
		Use:   "eq",
		Short: "Apply equalization (gain) to a specified frequency range in the WAV file",
		Run: func(cmd *cobra.Command, args []string) {
			cmdEq(inputFilePath, outputFilePath, freqStart, freqEnd, gain)
		},
	}

	eqCmd.Flags().StringVarP(&inputFilePath, "input-file", "i", "", "Path to the source WAV file (required)")
	eqCmd.Flags().StringVarP(&outputFilePath, "output-file", "o", "result.wav", "Path for the modified output WAV file")
	eqCmd.Flags().Float64Var(&freqStart, "freq-start", 20.0, "Starting frequency for the EQ band (Hz)")
	eqCmd.Flags().Float64Var(&freqEnd, "freq-end", 20000.0, "Ending frequency for the EQ band (Hz)")
	eqCmd.Flags().Float64Var(&gain, "gain", 0.0, "Gain to apply to the frequency band (in dB)")

	eqCmd.MarkFlagRequired("input-file")

	rootCmd.AddCommand(eqCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Whoops. There was an error while executing your CLI '%s'\n", err)
		os.Exit(1)
	}
}

func cmdEq(inputFilePath, outputFilePath string, freqStart, freqEnd, gain float64) {

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

	outputBuffer := make([][]float64, decoder.NumChans)

	stepSize := ChunkSize - Overlap

	for ch := 0; ch < int(decoder.NumChans); ch++ {
		log.Printf("Processing channel %d of %d\n", ch+1, decoder.NumChans)
		outputBuffer[ch] = make([]float64, numSamples+ChunkSize) // Initialize per-channel output buffer with enough space

		// This loop processes the input 'channelsData' in chunks
		for i := 0; i < numSamples; i += stepSize {
			if i > 0 && i%(numSamples/10) == 0 {
				log.Printf("  ...channel %d progress: %d%%", ch+1, int(float64(i)/float64(numSamples)*100))
			}

			// Ensure chunk does not go out of bounds of channelsData
			end := i + ChunkSize
			if end > len(channelsData[ch]) {
				end = len(channelsData[ch])
			}
			chunk := channelsData[ch][i:end]

			// Pad chunk if smaller than chunkSize for FFT
			if len(chunk) < ChunkSize {
				paddedChunk := make([]float64, ChunkSize)
				copy(paddedChunk, chunk)
				chunk = paddedChunk
			}

			windowedChunk := ApplyHannWindow(chunk)
			fftData := PerformFFT(windowedChunk)
			ApplyEQ(fftData, int(decoder.SampleRate), ChunkSize, freqStart, freqEnd, gain)
			ifftResult := PerformIFFT(fftData)

			// Overlap-add implementation:
			// Add the IFFT result to the output buffer, summing in overlapping regions
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
			// Read from outputBuffer[ch] instead of processedChannelsData[ch]
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
