package main

import (
	"math"
	"testing"
)

// Helper function to compare float64 slices with a tolerance
func compareFloat64Slices(t *testing.T, got, want []float64, tolerance float64, msg string) {
	if len(got) != len(want) {
		t.Fatalf("%s: Mismatched slice lengths. Got %d, want %d", msg, len(got), len(want))
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > tolerance {
			t.Errorf("%s: Mismatched value at index %d. Got %f, want %f (tolerance %f)", msg, i, got[i], want[i], tolerance)
		}
	}
}

// Helper function to compare complex128 slices with a tolerance
func compareComplex128Slices(t *testing.T, got, want []complex128, tolerance float64, msg string) {
	if len(got) != len(want) {
		t.Fatalf("%s: Mismatched slice lengths. Got %d, want %d", msg, len(got), len(want))
	}
	for i := range got {
		if math.Abs(real(got[i])-real(want[i])) > tolerance || math.Abs(imag(got[i])-imag(want[i])) > tolerance {
			t.Errorf("%s: Mismatched value at index %d. Got %v, want %v (tolerance %f)", msg, i, got[i], want[i], tolerance)
		}
	}
}

func TestApplyHannWindow(t *testing.T) {
	// Test case 1: Empty slice
	input1 := []float64{}
	expected1 := []float64{}
	got1 := ApplyHannWindow(input1)
	compareFloat64Slices(t, got1, expected1, 1e-9, "Empty slice test")

	// Test case 2: Single element slice
	input2 := []float64{1.0}
	expected2 := []float64{0.0} // Hann window for single element is 0
	got2 := ApplyHannWindow(input2)
	compareFloat64Slices(t, got2, expected2, 1e-9, "Single element slice test")

	// Test case 3: Two elements slice
	input3 := []float64{1.0, 1.0}
	expected3 := []float64{0.0, 0.0} // Hann window for two elements
	got3 := ApplyHannWindow(input3)
	compareFloat64Slices(t, got3, expected3, 1e-9, "Two elements slice test")

	// Test case 4: Typical case (e.g., 4 elements)
	input4 := []float64{1.0, 2.0, 3.0, 4.0}
	// Expected Hann window values for N=4: w[n] = 0.5 * (1 - cos(2*pi*n/(N-1)))
	// n=0: 0.5 * (1 - cos(0)) = 0.0
	// n=1: 0.5 * (1 - cos(2*pi/3)) = 0.5 * (1 - (-0.5)) = 0.75
	// n=2: 0.5 * (1 - cos(4*pi/3)) = 0.5 * (1 - (-0.5)) = 0.75
	// n=3: 0.5 * (1 - cos(6*pi/3)) = 0.5 * (1 - 1) = 0.0
	expected4 := []float64{0.0, 1.5, 2.25, 0.0} // input * window
	got4 := ApplyHannWindow(input4)
	compareFloat64Slices(t, got4, expected4, 1e-9, "Four elements slice test")
}

func TestPerformFFT(t *testing.T) {
	// Test case 1: Empty slice
	input1 := []float64{}
	expected1 := []complex128{}
	got1 := PerformFFT(input1)
	compareComplex128Slices(t, got1, expected1, 1e-9, "FFT Empty slice test")

	// Test case 2: DC signal (constant value)
	input2 := []float64{1.0, 1.0, 1.0, 1.0} // N=4
	// FFT of [1, 1, 1, 1] is [4, 0, 0, 0]
	expected2 := []complex128{complex(4.0, 0.0), complex(0.0, 0.0), complex(0.0, 0.0), complex(0.0, 0.0)}
	got2 := PerformFFT(input2)
	compareComplex128Slices(t, got2, expected2, 1e-9, "FFT DC signal test")

	// Test case 3: Impulse at the beginning
	input3 := []float64{1.0, 0.0, 0.0, 0.0} // N=4
	// FFT of [1, 0, 0, 0] is [1, 1, 1, 1]
	expected3 := []complex128{complex(1.0, 0.0), complex(1.0, 0.0), complex(1.0, 0.0), complex(1.0, 0.0)}
	got3 := PerformFFT(input3)
	compareComplex128Slices(t, got3, expected3, 1e-9, "FFT Impulse test")

	// Test case 4: Simple sine wave (e.g., 2 samples per cycle, N=4)
	// sin(2*pi*n*k/N)
	// For k=1, N=4: sin(pi*n/2) -> n=0:0, n=1:1, n=2:0, n=3:-1
	input4 := []float64{0.0, 1.0, 0.0, -1.0} // Represents a sine wave
	// Expected FFT for a real sine wave:
	// X[1] should have positive imaginary part, X[N-1] (X[3]) should have negative imaginary part
	// X[0] and X[2] should be zero
	// The magnitude at k=1 and k=3 should be N/2 (2 in this case)
	// For [0, 1, 0, -1]:
	// X[0] = 0
	// X[1] = 0 - 2i
	// X[2] = 0
	// X[3] = 0 + 2i
	// Note: go-dsp's FFT output might be scaled differently or use different sign for imaginary part.
	// We'll test against the expected complex values based on common FFT implementations.
	expected4 := []complex128{complex(0.0, 0.0), complex(0.0, -2.0), complex(0.0, 0.0), complex(0.0, 2.0)}
	got4 := PerformFFT(input4)
	compareComplex128Slices(t, got4, expected4, 1e-9, "FFT Sine wave test")
}

