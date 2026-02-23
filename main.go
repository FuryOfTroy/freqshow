package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"furyoftroy/freqshow/audio"
	"github.com/ebitengine/oto/v3"
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

	var playCmd = &cobra.Command{
		Use:   "play",
		Short: "Play the WAV file with real-time EQ applied",
		Run: func(cmd *cobra.Command, args []string) {
			cmdPlay(inputFilePath, freqStart, freqEnd, gain)
		},
	}

	playCmd.Flags().StringVarP(&inputFilePath, "input-file", "i", "", "Path to the source WAV file (required)")
	playCmd.Flags().Float64Var(&freqStart, "freq-start", 20.0, "Starting frequency for the EQ band (Hz)")
	playCmd.Flags().Float64Var(&freqEnd, "freq-end", 20000.0, "Ending frequency for the EQ band (Hz)")
	playCmd.Flags().Float64Var(&gain, "gain", 0.0, "Gain to apply to the frequency band (in dB)")

	playCmd.MarkFlagRequired("input-file")

	rootCmd.AddCommand(eqCmd)
	rootCmd.AddCommand(playCmd)

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

func cmdPlay(inputFilePath string, freqStart, freqEnd, gain float64) {
	f, err := os.Open(inputFilePath)
	if err != nil {
		log.Fatalf("failed to open WAV file: %v", err)
	}
	defer f.Close()

	originalDecoder := wav.NewDecoder(f)
	if !originalDecoder.IsValidFile() {
		log.Fatalf("invalid WAV file")
	}
	
	// Wrap the original decoder to satisfy our PcmDecoder interface
	decoder := &audio.WavDecoderWrapper{originalDecoder}

	// Initialize oto context
	op := &oto.NewContextOptions{
		SampleRate:   int(decoder.SampleRate()),
		ChannelCount: int(decoder.NumChans()),
		Format:       oto.FormatSignedInt16LE,
	}

	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		log.Fatalf("Failed to create oto context: %v", err)
	}
	<-ready

	// The stream will be responsible for closing the file now.
	stream := audio.NewEQStream(decoder, f, freqStart, freqEnd, gain)
	player := ctx.NewPlayer(stream)

	log.Printf("Playing with EQ: %f - %f Hz, %f dB\n", freqStart, freqEnd, gain)
	player.Play()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}

	if err := player.Close(); err != nil {
		log.Printf("Error closing player: %v", err)
	}
}
