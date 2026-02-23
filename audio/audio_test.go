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

	// In a real sine wave, we expect peaks at the positive and negative frequencies.
	// This is a basic sanity check, not a full spectral analysis.
	// For 440Hz in 1024 samples at 44100Hz:
	// bin = freq * N / sampleRate = 440 * 1024 / 44100 ~= 10.18
	// So we expect energy around bin 10 or 11.
	peakBin := int(freq * float64(numSamples) / float64(sampleRate))
	if peakBin >= numSamples/2 {
		peakBin = numSamples/2 - 1 // Avoid out of bounds if freq is too high
	}

	// Very basic check: ensure some magnitude exists where we expect it
	magnitude := math.Sqrt(real(fftData[peakBin])*real(fftData[peakBin]) + imag(fftData[peakBin])*imag(fftData[peakBin]))
	if magnitude < 10.0 { // Arbitrary threshold
		t.Errorf("Expected significant magnitude at peak bin %d, got %f", peakBin, magnitude)
	}

	// Perform IFFT
	ifftResult := PerformIFFT(fftData)

	// Check IFFT output length
	if len(ifftResult) != numSamples {
		t.Fatalf("IFFT output length mismatch: expected %d, got %d", numSamples, len(ifftResult))
	}

	// Compare IFFT result with original sine wave (should be very close)
	// Due to floating point math and windowing effects (if any applied before FFT in a real scenario),
	// they won't be identical, so we check with a tolerance.
	tolerance := 0.1 // This tolerance might need adjustment based on FFT library specifics and Go's float64 precision.
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
	testFreq := 1000.0 // Frequency to apply gain
	gainDb := 6.0       // +6dB gain
	gainLinear := math.Pow(10, gainDb/20.0)

	// Create dummy FFT data: a single peak at testFreq
	fftData := make([]complex128, numSamples)
	testBin := int(testFreq * float64(numSamples) / float64(sampleRate))
	if testBin >= numSamples {
		t.Fatalf("Test frequency %fHz is too high for sample rate %d and numSamples %d", testFreq, sampleRate, numSamples)
	}
	originalMagnitude := 1.0
	fftData[testBin] = complex(originalMagnitude, 0)
	// Also mirror for real-valued signal FFT
	if testBin > 0 && testBin < numSamples/2 {
		fftData[numSamples-testBin] = complex(originalMagnitude, 0)
	}

	// Store a copy for comparison
	originalFftData := make([]complex128, numSamples)
	copy(originalFftData, fftData)

	// Apply EQ
	ApplyEQToFFT(fftData, sampleRate, numSamples, testFreq-10, testFreq+10, gainDb) // Apply gain around testFreq

	// Check if gain was applied correctly at testFreq
	if math.Abs(real(fftData[testBin])-(originalMagnitude*gainLinear)) > 1e-9 {
		t.Errorf("Gain not applied correctly at testFreq bin %d: expected %f, got %f", testBin, originalMagnitude*gainLinear, real(fftData[testBin]))
	}

	// Check if other frequencies were unaffected
	if testBin > 0 && math.Abs(real(fftData[testBin-1])-real(originalFftData[testBin-1])) > 1e-9 {
		t.Errorf("Unexpected change at bin %d: expected %f, got %f", testBin-1, real(originalFftData[testBin-1]), real(fftData[testBin-1]))
	}
	if testBin < numSamples-1 && math.Abs(real(fftData[testBin+1])-real(originalFftData[testBin+1])) > 1e-9 {
		t.Errorf("Unexpected change at bin %d: expected %f, got %f", testBin+1, real(originalFftData[testBin+1]), real(fftData[testBin+1]))
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
	err        error // Error to return on PCMBuffer call
}

func (m *mockPcmDecoder) PCMBuffer(buf *audio.IntBuffer) (n int, err error) {
	if m.err != nil {
		return 0, m.err
	}
	if m.pos >= len(m.data) {
		return 0, io.EOF // Simulate explicit io.EOF
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


// Test EQStream Read method (basic functionality and EOF handling)
func TestEQStreamRead(t *testing.T) {
	sampleRate := uint32(44100)
	numChans := uint16(1)
	bitDepth := uint16(16)
	freqStart := 100.0
	freqEnd := 1000.0
	gain := 6.0

	// Create dummy PCM data: 2 chunks + some overlap + less than a chunk for end
	// Total samples: (ChunkSize - Overlap) * 2 + (ChunkSize / 2) = 512 * 2 + 256 = 1280 samples for 1 channel
	stepSize := ChunkSize - Overlap // 512
	totalTestSamples := (stepSize * 2) + (ChunkSize / 2) // Enough for two full chunk processes + some remainder

	// Create some dummy audio data (e.g., sine wave)
	testPCMData := make([]int, totalTestSamples*int(numChans))
	sineAmp := 10000.0 // MaxInt16 is 32767
	sineFreq := 500.0 // Hz
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

	// buffer to read into from the EQStream
	outputBuf := make([]byte, stepSize*int(numChans)*2) // Request one stepSize worth of bytes

	// Read first chunk (should return first processed stepSize data)
	n, err := stream.Read(outputBuf)
	if err != nil {
		t.Fatalf("Read #1 failed: %v", err)
	}
	if n == 0 {
		t.Fatal("Read #1 returned 0 bytes")
	}
	if stream.decoderEOF {
		t.Error("decoderEOF should not be true after first read (not enough data exhausted yet)")
	}

	// Read second chunk (should continue processing)
	n, err = stream.Read(outputBuf)
	if err != nil {
		t.Fatalf("Read #2 failed: %v", err)
	}
	if n == 0 {
		t.Fatal("Read #2 returned 0 bytes")
	}
	// At this point, the underlying mock decoder should have been fully read.
	// So decoderEOF should be true.
	if !stream.decoderEOF {
		t.Error("decoderEOF should be true after second read as mock data should be exhausted")
	}

	// Read remaining chunks until EOF
	totalBytesRead := n
	for {
		n, err = stream.Read(outputBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read during EOF loop failed: %v", err)
		}
		if n == 0 {
			t.Fatal("Read returned 0 bytes unexpectedly during EOF loop")
		}
		totalBytesRead += n
	}

	if !stream.decoderEOF {
		t.Error("decoderEOF should be true at EOF")
	}
	if !mockCls.closed {
		t.Error("mockCloser should have been closed at EOF")
	}
	// Basic check that total bytes read is reasonable (not 0)
	if totalBytesRead == 0 {
		t.Error("Expected to read more than 0 bytes of processed audio")
	}
}