func TestPerformIFFT(t *testing.T) {
	// Test case 1: Empty slice
	input1 := []complex128{}
	expected1 := []float64{}
	got1 := PerformIFFT(input1)
	compareFloat64Slices(t, got1, expected1, 1e-9, "IFFT Empty slice test")

	// Test case 2: IFFT of a DC signal's FFT
	fftInput2 := []complex128{complex(4.0, 0.0), complex(0.0, 0.0), complex(0.0, 0.0), complex(0.0, 0.0)}
	expected2 := []float64{1.0, 1.0, 1.0, 1.0}
	got2 := PerformIFFT(fftInput2)
	compareFloat64Slices(t, got2, expected2, 1e-9, "IFFT DC signal test")

	// Test case 3: IFFT of an Impulse signal's FFT
	fftInput3 := []complex128{complex(1.0, 0.0), complex(1.0, 0.0), complex(1.0, 0.0), complex(1.0, 0.0)}
	expected3 := []float64{1.0, 0.0, 0.0, 0.0}
	got3 := PerformIFFT(fftInput3)
	compareFloat64Slices(t, got3, expected3, 1e-9, "IFFT Impulse test")

	// Test case 4: IFFT of a sine wave's FFT
	fftInput4 := []complex128{complex(0.0, 0.0), complex(0.0, -2.0), complex(0.0, 0.0), complex(0.0, 2.0)}
	expected4 := []float64{0.0, 1.0, 0.0, -1.0}
	got4 := PerformIFFT(fftInput4)
	compareFloat64Slices(t, got4, expected4, 1e-9, "IFFT Sine wave test")
}

func TestApplyEQ(t *testing.T) {
	sampleRate := 44100
	numSamples := 1024 // Corresponds to the FFT size

	// Test case 1: No gain, no frequency range (should not change anything)
	fftData1 := make([]complex128, numSamples)
	for i := range fftData1 {
		fftData1[i] = complex(float64(i), float64(i*2))
	}
	originalFFTData1 := make([]complex128, numSamples)
	copy(originalFFTData1, fftData1)

	ApplyEQ(fftData1, sampleRate, numSamples, 0.0, float64(sampleRate/2), 0.0) // 0dB gain
	compareComplex128Slices(t, fftData1, originalFFTData1, 1e-9, "ApplyEQ No gain test")

	// Test case 2: Positive gain in a specific frequency range
	fftData2 := make([]complex128, numSamples)
	// Create some dummy data. Let's make one bin strong in the target range.
	// For N=1024, SampleRate=44100, binFreq = k * SampleRate / N
	// Let's target 1000 Hz. k = 1000 * 1024 / 44100 = ~23
	targetBin := 23
	fftData2[targetBin] = complex(10.0, 10.0)
	fftData2[numSamples-targetBin] = complex(10.0, -10.0) // Symmetric for real input

	originalFFTData2 := make([]complex128, numSamples)
	copy(originalFFTData2, fftData2)

	freqStart := 900.0
	freqEnd := 1100.0
	gainDB := 6.0 // 6dB gain means amplitude doubles (2x)
	gainLinear := math.Pow(10, gainDB/20)

	ApplyEQ(fftData2, sampleRate, numSamples, freqStart, freqEnd, gainDB)

	// Check target bin
	expectedTargetBin := originalFFTData2[targetBin] * complex(gainLinear, 0.0)
	if math.Abs(real(fftData2[targetBin])-real(expectedTargetBin)) > 1e-9 ||
		math.Abs(imag(fftData2[targetBin])-imag(expectedTargetBin)) > 1e-9 {
		t.Errorf("ApplyEQ Positive gain: Mismatched target bin. Got %v, want %v", fftData2[targetBin], expectedTargetBin)
	}

	// Check non-target bin (should be unchanged)
	nonTargetBin := targetBin + 1
	if nonTargetBin >= numSamples {
		nonTargetBin = targetBin - 1
	}
	compareComplex128Slices(t, []complex128{fftData2[nonTargetBin]}, []complex128{originalFFTData2[nonTargetBin]}, 1e-9, "ApplyEQ non-target bin should be unchanged")

	// Test case 3: Negative gain in a specific frequency range
	fftData3 := make([]complex128, numSamples)
	fftData3[targetBin] = complex(10.0, 10.0)
	fftData3[numSamples-targetBin] = complex(10.0, -10.0)

	originalFFTData3 := make([]complex128, numSamples)
	copy(originalFFTData3, fftData3)

	gainDB_neg := -6.0 // -6dB gain means amplitude halves (0.5x)
	gainLinear_neg := math.Pow(10, gainDB_neg/20)

	ApplyEQ(fftData3, sampleRate, numSamples, freqStart, freqEnd, gainDB_neg)

	// Check target bin
	expectedTargetBin_neg := originalFFTData3[targetBin] * complex(gainLinear_neg, 0.0)
	if math.Abs(real(fftData3[targetBin])-real(expectedTargetBin_neg)) > 1e-9 ||
		math.Abs(imag(fftData3[targetBin])-imag(expectedTargetBin_neg)) > 1e-9 {
		t.Errorf("ApplyEQ Negative gain: Mismatched target bin. Got %v, want %v", fftData3[targetBin], expectedTargetBin_neg)
	}

	// Test case 4: Edge case - freqStart > freqEnd
	fftData4 := make([]complex128, numSamples)
	fftData4[targetBin] = complex(10.0, 10.0)
	originalFFTData4 := make([]complex128, numSamples)
	copy(originalFFTData4, fftData4)

	ApplyEQ(fftData4, sampleRate, numSamples, 2000.0, 1000.0, 6.0) // Invalid range
	compareComplex128Slices(t, fftData4, originalFFTData4, 1e-9, "ApplyEQ Invalid frequency range test")
}
