package shazam

// Audio Fingerprinting Algorithm
//
// This package implements a Shazam-like audio fingerprinting system for identifying
// audio samples by creating compact, robust signatures.
//
// How Fingerprinting Works:
//
// 1. Peak Extraction:
//    - Spectrogram is analyzed to find frequency peaks (local maxima)
//    - Peaks represent distinctive frequency components at specific times
//
// 2. Target Zone Construction:
//    - For each anchor peak, create pairs with nearby peaks (within targetZoneSize=5)
//    - Each pair encodes: (anchor_frequency, target_frequency, time_delta)
//
// 3. Address Generation:
//    - Create a 32-bit hash address from the peak pair:
//      * Bits 23-31: Anchor frequency (9 bits, max 512 Hz bins)
//      * Bits 14-22: Target frequency (9 bits)
//      * Bits 0-13: Time delta in milliseconds (14 bits, max ~16 seconds)
//    - This creates a unique identifier for the acoustic pattern
//
// 4. Fingerprint Storage:
//    - Each fingerprint maps: address -> (anchor_time_ms, song_id)
//    - Multiple songs can share the same address (collision handling)
//
// Why This Works:
// - Time-frequency pairs are robust to noise and distortion
// - Relative timing between peaks is preserved even with tempo changes
// - Compact representation allows fast database lookups
// - Works well for identifying drone acoustic signatures with distinct rotor patterns
//
// The fingerprinting system is used internally by the drone detection system to
// identify and match audio patterns, though the primary classification uses the
// feature-based KNN approach in the drone package.

import (
	"song-recognition/models"
)

const (
	maxFreqBits    = 9
	maxDeltaBits   = 14
	targetZoneSize = 5
)

// Fingerprint generates fingerprints from a list of peaks and stores them in an array.
// Each fingerprint consists of an address and a couple.
// The address is a hash. The couple contains the anchor time and the song ID.
func Fingerprint(peaks []Peak, songID uint32) map[uint32]models.Couple {
	fingerprints := map[uint32]models.Couple{}

	for i, anchor := range peaks {
		for j := i + 1; j < len(peaks) && j <= i+targetZoneSize; j++ {
			target := peaks[j]

			address := createAddress(anchor, target)
			anchorTimeMs := uint32(anchor.Time * 1000)

			fingerprints[address] = models.Couple{anchorTimeMs, songID}
		}
	}

	return fingerprints
}

// createAddress generates a unique address for a pair of anchor and target points.
// The address is a 32-bit integer where certain bits represent the frequency of
// the anchor and target points, and other bits represent the time difference (delta time)
// between them. This function combines these components into a single address (a hash).
func createAddress(anchor, target Peak) uint32 {
	anchorFreq := int(real(anchor.Freq))
	targetFreq := int(real(target.Freq))
	deltaMs := uint32((target.Time - anchor.Time) * 1000)

	// Combine the frequency of the anchor, target, and delta time into a 32-bit address
	address := uint32(anchorFreq<<23) | uint32(targetFreq<<14) | deltaMs

	return address
}
