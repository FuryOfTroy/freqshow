package audio

import (
	"io"
	"math"
	"testing"

	"github.com/go-audio/audio"
)

// Helper function to create a simple sine wave for testing
func createSineWave(freq, amplitude float64, sampleRate, numSamples int) []float64 {
	data := make([]float64, numSamples)
	for i := 0; i < numSamples; i++ {
		data[i] = amplitude * math.Sin(2*math.Pi*freq*float64(i)/float64(sampleRate))
	}
	return data
}

// Test ApplyHannWindow
func TestApplyHannWindow(t *testing.T) {
	input := make([]float64, 4)
	for i := 0; i < 4; i++ {
		input[i] = 1.0 // Flat signal
	}
	
	// Recalculate expected more precisely
	N := 4
	expectedPrecise := make([]float64, N)
	for n := 0; n < N; n++ {
		expectedPrecise[n] = 1.0 * 0.5 * (1 - math.Cos(2*math.Pi*float64(n)/float64(N-1)))
	}

	output := ApplyHannWindow(input)

	if len(output) != len(expectedPrecise) {
		t.Fatalf("Expected output length %d, got %d", len(expectedPrecise), len(output))
	}
	for i := 0; i < len(output); i++ {
		if math.Abs(output[i]-expectedPrecise[i]) > 1e-9 {
			t.Errorf("Mismatch at index %d: expected %f, got %f", i, expectedPrecise[i], output[i])
		}
	}
}

// Test PerformFFT and PerformIFFT (sanity check with a simple sine wave)
func TestFFTAndIFFT(t *testing.T) {
	sampleRate := 44100
	freq := 440.0
	amplitude := 0.5
	numSamples := 1024 // A power of 2 for FFT efficiency

	// Create a sine wave
	sineWave := createSineWave(freq, amplitude, sampleRate, numSamples)

	// Perform FFT
	fftData := PerformFFT(sineWave)

	// Check if FFT output has expected length
	if len(fftData) != numSamples {
		t.Fatalf("FFT output length mismatch: expected %d, got %d", numSamples, len(fftData))
	}

	peakBin := int(freq * float64(numSamples) / float64(sampleRate))
	if peakBin >= numSamples/2 {
		peakBin = numSamples/2 - 1
	}

	magnitude := math.Sqrt(real(fftData[peakBin])*real(fftData[peakBin]) + imag(fftData[peakBin])*imag(fftData[peakBin]))
	if magnitude < 10.0 {
		t.Errorf("Expected significant magnitude at peak bin %d, got %f", peakBin, magnitude)
	}

	// Perform IFFT
	ifftResult := PerformIFFT(fftData)

	if len(ifftResult) != numSamples {
		t.Fatalf("IFFT output length mismatch: expected %d, got %d", numSamples, len(ifftResult))
	}

	tolerance := 0.1
	for i := 0; i < numSamples; i++ {
		if math.Abs(ifftResult[i]-sineWave[i]) > tolerance {
			t.Errorf("IFFT mismatch at index %d: original %f, IFFT %f, diff %f", i, sineWave[i], ifftResult[i], math.Abs(ifftResult[i]-sineWave[i]))
		}
	}
}

// Test ApplyEQToFFT
func TestApplyEQToFFT(t *testing.T) {
	sampleRate := 44100
	numSamples := 1024
	testFreq := 1000.0
	gainDb := 6.0
	gainLinear := math.Pow(10, gainDb/20.0)

	fftData := make([]complex128, numSamples)
	testBin := int(testFreq * float64(numSamples) / float64(sampleRate))
	if testBin >= numSamples {
		t.Fatalf("Test frequency %fHz is too high", testFreq)
	}
	originalMagnitude := 1.0
	fftData[testBin] = complex(originalMagnitude, 0)
	if testBin > 0 && testBin < numSamples/2 {
		fftData[numSamples-testBin] = complex(originalMagnitude, 0)
	}

	originalFftData := make([]complex128, numSamples)
	copy(originalFftData, fftData)

	ApplyEQToFFT(fftData, sampleRate, numSamples, testFreq-10, testFreq+10, gainDb)

	if math.Abs(real(fftData[testBin])-(originalMagnitude*gainLinear)) > 1e-9 {
		t.Errorf("Gain not applied correctly: expected %f, got %f", originalMagnitude*gainLinear, real(fftData[testBin]))
	}

	if testBin > 0 && math.Abs(real(fftData[testBin-1])-real(originalFftData[testBin-1])) > 1e-9 {
		t.Errorf("Unexpected change at bin %d", testBin-1)
	}
}

// Mock implementation of io.Closer for EQStream testing
type mockCloser struct {
	closed bool
}

func (mc *mockCloser) Close() error {
	mc.closed = true
	return nil
}

// Mock implementation of PcmDecoder for EQStream testing
type mockPcmDecoder struct {
	data       []int
	pos        int
	sampleRate uint32
	numChans   uint16
	bitDepth   uint16
	err        error
}

func (m *mockPcmDecoder) PCMBuffer(buf *audio.IntBuffer) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}

	toRead := len(buf.Data)
	if m.pos+toRead > len(m.data) {
		toRead = len(m.data) - m.pos
	}
	copy(buf.Data, m.data[m.pos:m.pos+toRead])
	m.pos += toRead
	return toRead, nil
}

func (m *mockPcmDecoder) SampleRate() uint32 { return m.sampleRate }
func (m *mockPcmDecoder) NumChans() uint16 { return m.numChans }
func (m *mockPcmDecoder) BitDepth() uint16 { return m.bitDepth }


// Test EQStream Read method
func TestEQStreamRead(t *testing.T) {
	sampleRate := uint32(44100)
	numChans := uint16(1)
	bitDepth := uint16(16)
	freqStart := 100.0
	freqEnd := 1000.0
	gain := 6.0

	stepSize := ChunkSize - Overlap // 512
	totalTestSamples := (stepSize * 3) // Enough for several reads

	testPCMData := make([]int, totalTestSamples*int(numChans))
	sineAmp := 10000.0
	sineFreq := 500.0
	for i := 0; i < totalTestSamples; i++ {
		testPCMData[i*int(numChans)] = int(sineAmp * math.Sin(2*math.Pi*sineFreq*float64(i)/float64(sampleRate)))
	}

	mockDec := &mockPcmDecoder{
		data:       testPCMData,
		sampleRate: sampleRate,
		numChans:   numChans,
		bitDepth:   bitDepth,
	}
	mockCls := &mockCloser{}

	stream := NewEQStream(mockDec, mockCls, freqStart, freqEnd, gain)

	outputBuf := make([]byte, stepSize*int(numChans)*2)

	// Read until EOF
	totalBytesRead := 0
	for {
		n, err := stream.Read(outputBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if n > 0 {
			totalBytesRead += n
		}
	}

	if !mockCls.closed {
		t.Error("mockCloser should have been closed at EOF")
	}
	if totalBytesRead == 0 {
		t.Error("Expected to read more than 0 bytes")
	}
}
