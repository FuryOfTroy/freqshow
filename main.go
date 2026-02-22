package main

import (
	"fmt"
	"log"
	"os"

	"furyoftroy/freqshow/audio"
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
	if err := audio.ApplyEqualization(inputFilePath, outputFilePath, freqStart, freqEnd, gain); err != nil {
		log.Fatalf("Error applying EQ: %v", err)
	}
}
