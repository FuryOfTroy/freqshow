package audio

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/go-audio/audio"
)

// EQStream implements io.Reader and io.Closer for real-time EQ processing.
type EQStream struct {
	decoder   PcmDecoder // Use the interface here
	closer    io.Closer  // The file or resource to close when we are done.
	freqStart float64
	freqEnd   float64
	gain      float64

	decoderEOF bool // New field to track if the underlying decoder has hit EOF

	// Internal state
	channelsData [][]float64
	overlapBuf   [][]float64
	outBuf       []byte
	pos          int
	intBuf       *audio.IntBuffer
}

// NewEQStream creates a new EQStream from a PcmDecoder and a closer (the file).
func NewEQStream(decoder PcmDecoder, closer io.Closer, freqStart float64, freqEnd float64, gain float64) *EQStream { // Removed streamID param
	numChans := int(decoder.NumChans())
	stepSize := ChunkSize - Overlap
	return &EQStream{
		decoder:    decoder,
		closer:     closer,
		freqStart:  freqStart,
		freqEnd:    freqEnd,
		gain:       gain,
		decoderEOF: false, // Initialize to false
		channelsData: make([][]float64, numChans),
		overlapBuf:   make([][]float64, numChans),
		outBuf:       make([]byte, 0),
		intBuf: &audio.IntBuffer{
			Format: &audio.Format{
				NumChannels: numChans,
				SampleRate:  int(decoder.SampleRate()),
			},
			Data: make([]int, stepSize*numChans), // Still read in stepSize chunks
		},
	}
}

// Close closes the underlying file resource.
func (s *EQStream) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func (s *EQStream) Read(p []byte) (n int, err error) {
	for { // Loop to ensure we either provide data or signal definitive EOF
		// 1. If we have processed output already, serve it.
		if s.pos < len(s.outBuf) {
			n = copy(p, s.outBuf[s.pos:])
			s.pos += n
			return n, nil
		}

		// 2. Our internal output buffer is empty, clear it and reset position.
		s.pos = 0
		s.outBuf = make([]byte, 0)

		// 3. Accumulate enough raw PCM data to process a ChunkSize or until underlying decoder EOFs.
		// This inner loop ensures s.channelsData gets enough samples for processing.
		for len(s.channelsData[0]) < ChunkSize && !s.decoderEOF {
			nRead, readErr := s.decoder.PCMBuffer(s.intBuf) // Reads up to stepSize samples

			if readErr != nil && readErr != io.EOF {
				s.Close()
				return 0, readErr
			}

			// If the decoder returned 0 bytes and no error, or explicitly io.EOF, treat it as EOF.
			if nRead == 0 && readErr == nil {
				s.decoderEOF = true
				break // Exit inner accumulation loop
			} else if readErr == io.EOF {
				s.decoderEOF = true
				break // Exit inner accumulation loop
			}

			if nRead > 0 {
				rawSamples := s.intBuf.Data[:nRead]
				numSamplesRead := nRead / int(s.decoder.NumChans())
				for ch := 0; ch < int(s.decoder.NumChans()); ch++ {
					newSamples := make([]float64, numSamplesRead)
					for i := 0; i < numSamplesRead; i++ {
						newSamples[i] = float64(rawSamples[i*int(s.decoder.NumChans())+ch]) / math.MaxInt16
					}
					s.channelsData[ch] = append(s.channelsData[ch], newSamples...)
				}
			}
		}

		// 4. Now, if we have enough data (or it's EOF and we have remaining data), process a chunk.
		// The condition len(s.channelsData[0]) > 0 is important for final partial chunks after decoderEOF
		if len(s.channelsData[0]) >= ChunkSize || (s.decoderEOF && len(s.channelsData[0]) > 0) {
			processedSamples := make([][]float64, s.decoder.NumChans())
			stepSize := ChunkSize - Overlap
			for ch := 0; ch < int(s.decoder.NumChans()); ch++ {
				chunk := s.channelsData[ch];
				// Pad chunk if it's the last, partial one due to EOF
				if len(chunk) < ChunkSize {
					padded := make([]float64, ChunkSize)
					copy(padded, chunk)
					chunk = padded
				} else { // Take a full chunk if available
					chunk = chunk[:ChunkSize]
				}

				windowedChunk := ApplyHannWindow(chunk); fftData := PerformFFT(windowedChunk);
				ApplyEQToFFT(fftData, int(s.decoder.SampleRate()), ChunkSize, s.freqStart, s.freqEnd, s.gain);
				ifftResult := PerformIFFT(fftData);
				if len(s.overlapBuf[ch]) == 0 { s.overlapBuf[ch] = make([]float64, ChunkSize) };
				for i := 0; i < ChunkSize; i++ { s.overlapBuf[ch][i] += ifftResult[i] };
				outCount := stepSize; if outCount > len(s.overlapBuf[ch]) { outCount = len(s.overlapBuf[ch]) };
				processedSamples[ch] = s.overlapBuf[ch][:outCount];
				newOverlap := make([]float64, ChunkSize);
				if len(s.overlapBuf[ch]) > stepSize { copy(newOverlap, s.overlapBuf[ch][stepSize:]) };
				s.overlapBuf[ch] = newOverlap;
				// Remove processed samples from channelsData - always remove stepSize if processed a full chunk,
				// otherwise remove whatever was available if it was a partial/padded chunk at EOF.
				if len(s.channelsData[ch]) >= stepSize {
					s.channelsData[ch] = s.channelsData[ch][stepSize:]
				} else { // It was a partial chunk, now consumed
					s.channelsData[ch] = []float64{}
				}
			}

			outSamplesCount := len(processedSamples[0])
			if outSamplesCount > 0 {
				s.outBuf = make([]byte, outSamplesCount*int(s.decoder.NumChans())*2)
				for i := 0; i < outSamplesCount; i++ {
					for ch := 0; ch < int(s.decoder.NumChans()); ch++ {
						sample := processedSamples[ch][i] * math.MaxInt16;
						if sample > math.MaxInt16 { sample = math.MaxInt16 } else if sample < math.MinInt16 { sample = math.MinInt16 };
						binary.LittleEndian.PutUint16(s.outBuf[(i*int(s.decoder.NumChans())+ch)*2:], uint16(int16(sample)))
					}
				}
			}
		}

		// 5. If we produced new output, return it.
		if s.pos < len(s.outBuf) {
			n = copy(p, s.outBuf[s.pos:])
			s.pos += n
			return n, nil
		}

		// 6. If we're truly at EOF (decoderEOF is true and no more output), signal EOF.
		if s.decoderEOF && len(s.outBuf) == 0 {
			s.Close()
			return 0, io.EOF
		}

		// 7. Otherwise, we didn't have enough data to fill p, so ask oto for another call.
		// This should only happen if decoder is not EOF and we haven't accumulated enough to process.
		return 0, nil
	}
}
