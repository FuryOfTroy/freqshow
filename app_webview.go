package main

import (
	"furyoftroy/freqshow/audio"
)

// App struct to hold methods callable from JavaScript
type App struct{}

// RunEQCommand processes a WAV file and applies equalization.
// This function will be callable from the JavaScript frontend.
func (a *App) RunEQCommand(inputFilePath, outputFilePath string, freqStart, freqEnd, gain float64) error {
	return audio.ApplyEqualization(inputFilePath, outputFilePath, freqStart, freqEnd, gain)
}
