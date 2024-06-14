package main

import (
	"fmt"
	"math"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mjibson/go-dsp/fft"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "freqshow",
		Short: "freqshow - a CLI that SHOWs FREQuencies ",
		Long: `freqshow is CLI that displays the frequencies of sounds in WAV files in 3 dimensions (frequency, amplitude, and time), so you can "see" the sound

This tool hasn't been tested in a professional or academic setting, so it likely has bugs. Please test your results against industry-trusted, battle-tested alternatives`,
	}

	var evalCmd = &cobra.Command{
		Use:   "display",
		Short: "Display the provided WAV file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			displayWavFile(args[0])
		},
	}

	rootCmd.AddCommand(evalCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Whoops. There was an error while executing your CLI '%s'\n", err)
		os.Exit(1)
	}
}

func displayWavFile(wavFilePath string) {
	wavFile, err := os.Open(wavFilePath)
	exitOnError(err)
	defer wavFile.Close()

	decoder := wav.NewDecoder(wavFile)

	if !decoder.IsValidFile() {
		fmt.Println("Invalid WAV file")
		os.Exit(1)
	}

	spew.Config.DisableMethods = true

	samplesPerBlock := int(decoder.SampleRate) / 60

	pcmBuffer := audio.IntBuffer{Data: make([]int, samplesPerBlock)}
	n, err := decoder.PCMBuffer(&pcmBuffer)
	fmt.Printf("Tried to get %d samples, acquired %d", samplesPerBlock, n)
	exitOnError(err)

	var leftPCMData []float64
	var rightPCMData []float64
	for _, sample := range pcmBuffer.Data {
		leftPCMData = append(leftPCMData, float64(sample>>16)/math.MaxInt16)
		rightPCMData = append(rightPCMData, float64(sample&math.MaxInt16)/math.MaxInt16)
	}

	leftFFTResult := fft.FFTReal(leftPCMData)
	rightFFTResult := fft.FFTReal(rightPCMData)

	fmt.Printf("leftFFTResult: %d\n", len(leftFFTResult))
	fmt.Printf("rightFFTResult: %d\n", len(rightFFTResult))
}

func exitOnError(err error) {
	if err != nil {
		fmt.Printf("Uh oh, decoder threw an error: '%s'\n", err)
		os.Exit(1)
	}
}
