package main

import (
	"log"
	"os"
	"reflect"
	"time"

	"furyoftroy/freqshow/audio"
	"github.com/ebitengine/oto/v3"
	"github.com/go-audio/wav"
	"github.com/webview/webview_go"
)

// App struct now holds the webview instance to dispatch calls to the main thread.
type App struct {
	w webview.WebView

	// State is now only accessed on the main thread via Dispatch, so no mutex is needed.
	player *oto.Player
	otoCtx *oto.Context
	otoCtxOpts oto.NewContextOptions
}

// Shutdown gracefully stops any active audio. Must be called from the main thread.
func (a *App) Shutdown() {
	if a.player != nil {
		a.player.Close()
		a.player = nil
	}
}

// RunEQCommand dispatches the save operation to the main thread.
func (a *App) RunEQCommand(inputFilePath, outputFilePath string, freqStart, freqEnd, gain float64) error {
	a.w.Dispatch(func() {
		// Stop any active playback before saving.
		if a.player != nil {
			a.player.Close()
			a.player = nil
		}
		// Since this is dispatched, we can't easily return the error to JS.
		// We will log it instead.
		if err := audio.ApplyEqualization(inputFilePath, outputFilePath, freqStart, freqEnd, gain); err != nil {
			log.Printf("Error during EQ save: %v", err)
		} else {
			log.Printf("Successfully saved EQ file to %s", outputFilePath)
		}
	})
	return nil
}

// RunPlayCommand is called by JS and dispatches the actual work to the main thread.
func (a *App) RunPlayCommand(inputFilePath string, freqStart, freqEnd, gain float64) error {
	a.w.Dispatch(func() {
		a.runPlay(inputFilePath, freqStart, freqEnd, gain)
	})
	return nil
}

// runPlay contains the actual audio logic and MUST run on the main thread.
func (a *App) runPlay(inputFilePath string, freqStart, freqEnd, gain float64) {
	// 1. Stop any existing player.
	if a.player != nil {
		a.player.Close()
		a.player = nil
	}

	// 2. Open file and decoder.
	f, err := os.Open(inputFilePath)
	if err != nil {
		log.Printf("ERROR: failed to open WAV file: %v", err)
		return
	}
	originalDecoder := wav.NewDecoder(f)
	if !originalDecoder.IsValidFile() {
		f.Close()
		log.Printf("ERROR: invalid WAV file: %v", err)
		return
	}
	// Wrap the original decoder to satisfy our PcmDecoder interface
	decoder := &audio.WavDecoderWrapper{originalDecoder}


	// 3. Set up or reuse the audio context.
	newOpts := oto.NewContextOptions{
		SampleRate:   int(decoder.SampleRate()),
		ChannelCount: int(decoder.NumChans()),
		Format:       oto.FormatSignedInt16LE,
	}
	if a.otoCtx == nil || !reflect.DeepEqual(a.otoCtxOpts, newOpts) {
		ctx, ready, err := oto.NewContext(&newOpts)
		if err != nil {
			f.Close()
			log.Printf("ERROR: failed to create oto context: %v", err)
			return
		}
		<-ready
		a.otoCtx = ctx
		a.otoCtxOpts = newOpts
	}

	// 4. Ensure the context is running.
	if err := a.otoCtx.Resume(); err != nil {
		log.Printf("ERROR: failed to resume oto context: %v", err)
		// If we can't resume, the context is likely broken.
		// Let's try to recover by nil-ing it out, so it gets recreated next time.
		a.otoCtx = nil
		f.Close()
		return
	}

	// 5. Create the stream and a new player.
	stream := audio.NewEQStream(decoder, f, freqStart, freqEnd, gain)
	player := a.otoCtx.NewPlayer(stream)
	a.player = player

	log.Printf("Playing with EQ: %f - %f Hz, %f dB", freqStart, freqEnd, gain)
	player.Play()

	// 6. Monitor for completion in a background goroutine.
	go func() {
		for player.IsPlaying() {
			time.Sleep(100 * time.Millisecond)
		}
		// When done, dispatch the cleanup code back to the main thread.
		a.w.Dispatch(func() {
			// Only nil out the player if it's the one we started.
			if a.player == player {
				a.player = nil
				log.Println("Playback finished.")
			}
		})
	}()
}
