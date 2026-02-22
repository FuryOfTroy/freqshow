package audio

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
	ChunkSize = 1024
	Overlap   = 512
)

// ApplyEqualization processes a WAV file and applies equalization to the specified frequency range.
func ApplyEqualization(inputFilePath, outputFilePath string, freqStart, freqEnd, gain float64) error {
	log.Printf(`Processing EQ:
	Input: %s
	Output: %s
	Freq: %f - %f Hz
	Gain: %f dB
`,
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
	log.Printf("PCM size: %d, Sample rate: %d, Bit depth: %d, Channels: %d\n",
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
	stepSize := ChunkSize - Overlap

	for ch := 0; ch < int(decoder.NumChans); ch++ {
		log.Printf("Processing channel %d of %d\n", ch+1, decoder.NumChans)
		outputBuffer[ch] = make([]float64, numSamples+ChunkSize)

		for i := 0; i < numSamples; i += stepSize {
			if i > 0 && i%(numSamples/10) == 0 {
				log.Printf("  ...channel %d progress: %d%%", ch+1, int(float64(i)/float64(numSamples)*100))
			}

			end := i + ChunkSize
			if end > len(channelsData[ch]) {
				end = len(channelsData[ch])
			}
			chunk := channelsData[ch][i:end]

			if len(chunk) < ChunkSize {
				paddedChunk := make([]float64, ChunkSize)
				copy(paddedChunk, chunk)
				chunk = paddedChunk
			}

			windowedChunk := ApplyHannWindow(chunk)
			fftData := PerformFFT(windowedChunk)
			ApplyEQToFFT(fftData, int(decoder.SampleRate), ChunkSize, freqStart, freqEnd, gain)
			ifftResult := PerformIFFT(fftData)

			for j := 0; j < len(ifftResult); j++ {
				if i+j < len(outputBuffer[ch]) {
					outputBuffer[ch][i+j] += ifftResult[j]
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
	return SaveWav(outputFilePath, outputIntBufferData, decoder)
}

// SaveWav saves PCM data to a new WAV file.
func SaveWav(filePath string, pcmData []int, decoder *wav.Decoder) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sampleRate := int(decoder.SampleRate)
	bitDepth := int(decoder.BitDepth)
	numChans := int(decoder.NumChans)

	encoder := wav.NewEncoder(file, sampleRate, bitDepth, numChans, 1)
	intBuffer := &audio.IntBuffer{
		Format: &audio.Format{SampleRate: sampleRate, NumChannels: numChans},
		Data:   pcmData,
	}

	if err := encoder.Write(intBuffer); err != nil {
		return err
	}
	return encoder.Close()
}

// ApplyHannWindow applies a Hann window to the given data.
func ApplyHannWindow(data []float64) []float64 {
	N := len(data)
	windowedData := make([]float64, N)
	for n := 0; n < N; n++ {
		windowedData[n] = data[n] * 0.5 * (1 - math.Cos(2*math.Pi*float64(n)/float64(N-1)))
	}
	return windowedData
}

// PerformFFT performs a Fast Fourier Transform on the PCM data.
func PerformFFT(pcmData []float64) []complex128 {
	return fft.FFTReal(pcmData)
}

// PerformIFFT performs an Inverse Fast Fourier Transform on the FFT data.
func PerformIFFT(fftData []complex128) []float64 {
	if len(fftData) == 0 {
		return []float64{}
	}
	ifftResult := fft.IFFT(fftData)
	pcmData := make([]float64, len(ifftResult))
	for i, value := range ifftResult {
		pcmData[i] = real(value)
	}
	return pcmData
}

// ApplyEQToFFT applies equalization (gain) to the frequency data.
func ApplyEQToFFT(fftData []complex128, sampleRate, numSamples int, freqStart, freqEnd, gain float64) {
	gainLinear := math.Pow(10, gain/20.0)

	for i := range fftData {
		freq := float64(i) * float64(sampleRate) / float64(numSamples)
		if freq >= freqStart && freq <= freqEnd {
			fftData[i] *= complex(gainLinear, 0)
		}
	}
}
